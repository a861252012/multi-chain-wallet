package web

import (
	"bytes"
	"encoding/json"
	"github.com/a861252012/multi-chain-wallet/internal/wallet"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFaucetRejectsForeignAndUnboundedRequests(t *testing.T) {
	h := SharedDemo(NewFaucet(&wallet.TestFaucet{}, nil, "fixture"), "")
	for _, path := range []string{"/api/faucet", "/api/faucet/solana", "/api/faucet/tron"} {
		for _, tc := range []struct {
			host, origin, csrf, body string
			code                     int
		}{
			{"attacker.example", "", "fixture", "{}", 400},
			{"localhost", "https://attacker.example", "fixture", "{}", 403},
			{"localhost", "", "", "{}", 403},
			{"localhost", "", "fixture", `{"amount":"1000000"}`, 400},
		} {
			r := httptest.NewRequest("POST", "http://"+tc.host+path, strings.NewReader(tc.body))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Origin", tc.origin)
			r.Header.Set("X-Wallet-CSRF", tc.csrf)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.code {
				t.Fatalf("%s %d %s", path, w.Code, w.Body.String())
			}
		}
	}
}

func TestSolanaFaucetLogsRequestCorrelation(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(previous)
	service, err := wallet.NewSolanaService("http://127.0.0.1:1", t.TempDir(), 2)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	h := NewFaucet(&wallet.TestFaucet{}, service, "fixture")
	for i, expected := range []int{502, 429} {
		r := httptest.NewRequest("POST", "http://localhost/api/faucet/solana", strings.NewReader("{}"))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Wallet-CSRF", "fixture")
		r.Header.Set("X-Request-ID", "untrusted-client-value")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		id := w.Header().Get("X-Request-ID")
		if w.Code != expected || id == "" || id == "untrusted-client-value" {
			t.Fatalf("%d: status %d id %q", i, w.Code, id)
		}
		if !strings.Contains(logs.String(), id) || !strings.Contains(logs.String(), "solana_faucet_response") {
			t.Fatal(logs.String())
		}
	}
	if !strings.Contains(logs.String(), "wallet_status") || strings.Contains(logs.String(), "solana_airdrop_call_started") {
		t.Fatal(logs.String())
	}
}

func TestFaucetTronCapability(t *testing.T) {
	for _, tc := range []struct {
		name    string
		faucet  *wallet.TestFaucet
		enabled bool
	}{
		{"EVM only", &wallet.TestFaucet{Sources: map[int64]*wallet.Service{1: {}}}, false},
		{"source only", &wallet.TestFaucet{TronSource: &wallet.TronService{}}, false},
		{"missing recipient", &wallet.TestFaucet{TronSource: &wallet.TronService{}, TronPassword: "fixture"}, false},
		{"missing password", &wallet.TestFaucet{TronSource: &wallet.TronService{}, TronRecipient: &wallet.TronService{}}, false},
		{"configured", &wallet.TestFaucet{TronSource: &wallet.TronService{}, TronRecipient: &wallet.TronService{}, TronPassword: "fixture"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			NewFaucet(tc.faucet, nil, "fixture").ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/api/faucet", nil))
			var body struct {
				Enabled     bool   `json:"enabled"`
				TronEnabled bool   `json:"tronEnabled"`
				CSRFToken   string `json:"csrfToken"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if w.Code != 200 || body.Enabled != (tc.name == "EVM only") || body.TronEnabled != tc.enabled || body.CSRFToken != "fixture" {
				t.Fatalf("unexpected capability: %d %s", w.Code, w.Body.String())
			}
		})
	}
}
