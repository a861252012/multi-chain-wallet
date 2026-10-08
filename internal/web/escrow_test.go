package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/a861252012/multi-chain-wallet/internal/wallet"
)

func TestEscrowRoutesDisabledAndAccess(t *testing.T) {
	client, err := chain.New("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	service, err := wallet.NewService(client, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	handler, err := New(client, service)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "http://localhost/api/wallet/escrow", nil))
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != `{"enabled":false,"contract":"","token":"","balance":"","allowance":""}` {
		t.Fatal(rec.Code, rec.Body.String())
	}
	for _, path := range []string{"/api/wallet/escrow/order", "/api/wallet/escrow/order?buyer=bad&orderId=valid"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest("GET", "http://localhost"+path, nil))
		if rec.Code != 400 {
			t.Fatal(rec.Code)
		}
	}
	request := httptest.NewRequest("GET", "http://localhost/api/wallet/escrow", nil)
	request.Header.Set("Origin", "https://attacker.example")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, request)
	if rec.Code != http.StatusForbidden {
		t.Fatal("cross-origin escrow access", rec.Code)
	}
	request = httptest.NewRequest("POST", "http://localhost/api/wallet/quote", strings.NewReader(`{"action":"escrow_fund","to":"0x1111111111111111111111111111111111111111","orderId":"test","amount":"1"}`))
	request.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, request)
	if rec.Code != http.StatusForbidden {
		t.Fatal("missing CSRF accepted", rec.Code)
	}
}
