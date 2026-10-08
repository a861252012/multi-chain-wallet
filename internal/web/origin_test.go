package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicOriginBehindHTTPProxy(t *testing.T) {
	const origin = "https://wallet.example"
	const token = "0123456789abcdef0123456789abcdef"
	endpoint := localWalletFilter("csrf", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	h := WithPublicOrigin(RequireAccessToken(LimitTraffic(endpoint), token), origin)
	login := httptest.NewRequest("POST", "http://wallet.example/login", strings.NewReader("token="+token))
	login.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	login.Header.Set("Origin", origin)
	login.Header.Set("X-Forwarded-Proto", "http")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, login)
	if w.Code != 303 || len(w.Result().Cookies()) != 1 {
		t.Fatalf("proxy login: %d %s", w.Code, w.Body.String())
	}
	cookie := w.Result().Cookies()[0]
	if !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatal("insecure public cookie", cookie)
	}
	for _, tc := range []struct {
		name, host, origin, csrf string
		authorized               bool
		want                     int
	}{
		{"valid", "wallet.example", origin, "csrf", true, 204},
		{"no origin with csrf", "wallet.example", "", "csrf", true, 204},
		{"wrong host", "evil.example", origin, "csrf", true, 403},
		{"localhost bypass", "localhost", origin, "csrf", true, 403},
		{"wrong origin", "wallet.example", "https://evil.example", "csrf", true, 403},
		{"downgrade", "wallet.example", "http://wallet.example", "csrf", true, 403},
		{"origin path", "wallet.example", origin + "/", "csrf", true, 403},
		{"no csrf", "wallet.example", origin, "", true, 403},
		{"anonymous", "wallet.example", origin, "csrf", false, 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "http://"+tc.host+"/api/wallet", nil)
			r.Header.Set("Origin", tc.origin)
			r.Header.Set("X-Wallet-CSRF", tc.csrf)
			r.Header.Set("X-Forwarded-Host", "wallet.example")
			r.Header.Set("X-Forwarded-Proto", "https")
			if tc.authorized {
				r.AddCookie(cookie)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("got %d, want %d", w.Code, tc.want)
			}
		})
	}
}

func TestLoginBudgetDoesNotBlockExistingSessionOrTrustForwardedIP(t *testing.T) {
	const token = "0123456789abcdef0123456789abcdef"
	h := RequireAccessToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }), token)
	r := httptest.NewRequest("POST", "http://localhost/login", strings.NewReader("token="+token))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	cookie := w.Result().Cookies()[0]
	for i := range 10 {
		r := httptest.NewRequest("POST", "http://localhost/login", strings.NewReader("token=wrong"))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("X-Forwarded-For", strings.Repeat("1", i+1))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		want := 401
		if i == 9 {
			want = 429
		}
		if w.Code != want {
			t.Fatalf("attempt %d: %d", i, w.Code)
		}
	}
	r = httptest.NewRequest("GET", "http://localhost/api/wallet", nil)
	r.AddCookie(cookie)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatal("existing session blocked")
	}
}

func TestAnonymousRequestsCannotConsumeAuthenticatedAPIBudget(t *testing.T) {
	const token = "0123456789abcdef0123456789abcdef"
	h := WithPublicOrigin(RequireAccessToken(LimitTraffic(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })), token), "https://wallet.example")
	for range 301 {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "http://wallet.example/api/wallet", nil))
		if w.Code != 401 {
			t.Fatal(w.Code)
		}
	}
	r := httptest.NewRequest("GET", "http://wallet.example/api/wallet", nil)
	r.SetBasicAuth("multi-chain-wallet", token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatal("anonymous requests exhausted API budget", w.Code)
	}
}

func TestValidBasicAuthSurvivesFailedLoginBudget(t *testing.T) {
	const token = "0123456789abcdef0123456789abcdef"
	h := RequireAccessToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }), token)
	for i := range 11 {
		r := httptest.NewRequest("GET", "http://localhost/api/wallet", nil)
		r.SetBasicAuth("multi-chain-wallet", "wrong")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		want := 401
		if i == 10 {
			want = 429
		}
		if w.Code != want {
			t.Fatalf("attempt %d: %d", i, w.Code)
		}
	}
	for range 11 {
		r := httptest.NewRequest("GET", "http://localhost/api/wallet", nil)
		r.SetBasicAuth("multi-chain-wallet", token)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 204 {
			t.Fatal("valid CLI credentials blocked", w.Code)
		}
	}
}
