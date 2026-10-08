package web

import (
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/a861252012/multi-chain-wallet/internal/wallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestVaultWebRouteDisabled(t *testing.T) {
	c, err := chain.New("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	dir := t.TempDir()
	ws, err := wallet.NewService(c, dir, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	h, err := New(c, ws)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://localhost:8090/api/wallet/vault", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var res struct {
		Enabled    bool   `json:"enabled"`
		Contract   string `json:"contract"`
		Balance    string `json:"balance"`
		BalanceRaw string `json:"balanceRaw"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Enabled || res.Contract != "" || res.Balance != "" || res.BalanceRaw != "" {
		t.Fatalf("expected disabled vault info, got %#v", res)
	}

	created, err := ws.Create("test-password-12345")
	if err != nil {
		t.Fatal(err)
	}

	// 1. vault_deposit when unconfigured returns 400 with 尚未設定合約地址
	depositBody := strings.NewReader(`{"action":"vault_deposit","to":"` + created.Address + `","amount":"0.5"}`)
	depReq := httptest.NewRequest("POST", "http://localhost:8090/api/wallet/quote", depositBody)
	depReq.Header.Set("Content-Type", "application/json")
	depReq.Header.Set("X-Wallet-CSRF", ws.CSRFToken())
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, depReq)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unconfigured vault deposit, got %d", rec.Code)
	}
	var depErr map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &depErr)
	if depErr["error"] != "尚未設定合約地址，暫時無法操作" {
		t.Fatalf("unexpected error message: %v", depErr["error"])
	}

	// 2. vault_withdraw when unconfigured returns 400 with 尚未設定合約地址
	withdrawBody := strings.NewReader(`{"action":"vault_withdraw","to":"` + created.Address + `","amount":"0.5"}`)
	withReq := httptest.NewRequest("POST", "http://localhost:8090/api/wallet/quote", withdrawBody)
	withReq.Header.Set("Content-Type", "application/json")
	withReq.Header.Set("X-Wallet-CSRF", ws.CSRFToken())
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withReq)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unconfigured vault withdraw, got %d", rec.Code)
	}
	var withErr map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &withErr)
	if withErr["error"] != "尚未設定合約地址，暫時無法操作" {
		t.Fatalf("unexpected error message: %v", withErr["error"])
	}

	// 3. Client cannot specify contract
	badContractBody := strings.NewReader(`{"action":"vault_deposit","to":"` + created.Address + `","amount":"0.5","contract":"0x1111111111111111111111111111111111111111"}`)
	badReq := httptest.NewRequest("POST", "http://localhost:8090/api/wallet/quote", badContractBody)
	badReq.Header.Set("Content-Type", "application/json")
	badReq.Header.Set("X-Wallet-CSRF", ws.CSRFToken())
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, badReq)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for client contract specification, got %d", rec.Code)
	}
	var badErr map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &badErr)
	if badErr["error"] != "合約地址由伺服器設定，無法在操作時變更" {
		t.Fatalf("unexpected error message: %v", badErr["error"])
	}
}

func TestVaultWebRouteEnabled(t *testing.T) {
	const testVault = "0x1111111111111111111111111111111111111111"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var res any
		switch req.Method {
		case "eth_chainId":
			res = "0xaa36a7"
		case "eth_getCode":
			res = "0x60806040"
		case "eth_call":
			val := new(big.Int)
			val.SetString("2500000000000000000", 10)
			res = hexutil.Encode(common.LeftPadBytes(val.Bytes(), 32))
		default:
			res = "0x0"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  res,
		})
	}))
	defer server.Close()

	c, err := chain.NewNetwork(chain.SepoliaID, []string{server.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	dir := t.TempDir()
	ws, err := wallet.NewService(c, dir, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	if err := ws.SetVaultAddress(testVault); err != nil {
		t.Fatal(err)
	}
	_, err = ws.Create("password-12345")
	if err != nil {
		t.Fatal(err)
	}

	h, err := New(c, ws)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://localhost:8090/api/wallet/vault", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var res struct {
		Enabled    bool   `json:"enabled"`
		Contract   string `json:"contract"`
		Balance    string `json:"balance"`
		BalanceRaw string `json:"balanceRaw"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if !res.Enabled || res.Contract != testVault || res.Balance != "2.5" || res.BalanceRaw != "2500000000000000000" {
		t.Fatalf("unexpected vault response: %#v", res)
	}
}
