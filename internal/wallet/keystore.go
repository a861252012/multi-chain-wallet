package wallet

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

// KeystoreManager handles wallet generation, derivation, encryption, and local disk persistence.
type KeystoreManager struct {
	coinType   uint32
	catalogMu  *sync.Mutex
	catalogDir string
	mu         sync.Mutex
	walletDir  string
	scryptN    int
	scryptP    int
	scryptSem  chan struct{}
}

func NewKeystoreManager(walletDir string, scryptN, scryptP int) *KeystoreManager {
	if scryptN <= 0 {
		scryptN = keystore.StandardScryptN
	}
	if scryptP <= 0 {
		scryptP = keystore.StandardScryptP
	}
	return &KeystoreManager{
		walletDir: walletDir, catalogDir: walletDir, catalogMu: &sync.Mutex{},
		scryptN:   scryptN,
		scryptP:   scryptP,
		scryptSem: make(chan struct{}, 1), // Bounded concurrency: max 1 simultaneous scrypt operations
	}
}

func (km *KeystoreManager) keystorePath() string {
	return filepath.Join(km.walletDir, "keystore.json")
}

// ValidatePassword enforces:
// Password must be 12 to 128 characters, spaces allowed, never trimmed.
func ValidatePassword(password string) error {
	count := utf8.RuneCountInString(password)
	if count < 12 || count > 128 {
		return ErrInvalidPassword
	}
	return nil
}

// DeriveKey derives an Ethereum address and private key using BIP39 mnemonic and BIP44 path m/44'/60'/0'/0/0.
// The BIP39 passphrase is fixed to empty string ("").
func DeriveKey(mnemonic string) (common.Address, *ecdsa.PrivateKey, error) {
	return deriveCoinKey(mnemonic, 60)
}

func deriveCoinKey(mnemonic string, coin uint32) (common.Address, *ecdsa.PrivateKey, error) {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(mnemonic)), " ")
	if !bip39.IsMnemonicValid(normalized) {
		return common.Address{}, nil, ErrInvalidMnemonic
	}

	// Empty optional BIP39 passphrase ONLY, explicitly named scope.
	seed := bip39.NewSeed(normalized, "")
	defer wipeBytes(seed)

	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return common.Address{}, nil, err
	}

	defer wipeBytes(masterKey.Key)
	// BIP44 path: m/44'/60'/0'/0/0
	// 44' (purpose)
	purpose, err := masterKey.NewChildKey(bip32.FirstHardenedChild + 44)
	if err != nil {
		return common.Address{}, nil, err
	}
	defer wipeBytes(purpose.Key)
	// 60' (coin_type: Ethereum)
	coinType, err := purpose.NewChildKey(bip32.FirstHardenedChild + coin)
	if err != nil {
		return common.Address{}, nil, err
	}
	defer wipeBytes(coinType.Key)
	// 0' (account 0)
	account, err := coinType.NewChildKey(bip32.FirstHardenedChild + 0)
	if err != nil {
		return common.Address{}, nil, err
	}
	defer wipeBytes(account.Key)
	// 0 (external change)
	change, err := account.NewChildKey(0)
	if err != nil {
		return common.Address{}, nil, err
	}
	defer wipeBytes(change.Key)
	// 0 (address index 0)
	addressKey, err := change.NewChildKey(0)
	if err != nil {
		return common.Address{}, nil, err
	}
	defer wipeBytes(addressKey.Key)

	privKey, err := crypto.ToECDSA(addressKey.Key)
	if err != nil {
		return common.Address{}, nil, err
	}
	addr := crypto.PubkeyToAddress(privKey.PublicKey)
	return addr, privKey, nil
}

func (km *KeystoreManager) acquireScrypt() error {
	select {
	case km.scryptSem <- struct{}{}:
		return nil
	default:
		return ErrTooManyScryptRequests
	}
}

func (km *KeystoreManager) releaseScrypt() {
	<-km.scryptSem
}

// Callers hold the scrypt admission gate. Collect each completed KDF's large
// scratch allocation before the same manager's next KDF, including decrypt/encrypt
// pairs in password changes and imports. The returned key remains live.
func encryptKeystoreWithGC(key *keystore.Key, password string, n, p int) ([]byte, error) {
	defer runtime.GC()
	return keystore.EncryptKey(key, password, n, p)
}

func decryptKeystoreWithGC(data []byte, password string) (*keystore.Key, error) {
	defer runtime.GC()
	return keystore.DecryptKey(data, password)
}

// Exists returns true if the keystore file exists on disk.
func (km *KeystoreManager) Exists() bool {
	km.mu.Lock()
	defer km.mu.Unlock()
	_, err := os.Stat(km.keystorePath())
	return err == nil
}

// Address returns the checksummed Ethereum address of the stored wallet.
func (km *KeystoreManager) Address() (string, error) {
	km.mu.Lock()
	defer km.mu.Unlock()
	data, err := os.ReadFile(km.keystorePath())
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrWalletNotFound
		}
		return "", err
	}
	var meta struct {
		Address string `json:"address"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return "", errors.New("無法讀取既有金鑰檔")
	}
	if !strings.HasPrefix(meta.Address, "0x") {
		meta.Address = "0x" + meta.Address
	}
	addr, err := ValidateAddress(meta.Address)
	if err != nil {
		return "", errors.New("金鑰檔地址格式錯誤")
	}
	return addr.Hex(), nil
}

// Create generates a new BIP39 12-word mnemonic, derives m/44'/60'/0'/0/0,
// encrypts using StandardScrypt into keystore.json, and returns address and mnemonic.
// Never overwrites an existing wallet. Mnemonic is returned once and never persisted.
func (km *KeystoreManager) Create(password string) (*CreateResponse, error) {
	km.catalogMu.Lock()
	defer km.catalogMu.Unlock()
	if err := ValidatePassword(password); err != nil {
		return nil, err
	}

	km.mu.Lock()
	defer km.mu.Unlock()

	if _, err := os.Lstat(km.keystorePath()); !os.IsNotExist(err) {
		if err != nil {
			return nil, errors.New("無法安全讀取錢包儲存狀態")
		}
		return nil, ErrWalletExists
	}

	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		return nil, err
	}
	defer wipeBytes(entropy)

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return nil, err
	}

	coin := km.coinType
	if coin == 0 {
		coin = 60
	}
	addr, privKey, err := deriveCoinKey(mnemonic, coin)
	if err != nil {
		return nil, err
	}
	defer wipePrivateKey(privKey)

	if err := km.acquireScrypt(); err != nil {
		return nil, err
	}
	defer km.releaseScrypt()

	key := &keystore.Key{
		Id:         uuid.New(),
		Address:    addr,
		PrivateKey: privKey,
	}
	if err := km.checkDuplicateAddress(addr); err != nil {
		return nil, err
	}
	keyJSON, err := encryptKeystoreWithGC(key, password, km.scryptN, km.scryptP)
	if err != nil {
		return nil, err
	}

	if err := atomicWriteFile(km.keystorePath(), keyJSON, 0600); err != nil {
		return nil, err
	}

	return &CreateResponse{
		Address:  addr.Hex(),
		Mnemonic: mnemonic,
		Path:     "m/44'/60'/0'/0/0",
	}, nil
}

// Import restores a wallet from an existing 12/15/18/21/24-word mnemonic.
// Never overwrites an existing wallet.
func (km *KeystoreManager) Import(mnemonic, password string) (*ImportResponse, error) {
	km.catalogMu.Lock()
	defer km.catalogMu.Unlock()
	if err := ValidatePassword(password); err != nil {
		return nil, err
	}

	words := strings.Fields(strings.TrimSpace(mnemonic))
	wCount := len(words)
	if wCount != 12 && wCount != 15 && wCount != 18 && wCount != 21 && wCount != 24 {
		return nil, ErrInvalidMnemonic
	}

	coin := km.coinType
	if coin == 0 {
		coin = 60
	}
	addr, privKey, err := deriveCoinKey(mnemonic, coin)
	if err != nil {
		return nil, err
	}
	defer wipePrivateKey(privKey)

	km.mu.Lock()
	defer km.mu.Unlock()

	if _, err := os.Lstat(km.keystorePath()); !os.IsNotExist(err) {
		if err != nil {
			return nil, errors.New("無法安全讀取錢包儲存狀態")
		}
		return nil, ErrWalletExists
	}

	if err := km.acquireScrypt(); err != nil {
		return nil, err
	}
	defer km.releaseScrypt()

	key := &keystore.Key{
		Id:         uuid.New(),
		Address:    addr,
		PrivateKey: privKey,
	}
	if err := km.checkDuplicateAddress(addr); err != nil {
		return nil, err
	}
	keyJSON, err := encryptKeystoreWithGC(key, password, km.scryptN, km.scryptP)
	if err != nil {
		return nil, err
	}

	if err := atomicWriteFile(km.keystorePath(), keyJSON, 0600); err != nil {
		return nil, err
	}

	return &ImportResponse{
		Address: addr.Hex(),
		Path:    "m/44'/60'/0'/0/0",
	}, nil
}

// Backup verifies the password by decrypting, then returns the raw encrypted keystore JSON object.
func (km *KeystoreManager) Backup(password string) (json.RawMessage, error) {
	// Existing keystores may use passwords accepted by earlier creation rules.
	km.mu.Lock()
	data, err := os.ReadFile(km.keystorePath())
	km.mu.Unlock()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrWalletNotFound
		}
		return nil, err
	}

	if err := km.acquireScrypt(); err != nil {
		return nil, err
	}
	defer km.releaseScrypt()

	key, err := decryptKeystoreWithGC(data, password)
	if err != nil {
		return nil, ErrPasswordMismatch
	}
	wipePrivateKey(key.PrivateKey)

	return json.RawMessage(data), nil
}

// ImportKeystore accepts a bounded V3 scrypt file and re-encrypts it with local parameters.
func (km *KeystoreManager) ImportKeystore(data json.RawMessage, password, newPassword string) (*ImportResponse, error) {
	km.catalogMu.Lock()
	defer km.catalogMu.Unlock()
	if err := ValidatePassword(newPassword); err != nil {
		return nil, err
	}
	var meta struct {
		Version int                 `json:"version"`
		Crypto  keystore.CryptoJSON `json:"crypto"`
	}
	if len(data) > 8192 || json.Unmarshal(data, &meta) != nil || meta.Version != 3 || meta.Crypto.KDF != "scrypt" || meta.Crypto.Cipher != "aes-128-ctr" {
		return nil, errors.New("請選擇 V3 scrypt 加密的 Keystore JSON")
	}
	params := map[string]int{}
	for _, name := range []string{"n", "r", "p", "dklen"} {
		value, ok := meta.Crypto.KDFParams[name].(float64)
		if !ok || value != math.Trunc(value) || value < 1 || value > keystore.StandardScryptN {
			return nil, errors.New("Keystore 密碼運算參數無效")
		}
		params[name] = int(value)
	}
	n := params["n"]
	if n < 2 || n&(n-1) != 0 || params["r"] != 8 || params["p"] > 6 || params["dklen"] != 32 {
		return nil, errors.New("Keystore 密碼運算參數不在支援範圍")
	}
	salt, ok := meta.Crypto.KDFParams["salt"].(string)
	if !ok {
		return nil, errors.New("Keystore salt 格式錯誤")
	}
	saltBytes, saltErr := hex.DecodeString(salt)
	iv, ivErr := hex.DecodeString(meta.Crypto.CipherParams.IV)
	mac, macErr := hex.DecodeString(meta.Crypto.MAC)
	ciphertext, cipherErr := hex.DecodeString(meta.Crypto.CipherText)
	if saltErr != nil || len(saltBytes) < 16 || len(saltBytes) > 64 || ivErr != nil || len(iv) != 16 || macErr != nil || len(mac) != 32 || cipherErr != nil || len(ciphertext) != 32 {
		return nil, errors.New("Keystore 加密欄位格式錯誤")
	}
	km.mu.Lock()
	defer km.mu.Unlock()
	if _, err := os.Lstat(km.keystorePath()); !os.IsNotExist(err) {
		return nil, ErrWalletExists
	}
	if err := km.acquireScrypt(); err != nil {
		return nil, err
	}
	defer km.releaseScrypt()
	key, err := decryptKeystoreWithGC(data, password)
	if err != nil {
		return nil, ErrPasswordMismatch
	}
	defer wipePrivateKey(key.PrivateKey)
	key.Address = crypto.PubkeyToAddress(key.PrivateKey.PublicKey)
	if err := km.checkDuplicateAddress(key.Address); err != nil {
		return nil, err
	}
	encrypted, err := encryptKeystoreWithGC(key, newPassword, km.scryptN, km.scryptP)
	if err != nil {
		return nil, err
	}
	if err := atomicWriteFile(km.keystorePath(), encrypted, 0600); err != nil {
		return nil, err
	}
	return &ImportResponse{Address: key.Address.Hex()}, nil
}

func (km *KeystoreManager) ChangePassword(password, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	km.mu.Lock()
	defer km.mu.Unlock()
	data, err := os.ReadFile(km.keystorePath())
	if err != nil {
		return err
	}
	if err := km.acquireScrypt(); err != nil {
		return err
	}
	defer km.releaseScrypt()
	key, err := decryptKeystoreWithGC(data, password)
	if err != nil {
		return ErrPasswordMismatch
	}
	defer wipePrivateKey(key.PrivateKey)
	encrypted, err := encryptKeystoreWithGC(key, newPassword, km.scryptN, km.scryptP)
	if err != nil {
		return err
	}
	return atomicWriteFile(km.keystorePath(), encrypted, 0600)
}

// DecryptKey decrypts the keystore file using the provided password.
// The caller is responsible for wiping the returned private key after use.
func (km *KeystoreManager) DecryptKey(password string) (*keystore.Key, error) {
	// Existing keystores may use passwords accepted by earlier creation rules.
	km.mu.Lock()
	data, err := os.ReadFile(km.keystorePath())
	km.mu.Unlock()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrWalletNotFound
		}
		return nil, err
	}

	if err := km.acquireScrypt(); err != nil {
		return nil, err
	}
	defer km.releaseScrypt()

	key, err := decryptKeystoreWithGC(data, password)
	if err != nil {
		return nil, ErrPasswordMismatch
	}
	return key, nil
}

// atomicWriteFile writes data to a temp file, syncs to disk, renames to dest, and syncs the parent directory.
func atomicWriteFile(dest string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(dir, "tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if err := tmpFile.Chmod(perm); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, dest); err != nil {
		return err
	}

	// Sync parent directory to persist directory entry metadata
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func wipeBytes(b []byte) {
	clear(b)
}

func wipePrivateKey(k *ecdsa.PrivateKey) {
	if k == nil || k.D == nil {
		return
	}
	b := k.D.Bits()
	clear(b)
}

// Multiple aliases for one signing address would otherwise have independent nonce journals.
func (km *KeystoreManager) checkDuplicateAddress(address common.Address) error {
	accountsDir := filepath.Join(km.catalogDir, "accounts")
	entries, err := os.ReadDir(accountsDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			paths = append(paths, filepath.Join(accountsDir, entry.Name(), "keystore.json"))
		}
	}
	paths = append(paths, filepath.Join(km.catalogDir, "keystore.json"))
	for _, path := range paths {
		if path == km.keystorePath() {
			continue
		}
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		var meta struct {
			Address string `json:"address"`
		}
		if json.Unmarshal(data, &meta) != nil {
			return errors.New("既有帳戶資料無法讀取")
		}
		if common.HexToAddress(meta.Address) == address {
			return errors.New("此地址已存在另一個帳戶，請切換既有帳戶")
		}
	}
	return nil
}
