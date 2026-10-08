package web

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/a861252012/multi-chain-wallet/internal/wallet"
)

func TestRoutesAndInputErrors(t *testing.T) {
	// Invalid input must be rejected without contacting this unreachable RPC.
	c, err := chain.New("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	h, err := New(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path, contains string
		status                 int
	}{
		{"GET", "/", "Multi-Chain Wallet", 200}, {"GET", "/static/app.css", ":root", 200},
		{"GET", "/static/app.js", "refreshNetwork", 200}, {"GET", "/healthz", "wallet", 200},
		{"GET", "/api/balance?address=invalid", "地址格式", 400},
		{"GET", "/api/transactions/bad", "交易雜湊格式", 400},
		{"POST", "/api/network", "Method Not Allowed", 405}, {"GET", "/missing", "404", 404},
	} {
		t.Run(tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.contains) {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
			if w.Header().Get("Content-Security-Policy") == "" {
				t.Fatal("missing CSP")
			}
		})
	}
}

func TestWalletSecurityAndRestrictions(t *testing.T) {
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

	csrfToken := ws.CSRFToken()

	// 1. Host validation
	t.Run("host validation", func(t *testing.T) {
		// Invalid host rejected
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "http://localhost:8090/api/wallet", nil)
		req.Host = "evil-domain.com:8090"
		h.ServeHTTP(w, req)
		if w.Code != 400 {
			t.Fatalf("expected 400 for evil host, got %d", w.Code)
		}

		// Allowed host accepted
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "http://localhost:8090/api/wallet", nil)
		req2.Host = "127.0.0.1:8090"
		h.ServeHTTP(w2, req2)
		if w2.Code != 200 {
			t.Fatalf("expected 200 for 127.0.0.1 host, got %d", w2.Code)
		}

		// Status endpoint alias accepted
		wStatus := httptest.NewRecorder()
		reqStatus := httptest.NewRequest("GET", "http://localhost:8090/api/wallet/status", nil)
		reqStatus.Host = "127.0.0.1:8090"
		h.ServeHTTP(wStatus, reqStatus)
		if wStatus.Code != 200 {
			t.Fatalf("expected 200 for /api/wallet/status, got %d", wStatus.Code)
		}
	})

	// 2. Cross-origin rejection
	t.Run("origin and sec-fetch-site", func(t *testing.T) {
		// Cross-site Sec-Fetch-Site rejected
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "http://localhost:8090/api/wallet", nil)
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		h.ServeHTTP(w, req)
		if w.Code != 403 {
			t.Fatalf("expected 403 for cross-site, got %d", w.Code)
		}

		// Untrusted Origin rejected
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "http://localhost:8090/api/wallet", nil)
		req2.Header.Set("Origin", "https://attacker.org")
		h.ServeHTTP(w2, req2)
		if w2.Code != 403 {
			t.Fatalf("expected 403 for attacker origin, got %d", w2.Code)
		}
	})

	// 3. CSRF protection
	t.Run("csrf protection", func(t *testing.T) {
		body := bytes.NewBufferString(`{"password":"test-password-123"}`)
		// Missing CSRF
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "http://localhost:8090/api/wallet/create", body)
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(w, req)
		if w.Code != 403 {
			t.Fatalf("expected 403 for missing CSRF, got %d", w.Code)
		}

		// Invalid CSRF
		body2 := bytes.NewBufferString(`{"password":"test-password-123"}`)
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("POST", "http://localhost:8090/api/wallet/create", body2)
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("X-Wallet-CSRF", "bad-token")
		h.ServeHTTP(w2, req2)
		if w2.Code != 403 {
			t.Fatalf("expected 403 for invalid CSRF, got %d", w2.Code)
		}
	})

	// 4. Content-Type and Body size & trailing input restrictions
	t.Run("body restrictions", func(t *testing.T) {
		// Wrong Content-Type
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "http://localhost:8090/api/wallet/create", bytes.NewBufferString(`{"password":"test-password-123"}`))
		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("X-Wallet-CSRF", csrfToken)
		h.ServeHTTP(w, req)
		if w.Code != 400 {
			t.Fatalf("expected 400 for text/plain, got %d", w.Code)
		}

		// Trailing input
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("POST", "http://localhost:8090/api/wallet/create", bytes.NewBufferString(`{"password":"test-password-123"} trailing`))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("X-Wallet-CSRF", csrfToken)
		h.ServeHTTP(w2, req2)
		if w2.Code != 400 {
			t.Fatalf("expected 400 for trailing input, got %d", w2.Code)
		}

		// Unknown fields
		w3 := httptest.NewRecorder()
		req3 := httptest.NewRequest("POST", "http://localhost:8090/api/wallet/create", bytes.NewBufferString(`{"password":"test-password-123","unknown":true}`))
		req3.Header.Set("Content-Type", "application/json")
		req3.Header.Set("X-Wallet-CSRF", csrfToken)
		h.ServeHTTP(w3, req3)
		if w3.Code != 400 {
			t.Fatalf("expected 400 for unknown fields, got %d", w3.Code)
		}

		// Body > 16KiB
		hugePass := strings.Repeat("x", 20*1024)
		payload, _ := json.Marshal(map[string]string{"password": hugePass})
		w4 := httptest.NewRecorder()
		req4 := httptest.NewRequest("POST", "http://localhost:8090/api/wallet/create", bytes.NewReader(payload))
		req4.Header.Set("Content-Type", "application/json")
		req4.Header.Set("X-Wallet-CSRF", csrfToken)
		h.ServeHTTP(w4, req4)
		if w4.Code != 400 {
			t.Fatalf("expected 400 for body > 16KiB, got %d", w4.Code)
		}
	})
}

func TestWalletRejectsSameSiteDifferentOriginAndAmbiguousJSON(t *testing.T) {
	c, err := chain.New("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ws, err := wallet.NewService(c, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	h, err := New(c, ws)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		origin, ctype, body string
		status              int
	}{
		{"http://localhost:9999", "application/json", `{"password":"fixture-password-123"}`, 403},
		{"http://127.0.0.1:8090", "application/json", `{"password":"fixture-password-123"}`, 403},
		{"http://localhost:8090", "application/json-invalid", `{}`, 400},
		{"http://localhost:8090", "application/json", `{} {}`, 400},
		{"null", "application/json", `{}`, 403},
	} {
		req := httptest.NewRequest("POST", "http://localhost:8090/api/wallet/create", strings.NewReader(tc.body))
		req.Header.Set("Origin", tc.origin)
		req.Header.Set("Content-Type", tc.ctype)
		req.Header.Set("X-Wallet-CSRF", ws.CSRFToken())
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Fatalf("%+v got %d %s", tc, w.Code, w.Body.String())
		}
	}
}

func TestActivityRoutesStayLocalAndExportExactCSV(t *testing.T) {
	c, err := chain.New("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ws, err := wallet.NewService(c, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	if _, err := ws.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "fixture-password-123"); err != nil {
		t.Fatal(err)
	}
	handler, err := New(c, ws)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/wallet/activity/import", "/api/wallet/activity/sync"} {
		req := httptest.NewRequest("POST", "http://localhost:8090"+path, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != 403 {
			t.Fatalf("missing CSRF accepted: %s %d", path, response.Code)
		}
		req = httptest.NewRequest("POST", "http://localhost:8090"+path, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Wallet-CSRF", ws.CSRFToken())
		req.Header.Set("Origin", "http://localhost:9999")
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != 403 {
			t.Fatalf("cross-origin accepted: %s", path)
		}
	}
	for _, path := range []string{"/api/wallet/activity?page=0", "/api/wallet/activity?page=9999999999999999999999"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", "http://localhost:8090"+path, nil))
		if response.Code != 400 {
			t.Fatalf("invalid page accepted: %s", path)
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "http://localhost:8090/api/wallet/activity?format=csv", nil))
	if response.Code != 200 || !strings.HasPrefix(response.Header().Get("Content-Type"), "text/csv") || !strings.Contains(response.Header().Get("Content-Disposition"), "multi-chain-wallet-sepolia-activity.csv") || !strings.Contains(response.Body.String(), "amount_raw") || strings.Contains(response.Body.String(), "fixture-password") {
		t.Fatalf("bad CSV: %d %s headers=%v", response.Code, response.Body.String(), response.Header())
	}
	response = httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://localhost:8090/api/wallet/activity?format=csv", nil)
	req.Host = "example.com"
	handler.ServeHTTP(response, req)
	if response.Code != 400 {
		t.Fatal("foreign host accessed export")
	}
}

// TestActivityCSVExportPhaseBInterruptedCompaction 驗證階段 b 封存成功但活躍檔更新中斷時，跨頁匯出 CSV 正確有序去重、穩定不跳頁且無重複交易行。
func TestActivityCSVExportPhaseBInterruptedCompaction(t *testing.T) {
	c, err := chain.New("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	walletDir := t.TempDir()
	ws, err := wallet.NewService(c, walletDir, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	if _, err := ws.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "fixture-password-123"); err != nil {
		t.Fatal(err)
	}

	// 產生 30 筆合法雜湊
	total := 30
	ids := make([]string, total)
	for i := range ids {
		ids[i] = fmt.Sprintf("0x%064x", i+1)
	}

	// 模擬階段 b：封存檔已寫入前 25 筆，活躍檔未縮減仍有全部 30 筆
	archiveData, _ := json.Marshal(ids[:25])
	archivePath := filepath.Join(walletDir, "activity-archive-00000000000000000001.json")
	if err := os.WriteFile(archivePath, archiveData, 0600); err != nil {
		t.Fatal(err)
	}
	activeData, _ := json.Marshal(ids)
	activePath := filepath.Join(walletDir, "activity.json")
	if err := os.WriteFile(activePath, activeData, 0600); err != nil {
		t.Fatal(err)
	}

	handler, err := New(c, ws)
	if err != nil {
		t.Fatal(err)
	}

	// 請求第 1 頁 CSV（20 筆）與第 2 頁 CSV（10 筆）
	seenHashes := map[string]int{}
	pageRowCounts := map[int]int{}

	for page := 1; page <= 2; page += 1 {
		req := httptest.NewRequest("GET", fmt.Sprintf("http://localhost:8090/api/wallet/activity?format=csv&page=%d", page), nil)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)

		if resp.Code != 200 {
			t.Fatalf("page %d failed with status %d: %s", page, resp.Code, resp.Body.String())
		}
		if !strings.HasPrefix(resp.Header().Get("Content-Type"), "text/csv") {
			t.Fatalf("page %d invalid Content-Type: %s", page, resp.Header().Get("Content-Type"))
		}

		reader := csv.NewReader(resp.Body)
		rows, err := reader.ReadAll()
		if err != nil {
			t.Fatalf("page %d CSV parse failed: %v", page, err)
		}
		if len(rows) < 1 {
			t.Fatalf("page %d CSV has no header", page)
		}
		// 檢查標題列
		header := rows[0]
		expectedHeader := []string{"chain_id", "hash", "state", "block", "block_time", "kind", "asset", "amount_raw", "counterparty", "evidence"}
		if !slices.Equal(header, expectedHeader) {
			t.Fatalf("page %d header mismatch: got %v, want %v", page, header, expectedHeader)
		}

		// 資料列
		dataRows := rows[1:]
		pageRowCounts[page] = len(dataRows)
		for _, row := range dataRows {
			hash := row[1]
			seenHashes[hash] += 1
		}
	}

	// 驗證分頁行數
	if pageRowCounts[1] != 20 {
		t.Fatalf("page 1 expected 20 rows, got %d", pageRowCounts[1])
	}
	if pageRowCounts[2] != 10 {
		t.Fatalf("page 2 expected 10 rows, got %d", pageRowCounts[2])
	}

	// 驗證跨頁完全無重複交易行，且總數與 30 筆完全一致
	if len(seenHashes) != total {
		t.Fatalf("expected %d unique hashes across CSV pages, got %d", total, len(seenHashes))
	}
	for h, count := range seenHashes {
		if count != 1 {
			t.Fatalf("hash %s appeared %d times across CSV pages", h, count)
		}
	}
}

func TestNewWalletMutationRoutesRequireCSRF(t *testing.T) {
	c, err := chain.New("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ws, err := wallet.NewService(c, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	handler, err := New(c, ws)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/wallet/import-keystore", "/api/wallet/password", "/api/wallet/accounts", "/api/wallet/scan", "/api/wallet/exchange/pools"} {
		req := httptest.NewRequest("POST", "http://localhost:8090"+path, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != 403 {
			t.Fatalf("%s accepted without CSRF: %d", path, response.Code)
		}
	}
}

func TestNormalizeWorkspaceRedirect(t *testing.T) {
	cases := []struct {
		path       string
		wantTarget string
		wantOK     bool
	}{
		{path: "/solana", wantTarget: "/solana/", wantOK: true},
		{path: "/tron", wantTarget: "/tron/", wantOK: true},
		{path: "/solana/", wantTarget: "", wantOK: false},
		{path: "/tron/", wantTarget: "", wantOK: false},
		{path: "/solana/api/status", wantTarget: "", wantOK: false},
		{path: "/tron/api/status", wantTarget: "", wantOK: false},
		{path: "/accounts/acc1", wantTarget: "/accounts/acc1/", wantOK: true},
		{path: "/accounts/acc1/", wantTarget: "", wantOK: false},
		{path: "/accounts/acc1/net/arbitrum", wantTarget: "/accounts/acc1/net/arbitrum/", wantOK: true},
		{path: "/accounts/acc1/net/arbitrum/", wantTarget: "", wantOK: false},
		{path: "/accounts/acc1/net/arbitrum/api/network", wantTarget: "", wantOK: false},
		{path: "/accounts", wantTarget: "", wantOK: false},
		{path: "/accounts/", wantTarget: "", wantOK: false},
		{path: "/", wantTarget: "", wantOK: false},
		{path: "/healthz", wantTarget: "", wantOK: false},
		{path: "/api/network", wantTarget: "", wantOK: false},
	}

	for _, tc := range cases {
		gotTarget, gotOK := NormalizeWorkspaceRedirect(tc.path)
		if gotOK != tc.wantOK || gotTarget != tc.wantTarget {
			t.Errorf("NormalizeWorkspaceRedirect(%q) = (%q, %v), want (%q, %v)", tc.path, gotTarget, gotOK, tc.wantTarget, tc.wantOK)
		}
	}

	// Test redirect handler behavior with query preservation
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if target, ok := NormalizeWorkspaceRedirect(r.URL.Path); ok {
			if r.URL.RawQuery != "" {
				target += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	for _, tc := range []struct {
		requestPath  string
		wantLocation string
	}{
		{"/solana", "/solana/"},
		{"/tron", "/tron/"},
		{"/accounts/acc1", "/accounts/acc1/"},
		{"/accounts/acc1/net/base", "/accounts/acc1/net/base/"},
		{"/accounts/acc1/net/base?tab=history", "/accounts/acc1/net/base/?tab=history"},
	} {
		req := httptest.NewRequest("GET", tc.requestPath, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusMovedPermanently {
			t.Fatalf("%s status code: got %d, want %d", tc.requestPath, rec.Code, http.StatusMovedPermanently)
		}
		if got := rec.Header().Get("Location"); got != tc.wantLocation {
			t.Fatalf("%s Location header: got %s, want %s", tc.requestPath, got, tc.wantLocation)
		}
	}

	for _, okPath := range []string{"/solana/", "/tron/", "/accounts/acc1/", "/accounts/acc1/net/base/"} {
		req := httptest.NewRequest("GET", okPath, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status code: got %d, want %d", okPath, rec.Code, http.StatusOK)
		}
	}
}

func TestWalletTokenQueryRoutes(t *testing.T) {
	c, err := chain.New("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ws, err := wallet.NewService(c, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	if _, err := ws.Import("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "fixture-password-123"); err != nil {
		t.Fatal(err)
	}
	handler, err := New(c, ws)
	if err != nil {
		t.Fatal(err)
	}

	// 1. GET 缺少 contract 參數應回傳 400
	getReq := httptest.NewRequest("GET", "http://127.0.0.1:8090/api/wallet/token", nil)
	getReq.Host = "127.0.0.1:8090"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, getReq)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "缺少 contract 參數") {
		t.Fatalf("GET token without contract: got %d %s", rec.Code, rec.Body.String())
	}

	// 2. GET 請求不強制要求 X-Wallet-CSRF
	getReqWithParam := httptest.NewRequest("GET", "http://127.0.0.1:8090/api/wallet/token?contract=0x0000000000000000000000000000000000000001", nil)
	getReqWithParam.Host = "127.0.0.1:8090"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, getReqWithParam)
	// 由於 local RPC 127.0.0.1:1 無法連線，此處預期為 RPC 連線失敗 (502/504) 或 400，但絕不能是 403 CSRF 錯誤
	if rec.Code == http.StatusForbidden {
		t.Fatalf("GET token should not be rejected with 403 CSRF: %d %s", rec.Code, rec.Body.String())
	}

	// 3. POST 請求在缺少 CSRF 時必須被阻擋為 403
	postReqNoCSRF := httptest.NewRequest("POST", "http://127.0.0.1:8090/api/wallet/token", strings.NewReader(`{"contract":"0x0000000000000000000000000000000000000001"}`))
	postReqNoCSRF.Host = "127.0.0.1:8090"
	postReqNoCSRF.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, postReqNoCSRF)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("POST token without CSRF must be 403: got %d", rec.Code)
	}

	// 4. GET 請求帶前後空格應正確被 TrimSpace 處理，且無效合約地址應回傳 400
	getReqTrim := httptest.NewRequest("GET", "http://127.0.0.1:8090/api/wallet/token?contract=%20invalid-addr%20", nil)
	getReqTrim.Host = "127.0.0.1:8090"
	recTrim := httptest.NewRecorder()
	handler.ServeHTTP(recTrim, getReqTrim)
	if recTrim.Code != http.StatusBadRequest {
		t.Fatalf("GET token with invalid address: got %d %s", recTrim.Code, recTrim.Body.String())
	}
}
