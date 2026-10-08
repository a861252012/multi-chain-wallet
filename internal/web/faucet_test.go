package web

import (
	"github.com/a861252012/multi-chain-wallet/internal/wallet"
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
