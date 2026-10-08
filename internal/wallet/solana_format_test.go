package wallet

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	sol "github.com/gagliardetto/solana-go"
)

// Known-answer synthetic test vector independently constructed from the pre-change format.
// This vector contains strictly synthetic, test-only keys and random seed material.
const (
	testSyntheticSeedHex   = "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"
	testSyntheticPassword  = "test-synthetic-solana-pass-2026"
	testSyntheticAddress   = "9C6hybhQ6Aycep9jaUnP6uL9ZYvDjUp1aSkFWPUFJtpj"
	testSyntheticSaltHex   = "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	testSyntheticNonceHex  = "112233445566778899aabbcc"
	testSyntheticCipherHex = "9455b2ccec49f55ff0e809604dd3fd26b289823f3c8537e973cdbcc697971b83e9680556980fad44053f471e048c6e80"
	testSyntheticScryptN   = 16
	testProtocolHexDomain  = "666c6f776c65646765722d736f6c616e612d76313a"
)

func newSyntheticKeyFile() *solanaKeyFile {
	return &solanaKeyFile{
		Version:    1,
		Address:    testSyntheticAddress,
		N:          testSyntheticScryptN,
		Salt:       testSyntheticSaltHex,
		Nonce:      testSyntheticNonceHex,
		Ciphertext: testSyntheticCipherHex,
	}
}

func TestSolanaFormat_ProtocolConstant(t *testing.T) {
	if len(solanaV1AuthDomain) != 21 {
		t.Fatalf("expected 21 bytes for solanaV1AuthDomain, got %d", len(solanaV1AuthDomain))
	}
	if got := hex.EncodeToString([]byte(solanaV1AuthDomain)); got != testProtocolHexDomain {
		t.Fatalf("protocol domain mismatch: got %s, want %s", got, testProtocolHexDomain)
	}

	addr := testSyntheticAddress
	ad := solanaV1AuthData(addr)
	expectedPrefix, err := hex.DecodeString(testProtocolHexDomain)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(ad, expectedPrefix) {
		t.Fatal("auth data missing required protocol domain prefix")
	}
	if string(ad[len(expectedPrefix):]) != addr {
		t.Fatal("auth data missing address suffix")
	}
}

func TestSolanaFormat_KnownAnswerCompatibility(t *testing.T) {
	wantSeed, err := hex.DecodeString(testSyntheticSeedHex)
	if err != nil {
		t.Fatalf("invalid hex seed: %v", err)
	}

	keyFile := newSyntheticKeyFile()
	gotSeed, err := decryptSolanaKey(keyFile, testSyntheticPassword)
	if err != nil {
		t.Fatalf("failed to decrypt known-answer vector: %v", err)
	}
	defer wipeBytes(gotSeed)

	if !bytes.Equal(gotSeed, wantSeed) {
		t.Fatalf("decrypted seed mismatch: got %x, want %x", gotSeed, wantSeed)
	}

	priv := ed25519.NewKeyFromSeed(gotSeed)
	derivedAddr := sol.PrivateKey(priv).PublicKey().String()
	if derivedAddr != testSyntheticAddress {
		t.Fatalf("derived address mismatch: got %s, want %s", derivedAddr, testSyntheticAddress)
	}
}

func TestSolanaFormat_RestoreBackupWithKnownAnswer(t *testing.T) {
	keyFile := newSyntheticKeyFile()
	rawJSON, err := json.Marshal(keyFile)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	dir := t.TempDir()
	s, err := NewSolanaService("http://127.0.0.1:8899", dir, testSyntheticScryptN)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	newPass := "new-restored-password-456"
	if err := s.RestoreBackup(rawJSON, testSyntheticPassword, newPass); err != nil {
		t.Fatalf("RestoreBackup failed with synthetic known-answer: %v", err)
	}

	status, err := s.Status()
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if status.Address != testSyntheticAddress {
		t.Fatalf("restored address mismatch: got %s, want %s", status.Address, testSyntheticAddress)
	}

	backupBytes, err := s.Backup(newPass)
	if err != nil {
		t.Fatalf("Backup failed after restore: %v", err)
	}
	var reloadedKey solanaKeyFile
	if err := json.Unmarshal(backupBytes, &reloadedKey); err != nil {
		t.Fatalf("unmarshal restored backup failed: %v", err)
	}
	if reloadedKey.Address != testSyntheticAddress {
		t.Fatalf("reloaded key address mismatch: got %s, want %s", reloadedKey.Address, testSyntheticAddress)
	}
}

func TestSolanaFormat_WrongPassword(t *testing.T) {
	keyFile := newSyntheticKeyFile()
	_, err := decryptSolanaKey(keyFile, "incorrect-password")
	if !errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("expected ErrPasswordMismatch on wrong password, got: %v", err)
	}
}

func TestSolanaFormat_TamperedAddress(t *testing.T) {
	// Case 1: Corrupted non-base58 address
	rawCorrupted := []byte(`{
		"version": 1,
		"address": "invalid_base58_address_!!",
		"scryptN": 16,
		"salt": "` + testSyntheticSaltHex + `",
		"nonce": "` + testSyntheticNonceHex + `",
		"ciphertext": "` + testSyntheticCipherHex + `"
	}`)
	if _, err := parseSolanaKey(rawCorrupted); err == nil {
		t.Fatal("expected error parsing invalid base58 address")
	}

	// Case 2: Valid alternative Solana address (ciphertext transplanted or tampered)
	// Because authenticated data binds address to AES-GCM ciphertext, GCM Open fails.
	altPriv, err := sol.NewRandomPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	altAddr := altPriv.PublicKey().String()

	keyFile := newSyntheticKeyFile()
	keyFile.Address = altAddr

	_, err = decryptSolanaKey(keyFile, testSyntheticPassword)
	if !errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("expected ErrPasswordMismatch when address is tampered, got: %v", err)
	}
}

func TestSolanaFormat_TamperedCiphertext(t *testing.T) {
	ct, err := hex.DecodeString(testSyntheticCipherHex)
	if err != nil {
		t.Fatal(err)
	}
	ct[0] ^= 0x01

	keyFile := newSyntheticKeyFile()
	keyFile.Ciphertext = hex.EncodeToString(ct)

	_, err = decryptSolanaKey(keyFile, testSyntheticPassword)
	if !errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("expected ErrPasswordMismatch when ciphertext is tampered, got: %v", err)
	}
}

func TestSolanaFormat_TamperedNonce(t *testing.T) {
	nonce, err := hex.DecodeString(testSyntheticNonceHex)
	if err != nil {
		t.Fatal(err)
	}
	nonce[0] ^= 0x01

	keyFile := newSyntheticKeyFile()
	keyFile.Nonce = hex.EncodeToString(nonce)

	_, err = decryptSolanaKey(keyFile, testSyntheticPassword)
	if !errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("expected ErrPasswordMismatch when nonce is tampered, got: %v", err)
	}
}

func TestSolanaFormat_TamperedSalt(t *testing.T) {
	salt, err := hex.DecodeString(testSyntheticSaltHex)
	if err != nil {
		t.Fatal(err)
	}
	salt[0] ^= 0x01

	keyFile := newSyntheticKeyFile()
	keyFile.Salt = hex.EncodeToString(salt)

	_, err = decryptSolanaKey(keyFile, testSyntheticPassword)
	if !errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("expected ErrPasswordMismatch when salt is tampered, got: %v", err)
	}
}
