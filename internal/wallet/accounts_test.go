package wallet

import (
	"bytes"
	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"os"
	"strings"
	"testing"
)

func TestAccountAndNetworkIsolation(t *testing.T) {
	client, err := chain.New("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	root, err := NewService(client, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	first, err := root.Create("test-password-123")
	if err != nil {
		t.Fatal(err)
	}
	info, err := root.AddAccount("轉帳測試", "")
	if err != nil {
		t.Fatal(err)
	}
	account, err := NewAccountService(client, root, info.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer account.Close()
	account.keystore.scryptN = 2
	account.keystore.scryptP = 1
	backup, err := root.Backup("test-password-123")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := account.ImportKeystore(backup, "test-password-123", "other-password-123"); err == nil {
		t.Fatal("accepted duplicate signing address")
	}
	second, err := account.Create("other-password-123")
	if err != nil || first.Address == second.Address {
		t.Fatal("accounts share key", err)
	}
	arb, err := chain.NewNetwork(421614, []string{"http://127.0.0.1:1"})
	if err != nil {
		t.Fatal(err)
	}
	defer arb.Close()
	linked, err := NewLinkedService(arb, t.TempDir(), account)
	if err != nil {
		t.Fatal(err)
	}
	defer linked.Close()
	status, err := linked.Status()
	if err != nil || status.Address != second.Address || status.ChainID != 421614 || len(status.Exchange) != 0 {
		t.Fatal("linked network configuration", err)
	}
	if linked.journal == account.journal || linked.quotes == account.quotes {
		t.Fatal("shared transaction state across chains")
	}
	if _, err := NewAccountService(client, root, "../"); err == nil {
		t.Fatal("accepted invalid account path")
	}
	if _, err := chain.NewNetwork(1, []string{"http://127.0.0.1:1"}); err == nil {
		t.Fatal("accepted mainnet")
	}
}

func TestAccountLifecycle(t *testing.T) {
	client, err := chain.New("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	dir := t.TempDir()
	root, err := NewService(client, dir, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, err := root.UpdateAccount("", "主要錢包", true); err == nil {
		t.Fatal("archived last active wallet")
	}
	for _, name := range []string{"", "  ", "a\nb", strings.Repeat("名", 41)} {
		if _, err := root.AddAccount(name, ""); err == nil {
			t.Fatalf("accepted invalid name %q", name)
		}
	}
	added, err := root.AddAccount("  收款測試  ", "")
	if err != nil || added.Name != "收款測試" {
		t.Fatalf("create: %+v %v", added, err)
	}
	child, err := NewAccountService(client, root, added.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer child.Close()
	child.keystore.scryptN, child.keystore.scryptP = 2, 1
	created, err := child.Create("test-password-123")
	if err != nil {
		t.Fatal(err)
	}
	before, err := child.Backup("test-password-123")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := child.journal.AppendAtomic(signedJournalRecord(t, "account-lifecycle")); err != nil {
		t.Fatal(err)
	}
	journal := child.journal.journalPath()
	journalBefore, err := os.ReadFile(journal)
	if err != nil {
		t.Fatal(err)
	}
	renamed, err := child.UpdateAccount(added.ID, "練習錢包", false)
	if err != nil || renamed.Address != created.Address || renamed.Name != "練習錢包" {
		t.Fatalf("rename: %+v %v", renamed, err)
	}
	if _, err := root.UpdateAccount("../", "非法", true); err == nil {
		t.Fatal("accepted traversal")
	}
	if _, err := root.UpdateAccount(strings.Repeat("a", 32), "不存在", true); err == nil {
		t.Fatal("accepted missing wallet")
	}
	for _, archived := range []bool{true, false} {
		changed, err := root.UpdateAccount(added.ID, "練習錢包", archived)
		if err != nil || changed.Archived != archived {
			t.Fatalf("archive/restore: %+v %v", changed, err)
		}
		// Read through a fresh service to verify persisted state, not in-memory cache.
		root.Close()
		reopened, err := NewService(client, dir, 2, 1)
		if err != nil {
			t.Fatal(err)
		}
		accounts, err := reopened.Accounts()
		root = reopened
		defer reopened.Close()
		if err != nil || len(accounts) != 2 || accounts[1].Archived != archived || accounts[1].Name != "練習錢包" {
			t.Fatalf("persisted accounts: %+v %v", accounts, err)
		}
		after, err := child.Backup("test-password-123")
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("key changed during metadata update", err)
		}
		data, err := os.ReadFile(journal)
		if err != nil || !bytes.Equal(data, journalBefore) {
			t.Fatal("journal changed", err)
		}
	}
}

func TestNamedAccountCreatedWithPassword(t *testing.T) {
	client, err := chain.New("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	root, err := NewService(client, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, err := root.AddAccount("bad password", "short"); err == nil {
		t.Fatal("accepted weak password")
	}
	before, err := root.Accounts()
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 1 {
		t.Fatal("failed creation left registered account")
	}
	added, err := root.AddAccount("我的測試錢包", "visitor-test-password")
	if err != nil {
		t.Fatal(err)
	}
	if added.Address == "" {
		t.Fatal("exposed uninitialized account")
	}
	account, err := NewAccountService(client, root, added.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer account.Close()
	if _, err := account.Backup("wrong-password"); err == nil {
		t.Fatal("wrong password accepted")
	}
	if _, err := account.Backup("visitor-test-password"); err != nil {
		t.Fatal(err)
	}
	list, err := root.Accounts()
	if err != nil || len(list) != 2 || list[1].Name != added.Name || list[1].Address != added.Address {
		t.Fatal("name/address not persisted", err)
	}
}
