package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/a861252012/multi-chain-wallet/internal/wallet"
)

func TestSharedDemoInvalidRequestsDoNotExhaustPasswordBudget(t *testing.T) {
	const password = "disposable-password-123"
	client, err := chain.New("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ws, err := wallet.NewService(client, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	if _, err := ws.Create(password); err != nil {
		t.Fatal(err)
	}
	evm, err := New(client, ws)
	if err != nil {
		t.Fatal(err)
	}
	sol, err := wallet.NewSolanaService("http://127.0.0.1:1", t.TempDir(), 2)
	if err != nil {
		t.Fatal(err)
	}
	defer sol.Close()
	if _, err := sol.Create("", password); err != nil {
		t.Fatal(err)
	}
	tron, err := wallet.NewTronService("http://127.0.0.1:1", "", t.TempDir(), 2)
	if err != nil {
		t.Fatal(err)
	}
	defer tron.Close()
	if _, err := tron.Create("", password); err != nil {
		t.Fatal(err)
	}
	sh, err := NewSolana(sol, ws.CSRFToken())
	if err != nil {
		t.Fatal(err)
	}
	th, err := NewTron(tron, ws.CSRFToken())
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", evm)
	mux.Handle("/solana/", http.StripPrefix("/solana", sh))
	mux.Handle("/tron/", http.StripPrefix("/tron", th))
	// Exercise the same prefix stripping used by the multi-network router.
	const prefix = "/accounts/0123456789abcdef0123456789abcdef/net/base"
	mux.Handle(prefix+"/", http.StripPrefix(prefix, evm))
	h := WithPublicOrigin(SharedDemo(mux, ""), "https://wallet.example")
	call := func(path, body, csrf string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "https://wallet.example"+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Wallet-CSRF", csrf)
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
		return res
	}
	paths := []string{"/api/wallet/send", "/api/wallet/backup", "/api/wallet/accounts", "/solana/api/send", "/solana/api/backup", "/tron/api/send", "/tron/api/backup", prefix + "/api/wallet/send"}
	for _, path := range paths {
		for range 10 {
			if res := call(path, "", ""); res.Code != 403 {
				t.Fatalf("missing CSRF %s: %d", path, res.Code)
			}
		}
		for _, body := range []string{"", "{", "null", "{}", `{"unknown":true}`, `{"password":"x"} {}`} {
			if res := call(path, body, ws.CSRFToken()); res.Code != 400 {
				t.Fatalf("invalid JSON/required fields %s %q: %d", path, body, res.Code)
			}
		}
	}
	// Public account creation remains available and shares the budget with backups.
	if res := call("/api/wallet/accounts", `{"name":"訪客測試","password":"`+password+`"}`, ws.CSRFToken()); res.Code != 200 {
		t.Fatalf("valid public creation: %d %s", res.Code, res.Body)
	}
	// Real backups still verify passwords and share the original ten-attempt budget.
	backups := []string{"/api/wallet/backup", "/solana/api/backup", "/tron/api/backup", prefix + "/api/wallet/backup"}
	for i := range 9 {
		if res := call(backups[i%len(backups)], `{"password":"`+password+`"}`, ws.CSRFToken()); res.Code != 200 {
			t.Fatalf("valid backup %d: %d %s", i, res.Code, res.Body)
		}
	}
	if res := call("/api/wallet/backup", `{"password":"`+password+`"}`, ws.CSRFToken()); res.Code != 429 || res.Header().Get("Retry-After") != "60" {
		t.Fatalf("missing attempt limit: %d", res.Code)
	}
}
