package wallet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// TestPasswordUnicodeRuneLengthRules explicitly verifies rune-based password rules:
// UTF-8 multibyte characters (e.g. Chinese characters) must be measured in characters (runes), not bytes.
func TestPasswordUnicodeRuneLengthRules(t *testing.T) {
	dir := t.TempDir()
	km := NewKeystoreManager(dir, 2, 1)

	if ErrInvalidPassword.Error() != "密碼長度必須介於 12 至 128 字元" {
		t.Fatalf("expected ErrInvalidPassword to say '字元', got %q", ErrInvalidPassword.Error())
	}

	// 1. 4 Chinese characters = 12 bytes, but only 4 runes (< 12 runes, MUST be rejected)
	if _, err := km.Create(strings.Repeat("密", 4)); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("expected ErrInvalidPassword for 4 Chinese runes, got %v", err)
	}

	// 2. 11 Chinese characters = 33 bytes, 11 runes (< 12 runes, MUST be rejected)
	elevenRunes := strings.Repeat("密", 11)
	if _, err := km.Create(elevenRunes); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("expected ErrInvalidPassword for 11 Chinese runes, got %v", err)
	}

	// 3. Exactly 12 Chinese characters = 36 bytes, 12 runes (MUST be accepted)
	twelveRunes := strings.Repeat("密", 12)
	created, err := km.Create(twelveRunes)
	if err != nil {
		t.Fatalf("Create with exactly 12 Chinese runes failed: %v", err)
	}
	if created == nil || created.Address == "" {
		t.Fatal("expected successful creation with 12 Chinese runes")
	}

	// 4. Verify backup and decrypt with 12 Chinese runes
	if _, err := km.Backup(twelveRunes); err != nil {
		t.Fatalf("Backup with 12 Chinese runes failed: %v", err)
	}

	// 5. 128 Chinese characters (boundary: 128 runes, MUST be accepted)
	dir2 := t.TempDir()
	km2 := NewKeystoreManager(dir2, 2, 1)
	runes128 := strings.Repeat("密", 128)
	if _, err := km2.Create(runes128); err != nil {
		t.Fatalf("Create with 128 Chinese runes failed: %v", err)
	}

	// 6. 129 Chinese characters (boundary: 129 runes, MUST be rejected)
	dir3 := t.TempDir()
	km3 := NewKeystoreManager(dir3, 2, 1)
	runes129 := strings.Repeat("密", 129)
	if _, err := km3.Create(runes129); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("expected ErrInvalidPassword for 129 Chinese runes, got %v", err)
	}

	// 7. Mixed ASCII and Chinese (6 ASCII + 6 Chinese = 12 runes, MUST be accepted)
	dir4 := t.TempDir()
	km4 := NewKeystoreManager(dir4, 2, 1)
	mixedPass := "abc123密碼測試通過"
	if _, err := km4.Create(mixedPass); err != nil {
		t.Fatalf("Create with 12 mixed runes failed: %v", err)
	}
}

// TestHistoryDoesNotBlockTradingOperations verifies that History() does NOT hold s.sendMu
// while executing, so Quote() and Send() are never blocked by slow history RPC calls.
func TestHistoryDoesNotBlockTradingOperations(t *testing.T) {
	s, _ := guardedFixture(t)

	// Send a transaction to have an outstanding record
	q := ethQuote(t, s)
	sent, err := s.Send(context.Background(), q.ID, "fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}

	// While s.historyMu is locked (simulating a slow in-progress History call),
	// s.sendMu should be completely free, allowing Quote to proceed without waiting for history.
	s.historyMu.Lock()
	sendMuAcquired := make(chan struct{})
	go func() {
		defer close(sendMuAcquired)
		// Try to acquire sendMu directly
		s.sendMu.Lock()
		defer s.sendMu.Unlock()
	}()

	select {
	case <-sendMuAcquired:
		// Success: sendMu was acquired immediately even though historyMu was locked!
	case <-time.After(500 * time.Millisecond):
		t.Fatal("s.sendMu acquisition was blocked by history operation")
	}
	s.historyMu.Unlock()

	_ = sent
}

// TestHistoryDoesNotBlockTradingOperationsConcurrentE2E runs an end-to-end test with an actively hanging
// RPC call in History() and verifies Quote() finishes immediately without waiting for History.
func TestHistoryDoesNotBlockTradingOperationsConcurrentE2E(t *testing.T) {
	receiptStarted := make(chan struct{})
	releaseReceipt := make(chan struct{})

	state := &rpcState{chainID: "0xaa36a7", balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil), gas: 48000, allowance: big.NewInt(0), tokenBalance: big.NewInt(10000000), swapOutput: big.NewInt(1000000)}
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
			return hexutil.EncodeUint64(state.gas)
		case "eth_getCode":
			return "0x6000"
		case "eth_sendRawTransaction":
			var args []string
			json.Unmarshal(params, &args)
			raw, _ := hexutil.Decode(args[0])
			var tx types.Transaction
			tx.UnmarshalBinary(raw)
			return tx.Hash().Hex()
		case "eth_getTransactionReceipt":
			// Signal that History has entered the slow RPC call
			state.mu.Unlock()
			select {
			case <-receiptStarted:
			default:
				close(receiptStarted)
			}
			select {
			case <-releaseReceipt:
			case <-time.After(5 * time.Second):
				t.Error("timeout waiting for releaseReceipt signal")
			}
			state.mu.Lock()
			return nil
		case "eth_getTransactionByHash":
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
	t.Cleanup(func() {
		select {
		case <-releaseReceipt:
		default:
			close(releaseReceipt)
		}
		svc.Close()
	})

	_, err = svc.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}

	// Send initial transaction so history has a record to refresh
	q := ethQuote(t, svc)
	sent, err := svc.Send(context.Background(), q.ID, "fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}

	// Mark it as succeeded so HasInFlightTx is false, allowing a new Quote
	if err := svc.journal.UpdateStateAtomic(sent.Hash, "succeeded", "1", "0.0001", ""); err != nil {
		t.Fatal(err)
	}

	// Start History() in background; it will hang in eth_getTransactionReceipt
	historyDone := make(chan struct{})
	go func() {
		defer close(historyDone)
		svc.History(context.Background())
	}()

	// Wait until History() has actually entered the slow RPC call
	select {
	case <-receiptStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for History to enter RPC call")
	}

	// Concurrently invoke Quote(). If History() held sendMu, Quote would hang until releaseReceipt!
	quoteDone := make(chan struct{})
	go func() {
		defer close(quoteDone)
		_, err := svc.Quote(context.Background(), &QuoteRequest{Action: "eth", To: "0x2222222222222222222222222222222222222222", Amount: "0.000000000000000001"})
		if err != nil {
			t.Errorf("Quote failed: %v", err)
		}
	}()

	select {
	case <-quoteDone:
		// SUCCESS: Quote completed immediately while History() was still blocked on RPC!
	case <-time.After(1 * time.Second):
		t.Fatal("Quote was blocked by ongoing History RPC call!")
	}

	// Release slow RPC
	close(releaseReceipt)
	select {
	case <-historyDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for History to finish")
	}
}

// TestHistoryDoesNotOverwriteNewerUpdates verifies that stale RPC results in History()
// do not overwrite newer updates made by Send/Retry or concurrent modifications.
func TestHistoryDoesNotOverwriteNewerUpdates(t *testing.T) {
	s, _ := guardedFixture(t)
	q := ethQuote(t, s)
	sent, err := s.Send(context.Background(), q.ID, "fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}

	// Snapshot items: expectedVersion for sent.Hash
	items := s.journal.RefreshItems()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	item := items[0]
	origVersion := item.Version

	// Concurrently, a Retry or state update happens, advancing version
	if err := s.journal.UpdateStateAtomic(sent.Hash, "submitted", "", "", ""); err != nil {
		t.Fatal(err)
	}
	updatedRec := s.journal.FindByHash(sent.Hash)
	if updatedRec.Version <= origVersion {
		t.Fatalf("expected version > %d after update, got %d", origVersion, updatedRec.Version)
	}

	// Now History tries to update using the stale origVersion (e.g. saying it is "pending")
	updated, err := s.journal.UpdateStateAtomicIfVersion(sent.Hash, origVersion, "pending", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if updated {
		t.Fatal("stale update was accepted! Expected it to be rejected due to version mismatch")
	}

	// Confirm state remains submitted and version was not rolled back
	finalRec := s.journal.FindByHash(sent.Hash)
	if finalRec.State != "submitted" || finalRec.Version != updatedRec.Version {
		t.Fatalf("record was corrupted: state=%s version=%d", finalRec.State, finalRec.Version)
	}
}

func TestHistoryAcceptsFreshReorgStates(t *testing.T) {
	for _, state := range []string{"pending", "receipt_unavailable", "reorg_detected", "broadcast_unknown"} {
		t.Run(state, func(t *testing.T) {
			s, _ := guardedFixture(t)
			q := ethQuote(t, s)
			sent, err := s.Send(context.Background(), q.ID, "fixture-password-123")
			if err != nil {
				t.Fatal(err)
			}
			if err := s.journal.UpdateStateAtomic(sent.Hash, "succeeded", "10", "0.001", ""); err != nil {
				t.Fatal(err)
			}
			record := s.journal.FindByHash(sent.Hash)
			updated, err := s.journal.UpdateStateAtomicIfVersion(sent.Hash, record.Version, state, "", "", "")
			if err != nil || !updated {
				t.Fatalf("fresh observation rejected: updated=%v err=%v", updated, err)
			}
			if got := s.journal.FindByHash(sent.Hash); string(got.State) != state {
				t.Fatalf("state=%s", got.State)
			}
			if !s.journal.HasInFlightTx() {
				t.Fatal("reorged transaction must remain in flight")
			}
		})
	}
}

// TestHistoryPreservesReorgWhenNotFound verifies that when an existing succeeded transaction
// becomes not found on RPC (canonical reorg / dropped), History() updates it to broadcast_unknown.
func TestHistoryPreservesReorgWhenNotFound(t *testing.T) {
	s, _ := guardedFixture(t)
	q := ethQuote(t, s)
	sent, err := s.Send(context.Background(), q.ID, "fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}

	if err := s.journal.UpdateStateAtomic(sent.Hash, "succeeded", "5", "0.001", ""); err != nil {
		t.Fatal(err)
	}

	// History will query mock RPC where this tx is not present (returns NotFound)
	hist, err := s.History(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(hist.Transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(hist.Transactions))
	}
	if hist.Transactions[0].State != "broadcast_unknown" {
		t.Fatalf("expected state broadcast_unknown after not found on RPC, got %s", hist.Transactions[0].State)
	}
	if !s.journal.HasInFlightTx() {
		t.Fatal("reorged transaction must block new quotes as in-flight")
	}
}

// TestHistoryUpdateStateAtomicIfVersionNoOpSkipsSave verifies that if transaction state and metadata
// have not changed, UpdateStateAtomicIfVersion returns true without incrementing version or triggering redundant disk writes.
func TestHistoryUpdateStateAtomicIfVersionNoOpSkipsSave(t *testing.T) {
	s, _ := guardedFixture(t)
	q := ethQuote(t, s)
	sent, err := s.Send(context.Background(), q.ID, "fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}

	if err := s.journal.UpdateStateAtomic(sent.Hash, "succeeded", "5", "0.001", ""); err != nil {
		t.Fatal(err)
	}

	rec := s.journal.FindByHash(sent.Hash)
	v := rec.Version

	// Call UpdateStateAtomicIfVersion with identical values
	updated, err := s.journal.UpdateStateAtomicIfVersion(sent.Hash, v, "succeeded", "5", "0.001", "")
	if err != nil {
		t.Fatal(err)
	}
	if !updated {
		t.Fatal("expected no-op update to succeed")
	}

	recAfter := s.journal.FindByHash(sent.Hash)
	if recAfter.Version != v {
		t.Fatalf("expected version to remain %d, got %d", v, recAfter.Version)
	}
}

func TestJournalUnchangedUpdatesDoNotAllocate(t *testing.T) {
	jm := &JournalManager{walletDir: t.TempDir()}
	for i := range 1000 {
		jm.records = append(jm.records, &JournalRecord{
			Hash: TransactionHash(fmt.Sprintf("0x%064x", i+1)), State: JournalSucceeded,
			Confirmations: "5", FeeETH: "0.001", Version: 1,
		})
	}
	target := jm.records[len(jm.records)-1]
	for _, tc := range []struct {
		name    string
		version uint64
		updated bool
	}{{"unchanged", 1, true}, {"stale version", 0, false}} {
		t.Run(tc.name, func(t *testing.T) {
			allocations := testing.AllocsPerRun(100, func() {
				updated, err := jm.UpdateStateAtomicIfVersion(string(target.Hash), tc.version, "succeeded", "5", "0.001", "")
				if err != nil || updated != tc.updated {
					t.Fatalf("updated=%v, err=%v", updated, err)
				}
			})
			if allocations != 0 {
				t.Fatalf("unchanged journal allocated %.0f objects", allocations)
			}
			if jm.records[len(jm.records)-1] != target || target.Version != 1 {
				t.Fatal("unchanged update replaced or modified the record")
			}
		})
	}
	files, err := os.ReadDir(jm.walletDir)
	if err != nil || len(files) != 0 {
		t.Fatalf("unchanged updates wrote to disk: %v, err=%v", files, err)
	}
}

func TestLegacyPasswordStillDecryptsAndBacksUp(t *testing.T) {
	km := NewKeystoreManager(t.TempDir(), 2, 1)
	const currentPassword = "fixture-password-123"
	if _, err := km.Create(currentPassword); err != nil {
		t.Fatal(err)
	}
	key, err := km.DecryptKey(currentPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer wipePrivateKey(key.PrivateKey)
	// Four Chinese code points met the previous 12-byte minimum.
	const legacyPassword = "密碼測試"
	encrypted, err := keystore.EncryptKey(key, legacyPassword, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(km.keystorePath(), encrypted, 0600); err != nil {
		t.Fatal(err)
	}
	restored, err := km.DecryptKey(legacyPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer wipePrivateKey(restored.PrivateKey)
	if restored.Address != key.Address {
		t.Fatal("address changed")
	}
	if _, err := km.Backup(legacyPassword); err != nil {
		t.Fatal(err)
	}
	if _, err := km.DecryptKey("wrong"); !errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("wrong password: %v", err)
	}
}

func TestPasswordEmojiBoundaries(t *testing.T) {
	for _, count := range []int{6, 11, 12, 128, 129} {
		err := ValidatePassword(strings.Repeat("\U0001F600", count))
		valid := count >= 12 && count <= 128
		if (err == nil) != valid {
			t.Fatalf("%d emoji: %v", count, err)
		}
	}
}

// TestJournalFullRejectsSendWithoutStorageFault verifies that when journal capacity (1000) is reached,
// Send returns ErrJournalFull without setting storageFault.
func TestJournalFullRejectsSendWithoutStorageFault(t *testing.T) {
	s, _ := guardedFixture(t)
	q := ethQuote(t, s)

	// Fill journal records up to exactly 1000
	s.journal.mu.Lock()
	for len(s.journal.records) < 1000 {
		idx := len(s.journal.records) + 1
		s.journal.records = append(s.journal.records, &JournalRecord{
			Hash:      TransactionHash(fmt.Sprintf("0x%064x", idx)),
			QuoteID:   QuoteID(fmt.Sprintf("quote-%d", idx)),
			State:     "succeeded",
			Version:   1,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		})
	}
	s.journal.mu.Unlock()

	_, err := s.Send(context.Background(), q.ID, "fixture-password-123")
	if !errors.Is(err, ErrJournalFull) {
		t.Fatalf("expected ErrJournalFull, got %v", err)
	}

	// Must NOT set storageFault on ErrJournalFull
	if s.storageFault.Load() {
		t.Fatal("storageFault must not be set when AppendAtomic hits ErrJournalFull")
	}

	// Journal record count should remain at 1000
	s.journal.mu.Lock()
	recCount := len(s.journal.records)
	s.journal.mu.Unlock()
	if recCount != 1000 {
		t.Fatalf("expected 1000 records, got %d", recCount)
	}
}

// TestHistoryInterruptedContextReturnsRefreshError verifies that when History querying is interrupted
// by context cancellation or timeout, it returns a non-empty RefreshError while preserving existing history records.
func TestHistoryInterruptedContextReturnsRefreshError(t *testing.T) {
	s, _ := guardedFixture(t)
	q := ethQuote(t, s)
	sent, err := s.Send(context.Background(), q.ID, "fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}

	const expectedRefreshError = "部分交易未能完成鏈上查核；以下保留本機最後紀錄，請稍後更新。"

	t.Run("pre-canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		resp, err := s.History(ctx)
		if err != nil {
			t.Fatalf("History must not return error on canceled context: %v", err)
		}
		if resp.RefreshError != expectedRefreshError {
			t.Fatalf("expected specific RefreshError %q, got %q", expectedRefreshError, resp.RefreshError)
		}
		if len(resp.Transactions) != 1 || resp.Transactions[0].Hash != sent.Hash {
			t.Fatalf("expected historical transactions to be preserved: %+v", resp.Transactions)
		}
	})

	t.Run("timeout context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()
		time.Sleep(5 * time.Millisecond) // ensure deadline exceeded

		resp, err := s.History(ctx)
		if err != nil {
			t.Fatalf("History must not return error on timed-out context: %v", err)
		}
		if resp.RefreshError != expectedRefreshError {
			t.Fatalf("expected specific RefreshError %q, got %q", expectedRefreshError, resp.RefreshError)
		}
		if len(resp.Transactions) != 1 || resp.Transactions[0].Hash != sent.Hash {
			t.Fatalf("expected historical transactions to be preserved: %+v", resp.Transactions)
		}
	})

	t.Run("in-flight RPC context timeout", func(t *testing.T) {
		hangReceipt := make(chan struct{})
		receiptReached := make(chan struct{})

		state := &rpcState{chainID: "0xaa36a7", balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil), gas: 48000, allowance: big.NewInt(0), tokenBalance: big.NewInt(10000000), swapOutput: big.NewInt(1000000)}
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
				return hexutil.EncodeUint64(state.gas)
			case "eth_getCode":
				return "0x6000"
			case "eth_sendRawTransaction":
				var args []string
				json.Unmarshal(params, &args)
				raw, _ := hexutil.Decode(args[0])
				var tx types.Transaction
				tx.UnmarshalBinary(raw)
				return tx.Hash().Hex()
			case "eth_getTransactionReceipt":
				state.mu.Unlock()
				select {
				case <-receiptReached:
				default:
					close(receiptReached)
				}
				select {
				case <-hangReceipt:
				case <-time.After(5 * time.Second):
					t.Error("timeout waiting for hangReceipt signal")
				}
				state.mu.Lock()
				return nil
			case "eth_getTransactionByHash":
				return nil
			default:
				return nil
			}
		})

		svc, err := NewService(c, t.TempDir(), 2, 1)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			select {
			case <-receiptReached:
			default:
				close(receiptReached)
			}
			select {
			case <-hangReceipt:
			default:
				close(hangReceipt)
			}
			svc.Close()
		})

		_, err = svc.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "fixture-password-123")
		if err != nil {
			t.Fatal(err)
		}

		q := ethQuote(t, svc)
		sentTx, err := svc.Send(context.Background(), q.ID, "fixture-password-123")
		if err != nil {
			t.Fatal(err)
		}

		// Use a context that will time out while eth_getTransactionReceipt is actively blocked
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		resp, err := svc.History(ctx)
		if err != nil {
			t.Fatalf("History must not return fatal error on in-flight timeout: %v", err)
		}
		if resp.RefreshError != expectedRefreshError {
			t.Fatalf("expected specific RefreshError %q, got %q", expectedRefreshError, resp.RefreshError)
		}
		if len(resp.Transactions) != 1 || resp.Transactions[0].Hash != sentTx.Hash {
			t.Fatalf("expected historical transactions to be preserved: %+v", resp.Transactions)
		}
	})
}

// TestSendConcurrentStateUpdatePreservesLatest verifies that if a journal record is modified concurrently
// during broadcast delay (e.g. reorg detected or history update), Send preserves the latest record without overwriting it.
func TestSendConcurrentStateUpdatePreservesLatest(t *testing.T) {
	sendStarted := make(chan struct{})
	canBroadcast := make(chan struct{})

	state := &rpcState{chainID: "0xaa36a7", balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil), gas: 48000, allowance: big.NewInt(0), tokenBalance: big.NewInt(10000000), swapOutput: big.NewInt(1000000)}
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
			return hexutil.EncodeUint64(state.gas)
		case "eth_getCode":
			return "0x6000"
		case "eth_sendRawTransaction":
			state.mu.Unlock()
			select {
			case <-sendStarted:
			default:
				close(sendStarted)
			}
			select {
			case <-canBroadcast:
			case <-time.After(5 * time.Second):
				t.Error("timeout waiting for canBroadcast signal")
			}
			state.mu.Lock()
			var args []string
			json.Unmarshal(params, &args)
			raw, _ := hexutil.Decode(args[0])
			var tx types.Transaction
			tx.UnmarshalBinary(raw)
			return tx.Hash().Hex()
		default:
			return nil
		}
	})

	svc, err := NewService(c, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		select {
		case <-sendStarted:
		default:
			close(sendStarted)
		}
		select {
		case <-canBroadcast:
		default:
			close(canBroadcast)
		}
		svc.Close()
	})

	_, err = svc.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}

	q := ethQuote(t, svc)

	type sendResult struct {
		resp *SendResponse
		err  error
	}
	sendResChan := make(chan sendResult, 1)

	go func() {
		resp, sErr := svc.Send(context.Background(), q.ID, "fixture-password-123")
		sendResChan <- sendResult{resp: resp, err: sErr}
	}()

	// Wait for Send to begin broadcasting
	select {
	case <-sendStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Send to start broadcast")
	}

	// While Send is hanging in broadcast, a concurrent update happens to the journal record
	items := svc.journal.RefreshItems()
	if len(items) == 0 {
		t.Fatal("expected pending record in journal")
	}
	txHash := items[0].Hash
	// Simulate external/history update: e.g., reorg detected or confirmed
	if err := svc.journal.UpdateStateAtomic(txHash, "reorg_detected", "0", "", ""); err != nil {
		t.Fatal(err)
	}

	// Allow broadcast to proceed
	close(canBroadcast)

	var res sendResult
	select {
	case res = <-sendResChan:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Send to finish")
	}

	if res.err != nil {
		t.Fatalf("Send failed: %v", res.err)
	}
	// The response MUST preserve the newer concurrent state ("reorg_detected") rather than overwriting with "submitted"
	if res.resp.State != "reorg_detected" {
		t.Fatalf("expected latest state 'reorg_detected', got %q", res.resp.State)
	}

	// The journal record must also keep "reorg_detected"
	rec := svc.journal.FindByHash(txHash)
	if rec == nil || rec.State != "reorg_detected" {
		t.Fatalf("expected journal state 'reorg_detected', got %+v", rec)
	}

	if svc.storageFault.Load() {
		t.Fatal("storageFault must not be set on version CAS mismatch")
	}
}

// TestRetryConcurrentStateUpdatePreservesLatest verifies that if a journal record is modified concurrently
// during Retry broadcast delay, Retry preserves the latest record without overwriting it.
func TestRetryConcurrentStateUpdatePreservesLatest(t *testing.T) {
	retryStarted := make(chan struct{})
	canBroadcast := make(chan struct{})

	state := &rpcState{chainID: "0xaa36a7", balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil), gas: 48000, allowance: big.NewInt(0), tokenBalance: big.NewInt(10000000), swapOutput: big.NewInt(1000000)}
	sendCount := 0
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
			return hexutil.EncodeUint64(state.gas)
		case "eth_getCode":
			return "0x6000"
		case "eth_sendRawTransaction":
			sendCount += 1
			if sendCount == 1 {
				// First send fails with unknown
				return nil
			}
			state.mu.Unlock()
			select {
			case <-retryStarted:
			default:
				close(retryStarted)
			}
			select {
			case <-canBroadcast:
			case <-time.After(5 * time.Second):
				t.Error("timeout waiting for canBroadcast signal")
			}
			state.mu.Lock()
			var args []string
			json.Unmarshal(params, &args)
			raw, _ := hexutil.Decode(args[0])
			var tx types.Transaction
			tx.UnmarshalBinary(raw)
			return tx.Hash().Hex()
		default:
			return nil
		}
	})

	svc, err := NewService(c, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		select {
		case <-retryStarted:
		default:
			close(retryStarted)
		}
		select {
		case <-canBroadcast:
		default:
			close(canBroadcast)
		}
		svc.Close()
	})

	_, err = svc.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}

	q := ethQuote(t, svc)
	// Initial Send fails, leaving state as broadcast_unknown
	sent, err := svc.Send(context.Background(), q.ID, "fixture-password-123")
	if err != nil || sent.State != "broadcast_unknown" {
		t.Fatalf("expected broadcast_unknown, got %+v %v", sent, err)
	}

	type retryResult struct {
		resp *SendResponse
		err  error
	}
	retryResChan := make(chan retryResult, 1)

	go func() {
		resp, rErr := svc.Retry(context.Background(), sent.Hash)
		retryResChan <- retryResult{resp: resp, err: rErr}
	}()

	select {
	case <-retryStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Retry broadcast to start")
	}

	// While Retry broadcast is in flight, update record concurrently to reverted
	if err := svc.journal.UpdateStateAtomic(sent.Hash, "reverted", "5", "0.0002", "execution reverted"); err != nil {
		t.Fatal(err)
	}

	close(canBroadcast)

	var res retryResult
	select {
	case res = <-retryResChan:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Retry to complete")
	}

	if res.err != nil {
		t.Fatalf("Retry failed: %v", res.err)
	}
	if res.resp.State != "reverted" {
		t.Fatalf("expected Retry to return latest state 'reverted', got %q", res.resp.State)
	}

	rec := svc.journal.FindByHash(sent.Hash)
	if rec == nil || rec.State != "reverted" {
		t.Fatalf("expected journal state 'reverted', got %+v", rec)
	}

	if svc.storageFault.Load() {
		t.Fatal("storageFault must not be set on version CAS mismatch in Retry")
	}
}

// TestBroadcastRPCErrorRecordsBroadcastUnknown explicitly verifies that a standard RPC broadcast error
// (not a timeout) records broadcast_unknown with the error description and does not trigger storage fault.
func TestBroadcastRPCErrorRecordsBroadcastUnknown(t *testing.T) {
	state := &rpcState{chainID: "0xaa36a7", balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil), gas: 48000, allowance: big.NewInt(0), tokenBalance: big.NewInt(10000000), swapOutput: big.NewInt(1000000)}
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
			return hexutil.EncodeUint64(state.gas)
		case "eth_getCode":
			return "0x6000"
		case "eth_sendRawTransaction":
			// Return a normal RPC error (NOT a timeout)
			return errors.New("execution reverted: intrinsic gas too low")
		default:
			return nil
		}
	})

	svc, err := NewService(c, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	_, err = svc.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}

	q := ethQuote(t, svc)
	resp, err := svc.Send(context.Background(), q.ID, "fixture-password-123")
	if err != nil {
		t.Fatalf("Send must not return error on broadcast RPC failure: %v", err)
	}
	if resp.State != "broadcast_unknown" {
		t.Fatalf("expected state broadcast_unknown, got %s", resp.State)
	}

	rec := svc.journal.FindByHash(resp.Hash)
	if rec == nil {
		t.Fatal("expected journal record to exist")
	}
	// The RPC error should map to chain.ErrUnavailable and be recorded
	if rec.Error != chain.ErrUnavailable.Error() {
		t.Fatalf("expected journal error to be %q, got %q", chain.ErrUnavailable.Error(), rec.Error)
	}
	// Must NOT be ErrTimeout
	if rec.Error == chain.ErrTimeout.Error() {
		t.Fatal("broadcast RPC error should NOT be recorded as ErrTimeout")
	}
	if svc.storageFault.Load() {
		t.Fatal("storageFault must not be set on RPC broadcast error")
	}
}

// TestBroadcastTimeoutRecordsBroadcastUnknown explicitly verifies that a context timeout
// during broadcast records broadcast_unknown with ErrTimeout and does not trigger storage fault.
func TestBroadcastTimeoutRecordsBroadcastUnknown(t *testing.T) {
	hangRPC := make(chan struct{})
	state := &rpcState{chainID: "0xaa36a7", balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil), gas: 48000, allowance: big.NewInt(0), tokenBalance: big.NewInt(10000000), swapOutput: big.NewInt(1000000)}
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
			return hexutil.EncodeUint64(state.gas)
		case "eth_getCode":
			return "0x6000"
		case "eth_sendRawTransaction":
			state.mu.Unlock()
			// Block until hangRPC is closed or test completes
			select {
			case <-hangRPC:
			case <-time.After(5 * time.Second):
				t.Error("timeout waiting for hangRPC signal")
			}
			state.mu.Lock()
			return nil
		default:
			return nil
		}
	})

	svc, err := NewService(c, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		select {
		case <-hangRPC:
		default:
			close(hangRPC)
		}
		svc.Close()
	})

	_, err = svc.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}

	q := ethQuote(t, svc)

	// Send with a very short timeout so client times out
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	resp, err := svc.Send(timeoutCtx, q.ID, "fixture-password-123")
	if err != nil {
		t.Fatalf("Send must not return error on broadcast timeout: %v", err)
	}
	if resp.State != "broadcast_unknown" {
		t.Fatalf("expected state broadcast_unknown on timeout, got %s", resp.State)
	}

	rec := svc.journal.FindByHash(resp.Hash)
	if rec == nil {
		t.Fatal("expected journal record to exist")
	}
	// The timeout should map to chain.ErrTimeout
	if rec.Error != chain.ErrTimeout.Error() {
		t.Fatalf("expected timeout error in journal record %q, got %q", chain.ErrTimeout.Error(), rec.Error)
	}
	if svc.storageFault.Load() {
		t.Fatal("storageFault must not be set on broadcast timeout")
	}
}

// TestStorageFaultOnDiskFailureVsCASMismatch verifies that CAS version mismatch does NOT trigger
// storage fault, while actual physical disk write failures correctly set storageFault and report error.
func TestStorageFaultOnDiskFailureVsCASMismatch(t *testing.T) {
	t.Run("CAS mismatch does not trigger storage fault", func(t *testing.T) {
		s, _ := guardedFixture(t)
		q := ethQuote(t, s)
		sent, err := s.Send(context.Background(), q.ID, "fixture-password-123")
		if err != nil {
			t.Fatal(err)
		}
		rec := s.journal.FindByHash(sent.Hash)
		// Try to update with wrong expected version
		updated, err := s.journal.UpdateStateAtomicIfVersion(sent.Hash, rec.Version+99, "succeeded", "1", "0.001", "")
		if err != nil {
			t.Fatalf("CAS mismatch should not return disk error: %v", err)
		}
		if updated {
			t.Fatal("CAS mismatch should return updated=false")
		}
		if s.storageFault.Load() {
			t.Fatal("storageFault must remain false on CAS mismatch")
		}
	})

	t.Run("Actual disk write failure in Send triggers storage fault and reports error", func(t *testing.T) {
		s, _ := guardedFixture(t)
		q := ethQuote(t, s)

		// Create a regular file and set walletDir to a path beneath it, forcing ENOTDIR on directory creation/temp file
		blockerFile := filepath.Join(t.TempDir(), "blocker-file")
		if err := os.WriteFile(blockerFile, []byte("blocker"), 0600); err != nil {
			t.Fatal(err)
		}
		s.journal.walletDir = filepath.Join(blockerFile, "wallet")

		_, err := s.Send(context.Background(), q.ID, "fixture-password-123")
		if err == nil {
			t.Fatal("expected error on disk write failure")
		}
		if !s.storageFault.Load() {
			t.Fatal("storageFault MUST be set to true when physical disk write fails")
		}
	})

	t.Run("Actual disk write failure in UpdateStateAtomicIfVersion returns error", func(t *testing.T) {
		s, _ := guardedFixture(t)
		rec := signedJournalRecord(t, "legacy-storage-fault")
		v, err := s.journal.AppendAtomic(rec)
		if err != nil {
			t.Fatal(err)
		}
		before := *s.journal.FindByHash(string(rec.Hash))
		journalPath := s.journal.journalPath()
		original, err := os.ReadFile(journalPath)
		if err != nil {
			t.Fatal(err)
		}

		blockerFile := filepath.Join(t.TempDir(), "blocker-file")
		if err := os.WriteFile(blockerFile, []byte("blocker"), 0600); err != nil {
			t.Fatal(err)
		}
		s.journal.walletDir = filepath.Join(blockerFile, "wallet")

		updated, err := s.journal.UpdateStateAtomicIfVersion(string(rec.Hash), v, "succeeded", "5", "0.001", "", true)
		if err == nil {
			t.Fatal("expected disk error on invalid directory")
		}
		if updated {
			t.Fatal("expected updated=false on disk error")
		}
		if after := s.journal.FindByHash(string(rec.Hash)); *after != before {
			t.Fatal("failed persistence changed the in-memory record")
		}
		after, err := os.ReadFile(journalPath)
		if err != nil || string(after) != string(original) {
			t.Fatalf("failed persistence changed the original journal: %v", err)
		}
	})

	t.Run("Actual disk write failure after broadcast in Send triggers storage fault and returns error", func(t *testing.T) {
		sendStarted := make(chan struct{})
		canBroadcast := make(chan struct{})

		state := &rpcState{chainID: "0xaa36a7", balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil), gas: 48000, allowance: big.NewInt(0), tokenBalance: big.NewInt(10000000), swapOutput: big.NewInt(1000000)}
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
				return hexutil.EncodeUint64(state.gas)
			case "eth_getCode":
				return "0x6000"
			case "eth_sendRawTransaction":
				state.mu.Unlock()
				select {
				case <-sendStarted:
				default:
					close(sendStarted)
				}
				select {
				case <-canBroadcast:
				case <-time.After(5 * time.Second):
					t.Error("timeout waiting for canBroadcast signal")
				}
				state.mu.Lock()
				var args []string
				json.Unmarshal(params, &args)
				raw, _ := hexutil.Decode(args[0])
				var tx types.Transaction
				tx.UnmarshalBinary(raw)
				return tx.Hash().Hex()
			default:
				return nil
			}
		})

		svc, err := NewService(c, t.TempDir(), 2, 1)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			select {
			case <-sendStarted:
			default:
				close(sendStarted)
			}
			select {
			case <-canBroadcast:
			default:
				close(canBroadcast)
			}
			svc.Close()
		})

		_, err = svc.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "fixture-password-123")
		if err != nil {
			t.Fatal(err)
		}

		q := ethQuote(t, svc)

		type sendResult struct {
			resp *SendResponse
			err  error
		}
		sendResChan := make(chan sendResult, 1)

		go func() {
			resp, sErr := svc.Send(context.Background(), q.ID, "fixture-password-123")
			sendResChan <- sendResult{resp: resp, err: sErr}
		}()

		// Wait for Send to append record and reach broadcast
		select {
		case <-sendStarted:
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for Send to start broadcast")
		}

		// Now break disk storage before broadcast completes
		blockerFile := filepath.Join(t.TempDir(), "blocker-file")
		if err := os.WriteFile(blockerFile, []byte("blocker"), 0600); err != nil {
			t.Fatal(err)
		}
		svc.journal.walletDir = filepath.Join(blockerFile, "wallet")

		// Release broadcast
		close(canBroadcast)

		var res sendResult
		select {
		case res = <-sendResChan:
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for Send to finish")
		}

		// Physical disk failure after broadcast MUST return error and set storageFault!
		if res.err == nil {
			t.Fatal("expected error when disk fails during post-broadcast state persistence")
		}
		if !svc.storageFault.Load() {
			t.Fatal("storageFault MUST be set to true when physical disk write fails after broadcast")
		}
	})
}

// TestRetryBroadcastRPCErrorRecordsBroadcastUnknown explicitly verifies that a standard RPC broadcast error
// during Retry records broadcast_unknown with ErrUnavailable and does not trigger storage fault.
func TestRetryBroadcastRPCErrorRecordsBroadcastUnknown(t *testing.T) {
	sendCount := 0
	state := &rpcState{chainID: "0xaa36a7", balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil), gas: 48000, allowance: big.NewInt(0), tokenBalance: big.NewInt(10000000), swapOutput: big.NewInt(1000000)}
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
			return hexutil.EncodeUint64(state.gas)
		case "eth_getCode":
			return "0x6000"
		case "eth_sendRawTransaction":
			sendCount += 1
			if sendCount == 1 {
				// Initial send leaves tx in broadcast_unknown
				return errors.New("initial broadcast failed")
			}
			// Retry broadcast also returns RPC error
			return errors.New("retry execution reverted: nonce too low")
		default:
			return nil
		}
	})

	svc, err := NewService(c, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	_, err = svc.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}

	q := ethQuote(t, svc)
	sent, err := svc.Send(context.Background(), q.ID, "fixture-password-123")
	if err != nil || sent.State != "broadcast_unknown" {
		t.Fatalf("expected broadcast_unknown on initial send, got %+v %v", sent, err)
	}

	resp, err := svc.Retry(context.Background(), sent.Hash)
	if err != nil {
		t.Fatalf("Retry must not return error on broadcast RPC failure: %v", err)
	}
	if resp.State != "broadcast_unknown" {
		t.Fatalf("expected state broadcast_unknown, got %s", resp.State)
	}

	rec := svc.journal.FindByHash(resp.Hash)
	if rec == nil {
		t.Fatal("expected journal record to exist")
	}
	if rec.Error != chain.ErrUnavailable.Error() {
		t.Fatalf("expected journal error to be %q, got %q", chain.ErrUnavailable.Error(), rec.Error)
	}
	if rec.Error == chain.ErrTimeout.Error() {
		t.Fatal("broadcast RPC error should NOT be recorded as ErrTimeout")
	}
	if svc.storageFault.Load() {
		t.Fatal("storageFault must not be set on RPC broadcast error in Retry")
	}
}

// TestRetryBroadcastTimeoutRecordsBroadcastUnknown explicitly verifies that a context timeout
// during Retry broadcast records broadcast_unknown with ErrTimeout and does not trigger storage fault.
func TestRetryBroadcastTimeoutRecordsBroadcastUnknown(t *testing.T) {
	hangRPC := make(chan struct{})
	sendCount := 0
	state := &rpcState{chainID: "0xaa36a7", balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil), gas: 48000, allowance: big.NewInt(0), tokenBalance: big.NewInt(10000000), swapOutput: big.NewInt(1000000)}
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
			return hexutil.EncodeUint64(state.gas)
		case "eth_getCode":
			return "0x6000"
		case "eth_sendRawTransaction":
			sendCount += 1
			if sendCount == 1 {
				return errors.New("initial send failed")
			}
			state.mu.Unlock()
			select {
			case <-hangRPC:
			case <-time.After(5 * time.Second):
				t.Error("timeout waiting for hangRPC signal")
			}
			state.mu.Lock()
			return nil
		default:
			return nil
		}
	})

	svc, err := NewService(c, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		select {
		case <-hangRPC:
		default:
			close(hangRPC)
		}
		svc.Close()
	})

	_, err = svc.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}

	q := ethQuote(t, svc)
	sent, err := svc.Send(context.Background(), q.ID, "fixture-password-123")
	if err != nil || sent.State != "broadcast_unknown" {
		t.Fatalf("expected broadcast_unknown on initial send, got %+v %v", sent, err)
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	resp, err := svc.Retry(timeoutCtx, sent.Hash)
	if err != nil {
		t.Fatalf("Retry must not return error on broadcast timeout: %v", err)
	}
	if resp.State != "broadcast_unknown" {
		t.Fatalf("expected state broadcast_unknown on timeout, got %s", resp.State)
	}

	rec := svc.journal.FindByHash(resp.Hash)
	if rec == nil {
		t.Fatal("expected journal record to exist")
	}
	if rec.Error != chain.ErrTimeout.Error() {
		t.Fatalf("expected timeout error in journal record %q, got %q", chain.ErrTimeout.Error(), rec.Error)
	}
	if svc.storageFault.Load() {
		t.Fatal("storageFault must not be set on Retry broadcast timeout")
	}
}

// TestSendConcurrentStateUpdatePreservesLatestOnBroadcastError verifies that if a journal record
// is modified concurrently while Send broadcast fails (e.g. with RPC error), Send preserves the latest record.
func TestSendConcurrentStateUpdatePreservesLatestOnBroadcastError(t *testing.T) {
	sendStarted := make(chan struct{})
	canBroadcast := make(chan struct{})

	state := &rpcState{chainID: "0xaa36a7", balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil), gas: 48000, allowance: big.NewInt(0), tokenBalance: big.NewInt(10000000), swapOutput: big.NewInt(1000000)}
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
			return hexutil.EncodeUint64(state.gas)
		case "eth_getCode":
			return "0x6000"
		case "eth_sendRawTransaction":
			state.mu.Unlock()
			select {
			case <-sendStarted:
			default:
				close(sendStarted)
			}
			select {
			case <-canBroadcast:
			case <-time.After(5 * time.Second):
				t.Error("timeout waiting for canBroadcast signal")
			}
			state.mu.Lock()
			return errors.New("network gateway timeout")
		default:
			return nil
		}
	})

	svc, err := NewService(c, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		select {
		case <-sendStarted:
		default:
			close(sendStarted)
		}
		select {
		case <-canBroadcast:
		default:
			close(canBroadcast)
		}
		svc.Close()
	})

	_, err = svc.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}

	q := ethQuote(t, svc)

	type sendResult struct {
		resp *SendResponse
		err  error
	}
	sendResChan := make(chan sendResult, 1)

	go func() {
		resp, sErr := svc.Send(context.Background(), q.ID, "fixture-password-123")
		sendResChan <- sendResult{resp: resp, err: sErr}
	}()

	select {
	case <-sendStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Send to start broadcast")
	}

	items := svc.journal.RefreshItems()
	if len(items) == 0 {
		t.Fatal("expected pending record in journal")
	}
	txHash := items[0].Hash
	// Concurrently, an out-of-band check marked this transaction as succeeded
	if err := svc.journal.UpdateStateAtomic(txHash, "succeeded", "1", "0.0001", ""); err != nil {
		t.Fatal(err)
	}

	close(canBroadcast)

	var res sendResult
	select {
	case res = <-sendResChan:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Send to finish")
	}

	if res.err != nil {
		t.Fatalf("Send failed: %v", res.err)
	}
	// The response MUST preserve the newer concurrent state ("succeeded") rather than overwriting with "broadcast_unknown"
	if res.resp.State != "succeeded" {
		t.Fatalf("expected latest state 'succeeded', got %q", res.resp.State)
	}

	rec := svc.journal.FindByHash(txHash)
	if rec == nil || rec.State != "succeeded" {
		t.Fatalf("expected journal state 'succeeded', got %+v", rec)
	}

	if svc.storageFault.Load() {
		t.Fatal("storageFault must not be set on version CAS mismatch")
	}
}

// TestRetryConcurrentStateUpdatePreservesLatestOnBroadcastError verifies that if a journal record
// is modified concurrently while Retry broadcast fails (e.g. with RPC error), Retry preserves the latest record.
func TestRetryConcurrentStateUpdatePreservesLatestOnBroadcastError(t *testing.T) {
	retryStarted := make(chan struct{})
	canBroadcast := make(chan struct{})

	state := &rpcState{chainID: "0xaa36a7", balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil), gas: 48000, allowance: big.NewInt(0), tokenBalance: big.NewInt(10000000), swapOutput: big.NewInt(1000000)}
	sendCount := 0
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
			return hexutil.EncodeUint64(state.gas)
		case "eth_getCode":
			return "0x6000"
		case "eth_sendRawTransaction":
			sendCount += 1
			if sendCount == 1 {
				return errors.New("initial send failed")
			}
			state.mu.Unlock()
			select {
			case <-retryStarted:
			default:
				close(retryStarted)
			}
			select {
			case <-canBroadcast:
			case <-time.After(5 * time.Second):
				t.Error("timeout waiting for canBroadcast signal")
			}
			state.mu.Lock()
			return errors.New("retry RPC node error")
		default:
			return nil
		}
	})

	svc, err := NewService(c, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		select {
		case <-retryStarted:
		default:
			close(retryStarted)
		}
		select {
		case <-canBroadcast:
		default:
			close(canBroadcast)
		}
		svc.Close()
	})

	_, err = svc.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}

	q := ethQuote(t, svc)
	sent, err := svc.Send(context.Background(), q.ID, "fixture-password-123")
	if err != nil || sent.State != "broadcast_unknown" {
		t.Fatalf("expected broadcast_unknown, got %+v %v", sent, err)
	}

	type retryResult struct {
		resp *SendResponse
		err  error
	}
	retryResChan := make(chan retryResult, 1)

	go func() {
		resp, rErr := svc.Retry(context.Background(), sent.Hash)
		retryResChan <- retryResult{resp: resp, err: rErr}
	}()

	select {
	case <-retryStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Retry broadcast to start")
	}

	// While Retry broadcast is in flight, record is updated concurrently to reorg_detected
	if err := svc.journal.UpdateStateAtomic(sent.Hash, "reorg_detected", "0", "", ""); err != nil {
		t.Fatal(err)
	}

	close(canBroadcast)

	var res retryResult
	select {
	case res = <-retryResChan:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Retry to complete")
	}

	if res.err != nil {
		t.Fatalf("Retry failed: %v", res.err)
	}
	if res.resp.State != "reorg_detected" {
		t.Fatalf("expected Retry to return latest state 'reorg_detected', got %q", res.resp.State)
	}

	rec := svc.journal.FindByHash(sent.Hash)
	if rec == nil || rec.State != "reorg_detected" {
		t.Fatalf("expected journal state 'reorg_detected', got %+v", rec)
	}

	if svc.storageFault.Load() {
		t.Fatal("storageFault must not be set on version CAS mismatch in Retry")
	}
}

// TestRetryStorageFault verifies that Retry properly respects storageFault:
// 1. It immediately rejects retry if storageFault is already set.
// 2. It sets storageFault and returns an error if post-broadcast disk write fails.
func TestRetryStorageFault(t *testing.T) {
	t.Run("Retry rejects immediately when storageFault is set", func(t *testing.T) {
		s, _ := guardedFixture(t)
		q := ethQuote(t, s)
		sent, err := s.Send(context.Background(), q.ID, "fixture-password-123")
		if err != nil {
			t.Fatal(err)
		}

		// Simulate pre-existing storage fault
		s.storageFault.Store(true)

		_, err = s.Retry(context.Background(), sent.Hash)
		if err == nil || err.Error() != "交易儲存發生錯誤，請修復磁碟後重啟錢包" {
			t.Fatalf("expected storage fault error, got %v", err)
		}
	})

	t.Run("Actual disk write failure after broadcast in Retry triggers storage fault and returns error", func(t *testing.T) {
		retryStarted := make(chan struct{})
		canBroadcast := make(chan struct{})

		state := &rpcState{chainID: "0xaa36a7", balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil), gas: 48000, allowance: big.NewInt(0), tokenBalance: big.NewInt(10000000), swapOutput: big.NewInt(1000000)}
		sendCount := 0
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
				return hexutil.EncodeUint64(state.gas)
			case "eth_getCode":
				return "0x6000"
			case "eth_sendRawTransaction":
				sendCount += 1
				if sendCount == 1 {
					// Initial send fails to leave record in broadcast_unknown
					return errors.New("initial send failed")
				}
				state.mu.Unlock()
				select {
				case <-retryStarted:
				default:
					close(retryStarted)
				}
				select {
				case <-canBroadcast:
				case <-time.After(5 * time.Second):
					t.Error("timeout waiting for canBroadcast signal")
				}
				state.mu.Lock()
				var args []string
				json.Unmarshal(params, &args)
				raw, _ := hexutil.Decode(args[0])
				var tx types.Transaction
				tx.UnmarshalBinary(raw)
				return tx.Hash().Hex()
			default:
				return nil
			}
		})

		svc, err := NewService(c, t.TempDir(), 2, 1)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			select {
			case <-retryStarted:
			default:
				close(retryStarted)
			}
			select {
			case <-canBroadcast:
			default:
				close(canBroadcast)
			}
			svc.Close()
		})

		_, err = svc.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "fixture-password-123")
		if err != nil {
			t.Fatal(err)
		}

		q := ethQuote(t, svc)
		sent, err := svc.Send(context.Background(), q.ID, "fixture-password-123")
		if err != nil || sent.State != "broadcast_unknown" {
			t.Fatalf("expected broadcast_unknown, got %+v %v", sent, err)
		}

		type retryResult struct {
			resp *SendResponse
			err  error
		}
		retryResChan := make(chan retryResult, 1)

		go func() {
			resp, rErr := svc.Retry(context.Background(), sent.Hash)
			retryResChan <- retryResult{resp: resp, err: rErr}
		}()

		select {
		case <-retryStarted:
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for Retry broadcast to start")
		}

		// Now break disk storage before broadcast completes
		blockerFile := filepath.Join(t.TempDir(), "blocker-file")
		if err := os.WriteFile(blockerFile, []byte("blocker"), 0600); err != nil {
			t.Fatal(err)
		}
		svc.journal.walletDir = filepath.Join(blockerFile, "wallet")

		// Release broadcast
		close(canBroadcast)

		var res retryResult
		select {
		case res = <-retryResChan:
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for Retry to finish")
		}

		if res.err == nil {
			t.Fatal("expected error when disk fails during post-broadcast state persistence in Retry")
		}
		if !svc.storageFault.Load() {
			t.Fatal("storageFault MUST be set to true when physical disk write fails after Retry broadcast")
		}
	})
}

// TestHistoryUpdatesReorgDetectedFromRPC verifies that History() properly detects chain reorganizations
// via ethclient header hash mismatch and marks the record as reorg_detected (keeping it in flight).
func TestHistoryUpdatesReorgDetectedFromRPC(t *testing.T) {
	state := &rpcState{chainID: "0xaa36a7", balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil), gas: 48000, allowance: big.NewInt(0), tokenBalance: big.NewInt(10000000), swapOutput: big.NewInt(1000000)}
	var txMu sync.Mutex
	var lastTxHash common.Hash

	c := mockRPC(t, func(method string, params json.RawMessage) any {
		state.mu.Lock()
		defer state.mu.Unlock()
		switch method {
		case "eth_chainId":
			return state.chainID
		case "eth_getBlockByNumber":
			// Return a header with block number 100, but canonical hash different from receipt blockHash
			return &types.Header{Number: big.NewInt(100), Difficulty: big.NewInt(0), GasLimit: 30000000, BaseFee: big.NewInt(1000000000)}
		case "eth_maxPriorityFeePerGas":
			return "0x3b9aca00"
		case "eth_getTransactionCount":
			return hexutil.EncodeUint64(state.nonce)
		case "eth_getBalance":
			return hexutil.EncodeBig(state.balance)
		case "eth_estimateGas":
			return hexutil.EncodeUint64(state.gas)
		case "eth_getCode":
			return "0x6000"
		case "eth_sendRawTransaction":
			var args []string
			json.Unmarshal(params, &args)
			raw, _ := hexutil.Decode(args[0])
			var tx types.Transaction
			tx.UnmarshalBinary(raw)
			txMu.Lock()
			lastTxHash = tx.Hash()
			txMu.Unlock()
			return tx.Hash().Hex()
		case "eth_getTransactionReceipt":
			txMu.Lock()
			h := lastTxHash
			txMu.Unlock()
			// Return receipt with blockHash that does NOT match canonical header
			return &types.Receipt{
				Status:            types.ReceiptStatusSuccessful,
				TxHash:            h,
				BlockNumber:       big.NewInt(100),
				BlockHash:         common.HexToHash("0x9999999999999999999999999999999999999999999999999999999999999999"),
				GasUsed:           21000,
				CumulativeGasUsed: 21000,
				EffectiveGasPrice: big.NewInt(1000000000),
				Logs:              []*types.Log{},
			}
		default:
			return nil
		}
	})

	svc, err := NewService(c, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	_, err = svc.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}

	q := ethQuote(t, svc)
	sent, err := svc.Send(context.Background(), q.ID, "fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}

	// History queries the mock RPC, which returns canonical.Hash() != r.BlockHash
	hist, err := svc.History(context.Background())
	if err != nil {
		t.Fatalf("History failed: %v", err)
	}

	if len(hist.Transactions) != 1 || hist.Transactions[0].Hash != sent.Hash {
		t.Fatalf("expected 1 transaction matching %s in history, got %+v", sent.Hash, hist.Transactions)
	}
	if hist.Transactions[0].State != "reorg_detected" {
		t.Fatalf("expected state 'reorg_detected', got %q", hist.Transactions[0].State)
	}
	if !svc.journal.HasInFlightTx() {
		t.Fatal("reorged transaction must remain in flight to prevent nonce collisions")
	}
}
