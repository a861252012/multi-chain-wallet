package web

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/a861252012/multi-chain-wallet/internal/wallet"
)

func TestSendRouteMarksOnlyKnownRejections(t *testing.T) {
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
	request := httptest.NewRequest("POST", "http://localhost:8090/api/wallet/send", strings.NewReader(`{"quoteId":"missing","password":"fixture-password"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Wallet-CSRF", service.CSRFToken())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != 400 || body["code"] != "send_rejected" || body["error"] != wallet.ErrQuoteNotFound.Error() {
		t.Fatalf("rejection contract: %d %s", response.Code, response.Body.String())
	}
}
