package wallet

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
)

type AccountID string

func ParseAccountID(value string) (AccountID, error) {
	if !opaqueIDPattern.MatchString(value) {
		return "", errors.New("帳戶 ID 無效")
	}
	return AccountID(value), nil
}

type AccountInfo struct {
	ID       string `json:"id"`
	Address  string `json:"address"`
	Name     string `json:"name"`
	Archived bool   `json:"archived"`
}

func (s *Service) Accounts() ([]AccountInfo, error) {
	root := s
	if s.catalog != nil {
		root = s.catalog
	}
	root.sendMu.Lock()
	defer root.sendMu.Unlock()
	return root.listAccounts()
}

func (s *Service) listAccounts() ([]AccountInfo, error) {
	address, err := s.keystore.Address()
	if err != nil && !errors.Is(err, ErrWalletNotFound) {
		return nil, err
	}
	metadata, err := readAccountMetadata(s.walletDir)
	if err != nil {
		return nil, err
	}
	if metadata.Name == "" {
		metadata.Name = "主要錢包"
	}
	result := []AccountInfo{{ID: "", Address: address, Name: metadata.Name, Archived: metadata.Archived}}
	entries, err := os.ReadDir(filepath.Join(s.walletDir, "accounts"))
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		id, parseErr := ParseAccountID(entry.Name())
		if !entry.IsDir() || parseErr != nil {
			continue
		}
		km := NewKeystoreManager(filepath.Join(s.walletDir, "accounts", string(id)), 0, 0)
		addr, err := km.Address()
		if err != nil && !errors.Is(err, ErrWalletNotFound) {
			return nil, err
		}
		metadata, err := readAccountMetadata(filepath.Join(s.walletDir, "accounts", string(id)))
		if err != nil {
			return nil, err
		}
		if metadata.Name == "" {
			metadata.Name = "錢包 " + string(id)[:6]
		}
		result = append(result, AccountInfo{ID: string(id), Address: addr, Name: metadata.Name, Archived: metadata.Archived})
	}
	return result, nil
}

func (s *Service) AddAccount(name, password string) (*AccountInfo, error) {
	name, err := accountName(name)
	if err != nil {
		return nil, err
	}
	if password != "" {
		if err := ValidatePassword(password); err != nil {
			return nil, err
		}
	}
	root := s
	if s.catalog != nil {
		root = s.catalog
	}
	root.sendMu.Lock()
	defer root.sendMu.Unlock()
	accounts, err := root.listAccounts()
	if err != nil {
		return nil, err
	}
	if len(accounts) >= 20 {
		return nil, errors.New("最多支援 20 個本機帳戶")
	}
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, err
	}
	id := hex.EncodeToString(idBytes)
	path := filepath.Join(root.walletDir, "accounts", id)
	if err := os.MkdirAll(path, 0700); err != nil {
		return nil, err
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(path)
		}
	}()
	address := ""
	if password != "" {
		keys := NewKeystoreManager(path, root.keystore.scryptN, root.keystore.scryptP)
		created, err := keys.Create(password)
		if err != nil {
			return nil, err
		}
		address = created.Address
	}
	// Publish only after the password-protected key is ready.
	data, err := json.Marshal(accountMetadata{Name: name})
	if err != nil {
		return nil, err
	}
	if err := atomicWriteFile(filepath.Join(path, "account.json"), data, 0600); err != nil {
		return nil, err
	}
	complete = true
	return &AccountInfo{ID: id, Name: name, Address: address}, nil
}

func NewAccountService(client *chain.Client, root *Service, id string) (*Service, error) {
	accountID, err := ParseAccountID(id)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(root.walletDir, "accounts", string(accountID))
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("找不到帳戶")
	}
	if _, err := os.Stat(filepath.Join(path, "account.json")); err != nil {
		return nil, errors.New("帳戶尚未完成登記")
	}
	service, err := NewService(client, path)
	if err != nil {
		return nil, err
	}
	service.catalog = root
	service.vaultAddress = root.vaultAddress
	service.escrowAddress, service.escrowToken = root.escrowAddress, root.escrowToken
	service.keystore.catalogDir = root.walletDir
	service.keystore.catalogMu = root.keystore.catalogMu
	return service, nil
}

// Account metadata is separate from keys and transaction journals. Archiving only hides a wallet.
type accountMetadata struct {
	Name     string `json:"name"`
	Archived bool   `json:"archived"`
}

func readAccountMetadata(dir string) (accountMetadata, error) {
	var metadata accountMetadata
	data, err := os.ReadFile(filepath.Join(dir, "account.json"))
	if os.IsNotExist(err) {
		return metadata, nil
	}
	if err != nil {
		return metadata, err
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return metadata, fmt.Errorf("讀取錢包名稱失敗: %w", err)
	}
	return metadata, nil
}

func accountName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 40 || strings.ContainsFunc(name, unicode.IsControl) {
		return "", errors.New("錢包名稱請填寫 1 至 40 個字，且不可含控制字元")
	}
	return name, nil
}

func (s *Service) UpdateAccount(id, name string, archived bool) (*AccountInfo, error) {
	name, err := accountName(name)
	if err != nil {
		return nil, err
	}
	if id != "" {
		if _, err := ParseAccountID(id); err != nil {
			return nil, err
		}
	}
	root := s
	if s.catalog != nil {
		root = s.catalog
	}
	root.sendMu.Lock()
	defer root.sendMu.Unlock()
	accounts, err := root.listAccounts()
	if err != nil {
		return nil, err
	}
	active := 0
	var target *AccountInfo
	for i := range accounts {
		if !accounts[i].Archived {
			active++
		}
		if accounts[i].ID == id {
			target = &accounts[i]
		}
	}
	if target == nil {
		return nil, errors.New("找不到錢包")
	}
	if archived && !target.Archived && active <= 1 {
		return nil, errors.New("請至少保留一個未封存的錢包")
	}
	dir := root.walletDir
	if id != "" {
		dir = filepath.Join(dir, "accounts", id)
	}
	data, err := json.Marshal(accountMetadata{Name: name, Archived: archived})
	if err != nil {
		return nil, err
	}
	if err := atomicWriteFile(filepath.Join(dir, "account.json"), data, 0600); err != nil {
		return nil, err
	}
	target.Name, target.Archived = name, archived
	return target, nil
}
