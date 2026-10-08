package wallet

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

type rpcState struct {
	mu             sync.Mutex
	chainID        string
	nonce          uint64
	balance        *big.Int
	gas            uint64
	unknown        bool
	raws           []string
	allowance      *big.Int
	tokenBalance   *big.Int
	tokenDecimals  uint8
	swapOutput     *big.Int
	missingPool    bool
	failSimulation bool
	broadcastCheck func(string) error
	checkRan       bool
}

func guardedFixture(t *testing.T) (*Service, *rpcState) {
	t.Helper()
	state := &rpcState{chainID: "0xaa36a7", balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil), gas: 48000, allowance: big.NewInt(0), tokenBalance: big.NewInt(10000000), tokenDecimals: 6, swapOutput: big.NewInt(1000000)}
	c := mockRPC(t, func(method string, params json.RawMessage) any {
		state.mu.Lock()
		defer state.mu.Unlock()
		switch method {
		case "eth_chainId":
			return state.chainID
		case "eth_getBlockByNumber":
			return &types.Header{Number: big.NewInt(100), Difficulty: big.NewInt(0), GasLimit: 30000000, BaseFee: big.NewInt(1000000000)}
		case "eth_maxPriorityFeePerGas":
			return "0x3b9aca00"
		case "eth_getTransactionCount":
			return hexutil.EncodeUint64(state.nonce)
		case "eth_getBalance":
			return hexutil.EncodeBig(state.balance)
		case "eth_estimateGas":
			if state.gas == 0 {
				return nil
			}
			return hexutil.EncodeUint64(state.gas)
		case "eth_getCode":
			return "0x6000"
		case "eth_call":
			var args []json.RawMessage
			json.Unmarshal(params, &args)
			var call map[string]string
			json.Unmarshal(args[0], &call)
			contract := common.HexToAddress(call["to"])
			input := call["input"]
			if input == "" {
				input = call["data"]
			}
			data, err := hexutil.Decode(input)
			if err != nil || len(data) < 4 {
				t.Error("invalid call data")
				return nil
			}
			if method, err := exchangeABI.MethodById(data[:4]); err == nil {
				switch method.Name {
				case "deposit", "withdraw":
					if state.failSimulation {
						return "0x01"
					}
					return "0x"
				case "getPool":
					address := common.HexToAddress("0x6666666666666666666666666666666666666666")
					if state.missingPool {
						address = common.Address{}
					}
					result, _ := method.Outputs.Pack(address)
					return hexutil.Encode(result)
				case "quoteExactInputSingle":
					result, _ := method.Outputs.Pack(state.swapOutput, big.NewInt(1), uint32(0), big.NewInt(150000))
					return hexutil.Encode(result)
				case "multicall":
					if state.failSimulation {
						return "0x01"
					}
					result, _ := method.Outputs.Pack([][]byte{common.LeftPadBytes(state.swapOutput.Bytes(), 32)})
					return hexutil.Encode(result)
				}
			}
			method, err := erc20ABI.MethodById(data[:4])
			if err != nil {
				t.Error(err)
				return nil
			}
			var result []byte
			tokenSymbol, tokenDecimals := "TST", state.tokenDecimals
			if contract == common.HexToAddress(USDCAddress) {
				tokenSymbol = "USDC"
			} else if contract == common.HexToAddress(WETHAddress) {
				tokenSymbol, tokenDecimals = "WETH", 18
			}
			switch method.Name {
			case "symbol":
				result, _ = method.Outputs.Pack(tokenSymbol)
			case "decimals":
				result, _ = method.Outputs.Pack(tokenDecimals)
			case "balanceOf":
				result, _ = method.Outputs.Pack(state.tokenBalance)
			case "allowance":
				result, _ = method.Outputs.Pack(state.allowance)
			case "transfer", "approve":
				result, _ = method.Outputs.Pack(true)
			default:
				t.Error("unexpected token call")
			}
			return hexutil.Encode(result)
		case "eth_sendRawTransaction":
			var args []string
			if err := json.Unmarshal(params, &args); err != nil {
				t.Error(err)
				return nil
			}
			if state.broadcastCheck != nil {
				state.checkRan = true
				if err := state.broadcastCheck(args[0]); err != nil {
					return err
				}
			}
			state.raws = append(state.raws, args[0])
			if state.unknown {
				return nil
			}
			raw, err := hexutil.Decode(args[0])
			if err != nil {
				t.Error(err)
				return nil
			}
			var tx types.Transaction
			if err := tx.UnmarshalBinary(raw); err != nil {
				t.Error(err)
				return nil
			}
			return tx.Hash().Hex()
		case "eth_getTransactionReceipt", "eth_getTransactionByHash":
			return nil
		default:
			t.Errorf("unexpected RPC %s", method)
			return nil
		}
	})
	svc, err := NewService(c, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { svc.Close() })
	_, err = svc.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}
	return svc, state
}
func ethQuote(t *testing.T, s *Service) *QuoteResponse {
	t.Helper()
	q, err := s.Quote(context.Background(), &QuoteRequest{Action: "eth", To: "0x2222222222222222222222222222222222222222", Amount: "0.000000000000000001"})
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func TestSendRejectsChangedConditions(t *testing.T) {
	for _, kind := range []string{"chain", "nonce", "funds", "expiry", "password", "persistence"} {
		t.Run(kind, func(t *testing.T) {
			s, state := guardedFixture(t)
			q := ethQuote(t, s)
			password := "fixture-password-123"
			switch kind {
			case "chain":
				state.chainID = "0x1"
			case "nonce":
				state.nonce = 1
			case "funds":
				state.balance = big.NewInt(0)
			case "expiry":
				s.quotes.quotes[QuoteID(q.ID)].ExpiresAt = time.Now().Add(-time.Second)
			case "password":
				password = "incorrect-password"
			case "persistence":
				block := filepath.Join(t.TempDir(), "file")
				if err := os.WriteFile(block, []byte("x"), 0600); err != nil {
					t.Fatal(err)
				}
				s.journal.walletDir = block
			}
			_, err := s.Send(context.Background(), q.ID, password)
			if err == nil {
				t.Fatalf("%s unexpectedly permitted", kind)
			}
			if len(state.raws) != 0 {
				t.Fatalf("%s broadcast a transaction", kind)
			}
		})
	}
}
func TestETHRequiresRealGasEstimate(t *testing.T) {
	s, state := guardedFixture(t)
	q := ethQuote(t, s)
	gas, _ := strconv.ParseUint(q.GasLimit, 10, 64)
	if gas < 48000 {
		t.Fatal("contract recipient gas underestimated")
	}
	state.gas = 0
	if _, err := s.Quote(context.Background(), &QuoteRequest{Action: "eth", To: q.To, Amount: "1"}); err == nil {
		t.Fatal("failed estimate silently accepted")
	}
}
func TestConcurrentSendSignsExactQuoteOnce(t *testing.T) {
	s, state := guardedFixture(t)
	q := ethQuote(t, s)
	var wg sync.WaitGroup
	results := make(chan *SendResponse, 4)
	for i := 0; i < 4; i += 1 {
		wg.Go(func() {
			result, err := s.Send(context.Background(), q.ID, "fixture-password-123")
			if err != nil {
				t.Error(err)
				return
			}
			results <- result
		})
	}
	wg.Wait()
	close(results)
	var hash string
	for result := range results {
		if hash != "" && hash != result.Hash {
			t.Fatal("duplicate request made different transfer")
		}
		hash = result.Hash
	}
	if len(state.raws) != 1 {
		t.Fatalf("broadcast count %d", len(state.raws))
	}
	raw, _ := hexutil.Decode(state.raws[0])
	var tx types.Transaction
	if err := tx.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	from, err := types.Sender(types.LatestSignerForChainID(big.NewInt(chain.SepoliaID)), &tx)
	if err != nil || from.Hex() != q.From || tx.Type() != types.DynamicFeeTxType || tx.ChainId().Int64() != chain.SepoliaID || tx.To().Hex() != q.To || tx.Value().String() != "1" || tx.GasFeeCap().String() != q.MaxFeePerGas || tx.GasTipCap().String() != q.MaxPriorityFeePerGas || strconv.FormatUint(tx.Gas(), 10) != q.GasLimit {
		t.Fatal("signed transaction did not match reviewed quote")
	}
}

func TestSendPersistsSignedTransactionBeforeBroadcast(t *testing.T) {
	s, state := guardedFixture(t)
	q := ethQuote(t, s)
	state.broadcastCheck = func(raw string) error {
		data, err := os.ReadFile(filepath.Join(s.walletDir, "journal.json"))
		if err != nil {
			return err
		}
		var records []*journalRecordDisk
		if err := json.Unmarshal(data, &records); err != nil {
			return err
		}
		if len(records) != 1 || records[0].State != string(JournalPending) || records[0].SignedRaw != raw || records[0].Version != 1 {
			return errors.New("broadcast observed before the exact signed transaction became durable")
		}
		return nil
	}
	result, err := s.Send(context.Background(), q.ID, "fixture-password-123")
	if err != nil || result.State != string(JournalSubmitted) {
		t.Fatalf("send result=%+v err=%v", result, err)
	}
	state.mu.Lock()
	checkRan := state.checkRan
	state.mu.Unlock()
	if !checkRan {
		t.Fatal("broadcast boundary was not exercised")
	}
}
func TestUnknownBroadcastRestartReusesRaw(t *testing.T) {
	s, state := guardedFixture(t)
	q := ethQuote(t, s)
	state.unknown = true
	sent, err := s.Send(context.Background(), q.ID, "fixture-password-123")
	if err != nil || sent.State != "broadcast_unknown" {
		t.Fatalf("%+v %v", sent, err)
	}
	dir := s.walletDir
	s.Close()
	restarted, err := NewService(s.client, dir, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	state.unknown = false
	result, err := restarted.Retry(context.Background(), sent.Hash)
	if err != nil || result.Hash != sent.Hash {
		t.Fatalf("retry %v", err)
	}
	if len(state.raws) != 2 || state.raws[0] != state.raws[1] {
		t.Fatal("retry changed signed payload")
	}
	if _, err := restarted.Quote(context.Background(), &QuoteRequest{Action: "eth", To: q.To, Amount: "1"}); !errors.Is(err, ErrTxInFlight) {
		t.Fatal("outstanding tx not blocked after restart")
	}
	if err := restarted.journal.UpdateStateAtomic(sent.Hash, "succeeded", "1", "0.001", ""); err != nil {
		t.Fatal(err)
	}
	hist, err := restarted.History(context.Background())
	if err != nil || hist.Transactions[0].State != "broadcast_unknown" || !restarted.journal.HasInFlightTx() {
		t.Fatal("missing previously mined tx should become uncertain and block")
	}
}
func TestWalletDirectoryExclusive(t *testing.T) {
	s, _ := guardedFixture(t)
	if second, err := NewService(s.client, s.walletDir, 2, 1); err == nil {
		second.Close()
		t.Fatal("two processes could use wallet directory")
	}
}

type tokenCaller struct{ result []byte }

func (f tokenCaller) CodeAt(context.Context, common.Address, *big.Int) ([]byte, error) {
	return []byte{1}, nil
}
func (f tokenCaller) CallContract(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
	return f.result, nil
}
func TestERC20SimulationStrictBoolean(t *testing.T) {
	for _, n := range []int64{0, 1, 2, 255} {
		raw := common.LeftPadBytes(big.NewInt(n).Bytes(), 32)
		err := SimulateERC20Call(context.Background(), tokenCaller{raw}, common.Address{}, common.Address{}, nil)
		if (err == nil) != (n == 1) {
			t.Fatalf("bool %d accepted=%v", n, err == nil)
		}
	}
	if err := SimulateERC20Call(context.Background(), tokenCaller{}, common.Address{}, common.Address{}, nil); err != nil {
		t.Fatal("optional empty return rejected")
	}
	data, _ := erc20ABI.Pack("transfer", common.HexToAddress("0x2222222222222222222222222222222222222222"), big.NewInt(1))
	if _, err := DecodeERC20Calldata(append(data, 0), 18); err == nil {
		t.Fatal("trailing calldata accepted")
	}
}

func TestERC20QuoteAndSignedRecipient(t *testing.T) {
	for _, action := range []string{"transfer", "approve"} {
		t.Run(action, func(t *testing.T) {
			s, state := guardedFixture(t)
			contract := "0x4444444444444444444444444444444444444444"
			recipient := "0x2222222222222222222222222222222222222222"
			info, err := s.Token(context.Background(), contract, "")
			if err != nil || info.Symbol != "TST" || info.Decimals != 6 || info.Balance != "10" {
				t.Fatalf("token %+v %v", info, err)
			}
			q, err := s.Quote(context.Background(), &QuoteRequest{Action: action, Contract: contract, To: recipient, AmountRaw: "1234567"})
			if err != nil {
				t.Fatal(err)
			}
			if q.AmountRaw != "1234567" || q.To != recipient || q.Contract != contract {
				t.Fatal("wrong token review fields")
			}
			result, err := s.Send(context.Background(), q.ID, "fixture-password-123")
			if err != nil || result.State != "submitted" {
				t.Fatalf("send %+v %v", result, err)
			}
			raw, _ := hexutil.Decode(state.raws[0])
			var tx types.Transaction
			tx.UnmarshalBinary(raw)
			decoded, err := DecodeERC20Calldata(tx.Data(), 6)
			if err != nil || decoded.Method != action || decoded.Target.Hex() != recipient || decoded.RawAmount.String() != "1234567" || tx.To().Hex() != contract || tx.Value().Sign() != 0 {
				t.Fatal("signed ERC20 recipient/contract/value mismatch")
			}
		})
	}
}

func TestCustomERC20RawAmountDoesNotDependOnRPCDecimals(t *testing.T) {
	for _, action := range []string{"transfer", "approve"} {
		t.Run(action, func(t *testing.T) {
			s, state := guardedFixture(t)
			req := &QuoteRequest{Action: action, Contract: "0x4444444444444444444444444444444444444444", To: "0x2222222222222222222222222222222222222222", AmountRaw: "1"}
			var last *QuoteResponse
			for _, decimals := range []uint8{6, 18} {
				state.tokenDecimals = decimals
				q, err := s.Quote(context.Background(), req)
				if err != nil {
					t.Fatal(err)
				}
				if q.AmountRaw != "1" {
					t.Fatalf("RPC decimals %d changed signed amount to %s", decimals, q.AmountRaw)
				}
				decoded, err := DecodeERC20Calldata(common.FromHex(q.Data), int(decimals))
				if err != nil || decoded.RawAmount.String() != "1" {
					t.Fatalf("RPC decimals %d changed calldata: %+v %v", decimals, decoded, err)
				}
				last = q
			}
			req.AmountRaw = ""
			if _, err := s.Quote(context.Background(), req); err == nil {
				t.Fatal("custom token quote accepted without exact raw amount")
			}
			if _, err := s.Send(context.Background(), last.ID, "fixture-password-123"); err != nil {
				t.Fatal(err)
			}
			raw, _ := hexutil.Decode(state.raws[0])
			var tx types.Transaction
			if tx.UnmarshalBinary(raw) != nil {
				t.Fatal("signed transaction could not be decoded")
			}
			decoded, err := DecodeERC20Calldata(tx.Data(), 18)
			if err != nil || decoded.RawAmount.String() != "1" {
				t.Fatalf("signed calldata amount changed: %+v %v", decoded, err)
			}
		})
	}
}

func TestBuiltInTokenRejectsRPCMetadataMismatch(t *testing.T) {
	s, state := guardedFixture(t)
	state.tokenDecimals = 18
	if _, err := s.Token(context.Background(), USDCAddress, ""); err == nil {
		t.Fatal("Sepolia USDC accepted RPC-provided decimals that differ from the built-in registry")
	}
}
func TestApprovalRevocationAndChangedAllowance(t *testing.T) {
	s, state := guardedFixture(t)
	req := &QuoteRequest{Action: "approve", Contract: "0x4444444444444444444444444444444444444444", To: "0x2222222222222222222222222222222222222222", AmountRaw: "1000000"}
	state.allowance = big.NewInt(5)
	if _, err := s.Quote(context.Background(), req); !errors.Is(err, ErrApprovalRace) {
		t.Fatalf("nonzero allowance accepted: %v", err)
	}
	req.AmountRaw = "0"
	if _, err := s.Quote(context.Background(), req); err != nil {
		t.Fatalf("revocation blocked: %v", err)
	}
	req.AmountRaw = maxUint256.String()
	if _, err := s.Quote(context.Background(), req); !errors.Is(err, ErrUnlimitedAllowanceNotAllowed) {
		t.Fatal("unlimited allowance accepted")
	}
	req.AmountRaw = "1000000"
	state.allowance = big.NewInt(0)
	q, err := s.Quote(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	state.allowance = big.NewInt(1)
	if _, err := s.Send(context.Background(), q.ID, "fixture-password-123"); !errors.Is(err, ErrApprovalRace) {
		t.Fatalf("changed allowance accepted: %v", err)
	}
	if len(state.raws) != 0 {
		t.Fatal("changed approval broadcast")
	}
}

func TestHistoryReportsUnverifiedRecords(t *testing.T) {
	s, _ := guardedFixture(t)
	q := ethQuote(t, s)
	sent, err := s.Send(context.Background(), q.ID, "fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}
	history, err := s.History(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if history.RefreshError == "" || len(history.Transactions) != 1 || history.Transactions[0].Hash != sent.Hash || history.Transactions[0].State != "submitted" {
		t.Fatalf("missing freshness warning or changed local record: %+v", history)
	}
}

func TestQuoteERC20InsufficientETHFunds(t *testing.T) {
	s, state := guardedFixture(t)
	state.balance = big.NewInt(0) // 0 ETH
	state.gas = 0                 // If estimateGas is invoked, it would fail

	req := &QuoteRequest{
		Action:    "transfer",
		Contract:  "0x4444444444444444444444444444444444444444",
		To:        "0x2222222222222222222222222222222222222222",
		AmountRaw: "1000000",
	}

	_, err := s.Quote(context.Background(), req)
	if !errors.Is(err, ErrInsufficientFunds) {
		t.Fatalf("expected ErrInsufficientFunds, got: %v", err)
	}
}
