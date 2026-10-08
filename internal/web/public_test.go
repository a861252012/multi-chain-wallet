package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicDemoProtectsAllPrivateRouteFamilies(t *testing.T) {
	const token = "0123456789abcdef0123456789abcdef"
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	h := WithPublicOrigin(RequireAccessToken(next, token), "https://wallet.example")
	for _, path := range []string{"/api/wallet", "/api/wallet/accounts", "/api/wallet/history", "/api/wallet/scan", "/api/diagnostics", "/api/faucet", "/solana/api/status", "/tron/api/status", "/accounts/test/api/network", "/net/base/api/wallet", "/net/unknown/api/network", "/api/transactions/../wallet", "/static/../api/wallet"} {
		for _, method := range []string{"GET", "HEAD", "POST", "PUT", "DELETE"} {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(method, "http://wallet.example"+path, nil))
			if w.Code != 401 {
				t.Errorf("anonymous %s %s: %d", method, path, w.Code)
			}
		}
	}
	for _, path := range []string{"/api/network", "/api/balance", "/api/watch/token", "/net/base/api/network", "/api/transactions/0x" + strings.Repeat("a", 64)} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "http://wallet.example"+path, nil))
		if w.Code != 204 {
			t.Errorf("public query %s: %d", path, w.Code)
		}
		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("POST", "http://wallet.example"+path, nil))
		if w.Code != 401 {
			t.Errorf("public POST %s: %d", path, w.Code)
		}
	}
	for _, path := range []string{"/api/wallet", "/solana/api/status", "/tron/api/status", "/accounts/test/api/wallet"} {
		r := httptest.NewRequest("GET", "http://wallet.example"+path, nil)
		r.SetBasicAuth("multi-chain-wallet", token)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 204 {
			t.Errorf("owner rejected %s: %d", path, w.Code)
		}
	}
}

func TestPublicNavigationAndSeparateBudgets(t *testing.T) {
	const token = "0123456789abcdef0123456789abcdef"
	h := WithPublicOrigin(RequireAccessToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }), token), "https://wallet.example")
	r := httptest.NewRequest("GET", "http://wallet.example/", nil)
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	r.Header.Set("Sec-Fetch-Mode", "navigate")
	r.Header.Set("Sec-Fetch-Dest", "document")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "公開查詢不需登入") {
		t.Fatal("external portfolio link blocked", w.Code)
	}
	r.Header.Set("Sec-Fetch-Mode", "cors")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross-site fetch accepted")
	}
	for i := range 301 {
		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "http://wallet.example/api/network", nil))
		if i == 300 && w.Code != 429 {
			t.Fatal("public budget absent", w.Code)
		}
	}
	r = httptest.NewRequest("GET", "http://wallet.example/api/wallet", nil)
	r.SetBasicAuth("multi-chain-wallet", token)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatal("public traffic exhausted admin budget")
	}
}

func TestPublicWorkspaceUsesSharedTemplate(t *testing.T) {
	h := WithPublicOrigin(RequireAccessToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }), "0123456789abcdef0123456789abcdef"), "https://wallet.example")
	for _, path := range []string{"/", "/net/base/", "/net/polygon/"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "http://wallet.example"+path, nil))
		body := w.Body.String()
		for _, marker := range []string{`id="app-sidebar"`, `id="wallet-dashboard"`, `id="public-query"`, `/static/public.js`} {
			if w.Code != 200 || !strings.Contains(body, marker) {
				t.Fatalf("shared public workspace %s missing %s: %d", path, marker, w.Code)
			}
		}
		if strings.Contains(body, `/static/wallet.js`) || strings.Contains(body, `/static/workspace.js`) {
			t.Fatal("private controllers loaded for visitor")
		}
	}
}
