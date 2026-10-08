package web

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/a861252012/multi-chain-wallet/internal/wallet"
)

func TestAccountManagementAPI(t *testing.T) {
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
	handler, err := New(client, ws)
	if err != nil {
		t.Fatal(err)
	}
	call := func(path string, body any, csrf bool) *httptest.ResponseRecorder {
		t.Helper()
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("POST", "http://localhost:8090"+path, bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://localhost:8090")
		if csrf {
			req.Header.Set("X-Wallet-CSRF", ws.CSRFToken())
		}
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		return res
	}
	for _, path := range []string{"/api/wallet/accounts", "/api/wallet/accounts/update"} {
		if res := call(path, map[string]any{"name": "測試"}, false); res.Code != 403 {
			t.Fatalf("missing CSRF: %d", res.Code)
		}
	}
	if res := call("/api/wallet/accounts", map[string]any{"name": ""}, true); res.Code != 400 {
		t.Fatal("empty name accepted")
	}
	res := call("/api/wallet/accounts", map[string]any{"name": "測試"}, true)
	if res.Code != 200 {
		t.Fatalf("create: %d %s", res.Code, res.Body)
	}
	var added evmAccount
	if err := json.Unmarshal(res.Body.Bytes(), &added); err != nil {
		t.Fatal(err)
	}
	for _, archived := range []bool{false, true, false} {
		res = call("/api/wallet/accounts/update", map[string]any{"id": added.ID, "name": "新名稱", "archived": archived}, true)
		if res.Code != 200 {
			t.Fatalf("update: %d %s", res.Code, res.Body)
		}
		var updated evmAccount
		if err := json.Unmarshal(res.Body.Bytes(), &updated); err != nil {
			t.Fatal(err)
		}
		if updated.Name != "新名稱" || updated.Archived != archived {
			t.Fatalf("response: %+v", updated)
		}
	}
	res = call("/api/wallet/accounts/update", map[string]any{"id": "../", "name": "測試", "archived": true}, true)
	if res.Code != 400 {
		t.Fatal("invalid id accepted")
	}
}
