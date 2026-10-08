package web

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/a861252012/multi-chain-wallet/internal/wallet"
	sol "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
)

func TestSolanaLocalRequestBoundary(t *testing.T) {
	service, err := wallet.NewSolanaService("http://127.0.0.1:1", t.TempDir(), 2)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	handler, err := NewSolana(service, "fixture-csrf")
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

func TestSolanaHistoryReportsSanitizedFailureThenRecovers(t *testing.T) {
	var recovered atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		var result any
		switch req.Method {
		case "getGenesisHash":
			result = wallet.SolanaDevnetGenesis
		case "getBlockHeight":
			result = 100
		case "getSignatureStatuses":
			if !recovered.Load() {
				json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": map[string]any{"code": -32000, "message": "https://rpc.example/?token=private-secret"}})
				return
			}
			result = map[string]any{"context": map[string]int{"slot": 100}, "value": []any{map[string]any{"slot": 100, "err": nil, "confirmationStatus": "finalized", "confirmations": nil}}}
		default:
			t.Errorf("unexpected Solana RPC %s", req.Method)
		}
		json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	defer server.Close()

	private := sol.PrivateKey(ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)))
	address := private.PublicKey()
	tx, err := sol.NewTransaction([]sol.Instruction{system.NewTransferInstruction(1, address, address).Build()}, sol.Hash{}, sol.TransactionPayer(address))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Sign(func(sol.PublicKey) *sol.PrivateKey { return &private }); err != nil {
		t.Fatal(err)
	}
	raw, err := tx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	journal, err := json.Marshal([]map[string]any{{
		"signature": tx.Signatures[0].String(), "quoteId": "history-fixture", "to": address.String(),
		"amount": "0.000000001", "state": "submitted", "lastValidBlockHeight": 200,
		"signedRaw": base64.StdEncoding.EncodeToString(raw),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "transactions.json"), journal, 0600); err != nil {
		t.Fatal(err)
	}
	service, err := wallet.NewSolanaService(server.URL, dir, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	handler, err := NewSolana(service, "fixture-csrf")
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "http://localhost/api/history", nil))
	var failure map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &failure); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusBadGateway || len(failure) != 1 || failure["error"] != "Solana RPC 查詢失敗或逾時；目前結果未知" {
		t.Fatalf("expected sanitized history failure: %d %s", response.Code, response.Body.String())
	}

	recovered.Store(true)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "http://localhost/api/history", nil))
	var history []struct {
		Signature string `json:"signature"`
		State     string `json:"state"`
		Finalized bool   `json:"finalized"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &history); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || len(history) != 1 || history[0].Signature != tx.Signatures[0].String() || history[0].State != "finalized" || !history[0].Finalized {
		t.Fatalf("recovered history must retain the array contract and transaction: %d %s", response.Code, response.Body.String())
	}
}
