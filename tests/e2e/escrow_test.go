package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/a861252012/multi-chain-wallet/internal/wallet"
	"github.com/a861252012/multi-chain-wallet/internal/web"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/node"
)

const escrowPassword = "StrongEscrowTestPass123!"

type escrowE2E struct {
	t                      *testing.T
	sim                    *simulated.Backend
	mu                     sync.Mutex
	client                 *chain.Client
	tokenAddress, contract common.Address
	token, escrow          *bind.BoundContract
	escrowABI              abi.ABI
	services               [2]*wallet.Service
	servers                [2]*httptest.Server
	dirs                   [2]string
	addresses              [2]common.Address
	dropResponse           atomic.Bool
	sendCount              atomic.Int64
	finalizedHeader        atomic.Pointer[types.Header]
	failFinality           atomic.Bool
}

func newEscrowE2E(t *testing.T) *escrowE2E {
	t.Helper()
	f := &escrowE2E{t: t}
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	deployer, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(chain.SepoliaID))
	if err != nil {
		t.Fatal(err)
	}
	f.sim = simulated.NewBackend(types.GenesisAlloc{deployer.From: {Balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(21), nil)}}, func(_ *node.Config, cfg *ethconfig.Config) { cfg.Genesis.Config.ChainID = big.NewInt(chain.SepoliaID) })
	t.Cleanup(func() { f.sim.Close() })
	tokenAddr, tokenABI := deployContract(t, f.sim, deployer, filepath.Join("..", "..", "contracts", "artifacts", "EscrowTestToken.json"))
	contract, escrowABI := deployContract(t, f.sim, deployer, filepath.Join("..", "..", "contracts", "artifacts", "PaymentEscrow.json"), tokenAddr)
	f.tokenAddress, f.contract, f.escrowABI = tokenAddr, contract, escrowABI
	f.token = bind.NewBoundContract(tokenAddr, tokenABI, f.sim.Client(), f.sim.Client(), f.sim.Client())
	f.escrow = bind.NewBoundContract(contract, escrowABI, f.sim.Client(), f.sim.Client(), f.sim.Client())
	rpcServer := startSimulatedRPC(t, f.sim, &f.mu)
	t.Cleanup(rpcServer.Close)
	target, err := url.Parse(rpcServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ModifyResponse = func(response *http.Response) error {
		if response.Request.Header.Get("X-Fixture-Send") == "1" && f.dropResponse.CompareAndSwap(true, false) {
			response.Body.Close()
			return fmt.Errorf("fixture lost broadcast response after acceptance")
		}
		return nil
	}
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body bytes.Buffer
		if _, err := body.ReadFrom(r.Body); err != nil {
			t.Error(err)
			return
		}
		r.Body.Close()
		raw := append([]byte(nil), body.Bytes()...)
		var req rpcRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			t.Error(err)
			return
		}
		if req.Method == "eth_getBlockByNumber" && bytes.Contains(req.Params, []byte(`"finalized"`)) {
			if f.failFinality.Load() {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": map[string]any{"code": -32000, "message": "fixture finality unavailable"}})
				return
			}
			if head := f.finalizedHeader.Load(); head != nil {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": head})
				return
			}
		}
		r.Body = http.NoBody
		if len(raw) > 0 {
			r.Body = io.NopCloser(bytes.NewReader(raw))
		}
		if req.Method == "eth_sendRawTransaction" {
			f.sendCount.Add(1)
			r.Header.Set("X-Fixture-Send", "1")
		}
		proxy.ServeHTTP(w, r)
	}))
	t.Cleanup(proxyServer.Close)
	f.client, err = chain.NewNetwork(chain.SepoliaID, []string{proxyServer.URL})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(f.client.Close)
	for i := range 2 {
		f.dirs[i] = t.TempDir()
		f.open(i)
		created, err := f.services[i].Create(escrowPassword)
		if err != nil {
			t.Fatal(err)
		}
		f.addresses[i] = common.HexToAddress(created.Address)
		nonce, err := f.sim.Client().PendingNonceAt(context.Background(), deployer.From)
		if err != nil {
			t.Fatal(err)
		}
		tx, err := types.SignTx(types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(chain.SepoliaID), Nonce: nonce, GasTipCap: big.NewInt(1e9), GasFeeCap: big.NewInt(2e9), Gas: 21000, To: &f.addresses[i], Value: big.NewInt(1e18)}), types.LatestSignerForChainID(big.NewInt(chain.SepoliaID)), key)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.sim.Client().SendTransaction(context.Background(), tx); err != nil {
			t.Fatal(err)
		}
		f.sim.Commit()
		tx, err = f.token.Transact(deployer, "mint", f.addresses[i], big.NewInt(100_000_000))
		if err != nil {
			t.Fatal(err)
		}
		f.sim.Commit()
		receipt, err := f.sim.Client().TransactionReceipt(context.Background(), tx.Hash())
		if err != nil || receipt.Status != 1 {
			t.Fatal("mint failed", err)
		}
	}
	t.Cleanup(func() {
		for i := range 2 {
			f.servers[i].Close()
			f.services[i].Close()
		}
	})
	return f
}

func (f *escrowE2E) open(i int) {
	f.t.Helper()
	service, err := wallet.NewService(f.client, f.dirs[i], 2, 1)
	if err != nil {
		f.t.Fatal(err)
	}
	if err := service.SetEscrow(f.contract.Hex(), f.tokenAddress.Hex()); err != nil {
		f.t.Fatal(err)
	}
	handler, err := web.New(f.client, service)
	if err != nil {
		f.t.Fatal(err)
	}
	f.services[i] = service
	f.servers[i] = httptest.NewServer(handler)
}

func (f *escrowE2E) request(person int, method, path string, body any, want int) map[string]any {
	f.t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		f.t.Fatal(err)
	}
	req, err := http.NewRequest(method, f.servers[person].URL+path, bytes.NewReader(data))
	if err != nil {
		f.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Wallet-CSRF", f.services[person].CSRFToken())
	response, err := f.servers[person].Client().Do(req)
	if err != nil {
		f.t.Fatal(err)
	}
	defer response.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(response.Body).Decode(&out); err != nil {
		f.t.Fatal(err)
	}
	if response.StatusCode != want {
		f.t.Fatalf("%s %s got=%d want=%d body=%v", method, path, response.StatusCode, want, out)
	}
	return out
}

func (f *escrowE2E) quote(person int, body map[string]string, want int) map[string]any {
	return f.request(person, "POST", "/api/wallet/quote", body, want)
}
func (f *escrowE2E) send(person int, quote map[string]any) map[string]any {
	f.t.Helper()
	return f.request(person, "POST", "/api/wallet/send", map[string]any{"quoteId": quote["id"], "password": escrowPassword}, 200)
}
func (f *escrowE2E) history(person int) { f.request(person, "GET", "/api/wallet/history", nil, 200) }
func (f *escrowE2E) approve(person int, amount string) {
	f.t.Helper()
	quote := f.quote(person, map[string]string{"action": "approve", "to": f.contract.Hex(), "contract": f.tokenAddress.Hex(), "amountRaw": amount}, 200)
	f.send(person, quote)
	f.history(person)
}
func (f *escrowE2E) order(person int, reference string) map[string]any {
	return f.request(person, "GET", "/api/wallet/escrow/order?"+url.Values{"buyer": {f.addresses[0].Hex()}, "orderId": {reference}}.Encode(), nil, 200)
}
func (f *escrowE2E) call(contract *bind.BoundContract, method string, args ...any) []any {
	f.t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []any
	if err := contract.Call(&bind.CallOpts{}, &out, method, args...); err != nil {
		f.t.Fatal(err)
	}
	return out
}
func (f *escrowE2E) checkReceipt(sent map[string]any, eventName, reference string, amount int64) {
	f.t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	receipt, err := f.sim.Client().TransactionReceipt(context.Background(), common.HexToHash(sent["hash"].(string)))
	if err != nil || receipt.Status != 1 {
		f.t.Fatal("receipt failed", err)
	}
	var matching int
	for _, log := range receipt.Logs {
		if log.Address == f.contract && log.Topics[0] == f.escrowABI.Events[eventName].ID {
			matching++
			if len(log.Topics) != 4 || log.Topics[1] != common.BytesToHash(f.addresses[0].Bytes()) || log.Topics[2] != crypto.Keccak256Hash([]byte(reference)) || log.Topics[3] != common.BytesToHash(f.addresses[1].Bytes()) || new(big.Int).SetBytes(log.Data).Int64() != amount {
				f.t.Fatal("event mismatch")
			}
		}
	}
	if matching != 1 {
		f.t.Fatal("expected exactly one escrow event")
	}
}

func TestE2EEscrowPaymentReleaseRefundRecovery(t *testing.T) {
	f := newEscrowE2E(t)
	fund := map[string]string{"action": "escrow_fund", "to": f.addresses[1].Hex(), "orderId": "refund-001", "amount": "5"}
	// Validation rejects bad precision, zero, self payment, changed server configuration, and absent approvals.
	f.quote(0, fund, 400)
	for _, change := range []map[string]string{{"amount": "0"}, {"amount": "1.0000001"}, {"to": f.addresses[0].Hex()}, {"contract": f.contract.Hex()}, {"orderId": "bad order"}, {"buyer": f.addresses[1].Hex()}} {
		body := maps.Clone(fund)
		maps.Copy(body, change)
		f.quote(0, body, 400)
	}
	if f.sendCount.Load() != 0 {
		t.Fatal("invalid quotes broadcast transactions")
	}
	f.approve(0, "5000000")
	q := f.quote(0, fund, 200)
	// Wrong password must leave the original quote available and send nothing.
	before := f.sendCount.Load()
	f.request(0, "POST", "/api/wallet/send", map[string]any{"quoteId": q["id"], "password": "wrong-password-123"}, 401)
	if f.sendCount.Load() != before {
		t.Fatal("bad password broadcast")
	}
	// RPC accepts and mines, but the response is lost. Reopen the real durable journal before retrying.
	f.dropResponse.Store(true)
	first := f.send(0, q)
	acceptedSends := f.sendCount.Load()
	f.servers[0].Close()
	f.services[0].Close()
	f.open(0)
	retry := f.send(0, q)
	if retry["hash"] != first["hash"] || f.sendCount.Load() != acceptedSends {
		t.Fatalf("restart: before=%d after=%d first=%v retry=%v", before, f.sendCount.Load(), first, retry)
	}
	f.history(0)
	f.checkReceipt(first, "Funded", "refund-001", 5_000_000)
	if f.order(0, "refund-001")["state"] != "funded" {
		t.Fatal("order not funded")
	}
	f.quote(0, fund, 400)
	refund := map[string]string{"action": "escrow_refund", "to": f.addresses[0].Hex(), "buyer": f.addresses[0].Hex(), "orderId": "refund-001"}
	f.quote(0, refund, 400)
	releasedBySeller := map[string]string{"action": "escrow_release", "to": f.addresses[1].Hex(), "buyer": f.addresses[0].Hex(), "orderId": "refund-001"}
	f.quote(1, releasedBySeller, 400)
	refunded := f.send(1, f.quote(1, refund, 200))
	f.history(1)
	f.checkReceipt(refunded, "Refunded", "refund-001", 5_000_000)
	f.quote(1, refund, 400)
	if f.order(0, "refund-001")["state"] != "refunded" || f.call(f.token, "balanceOf", f.addresses[0])[0].(*big.Int).Int64() != 100_000_000 {
		t.Fatal("refund accounting failed")
	}
	// Competing valid release/refund quotes: after one wins, the stale quote must fail before signing.
	fund["orderId"] = "release-001"
	fund["amount"] = "3.25"
	f.approve(0, "3250000")
	paid := f.send(0, f.quote(0, fund, 200))
	f.history(0)
	f.checkReceipt(paid, "Funded", "release-001", 3_250_000)
	refund["orderId"] = "release-001"
	staleRefund := f.quote(1, refund, 200)
	releasedBySeller["orderId"] = "release-001"
	releasedBySeller["amount"] = "999999"
	releaseQuote := f.quote(0, releasedBySeller, 200)
	if releaseQuote["amount"] != "3.25" {
		t.Fatal("client overwrote on-chain settlement amount")
	}
	released := f.send(0, releaseQuote)
	f.history(0)
	f.checkReceipt(released, "Released", "release-001", 3_250_000)
	before = f.sendCount.Load()
	f.request(1, "POST", "/api/wallet/send", map[string]any{"quoteId": staleRefund["id"], "password": escrowPassword}, 400)
	if f.sendCount.Load() != before {
		t.Fatal("stale settlement broadcast")
	}
	reused := f.send(0, releaseQuote)
	if reused["hash"] != released["hash"] || f.sendCount.Load() != before {
		t.Fatal("duplicate release not reused")
	}
	if f.order(0, "release-001")["state"] != "released" || f.call(f.token, "balanceOf", f.addresses[1])[0].(*big.Int).Int64() != 103_250_000 || f.call(f.escrow, "totalLocked")[0].(*big.Int).Sign() != 0 {
		t.Fatal("release accounting failed")
	}
}

func TestE2EEscrowOrderFinality(t *testing.T) {
	f := newEscrowE2E(t)
	f.approve(0, "1000000")
	f.send(0, f.quote(0, map[string]string{"action": "escrow_fund", "to": f.addresses[1].Hex(), "amount": "1", "orderId": "finality"}, 200))
	f.history(0)
	finalizeCurrent := func() {
		t.Helper()
		f.mu.Lock()
		defer f.mu.Unlock()
		head, err := f.sim.Client().HeaderByNumber(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		f.finalizedHeader.Store(head)
		f.sim.Commit()
		f.sim.Commit()
	}
	finalizeCurrent()
	if f.order(0, "finality")["finalized"] != true {
		t.Fatal("finalized order must remain finalized as latest advances")
	}
	f.send(0, f.quote(0, map[string]string{"action": "escrow_release", "to": f.addresses[1].Hex(), "buyer": f.addresses[0].Hex(), "orderId": "finality"}, 200))
	order := f.order(0, "finality")
	if order["state"] != "released" || order["finalized"] != false {
		t.Fatal("finalized funding must not imply finalized release", order)
	}
	finalizeCurrent()
	if f.order(0, "finality")["finalized"] != true {
		t.Fatal("release did not reach finality")
	}
	f.failFinality.Store(true)
	order = f.order(0, "finality")
	if order["state"] != "released" || order["finalized"] != false {
		t.Fatal("unavailable finality must retain known state without claiming finality", order)
	}
}

func TestE2EEscrowBrowser(t *testing.T) {
	required := os.Getenv("RUN_BROWSER_E2E") == "1"
	nodeBin, err := exec.LookPath("node")
	if testing.Short() || err != nil {
		if required {
			t.Fatal("browser test dependencies missing or short mode")
		}
		t.Skip("browser requires Node and normal test mode")
	}
	if _, err := os.Stat(filepath.Join("..", "browser", "node_modules", "playwright")); err != nil {
		if required {
			t.Fatal(err)
		}
		t.Skip("playwright missing")
	}
	f := newEscrowE2E(t)
	cmd := exec.Command(nodeBin, filepath.Join("..", "browser", "escrow-e2e.cjs"))
	cmd.Env = append(os.Environ(), "E2E_BACKEND=simulated", "E2E_BUYER_URL="+f.servers[0].URL, "E2E_SELLER_URL="+f.servers[1].URL, "E2E_BUYER="+f.addresses[0].Hex(), "E2E_SELLER="+f.addresses[1].Hex(), "E2E_PASSWORD="+escrowPassword)
	output, err := cmd.CombinedOutput()
	t.Log(string(output))
	if err != nil {
		t.Fatalf("browser: %v", err)
	}
	for person := range 2 {
		history, err := f.services[person].History(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if history.RefreshError != "" {
			t.Fatal(history.RefreshError)
		}
		expectedCount := 5
		if person == 1 {
			expectedCount = 1
		}
		if len(history.Transactions) != expectedCount {
			t.Fatalf("person %d history count=%d", person, len(history.Transactions))
		}
		for _, tx := range history.Transactions {
			if tx.State != "succeeded" {
				t.Fatalf("browser transaction %s state=%s", tx.Action, tx.State)
			}
			event := ""
			reference := "browser-refund"
			amount := int64(5_000_000)
			switch tx.Action {
			case "escrow_fund":
				event = "Funded"
				if tx.Amount == "3.25" {
					reference = "browser-release"
					amount = 3_250_000
				}
			case "escrow_refund":
				event = "Refunded"
			case "escrow_release":
				event = "Released"
				reference = "browser-release"
				amount = 3_250_000
			case "approve":
				continue
			default:
				t.Fatal("unexpected action", tx.Action)
			}
			f.checkReceipt(map[string]any{"hash": tx.Hash}, event, reference, amount)
		}
	}
	if f.call(f.token, "balanceOf", f.addresses[0])[0].(*big.Int).Int64() != 96_750_000 || f.call(f.token, "balanceOf", f.addresses[1])[0].(*big.Int).Int64() != 103_250_000 || f.call(f.escrow, "totalLocked")[0].(*big.Int).Sign() != 0 {
		t.Fatal("browser balances do not match independent chain state")
	}
}
