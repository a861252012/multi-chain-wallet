package wallet

import (
	"bytes"
	"errors"
	"os"
	"sync"
	"testing"
)

func TestKeystoreCollectedKDFLifecycle(t *testing.T) {
	oldPassword, newPassword := "synthetic-old-password-123", "synthetic-new-password-456"
	source := NewKeystoreManager(t.TempDir(), 16, 1)
	created, err := source.Create(oldPassword)
	if err != nil {
		t.Fatal(err)
	}
	backup, err := source.Backup(oldPassword)
	if err != nil {
		t.Fatal(err)
	}
	restored := NewKeystoreManager(t.TempDir(), 16, 1)
	imported, err := restored.ImportKeystore(backup, oldPassword, newPassword)
	if err != nil {
		t.Fatal(err)
	}
	if imported.Address != created.Address {
		t.Fatal("restored key changed")
	}
	if err = restored.ChangePassword(newPassword, oldPassword); err != nil {
		t.Fatal(err)
	}
	key, err := restored.DecryptKey(oldPassword)
	if err != nil {
		t.Fatal(err)
	}
	if key.Address.Hex() != created.Address {
		t.Fatal("changed key identity")
	}
	wipePrivateKey(key.PrivateKey)
	before, err := os.ReadFile(restored.keystorePath())
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(restored.ChangePassword("wrong-password", newPassword), ErrPasswordMismatch) {
		t.Fatal("wrong password not rejected")
	}
	after, err := os.ReadFile(restored.keystorePath())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("wrong password modified encrypted storage")
	}
	if err = restored.acquireScrypt(); err != nil {
		t.Fatal(err)
	}
	if _, err = restored.DecryptKey(oldPassword); !errors.Is(err, ErrTooManyScryptRequests) {
		t.Fatal("held gate did not reject decrypt")
	}
	if _, err = restored.Backup(oldPassword); !errors.Is(err, ErrTooManyScryptRequests) {
		t.Fatal("held gate did not reject backup")
	}
	if err = restored.ChangePassword(oldPassword, newPassword); !errors.Is(err, ErrTooManyScryptRequests) {
		t.Fatal("held gate did not reject password change")
	}
	restored.releaseScrypt()
	if _, err = restored.Backup(oldPassword); err != nil {
		t.Fatal("gate not released after errors")
	}
}

func TestKeystoreCollectedKDFConcurrentCalls(t *testing.T) {
	km := NewKeystoreManager(t.TempDir(), 16, 1)
	password := "synthetic-concurrent-password-123"
	created, err := km.Create(password)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 12)
	start := make(chan struct{})
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			key, err := km.DecryptKey(password)
			if err == nil {
				if key.Address.Hex() != created.Address {
					err = errors.New("key changed")
				}
				wipePrivateKey(key.PrivateKey)
			}
			results <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else if !errors.Is(err, ErrTooManyScryptRequests) {
			t.Fatal(err)
		}
	}
	if success == 0 {
		t.Fatal("all concurrent calls failed")
	}
	key, err := km.DecryptKey(password)
	if err != nil {
		t.Fatal("admission leaked")
	}
	wipePrivateKey(key.PrivateKey)
}
