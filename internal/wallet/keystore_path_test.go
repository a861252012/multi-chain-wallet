package wallet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/ethereum/go-ethereum/common"
)

func newKeystorePathTestAccount(t *testing.T, client *chain.Client, root *Service, name string) *Service {
	t.Helper()
	info, err := root.AddAccount(name, "")
	if err != nil {
		t.Fatal(err)
	}
	account, err := NewAccountService(client, root, info.ID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { account.Close() })
	account.keystore.scryptN, account.keystore.scryptP = 2, 1
	return account
}

func TestKeystoreDuplicateAddressUsesLiteralCatalogPath(t *testing.T) {
	for _, name := range []string{"wallet", "wallet[demo]", "wallet[", "wallet*"} {
		t.Run(name, func(t *testing.T) {
			client, err := chain.New("http://127.0.0.1:1")
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			root, err := NewService(client, filepath.Join(t.TempDir(), name), 2, 1)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			primary, err := root.Create("primary-password-123")
			if err != nil {
				t.Fatal(err)
			}
			first := newKeystorePathTestAccount(t, client, root, "first")
			created, err := first.Create("first-password-123")
			if err != nil {
				t.Fatal(err)
			}
			if primary.Address == created.Address {
				t.Fatal("fixture requires a different primary key")
			}
			backup, err := first.Backup("first-password-123")
			if err != nil {
				t.Fatal(err)
			}
			second := newKeystorePathTestAccount(t, client, root, "second")
			// Unrelated files under accounts are not account directories.
			if err := os.WriteFile(filepath.Join(root.walletDir, "accounts", ".DS_Store"), []byte("metadata"), 0600); err != nil {
				t.Fatal(err)
			}
			_, err = second.ImportKeystore(backup, "first-password-123", "second-password-123")
			if err == nil || !strings.Contains(err.Error(), "此地址已存在另一個帳戶") {
				t.Fatalf("duplicate signing address was not rejected: %v", err)
			}
			if second.keystore.Exists() {
				t.Fatal("rejected import left a keystore")
			}
			if err := first.keystore.checkDuplicateAddress(common.HexToAddress(created.Address)); err != nil {
				t.Fatalf("current account should be skipped: %v", err)
			}
			primaryBackup, err := root.Backup("primary-password-123")
			if err != nil {
				t.Fatal(err)
			}
			_, err = second.ImportKeystore(primaryBackup, "primary-password-123", "second-password-123")
			if err == nil || !strings.Contains(err.Error(), "此地址已存在另一個帳戶") {
				t.Fatalf("primary address was not rejected: %v", err)
			}
			if second.keystore.Exists() {
				t.Fatal("rejected primary import left a keystore")
			}
		})
	}
}

func TestKeystoreCatalogWildcardDoesNotReadSibling(t *testing.T) {
	client, err := chain.New("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	parent := t.TempDir()
	sibling, err := NewService(client, filepath.Join(parent, "wallet-other"), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer sibling.Close()
	outside := newKeystorePathTestAccount(t, client, sibling, "outside")
	created, err := outside.Create("outside-password-123")
	if err != nil {
		t.Fatal(err)
	}
	backup, err := outside.Backup("outside-password-123")
	if err != nil {
		t.Fatal(err)
	}
	root, err := NewService(client, filepath.Join(parent, "wallet*"), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	primary, err := root.Create("primary-password-123")
	if err != nil {
		t.Fatal(err)
	}
	if primary.Address == created.Address {
		t.Fatal("fixture requires a different primary key")
	}
	account := newKeystorePathTestAccount(t, client, root, "import")
	imported, err := account.ImportKeystore(backup, "outside-password-123", "import-password-123")
	if err != nil {
		t.Fatalf("sibling catalog incorrectly blocked import: %v", err)
	}
	if imported.Address != created.Address {
		t.Fatal("imported address changed")
	}
}

func TestKeystoreCatalogReadFailureIsNotIgnored(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "accounts"), []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	keys := NewKeystoreManager(dir, 2, 1)
	if err := keys.checkDuplicateAddress(common.HexToAddress("0x1111111111111111111111111111111111111111")); err == nil {
		t.Fatal("accounts directory read failure was ignored")
	}
}
