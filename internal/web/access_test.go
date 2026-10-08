package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequireAccessToken(t *testing.T) {
	const token = "0123456789abcdef0123456789abcdef"
	handler := RequireAccessToken(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), token)

	for _, test := range []struct {
		name, username, password string
		want                     int
	}{
		{name: "missing", want: http.StatusUnauthorized},
		{name: "wrong user", username: "admin", password: token, want: http.StatusUnauthorized},
		{name: "wrong token", username: "multi-chain-wallet", password: token + "x", want: http.StatusUnauthorized},
		{name: "valid", username: "multi-chain-wallet", password: token, want: http.StatusNoContent},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
			if test.username != "" || test.password != "" {
				req.SetBasicAuth(test.username, test.password)
			}
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)
			if res.Code != test.want {
				t.Fatalf("status %d, want %d", res.Code, test.want)
			}
			if test.want == http.StatusUnauthorized && res.Header().Get("WWW-Authenticate") != "" {
				t.Fatal("browser authentication challenge must not be sent")
			}
		})
	}

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "http://attacker.example/healthz", nil))
	if health.Code != http.StatusNoContent {
		t.Fatalf("health status %d", health.Code)
	}
}

func TestRequireAccessTokenCoversWalletRouteFamilies(t *testing.T) {
	const token = "0123456789abcdef0123456789abcdef"
	handler := RequireAccessToken(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), token)

	for _, path := range []string{
		"/", "/api/wallet", "/net/base/api/wallet", "/accounts/test/api/wallet",
		"/tron/api/status", "/solana/api/status", "/api/faucet", "/healthz/extra",
	} {
		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "http://localhost"+path, nil)
		req.Host = "localhost"
		req.RemoteAddr = "198.51.100.8:43210"
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated %s returned %d", path, res.Code)
		}
	}
}

func TestBrowserLoginSession(t *testing.T) {
	const token = "0123456789abcdef0123456789abcdef"
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	handler := RequireAccessToken(next, token)
	request := httptest.NewRequest("GET", "http://localhost/", nil)
	request.Header.Set("Accept", "text/html")
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, request)
	if page.Code != 200 || !strings.Contains(page.Body.String(), "存取憑證") || page.Header().Get("WWW-Authenticate") != "" {
		t.Fatal("expected inline login")
	}
	for _, value := range []string{"wrong", token} {
		req := httptest.NewRequest("POST", "http://localhost/login", strings.NewReader("token="+value))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "http://localhost")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Header().Get("WWW-Authenticate") != "" {
			t.Fatal("login triggered challenge")
		}
		if value == "wrong" {
			if res.Code != 401 {
				t.Fatal(res.Code)
			}
			continue
		}
		if res.Code != 303 {
			t.Fatal(res.Code)
		}
		cookies := res.Result().Cookies()
		if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode || cookies[0].Value == token {
			t.Fatal("invalid session cookie")
		}
		for _, path := range []string{"/", "/api/wallet", "/static/wallet.js"} {
			req := httptest.NewRequest("GET", "http://localhost"+path, nil)
			req.AddCookie(cookies[0])
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)
			if res.Code != 204 {
				t.Fatal("session not accepted", path, res.Code)
			}
		}
		req = httptest.NewRequest("GET", "http://localhost/api/wallet", nil)
		req.AddCookie(cookies[0])
		res = httptest.NewRecorder()
		RequireAccessToken(next, token).ServeHTTP(res, req)
		if res.Code != 401 || res.Header().Get("WWW-Authenticate") != "" {
			t.Fatal("stale session not rejected quietly")
		}
	}
	req := httptest.NewRequest("POST", "http://localhost/login", strings.NewReader("token="+token))
	req.Header.Set("Origin", "https://evil.example")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != 403 {
		t.Fatal("cross-origin login accepted")
	}
}

func TestLocalDemoNeedsNoLogin(t *testing.T) {
	handler := RequireAccessToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }), "")
	for _, path := range []string{"/", "/static/wallet.js", "/api/wallet"} {
		req := httptest.NewRequest("GET", "http://localhost:8090"+path, nil)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != 204 || res.Header().Get("WWW-Authenticate") != "" {
			t.Fatal("demo requires login", path, res.Code)
		}
	}
	for _, host := range []string{"evil.example", "localhost:8090"} {
		req := httptest.NewRequest("GET", "http://"+host+"/", nil)
		if host == "localhost:8090" {
			req.Header.Set("Sec-Fetch-Site", "cross-site")
		}
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != 403 {
			t.Fatal("unsafe demo request accepted")
		}
	}
}
