package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/a861252012/multi-chain-wallet/internal/wallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
)

// Use the real local txpool, but leave mining under test control. The shared
// fixture mines immediately and cannot exercise same-nonce replacements.
func escrowReplacementRPC(t *testing.T, f *escrowE2E) func() []*types.Transaction {
	t.Helper()
	upstream := startSimulatedRPC(t, f.sim, &f.mu)
	t.Cleanup(upstream.Close)
	var sentMu sync.Mutex
	var sent []*types.Transaction
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			t.Error(err)
			return
		}
		var request rpcRequest
		if err := json.Unmarshal(body, &request); err != nil {
			t.Error(err)
			return
		}
		if request.Method != "eth_sendRawTransaction" && request.Method != "eth_getTransactionCount" {
			r.Body = io.NopCloser(bytes.NewReader(body))
			upstream.Config.Handler.ServeHTTP(w, r)
			return
		}
		var args []json.RawMessage
		var result any
		callErr := json.Unmarshal(request.Params, &args)
		f.mu.Lock()
		if callErr == nil {
			switch request.Method {
			case "eth_getTransactionCount":
				var account common.Address
				var block rpc.BlockNumber
				if len(args) != 2 {
					callErr = fmt.Errorf("expected account and block tag")
				} else if callErr = json.Unmarshal(args[0], &account); callErr == nil {
					callErr = json.Unmarshal(args[1], &block)
					if callErr == nil {
						var nonce uint64
						if block == rpc.PendingBlockNumber {
							nonce, callErr = f.sim.Client().PendingNonceAt(r.Context(), account)
						} else {
							nonce, callErr = f.sim.Client().NonceAt(r.Context(), account, big.NewInt(block.Int64()))
						}
						result = hexutil.EncodeUint64(nonce)
					}
				}
			case "eth_sendRawTransaction":
				var raw hexutil.Bytes
				if len(args) != 1 {
					callErr = fmt.Errorf("expected signed transaction")
				} else if callErr = json.Unmarshal(args[0], &raw); callErr == nil {
					tx := new(types.Transaction)
					if callErr = tx.UnmarshalBinary(raw); callErr == nil {
						f.sendCount.Add(1)
						callErr = f.sim.Client().SendTransaction(r.Context(), tx)
						if callErr == nil {
							sentMu.Lock()
							sent = append(sent, tx)
							sentMu.Unlock()
							result = tx.Hash().Hex()
						}
					}
				}
			}
		}
		f.mu.Unlock()
		response := map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result}
		if callErr != nil {
			delete(response, "result")
			response["error"] = map[string]any{"code": -32000, "message": callErr.Error()}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	t.Cleanup(server.Close)
	client, err := chain.NewNetwork(chain.SepoliaID, []string{server.URL})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	f.client = client
	for person := range 2 {
		f.servers[person].Close()
		if err := f.services[person].Close(); err != nil {
			t.Fatal(err)
		}
		f.open(person)
	}
	return func() []*types.Transaction {
		sentMu.Lock()
		defer sentMu.Unlock()
		return append([]*types.Transaction(nil), sent...)
	}
}

func reopenEscrowReplacement(t *testing.T, f *escrowE2E, person int, legacy bool) {
	t.Helper()
	f.servers[person].Close()
	if err := f.services[person].Close(); err != nil {
		t.Fatal(err)
	}
	if legacy {
		path := filepath.Join(f.dirs[person], "journal.json")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var records []map[string]json.RawMessage
		if err := json.Unmarshal(data, &records); err != nil {
			t.Fatal(err)
		}
		for _, record := range records {
			delete(record, "orderId")
			delete(record, "escrowAction")
			// Legacy recovery must not mistake display fields (or injected
			// derived fields) for signed funding/settlement instructions.
			var action string
			if err := json.Unmarshal(record["action"], &action); err != nil {
				t.Fatal(err)
			}
			if action == "escrow_fund" || action == "escrow_release" || action == "escrow_refund" || action == "speedup" {
				record["amount"], record["amountRaw"] = json.RawMessage(`"999"`), json.RawMessage(`"999000000"`)
				record["to"], record["symbol"] = json.RawMessage(`"0x9999999999999999999999999999999999999999"`), json.RawMessage(`"FAKE"`)
				record["escrowAction"] = json.RawMessage(`"escrow_refund"`)
				record["escrowBuyer"], record["escrowContract"] = record["to"], record["to"]
			}
		}
		data, err = json.Marshal(records)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	f.open(person)
}

func TestE2EEscrowReplacementRecovery(t *testing.T) {
	for _, action := range []string{"escrow_fund", "escrow_release", "escrow_refund"} {
		for _, legacy := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/legacy=%t", action, legacy), func(t *testing.T) {
				f := newEscrowE2E(t)
				const reference = "replace-order"
				f.approve(0, "1000000")
				fund := map[string]string{"action": "escrow_fund", "to": f.addresses[1].Hex(), "amount": "1", "orderId": reference}
				request := fund
				person, event, state := 0, "Funded", "funded"
				if action != "escrow_fund" {
					f.send(0, f.quote(0, fund, 200))
					f.history(0)
					request = map[string]string{"action": action, "to": f.addresses[1].Hex(), "buyer": f.addresses[0].Hex(), "orderId": reference}
					event, state = "Released", "released"
					if action == "escrow_refund" {
						person, event, state = 1, "Refunded", "refunded"
						request["to"] = f.addresses[0].Hex()
					}
				}
				snapshot := escrowReplacementRPC(t, f)
				originalQuote := f.quote(person, request, 200)
				if preview, ok := originalQuote["escrow"].(map[string]any); !ok || preview["action"] != action {
					t.Fatal("ordinary escrow quote lost original action", originalQuote)
				}
				original := f.send(person, originalQuote)
				if original["state"] != "submitted" || len(snapshot()) != 1 {
					t.Fatal("original not accepted by txpool", original)
				}
				reopenEscrowReplacement(t, f, person, legacy)
				wantReference := reference
				if legacy {
					wantReference = crypto.Keccak256Hash([]byte(reference)).Hex()
				}
				previous := original
				var speedup map[string]any
				for round := range 2 {
					speedup = f.quote(person, map[string]string{"action": "speedup", "hash": previous["hash"].(string)}, 200)
					preview, ok := speedup["escrow"].(map[string]any)
					if !ok || preview["action"] != action || preview["orderId"] != wantReference || preview["buyer"] != f.addresses[0].Hex() || preview["seller"] != f.addresses[1].Hex() || preview["token"] != f.tokenAddress.Hex() {
						t.Fatalf("round %d lost signed escrow intent: %v", round, speedup)
					}
					for _, field := range []string{"data", "nonce", "to", "amount", "amountRaw", "symbol", "contract"} {
						if speedup[field] != originalQuote[field] {
							t.Fatalf("speedup changed %s: %v != %v", field, speedup[field], originalQuote[field])
						}
					}
					previous = f.send(person, speedup)
					if previous["state"] != "submitted" || previous["action"] != "speedup" || previous["escrowAction"] != action {
						t.Fatal("speedup response lost action or was not accepted", previous)
					}
					reopenEscrowReplacement(t, f, person, false)
					before := f.sendCount.Load()
					duplicate := f.send(person, speedup)
					if duplicate["hash"] != previous["hash"] || duplicate["escrowAction"] != action || f.sendCount.Load() != before {
						t.Fatal("restart retry changed identity or rebroadcast", duplicate)
					}
				}
				cancel := f.quote(person, map[string]string{"action": "cancel", "hash": previous["hash"].(string)}, 200)
				if _, exists := cancel["escrow"]; exists || cancel["data"] != "0x" || cancel["amount"] != "0" || cancel["to"] != f.addresses[person].Hex() {
					t.Fatal("cancel retained escrow intent", cancel)
				}
				txs := snapshot()
				if len(txs) != 3 {
					t.Fatal("expected original plus two accepted replacements", len(txs))
				}
				for i, tx := range txs {
					sender, err := types.Sender(types.LatestSignerForChainID(big.NewInt(chain.SepoliaID)), tx)
					if err != nil || sender != f.addresses[person] || tx.ChainId().Int64() != chain.SepoliaID || tx.Nonce() != txs[0].Nonce() || *tx.To() != *txs[0].To() || tx.Value().Cmp(txs[0].Value()) != 0 || !bytes.Equal(tx.Data(), txs[0].Data()) {
						t.Fatal("replacement changed signed payload or signer", i, err)
					}
					if i > 0 && (tx.GasFeeCap().Cmp(txs[i-1].GasFeeCap()) <= 0 || tx.GasTipCap().Cmp(txs[i-1].GasTipCap()) <= 0) {
						t.Fatal("replacement did not increase both fees")
					}
				}
				f.mu.Lock()
				f.sim.Commit()
				f.mu.Unlock()
				f.checkReceipt(previous, event, reference, 1_000_000)
				reopenEscrowReplacement(t, f, person, false)
				history := f.request(person, "GET", "/api/wallet/history", nil, 200)["transactions"].([]any)
				found := 0
				for _, value := range history {
					item := value.(map[string]any)
					for _, tx := range txs {
						if item["hash"] != tx.Hash().Hex() {
							continue
						}
						found++
						if item["escrowAction"] != action || item["orderId"] != wantReference || item["escrowBuyer"] != f.addresses[0].Hex() || item["escrowContract"] != f.contract.Hex() {
							t.Fatal("history lost escrow metadata", item)
						}
						if item["hash"] == previous["hash"] {
							if item["state"] != "succeeded" {
								t.Fatal("winning replacement not observed", item)
							}
						} else if item["state"] != "replaced" || item["replacedBy"] != previous["hash"] {
							t.Fatal("original relationship lost", item)
						}
					}
				}
				if found != 3 || f.order(person, wantReference)["state"] != state {
					t.Fatal("replacement history or recovered order missing")
				}
				before := f.sendCount.Load()
				if retry := f.send(person, speedup); retry["hash"] != previous["hash"] || retry["state"] != "succeeded" || f.sendCount.Load() != before {
					t.Fatal("mined quote retry changed identity", retry)
				}
			})
		}
	}
}

func TestE2EEscrowReplacementCancelHasNoOrder(t *testing.T) {
	f := newEscrowE2E(t)
	f.approve(0, "1000000")
	snapshot := escrowReplacementRPC(t, f)
	original := f.send(0, f.quote(0, map[string]string{"action": "escrow_fund", "to": f.addresses[1].Hex(), "amount": "1", "orderId": "cancel-order"}, 200))
	cancelQuote := f.quote(0, map[string]string{"action": "cancel", "hash": original["hash"].(string)}, 200)
	cancel := f.send(0, cancelQuote)
	if cancel["state"] != "submitted" {
		t.Fatal("cancel not accepted", cancel)
	}
	reopenEscrowReplacement(t, f, 0, false)
	// Speeding up a cancellation must remain a zero-value self-transfer.
	speedup := f.quote(0, map[string]string{"action": "speedup", "hash": cancel["hash"].(string)}, 200)
	if _, exists := speedup["escrow"]; exists {
		t.Fatal("speedup of cancel restored the original payment", speedup)
	}
	winner := f.send(0, speedup)
	if winner["state"] != "submitted" {
		t.Fatal("speedup of cancel not accepted", winner)
	}
	f.mu.Lock()
	f.sim.Commit()
	f.mu.Unlock()
	reopenEscrowReplacement(t, f, 0, false)
	history := f.request(0, "GET", "/api/wallet/history", nil, 200)["transactions"].([]any)
	found := 0
	for _, value := range history {
		item := value.(map[string]any)
		if item["hash"] != cancel["hash"] && item["hash"] != winner["hash"] {
			continue
		}
		found++
		for _, field := range []string{"escrowAction", "orderId", "escrowBuyer", "escrowContract"} {
			if _, exists := item[field]; exists {
				t.Fatal("cancel history contains escrow metadata", field, item)
			}
		}
	}
	if found != 2 {
		t.Fatal("cancel and its speedup missing from history")
	}
	for _, result := range []map[string]any{cancel, winner, f.send(0, speedup)} {
		if _, exists := result["escrowAction"]; exists {
			t.Fatal("cancel response contains escrow action", result)
		}
	}
	txs := snapshot()
	if len(txs) != 3 || txs[2].Value().Sign() != 0 || len(txs[2].Data()) != 0 || *txs[2].To() != f.addresses[0] || txs[2].Nonce() != txs[0].Nonce() {
		t.Fatal("cancel payload changed")
	}
	receipt, err := f.sim.Client().TransactionReceipt(context.Background(), txs[2].Hash())
	if err != nil || receipt.Status != types.ReceiptStatusSuccessful || len(receipt.Logs) != 0 || f.order(0, "cancel-order")["state"] != "none" {
		t.Fatal("cancellation funded the order or failed", err)
	}
}

func TestE2EEscrowReplacementRechecksConfiguration(t *testing.T) {
	f := newEscrowE2E(t)
	f.approve(0, "1000000")
	escrowReplacementRPC(t, f)
	original := f.send(0, f.quote(0, map[string]string{"action": "escrow_fund", "to": f.addresses[1].Hex(), "amount": "1", "orderId": "config-change"}, 200))
	speedup := f.quote(0, map[string]string{"action": "speedup", "hash": original["hash"].(string)}, 200)
	before := f.sendCount.Load()
	for _, change := range [][2]string{{common.Address{9}.Hex(), f.tokenAddress.Hex()}, {f.contract.Hex(), common.Address{9}.Hex()}, {"", ""}} {
		if err := f.services[0].SetEscrow(change[0], change[1]); err != nil {
			t.Fatal(err)
		}
		rejected := f.request(0, "POST", "/api/wallet/send", map[string]any{"quoteId": speedup["id"], "password": escrowPassword}, 400)
		if rejected["code"] != "send_rejected" || f.sendCount.Load() != before {
			t.Fatal("changed escrow configuration was not rejected before broadcast", rejected)
		}
	}
	// Disabling the escrow blocks a new speedup, but must still allow cancel.
	f.quote(0, map[string]string{"action": "speedup", "hash": original["hash"].(string)}, 400)
	cancel := f.quote(0, map[string]string{"action": "cancel", "hash": original["hash"].(string)}, 200)
	if _, exists := cancel["escrow"]; exists {
		t.Fatal("cancel acquired metadata from a disabled escrow")
	}
	if err := f.services[0].SetEscrow(f.contract.Hex(), f.tokenAddress.Hex()); err != nil {
		t.Fatal(err)
	}
	if sent := f.send(0, speedup); sent["state"] != "submitted" || f.sendCount.Load() != before+1 {
		t.Fatal("safe rejection consumed the original quote", sent)
	}
}

func TestE2EEscrowReplacementValidatesDiskAction(t *testing.T) {
	for _, savedAction := range []string{"escrow_fund", "", "eth", "cancel"} {
		t.Run("saved-action="+savedAction, func(t *testing.T) {
			f := newEscrowE2E(t)
			f.approve(0, "1000000")
			snapshot := escrowReplacementRPC(t, f)
			original := f.send(0, f.quote(0, map[string]string{"action": "escrow_fund", "to": f.addresses[1].Hex(), "amount": "1", "orderId": "signed-classification"}, 200))
			if original["state"] != "submitted" || len(snapshot()) != 1 {
				t.Fatal("original transaction was not accepted by the local txpool")
			}
			f.servers[0].Close()
			if err := f.services[0].Close(); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(f.dirs[0], "journal.json")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var records []map[string]json.RawMessage
			if err := json.Unmarshal(data, &records); err != nil {
				t.Fatal(err)
			}
			changed := 0
			for _, record := range records {
				var hash string
				if err := json.Unmarshal(record["hash"], &hash); err != nil {
					t.Fatal(err)
				}
				if hash == original["hash"] {
					changed++
					delete(record, "orderId")
					record["action"], err = json.Marshal(savedAction)
					if err != nil {
						t.Fatal(err)
					}
				}
			}
			if changed != 1 {
				t.Fatal("expected to change exactly the pending record")
			}
			data, err = json.Marshal(records)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			service, err := wallet.NewService(f.client, f.dirs[0], 2, 1)
			if savedAction == "eth" || savedAction == "cancel" {
				if service != nil {
					service.Close()
				}
				if err == nil || !strings.Contains(err.Error(), "託管操作與簽名不符") || len(snapshot()) != 1 {
					t.Fatalf("conflicting disk action was not rejected: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer service.Close()
			// Escrow is disabled in a newly opened service. Missing display fields
			// must not let a signed payment skip the configuration check.
			request := &wallet.QuoteRequest{Action: "speedup", Hash: original["hash"].(string)}
			if _, err := service.Quote(context.Background(), request); err == nil || !strings.Contains(err.Error(), "託管設定已變更") || len(snapshot()) != 1 {
				t.Fatalf("disabled escrow accepted a legacy payment: %v", err)
			}
			if err := service.SetEscrow(f.contract.Hex(), f.tokenAddress.Hex()); err != nil {
				t.Fatal(err)
			}
			quote, err := service.Quote(context.Background(), request)
			if err != nil || quote.Escrow == nil || quote.Escrow.Action != wallet.ActionEscrowFund || quote.Escrow.OrderID != crypto.Keccak256Hash([]byte("signed-classification")).Hex() {
				t.Fatalf("valid legacy record lost its signed order: quote=%+v err=%v", quote, err)
			}
		})
	}
}
