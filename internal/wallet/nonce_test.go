package wallet

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type nonceRecoveryRPC struct {
	latest     atomic.Uint64
	finalized  atomic.Uint64
	fail       atomic.Bool
	wrongChain atomic.Bool
	sends      atomic.Int64
	receipts   atomic.Int64
	minedHash  atomic.Value
}

func nonceRecoveryService(t *testing.T) (*Service, *nonceRecoveryRPC, *SendResponse) {
	t.Helper()
	rpc := new(nonceRecoveryRPC)
	head := &types.Header{Number: big.NewInt(100), Difficulty: big.NewInt(0), GasLimit: 30000000, BaseFee: big.NewInt(1000000000)}
	c := mockRPC(t, func(method string, params json.RawMessage) any {
		switch method {
		case "eth_chainId":
			if rpc.wrongChain.Load() {
				return "0x1"
			}
			return "0xaa36a7"
		case "eth_getBlockByNumber":
			return head
		case "eth_maxPriorityFeePerGas":
			return "0x3b9aca00"
		case "eth_getTransactionCount":
			var args []string
			if err := json.Unmarshal(params, &args); err != nil {
				t.Error(err)
				return nil
			}
			if args[1] == "finalized" {
				if rpc.fail.Load() {
					return errors.New("finalized unsupported or unavailable")
				}
				return hexutil.EncodeUint64(rpc.finalized.Load())
			}
			return hexutil.EncodeUint64(rpc.latest.Load())
		case "eth_getBalance":
			return "0xde0b6b3a76400000"
		case "eth_estimateGas":
			return "0x5208"
		case "eth_getTransactionReceipt":
			rpc.receipts.Add(1)
			var args []string
			if err := json.Unmarshal(params, &args); err != nil {
				t.Error(err)
				return nil
			}
			if hash, ok := rpc.minedHash.Load().(string); ok && hash == args[0] {
				return &types.Receipt{Type: types.DynamicFeeTxType, Status: types.ReceiptStatusSuccessful, TxHash: common.HexToHash(hash), BlockNumber: head.Number, BlockHash: head.Hash(), GasUsed: 21000, CumulativeGasUsed: 21000, EffectiveGasPrice: big.NewInt(1000000000), Logs: []*types.Log{}}
			}
			return nil
		case "eth_getTransactionByHash":
			return nil
		case "eth_sendRawTransaction":
			var args []string
			if err := json.Unmarshal(params, &args); err != nil {
				t.Error(err)
				return nil
			}
			raw, _ := hexutil.Decode(args[0])
			var tx types.Transaction
			if err := tx.UnmarshalBinary(raw); err != nil {
				t.Error(err)
				return nil
			}
			rpc.sends.Add(1)
			return tx.Hash().Hex()
		}
		return nil
	})
	svc, err := NewService(c, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { svc.Close() })
	if _, err := svc.Create("test-password-123"); err != nil {
		t.Fatal(err)
	}
	quote, err := svc.Quote(context.Background(), nonceRecoveryRequest())
	if err != nil {
		t.Fatal(err)
	}
	original, err := svc.Send(context.Background(), quote.ID, "test-password-123")
	if err != nil {
		t.Fatal(err)
	}
	rpc.latest.Store(1)
	return svc, rpc, original
}

func nonceRecoveryRequest() *QuoteRequest {
	return &QuoteRequest{Action: "eth", To: "0x2222222222222222222222222222222222222222", Amount: "0.000001"}
}

func TestFinalizedNonceRecoveryPreservesUnknownAndRestart(t *testing.T) {
	ctx := context.Background()
	svc, rpc, original := nonceRecoveryService(t)
	before := svc.journal.FindByHash(original.Hash)
	rpc.finalized.Store(1)
	for round := range 2 {
		history, err := svc.History(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(history.Transactions) != 1 || !history.Transactions[0].NonceConsumed || !history.CanCreateTransaction || history.Transactions[0].Finalized || history.Transactions[0].ReplacedBy != "" {
			t.Fatalf("bad recovery history: %+v", history)
		}
		if after := svc.journal.FindByHash(original.Hash); *before != *after {
			t.Fatalf("recovery changed payment journal: %+v", after)
		}
		quote, err := svc.Quote(ctx, nonceRecoveryRequest())
		if err != nil || quote.Nonce != "1" {
			t.Fatalf("new quote: %+v, %v", quote, err)
		}
		if rpc.sends.Load() != 1 {
			t.Fatal("recovery rebroadcast a payment")
		}
		if len(svc.journal.RefreshItems()) != 1 {
			t.Fatal("unknown payment lost receipt lookup")
		}
		if round == 0 {
			if err := svc.Close(); err != nil {
				t.Fatal(err)
			}
			svc, err = NewService(svc.client, svc.walletDir, 2, 1)
			if err != nil {
				t.Fatal(err)
			}
			defer svc.Close()
			rpc.fail.Store(true)
			if _, err := svc.Quote(ctx, nonceRecoveryRequest()); !errors.Is(err, ErrTxInFlight) {
				t.Fatalf("restart reused old evidence: %v", err)
			}
			rpc.fail.Store(false)
		} else {
			sent, err := svc.Send(ctx, quote.ID, "test-password-123")
			if err != nil || sent.Hash == original.Hash {
				t.Fatalf("new send: %+v, %v", sent, err)
			}
			if rpc.sends.Load() != 2 {
				t.Fatal("unexpected broadcasts")
			}
			if _, err := svc.Quote(ctx, nonceRecoveryRequest()); !errors.Is(err, ErrTxInFlight) {
				t.Fatalf("new pending nonce not protected: %v", err)
			}
			history, err = svc.History(ctx)
			if err != nil || history.CanCreateTransaction {
				t.Fatalf("new pending history: %+v %v", history, err)
			}
		}
	}
	if rpc.receipts.Load() < 2 {
		t.Fatal("stopped checking original receipt")
	}
	// A receipt found later still updates the original result through History.
	rpc.minedHash.Store(original.Hash)
	history, err := svc.History(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range history.Transactions {
		if item.Hash == original.Hash && (item.State != "succeeded" || item.NonceConsumed || !item.Finalized) {
			t.Fatalf("late receipt hidden: %+v", item)
		}
	}
}

func TestFinalizedNonceRecoveryFailsClosed(t *testing.T) {
	for _, name := range []string{"latest only", "RPC unsupported", "wrong network", "canceled", "stale pending nonce"} {
		t.Run(name, func(t *testing.T) {
			svc, rpc, _ := nonceRecoveryService(t)
			rpc.finalized.Store(1)
			ctx := context.Background()
			expected := ErrTxInFlight
			switch name {
			case "latest only":
				rpc.finalized.Store(0)
			case "RPC unsupported":
				rpc.fail.Store(true)
			case "wrong network":
				rpc.wrongChain.Store(true)
			case "canceled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "stale pending nonce":
				rpc.latest.Store(0)
				expected = ErrNonceMismatch
			}
			if _, err := svc.Quote(ctx, nonceRecoveryRequest()); !errors.Is(err, expected) {
				t.Fatalf("unsafe quote: %v", err)
			}
			if rpc.sends.Load() != 1 {
				t.Fatal("unexpected broadcast")
			}
		})
	}
}

func TestFinalizedNonceSendRechecksEvidenceAndSerializes(t *testing.T) {
	svc, rpc, _ := nonceRecoveryService(t)
	rpc.finalized.Store(1)
	ctx := context.Background()
	first, err := svc.Quote(ctx, nonceRecoveryRequest())
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Quote(ctx, nonceRecoveryRequest())
	if err != nil {
		t.Fatal(err)
	}
	rpc.fail.Store(true)
	if _, err := svc.Send(ctx, first.ID, "test-password-123"); !errors.Is(err, ErrTxInFlight) {
		t.Fatalf("send reused quote evidence: %v", err)
	}
	rpc.fail.Store(false)
	rpc.finalized.Store(2)
	if _, err := svc.Send(ctx, first.ID, "test-password-123"); !errors.Is(err, ErrNonceMismatch) {
		t.Fatalf("send reused consumed quote nonce: %v", err)
	}
	rpc.finalized.Store(1)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, id := range []string{first.ID, second.ID} {
		wg.Go(func() { _, err := svc.Send(ctx, id, "test-password-123"); results <- err })
	}
	wg.Wait()
	close(results)
	success, blocked := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrTxInFlight) {
			blocked++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || blocked != 1 || rpc.sends.Load() != 2 {
		t.Fatalf("concurrent sends: %d success, %d blocked, %d broadcasts", success, blocked, rpc.sends.Load())
	}
}

func TestFinalizedNonceEvidenceScope(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	account := crypto.PubkeyToAddress(key.PublicKey)
	to := common.Address{2}
	makeRecord := func(chainID int64, nonce uint64, value int64) *JournalRecord {
		tx, err := types.SignNewTx(key, types.LatestSignerForChainID(big.NewInt(chainID)), &types.DynamicFeeTx{ChainID: big.NewInt(chainID), Nonce: nonce, To: &to, Value: big.NewInt(value), Gas: 21000, GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(2)})
		if err != nil {
			t.Fatal(err)
		}
		raw, err := tx.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		return &JournalRecord{Hash: TransactionHash(tx.Hash().Hex()), Nonce: nonce, SignedRaw: hexutil.Encode(raw), State: JournalSubmitted}
	}
	original := makeRecord(chain.SepoliaID, 0, 1)
	replacement := makeRecord(chain.SepoliaID, 0, 2)
	proof := &finalizedNonce{account: account, chainID: chain.SepoliaID, nonce: 1}
	jm := &JournalManager{records: []*JournalRecord{original, replacement}}
	if jm.hasInFlightTx(proof) {
		t.Fatal("same nonce attempts did not unlock together")
	}
	for _, name := range []string{"account", "chain", "next nonce", "wrong stored nonce", "wrong hash", "invalid signature"} {
		t.Run(name, func(t *testing.T) {
			candidate := *original
			evidence := *proof
			switch name {
			case "account":
				evidence.account = common.Address{9}
			case "chain":
				evidence.chainID = chain.BaseSepoliaID
			case "next nonce":
				candidate = *makeRecord(chain.SepoliaID, 1, 1)
			case "wrong stored nonce":
				candidate = *makeRecord(chain.SepoliaID, 2, 1)
				candidate.Nonce = 0
			case "wrong hash":
				candidate.Hash = replacement.Hash
			case "invalid signature":
				candidate.SignedRaw = "0x00"
			}
			manager := &JournalManager{records: []*JournalRecord{&candidate}}
			if !manager.hasInFlightTx(&evidence) {
				t.Fatal("unrelated/invalid evidence released guard")
			}
		})
	}
	// History lookup cannot consume or install evidence for a later operation.
	if !jm.HasInFlightTx() {
		t.Fatal("ephemeral evidence leaked into journal")
	}
}
