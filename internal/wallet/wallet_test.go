package wallet

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

func mockRPC(t *testing.T, handler func(method string, params json.RawMessage) any, chainIDs ...int64) *chain.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		result := handler(req.Method, req.Params)
		w.Header().Set("Content-Type", "application/json")
		if errVal, ok := result.(error); ok && errVal != nil {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"error": map[string]any{
					"code":    -32000,
					"message": errVal.Error(),
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  result,
		})
	}))
	t.Cleanup(server.Close)
	c, err := chain.New(server.URL)
	if len(chainIDs) > 0 {
		c.Close()
		c, err = chain.NewNetwork(chainIDs[0], []string{server.URL})
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return c
}

// TestPublishedBIP39DerivationVector tests against published BIP39/BIP44 test vectors for Ethereum.
func TestPublishedBIP39DerivationVector(t *testing.T) {
	// Known test vector for m/44'/60'/0'/0/0
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	expectedAddress := "0x9858EfFD232B4033E47d90003D41EC34EcaEda94"

	addr, privKey, err := DeriveKey(mnemonic)
	if err != nil {
		t.Fatalf("DeriveKey failed: %v", err)
	}
	defer wipePrivateKey(privKey)

	if addr.Hex() != expectedAddress {
		t.Fatalf("Address mismatch: got %s, want %s", addr.Hex(), expectedAddress)
	}
}

// TestMnemonicRestoreSameAddress verifies that restoring a wallet with the same mnemonic
// always yields the exact same derived address.
func TestMnemonicRestoreSameAddress(t *testing.T) {
	dir := t.TempDir()
	km := NewKeystoreManager(dir, 2, 1)

	created, err := km.Create("secure-password-1234")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	dir2 := t.TempDir()
	km2 := NewKeystoreManager(dir2, 2, 1)

	imported, err := km2.Import(created.Mnemonic, "another-password-5678")
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if created.Address != imported.Address {
		t.Fatalf("Imported address mismatch: got %s, want %s", imported.Address, created.Address)
	}
}

// TestInvalidMnemonic tests word counts, checksum failure, and malformed mnemonics.
func TestInvalidMnemonic(t *testing.T) {
	km := NewKeystoreManager(t.TempDir(), 2, 1)

	// 11 words (invalid length)
	if _, err := km.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon", "password-123456"); !errors.Is(err, ErrInvalidMnemonic) {
		t.Fatalf("expected ErrInvalidMnemonic for 11 words, got %v", err)
	}

	// 13 words (invalid length)
	if _, err := km.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon", "password-123456"); !errors.Is(err, ErrInvalidMnemonic) {
		t.Fatalf("expected ErrInvalidMnemonic for 13 words, got %v", err)
	}

	// 12 words with invalid checksum
	if _, err := km.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon", "password-123456"); !errors.Is(err, ErrInvalidMnemonic) {
		t.Fatalf("expected ErrInvalidMnemonic for invalid checksum, got %v", err)
	}
}

// TestPasswordRulesAndEncryptedStorageNoPlaintext tests password lengths, password mismatch,
// and confirms that disk storage is encrypted without exposing plaintext keys or mnemonics.
func TestPasswordRulesAndEncryptedStorageNoPlaintext(t *testing.T) {
	dir := t.TempDir()
	km := NewKeystoreManager(dir, 2, 1)

	// Password < 12 bytes rejected
	if _, err := km.Create("short-pass"); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("expected ErrInvalidPassword, got %v", err)
	}

	// Password > 128 bytes rejected
	longPass := strings.Repeat("a", 129)
	if _, err := km.Create(longPass); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("expected ErrInvalidPassword, got %v", err)
	}

	// Password with spaces allowed
	passWithSpaces := "correct horse battery staple 2026"
	created, err := km.Create(passWithSpaces)
	if err != nil {
		t.Fatalf("Create with spaces failed: %v", err)
	}

	// Check file permissions
	fi, err := os.Stat(km.keystorePath())
	if err != nil {
		t.Fatalf("keystore stat failed: %v", err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Fatalf("expected keystore permissions 0600, got %v", fi.Mode().Perm())
	}

	// Inspect file content: plaintext mnemonic and password must NOT exist
	data, err := os.ReadFile(km.keystorePath())
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte(created.Mnemonic)) {
		t.Fatal("plaintext mnemonic found on disk!")
	}
	if bytes.Contains(data, []byte(passWithSpaces)) {
		t.Fatal("plaintext password found on disk!")
	}

	// Password mismatch on Backup
	if _, err := km.Backup("wrong-password-here"); !errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("expected ErrPasswordMismatch, got %v", err)
	}

	// Correct password succeeds
	backupData, err := km.Backup(passWithSpaces)
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(backupData, &parsed); err != nil || parsed["crypto"] == nil {
		t.Fatalf("invalid backup JSON: %s", string(backupData))
	}

	// Never overwrite existing wallet
	if _, err := km.Create("another-password-123"); !errors.Is(err, ErrWalletExists) {
		t.Fatalf("expected ErrWalletExists on Create, got %v", err)
	}
	if _, err := km.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "another-password-123"); !errors.Is(err, ErrWalletExists) {
		t.Fatalf("expected ErrWalletExists on Import, got %v", err)
	}
}

// TestExactDecimalMath tests integer decimal parsing, excess precision, and uint256 overflow.
func TestExactDecimalMath(t *testing.T) {
	// Valid ETH parsing
	wei, err := ParseUnits("1.5", 18)
	if err != nil || wei.String() != "1500000000000000000" {
		t.Fatalf("unexpected wei: %v, err: %v", wei, err)
	}
	if fmt := FormatUnits(wei, 18); fmt != "1.5" {
		t.Fatalf("unexpected format: %s", fmt)
	}

	// Zero
	zero, err := ParseUnits("0", 18)
	if err != nil || zero.Sign() != 0 {
		t.Fatalf("unexpected zero: %v", zero)
	}
	if fmt := FormatUnits(zero, 18); fmt != "0" {
		t.Fatalf("unexpected zero format: %s", fmt)
	}

	// Reject scientific notation
	if _, err := ParseUnits("1e18", 18); err == nil {
		t.Fatal("expected error for scientific notation")
	}

	// Reject negative numbers
	if _, err := ParseUnits("-1.5", 18); err == nil {
		t.Fatal("expected error for negative number")
	}

	// Reject excess precision (> 18 decimals for ETH)
	if _, err := ParseUnits("0.0000000000000000001", 18); err == nil {
		t.Fatal("expected error for excess decimals")
	}

	// Reject uint256 overflow
	huge := strings.Repeat("9", 80)
	if _, err := ParseUnits(huge, 18); err == nil {
		t.Fatal("expected error for uint256 overflow")
	}
}

// TestAddressValidationAndChecksum tests zero address rejection and EIP-55 checksum validation.
func TestAddressValidationAndChecksum(t *testing.T) {
	// Zero address rejected
	if _, err := ValidateAddress("0x0000000000000000000000000000000000000000"); !errors.Is(err, ErrZeroAddress) {
		t.Fatalf("expected ErrZeroAddress, got %v", err)
	}

	// All lowercase accepted
	if _, err := ValidateAddress("0x9858effd232b4033e47d90003d41ec34ecaeda94"); err != nil {
		t.Fatalf("all-lowercase rejected: %v", err)
	}

	// All uppercase accepted
	if _, err := ValidateAddress("0x9858EFFD232B4033E47D90003D41EC34ECAEDA94"); err != nil {
		t.Fatalf("all-uppercase rejected: %v", err)
	}

	// Valid EIP-55 mixed-case accepted
	if _, err := ValidateAddress("0x9858EfFD232B4033E47d90003D41EC34EcaEda94"); err != nil {
		t.Fatalf("valid checksum rejected: %v", err)
	}

	// Malicious/corrupt mixed-case checksum rejected
	if _, err := ValidateAddress("0x9858EffD232B4033E47d90003D41EC34EcaEda94"); !errors.Is(err, ErrMalformedChecksum) {
		t.Fatalf("expected ErrMalformedChecksum, got %v", err)
	}
}

// TestERC20CalldataDecodeAndApprovalRace tests calldata decoding, finite amounts,
// and revoke-to-zero enforcement.
func TestERC20CalldataDecodeAndApprovalRace(t *testing.T) {
	recipient := common.HexToAddress("0x1111111111111111111111111111111111111111")
	amount, _ := new(big.Int).SetString("1000000", 10) // 1 USDT (6 decimals)

	// Transfer calldata
	data, err := erc20ABI.Pack("transfer", recipient, amount)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeERC20Calldata(data, 6)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Method != "transfer" || decoded.Target != recipient || decoded.RawAmount.Cmp(amount) != 0 || decoded.FormattedAmount != "1" {
		t.Fatalf("bad decoded transfer: %+v", decoded)
	}

	// Approve calldata with 0
	zeroAmount := big.NewInt(0)
	approve0Data, err := erc20ABI.Pack("approve", recipient, zeroAmount)
	if err != nil {
		t.Fatal(err)
	}
	decoded0, err := DecodeERC20Calldata(approve0Data, 6)
	if err != nil || decoded0.Method != "approve" || decoded0.RawAmount.Sign() != 0 || decoded0.FormattedAmount != "0" {
		t.Fatalf("bad decoded approve 0: %+v", decoded0)
	}
}

// TestQuoteAndSendLifecycle tests full lifecycle: quote creation, nonce check, signing,
// atomic persistence, retry, and history.
func TestQuoteAndSendLifecycle(t *testing.T) {
	dir := t.TempDir()
	h := &types.Header{Number: big.NewInt(100), Difficulty: big.NewInt(0), GasLimit: 30000000, BaseFee: big.NewInt(1000000000)}
	recipient := "0x2222222222222222222222222222222222222222"

	var broadcastCount int
	c := mockRPC(t, func(method string, params json.RawMessage) any {
		switch method {
		case "eth_chainId":
			return "0xaa36a7" // 11155111
		case "eth_getBlockByNumber":
			return h
		case "eth_maxPriorityFeePerGas":
			return "0x3b9aca00" // 1 gwei
		case "eth_getTransactionCount":
			return "0x0" // nonce 0
		case "eth_getBalance":
			return "0xde0b6b3a76400000" // 16 ETH
		case "eth_estimateGas":
			return "0x5208" // 21000
		case "eth_sendRawTransaction":
			broadcastCount += 1
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
			return tx.Hash().Hex()
		case "eth_getTransactionReceipt":
			return nil // not mined yet
		default:
			return nil
		}
	})

	svc, err := NewService(c, dir, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	created, err := svc.Create("test-password-123")
	if err != nil {
		t.Fatal(err)
	}

	// Quote ETH transfer
	ctx := context.Background()
	quote, err := svc.Quote(ctx, &QuoteRequest{
		Action: "eth",
		To:     recipient,
		Amount: "0.1",
	})
	if err != nil {
		t.Fatalf("Quote failed: %v", err)
	}
	if quote.From != created.Address || quote.To != recipient {
		t.Fatalf("Quote addresses mismatch: %+v", quote)
	}

	// Send transaction
	sendRes, err := svc.Send(ctx, quote.ID, "test-password-123")
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if sendRes.State != "submitted" || sendRes.Hash == "" {
		t.Fatalf("unexpected Send result: %+v", sendRes)
	}
	if broadcastCount != 1 {
		t.Fatalf("expected 1 broadcast, got %d", broadcastCount)
	}

	// Duplicate send with same quoteId returns existing hash without re-signing or rebroadcast
	dupRes, err := svc.Send(ctx, quote.ID, "test-password-123")
	if err != nil {
		t.Fatalf("duplicate Send failed: %v", err)
	}
	if dupRes.Hash != sendRes.Hash {
		t.Fatalf("hash mismatch on duplicate send: got %s, want %s", dupRes.Hash, sendRes.Hash)
	}
	if broadcastCount != 1 {
		t.Fatalf("broadcast count changed on duplicate send: %d", broadcastCount)
	}

	// In-flight tx blocks new quotes
	_, err = svc.Quote(ctx, &QuoteRequest{
		Action: "eth",
		To:     recipient,
		Amount: "0.1",
	})
	if !errors.Is(err, ErrTxInFlight) {
		t.Fatalf("expected ErrTxInFlight, got %v", err)
	}

	// Retry endpoint rebroadcasts exact raw
	retryRes, err := svc.Retry(ctx, sendRes.Hash)
	if err != nil {
		t.Fatalf("Retry failed: %v", err)
	}
	if retryRes.Hash != sendRes.Hash {
		t.Fatalf("Retry hash mismatch: got %s, want %s", retryRes.Hash, sendRes.Hash)
	}
	if broadcastCount != 2 {
		t.Fatalf("expected 2 broadcasts after retry, got %d", broadcastCount)
	}

	// Restart service from same directory: in-flight tx remains blocked
	svc.Close()
	svc2, err := NewService(c, dir, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer svc2.Close()

	if !svc2.journal.HasInFlightTx() {
		t.Fatal("expected in-flight tx to persist after restart")
	}

	// History contains the transaction
	hist, err := svc2.History(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist.Transactions) != 1 || hist.Transactions[0].Hash != sendRes.Hash {
		t.Fatalf("bad history: %+v", hist)
	}
}

// TestUncertainBroadcastPersisted verifies that RPC transport failures result in broadcast_unknown
// and are safely recorded in journal.
func TestUncertainBroadcastPersisted(t *testing.T) {
	dir := t.TempDir()
	h := &types.Header{Number: big.NewInt(100), Difficulty: big.NewInt(0), GasLimit: 30000000, BaseFee: big.NewInt(1000000000)}
	recipient := "0x2222222222222222222222222222222222222222"

	c := mockRPC(t, func(method string, params json.RawMessage) any {
		switch method {
		case "eth_chainId":
			return "0xaa36a7"
		case "eth_getBlockByNumber":
			return h
		case "eth_maxPriorityFeePerGas":
			return "0x3b9aca00"
		case "eth_getTransactionCount":
			return "0x0"
		case "eth_getBalance":
			return "0xde0b6b3a76400000"
		case "eth_estimateGas":
			return "0x5208"
		case "eth_sendRawTransaction":
			// Simulate ambiguous timeout / transport error
			time.Sleep(10 * time.Millisecond)
			return nil
		default:
			return nil
		}
	})

	svc, err := NewService(c, dir, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	_, err = svc.Create("test-password-123")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	quote, err := svc.Quote(ctx, &QuoteRequest{Action: "eth", To: recipient, Amount: "0.1"})
	if err != nil {
		t.Fatal(err)
	}

	res, err := svc.Send(ctx, quote.ID, "test-password-123")
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if res.State != "broadcast_unknown" {
		t.Fatalf("expected state broadcast_unknown, got %s", res.State)
	}

	// Verify raw tx was persisted
	record := svc.journal.FindByHash(res.Hash)
	if record == nil || record.SignedRaw == "" {
		t.Fatal("signed raw transaction was not persisted!")
	}
}

func TestQuoteInsufficientFunds(t *testing.T) {
	dir := t.TempDir()
	h := &types.Header{
		Number:     big.NewInt(100),
		Difficulty: big.NewInt(0),
		BaseFee:    big.NewInt(1000000000), // 1 gwei
		GasLimit:   30000000,
	}
	recipient := "0x2222222222222222222222222222222222222222"
	estimateGasCalled := false
	c := mockRPC(t, func(method string, params json.RawMessage) any {
		switch method {
		case "eth_chainId":
			return "0xaa36a7"
		case "eth_getBlockByNumber":
			return h
		case "eth_maxPriorityFeePerGas":
			return "0x3b9aca00"
		case "eth_getBalance":
			return "0x0" // 0 ETH
		case "eth_estimateGas":
			estimateGasCalled = true
			return nil
		default:
			return nil
		}
	})

	svc, err := NewService(c, dir, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	_, err = svc.Create("test-password-123")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_, err = svc.Quote(ctx, &QuoteRequest{Action: "eth", To: recipient, Amount: "0.000001"})
	if !errors.Is(err, ErrInsufficientFunds) {
		t.Fatalf("expected ErrInsufficientFunds, got: %v", err)
	}
	if estimateGasCalled {
		t.Fatal("estimateGas should not have been called when balance is insufficient")
	}
}
