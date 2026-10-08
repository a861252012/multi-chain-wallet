package web

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a861252012/multi-chain-wallet/internal/wallet"
)

func TestTronLocalRequestBoundary(t *testing.T) {
	service, err := wallet.NewTronService("http://127.0.0.1:1", "", t.TempDir(), 2)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	handler, err := NewTron(service, "fixture-csrf")
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"create", "quote", "send", "retry", "backup", "restore", "password"} {
		for _, scenario := range []struct {
			name, host, origin, csrf string
			code                     int
		}{
			{"foreign-host", "attacker.example", "", "fixture-csrf", 400},
			{"foreign-origin", "localhost", "https://attacker.example", "fixture-csrf", 403},
			{"missing-csrf", "localhost", "http://localhost", "", 403},
			{"unknown-field", "localhost", "http://localhost", "fixture-csrf", 400},
		} {
			t.Run(action+"/"+scenario.name, func(t *testing.T) {
				req := httptest.NewRequest("POST", "http://"+scenario.host+"/api/"+action, strings.NewReader(`{"unexpected":true}`))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Origin", scenario.origin)
				req.Header.Set("X-Wallet-CSRF", scenario.csrf)
				res := httptest.NewRecorder()
				handler.ServeHTTP(res, req)
				if res.Code != scenario.code {
					t.Fatalf("status %d: %s", res.Code, res.Body.String())
				}
			})
		}
	}
	for _, host := range []string{"localhost", "attacker.example"} {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest("GET", "http://"+host+"/api/status", nil))
		if host == "localhost" {
			if res.Code != 200 || !strings.Contains(res.Body.String(), `"exists":false`) {
				t.Fatalf("status: %s", res.Body.String())
			}
		} else if res.Code != 400 || strings.Contains(res.Body.String(), "fixture-csrf") {
			t.Fatal("foreign host accessed status")
		}
	}
	status, err := service.Status()
	if err != nil || status.Exists {
		t.Fatal("rejected requests created a wallet")
	}
}
