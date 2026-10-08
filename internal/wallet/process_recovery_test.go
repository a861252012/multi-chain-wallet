package wallet

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// Kill the signing process after the mock receives the bytes but before it replies.
// No shutdown handlers run; only the parent's temporary directory survives.
func TestProcessKillRestartReusesRaw(t *testing.T) {
	if dir := os.Getenv("MULTI_CHAIN_WALLET_RECOVERY_CHILD_DIR"); dir != "" {
		client := mockRPC(t, func(method string, params json.RawMessage) any {
			switch method {
			case "eth_chainId":
				return "0xaa36a7"
			case "eth_getBlockByNumber":
				return &types.Header{Number: big.NewInt(100), Difficulty: big.NewInt(0), GasLimit: 30000000, BaseFee: big.NewInt(1000000000)}
			case "eth_maxPriorityFeePerGas":
				return "0x3b9aca00"
			case "eth_getTransactionCount":
				return "0x0"
			case "eth_getBalance":
				return "0x56bc75e2d63100000"
			case "eth_estimateGas":
				return "0x5208"
			case "eth_sendRawTransaction":
				if err := os.WriteFile(filepath.Join(dir, "received.json"), params, 0600); err != nil {
					t.Error(err)
					return nil
				}
				if err := syscall.Kill(os.Getpid(), syscall.SIGKILL); err != nil {
					t.Error(err)
				}
				return nil
			default:
				t.Errorf("unexpected RPC %s", method)
				return nil
			}
		})
		svc, err := NewService(client, dir, 2, 1)
		if err != nil {
			t.Fatal(err)
		}
		defer svc.Close()
		if _, err := svc.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "fixture-password-123"); err != nil {
			t.Fatal(err)
		}
		quote := ethQuote(t, svc)
		_, err = svc.Send(context.Background(), quote.ID, "fixture-password-123")
		t.Fatalf("child survived broadcast: %v", err)
	}
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestProcessKillRestartReusesRaw$", "-test.count=1")
	child.Env = append(os.Environ(), "MULTI_CHAIN_WALLET_RECOVERY_CHILD_DIR="+dir, "TMPDIR="+dir)
	output, err := child.CombinedOutput()
	exitErr, killed := errors.AsType[*exec.ExitError](err)
	if ctx.Err() != nil || !killed {
		t.Fatalf("expected killed child, got %v: %s", err, output)
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
		t.Fatalf("unexpected child exit: %v: %s", err, output)
	}
	received, err := os.ReadFile(filepath.Join(dir, "received.json"))
	if err != nil {
		t.Fatal(err)
	}
	var original []string
	if err := json.Unmarshal(received, &original); err != nil {
		t.Fatalf("decode mock broadcast evidence: %v", err)
	}
	if len(original) != 1 {
		t.Fatalf("expected one mock broadcast payload, got %d", len(original))
	}
	var raws []string
	var rawsMu sync.Mutex
	rawSnapshot := func() []string {
		rawsMu.Lock()
		defer rawsMu.Unlock()
		return append([]string(nil), raws...)
	}
	client := mockRPC(t, func(method string, params json.RawMessage) any {
		switch method {
		case "eth_chainId":
			return "0xaa36a7"
		case "eth_getTransactionCount":
			var args []string
			if err := json.Unmarshal(params, &args); err != nil || len(args) != 2 || args[1] != "finalized" {
				t.Errorf("unexpected recovery nonce query: %s", params)
				return nil
			}
			return "0x0" // The interrupted transaction's nonce is still unconsumed.
		case "eth_sendRawTransaction":
			var args []string
			if err := json.Unmarshal(params, &args); err != nil || len(args) != 1 {
				t.Error("invalid broadcast")
				return nil
			}
			rawsMu.Lock()
			raws = append(raws, args[0])
			rawsMu.Unlock()
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
		default:
			t.Errorf("unexpected recovery RPC %s", method)
			return nil
		}
	})
	restarted, err := NewService(client, dir, 2, 1)
	if err != nil {
		t.Fatalf("reopen after SIGKILL: %v", err)
	}
	defer restarted.Close()
	records := restarted.journal.NonceRecords(0)
	if len(records) != 1 {
		t.Fatalf("expected one durable record, got %d", len(records))
	}
	record := records[0]
	if record.State != "pending" || record.SignedRaw != original[0] {
		t.Fatalf("pending journal mismatch after restart: state=%q hash=%q", record.State, record.Hash)
	}
	if _, err := restarted.Quote(context.Background(), &QuoteRequest{Action: "eth", To: string(record.To), Amount: "1"}); !errors.Is(err, ErrTxInFlight) {
		t.Fatalf("pending nonce not protected: %v", err)
	}
	duplicate, err := restarted.Send(context.Background(), string(record.QuoteID), "")
	if err != nil {
		t.Fatalf("same quote recovery failed: %v", err)
	}
	if duplicate == nil {
		t.Fatal("same quote recovery returned no response")
	}
	if captured := rawSnapshot(); duplicate.Hash != string(record.Hash) || len(captured) != 0 {
		t.Fatalf("same quote recovery changed transaction: hash=%q want=%q broadcasts=%d", duplicate.Hash, record.Hash, len(captured))
	}
	result, err := restarted.Retry(context.Background(), string(record.Hash))
	if err != nil {
		t.Fatalf("retry recovered transaction: %v", err)
	}
	if result == nil {
		t.Fatal("retry returned no response")
	}
	captured := rawSnapshot()
	if result.Hash != string(record.Hash) || result.State != "submitted" || len(captured) != 1 {
		t.Fatalf("retry changed transaction identity or broadcast count: hash=%q want=%q state=%q broadcasts=%d", result.Hash, record.Hash, result.State, len(captured))
	}
	if captured[0] != original[0] {
		t.Fatal("retry changed signed transaction bytes")
	}
	if err := restarted.Close(); err != nil {
		t.Fatal(err)
	}
	journal, err := NewJournalManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	saved := journal.FindByHash(string(record.Hash))
	if saved == nil || saved.State != "submitted" || saved.SignedRaw != original[0] {
		state := "<missing>"
		if saved != nil {
			state = string(saved.State)
		}
		t.Fatalf("retry result was not persisted: state=%q hash=%q", state, record.Hash)
	}
}
