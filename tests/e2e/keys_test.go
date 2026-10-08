package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/wallet"
	"github.com/a861252012/multi-chain-wallet/internal/web"
)

func TestE2EChainKeyLifecycle(t *testing.T) {
	const (
		csrf             = "disposable-key-lifecycle-csrf"
		password         = "disposable-original-password"
		changedPassword  = "disposable-changed-password"
		restoredPassword = "disposable-restored-password"
	)
	for _, family := range []string{"solana", "tron"} {
		t.Run(family, func(t *testing.T) {
			var rpcCalls atomic.Int64
			rpcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				rpcCalls.Add(1)
				http.Error(w, "key lifecycle must not call RPC", http.StatusServiceUnavailable)
			}))
			t.Cleanup(rpcServer.Close)
			t.Cleanup(func() {
				if calls := rpcCalls.Load(); calls != 0 {
					t.Errorf("key lifecycle made %d RPC requests", calls)
				}
			})

			newServer := func(dir string) *httptest.Server {
				t.Helper()
				var handler http.Handler
				var err error
				if family == "solana" {
					service, createErr := wallet.NewSolanaService(rpcServer.URL, dir, 2)
					if createErr != nil {
						t.Fatal(createErr)
					}
					t.Cleanup(func() { _ = service.Close() })
					handler, err = web.NewSolana(service, csrf)
				} else {
					service, createErr := wallet.NewTronService(rpcServer.URL, "", dir, 2)
					if createErr != nil {
						t.Fatal(createErr)
					}
					t.Cleanup(func() { _ = service.Close() })
					handler, err = web.NewTron(service, csrf)
				}
				if err != nil {
					t.Fatal(err)
				}
				server := httptest.NewServer(handler)
				server.Client().Timeout = 10 * time.Second
				t.Cleanup(server.Close)
				return server
			}
			request := func(server *httptest.Server, method, action string, body any, want int) json.RawMessage {
				t.Helper()
				data, err := json.Marshal(body)
				if err != nil {
					t.Fatal(err)
				}
				req, err := http.NewRequest(method, server.URL+"/api/"+action, bytes.NewReader(data))
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Origin", server.URL)
				req.Header.Set("X-Wallet-CSRF", csrf)
				res, err := server.Client().Do(req)
				if err != nil {
					t.Fatal(err)
				}
				defer res.Body.Close()
				var result json.RawMessage
				if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
					t.Fatalf("%s %s response: %v", method, action, err)
				}
				if res.StatusCode != want {
					t.Fatalf("%s %s returned %d, want %d", method, action, res.StatusCode, want)
				}
				return result
			}
			object := func(raw json.RawMessage) map[string]any {
				t.Helper()
				var result map[string]any
				if err := json.Unmarshal(raw, &result); err != nil || result == nil {
					t.Fatalf("expected a JSON object, decode error: %v", err)
				}
				return result
			}

			source := newServer(t.TempDir())
			created := object(request(source, http.MethodPost, "create", map[string]string{"password": password}, http.StatusOK))
			address, ok := created["address"].(string)
			if !ok || address == "" {
				t.Fatal("create response has no address")
			}
			backup := request(source, http.MethodPost, "backup", map[string]string{"password": password}, http.StatusOK)
			object(backup) // Backup must remain a JSON object, not a quoted or base64 string.
			changed := object(request(source, http.MethodPost, "password", map[string]string{
				"password": password, "newPassword": changedPassword,
			}, http.StatusOK))
			if changed["changed"] != true {
				t.Fatal("password response did not confirm the change")
			}
			request(source, http.MethodPost, "backup", map[string]string{"password": password}, http.StatusBadRequest)
			request(source, http.MethodPost, "backup", map[string]string{"password": changedPassword}, http.StatusOK)

			restoreBody := map[string]any{"backup": backup, "password": password, "newPassword": restoredPassword}
			request(source, http.MethodPost, "restore", restoreBody, http.StatusBadRequest)
			request(source, http.MethodPost, "backup", map[string]string{"password": changedPassword}, http.StatusOK)

			restoredDir := t.TempDir()
			restored := newServer(restoredDir)
			request(restored, http.MethodPost, "restore", map[string]any{
				"backup": backup, "password": "incorrect-password", "newPassword": restoredPassword,
			}, http.StatusBadRequest)
			status := object(request(restored, http.MethodGet, "status", nil, http.StatusOK))
			if status["exists"] != false {
				t.Fatal("failed restore created a wallet")
			}
			keyFile := "keystore.json"
			if family == "solana" {
				keyFile = "key.json"
			}
			if _, err := os.Stat(filepath.Join(restoredDir, keyFile)); !os.IsNotExist(err) {
				t.Fatalf("failed restore wrote a key file: %v", err)
			}
			result := object(request(restored, http.MethodPost, "restore", restoreBody, http.StatusOK))
			if result["restored"] != true {
				t.Fatal("restore response did not confirm restoration")
			}
			status = object(request(restored, http.MethodGet, "status", nil, http.StatusOK))
			if status["exists"] != true || status["address"] != address {
				t.Fatal("restored wallet does not have the original address")
			}
			request(restored, http.MethodPost, "backup", map[string]string{"password": password}, http.StatusBadRequest)
			object(request(restored, http.MethodPost, "backup", map[string]string{"password": restoredPassword}, http.StatusOK))
		})
	}
}
