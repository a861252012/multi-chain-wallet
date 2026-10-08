package web

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
)

func TestObservationAvailableWithoutSigningWallet(t *testing.T) {
	client, err := chain.NewNetwork(84532, []string{"http://127.0.0.1:1"})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	handler, err := New(client)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/watch/token?address=bad", "/api/watch/activity?address=bad"} {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest("GET", path, nil))
		if res.Code != 400 {
			t.Fatalf("%s %d", path, res.Code)
		}
	}

	diagRes := httptest.NewRecorder()
	handler.ServeHTTP(diagRes, httptest.NewRequest("GET", "/api/diagnostics", nil))
	if diagRes.Code != 200 {
		t.Fatalf("/api/diagnostics returned %d", diagRes.Code)
	}
	var diag map[string]any
	if err := json.Unmarshal(diagRes.Body.Bytes(), &diag); err != nil {
		t.Fatal(err)
	}
	rpc, ok := diag["rpc"].(map[string]any)
	if !ok || rpc == nil {
		t.Fatalf("missing rpc map in diagnostics response: %v", diag)
	}
	for _, key := range []string{"requests", "transportFailures", "failovers", "lastRequestMs", "activeEndpoint", "endpointCount"} {
		if _, exists := rpc[key]; !exists {
			t.Fatalf("missing rpc metric %s in diagnostics response", key)
		}
	}

	// Verify uninstrumented client also maintains structured rpc contract
	uninstClient, err := chain.New("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	defer uninstClient.Close()
	uninstHandler, err := New(uninstClient)
	if err != nil {
		t.Fatal(err)
	}
	uninstRes := httptest.NewRecorder()
	uninstHandler.ServeHTTP(uninstRes, httptest.NewRequest("GET", "/api/diagnostics", nil))
	if uninstRes.Code != 200 {
		t.Fatalf("/api/diagnostics uninstrumented returned %d", uninstRes.Code)
	}
	var uninstDiag map[string]any
	if err := json.Unmarshal(uninstRes.Body.Bytes(), &uninstDiag); err != nil {
		t.Fatal(err)
	}
	uninstRPC, ok := uninstDiag["rpc"].(map[string]any)
	if !ok || uninstRPC == nil {
		t.Fatalf("missing rpc map in uninstrumented diagnostics response: %v", uninstDiag)
	}
	for _, key := range []string{"requests", "transportFailures", "failovers", "lastRequestMs", "activeEndpoint", "endpointCount"} {
		if _, exists := uninstRPC[key]; !exists {
			t.Fatalf("missing rpc metric %s in uninstrumented diagnostics response", key)
		}
	}
}
