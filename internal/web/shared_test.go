package web

import (
	"github.com/a861252012/multi-chain-wallet/internal/wallet"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSharedDemoBoundaries(t *testing.T) {
	h := WithPublicOrigin(SharedDemo(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }), ""), "https://wallet.example")
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/api/wallet", 204}, {"GET", "/api/wallet/accounts", 204}, {"GET", "/api/wallet/history", 204},
		{"POST", "/api/wallet/send", 204}, {"POST", "/net/base/api/wallet/send", 204},
		{"POST", "/api/wallet/quote", 204}, {"POST", "/api/wallet/backup", 204},
		{"POST", "/api/wallet/accounts", 204}, {"POST", "/api/wallet/accounts/update", 403},
		{"POST", "/api/wallet/create", 403}, {"POST", "/api/wallet/import", 403},
		{"POST", "/api/wallet/password", 403}, {"POST", "/api/wallet/scan", 403},
		{"POST", "/net/base/api/wallet/scan", 403}, {"POST", "/api/faucet", 204}, {"POST", "/api/faucet/solana", 204}, {"POST", "/api/faucet/tron", 204},
		{"POST", "/solana/api/create", 403}, {"POST", "/tron/api/send", 204}, {"POST", "/solana/api/send", 204}, {"POST", "/solana/api/backup", 204}, {"POST", "/tron/api/backup", 204}, {"POST", "/tron/api/restore", 403}, {"POST", "/solana/api/password", 403},
		{"GET", "/accounts/abc/api/wallet", 403},
		{"GET", "/accounts/0123456789abcdef0123456789abcdef/api/wallet", 204},
		{"POST", "/accounts/0123456789abcdef0123456789abcdef/net/base/api/wallet/send", 204},
		{"POST", "/accounts/0123456789abcdef0123456789abcdef/api/wallet/create", 403}, {"POST", "/api/../api/wallet/send", 403},
		{"DELETE", "/api/wallet/send", 403}, {"GET", "/login", 303},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(tc.method, "http://wallet.example"+tc.path, nil))
		if w.Code != tc.status {
			t.Errorf("%s %s: %d want %d", tc.method, tc.path, w.Code, tc.status)
		}
	}
}

func TestSharedChainWallets(t *testing.T) {
	const token = "shared-demo-initialization-token-32-chars"
	const password = "disposable-test-password"
	for _, family := range []string{"solana", "tron"} {
		t.Run(family, func(t *testing.T) {
			var handler http.Handler
			if family == "solana" {
				service, err := wallet.NewSolanaService("http://127.0.0.1:1", t.TempDir(), 2)
				if err != nil {
					t.Fatal(err)
				}
				defer service.Close()
				handler, err = NewSolana(service, "fixture-csrf")
				if err != nil {
					t.Fatal(err)
				}
			} else {
				service, err := wallet.NewTronService("http://127.0.0.1:1", "", t.TempDir(), 2)
				if err != nil {
					t.Fatal(err)
				}
				defer service.Close()
				handler, err = NewTron(service, "fixture-csrf")
				if err != nil {
					t.Fatal(err)
				}
			}
			h := WithPublicOrigin(SharedDemo(http.StripPrefix("/"+family, handler), token), "https://wallet.example")
			for _, tc := range []struct {
				action, body, credential, csrf string
				want                           int
			}{
				{"create", `{"password":"` + password + `"}`, "", "fixture-csrf", 401},
				{"create", `{"password":"` + password + `"}`, "wrong", "fixture-csrf", 401},
				{"create", `{"password":"` + password + `"}`, token, "", 403},
				{"create", `{"password":"` + password + `"}`, token, "fixture-csrf", 200},
				{"create", `{"password":"` + password + `"}`, token, "fixture-csrf", 400},
				{"backup", `{"password":"wrong"}`, "", "fixture-csrf", 400},
				{"backup", `{"password":"` + password + `"}`, "", "fixture-csrf", 200},
				{"send", `{}`, "", "", 403},
				{"restore", `{}`, token, "fixture-csrf", 403},
				{"password", `{}`, token, "fixture-csrf", 403},
			} {
				req := httptest.NewRequest("POST", "https://wallet.example/"+family+"/api/"+tc.action, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Wallet-CSRF", tc.csrf)
				if tc.credential != "" {
					req.SetBasicAuth("multi-chain-wallet", tc.credential)
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, req)
				if w.Code != tc.want {
					t.Fatalf("%s: status %d want %d", tc.action, w.Code, tc.want)
				}
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", "https://wallet.example/"+family+"/", nil))
			if w.Code != 200 || !strings.Contains(w.Body.String(), `/static/shared.js`) {
				t.Fatal("shared chain page missing shared controls")
			}
		})
	}
}
