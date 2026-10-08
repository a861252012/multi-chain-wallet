package web

import (
	"crypto/rand"
	"crypto/subtle"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"
)

var accessPage = template.Must(template.New("access").Parse(`<!doctype html><html lang="zh-TW"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>開啟 Multi-Chain Wallet</title></head><body><main><h1>開啟 Multi-Chain Wallet</h1><p>輸入管理者提供的存取憑證（WALLET_ACCESS_TOKEN）。這不是錢包密碼或助記詞。</p>{{if .}}<p role="alert">{{.}}</p>{{end}}<form method="post" action="/login"><label for="token">存取憑證</label><input id="token" name="token" type="password" autocomplete="current-password" required autofocus><button type="submit">開啟錢包</button></form><p>同一個瀏覽器工作階段只需登入一次。</p></main></body></html>`))

// RequireAccessToken accepts explicit Basic credentials for CLI clients and an
// HttpOnly session cookie for browsers. It never issues a browser auth challenge.
func RequireAccessToken(next http.Handler, token string) http.Handler {
	session := rand.Text()
	// Public unauthenticated queries and authenticated sessions maintain separate
	// traffic budgets so external public traffic cannot exhaust the administrator's quota.
	publicTraffic := LimitTraffic(next)
	authenticatedTraffic := LimitTraffic(next)
	var loginMu sync.Mutex
	window := time.Now()
	attempts := 0
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			authenticatedTraffic.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		if token == "" {
			if !allowedRequestHost(r) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				http.Error(w, "僅允許本機存取", http.StatusForbidden)
				return
			}
			authenticatedTraffic.ServeHTTP(w, r)
			return
		}
		username, password, ok := r.BasicAuth()
		authenticated := ok && subtle.ConstantTimeCompare([]byte(username), []byte("multi-chain-wallet")) == 1 && subtle.ConstantTimeCompare([]byte(password), []byte(token)) == 1
		// Bound login work globally without trusting spoofable IP headers.
		if (r.URL.Path == "/login" && r.Method == http.MethodPost) || (r.Header.Get("Authorization") != "" && !authenticated) {
			loginMu.Lock()
			if time.Since(window) >= time.Minute {
				window, attempts = time.Now(), 0
			}
			limited := attempts >= 10
			if !limited {
				attempts++
			}
			loginMu.Unlock()
			if limited {
				w.Header().Set("Retry-After", "60")
				http.Error(w, "登入嘗試過於頻繁", http.StatusTooManyRequests)
				return
			}
		}
		if r.URL.Path == "/login" && r.Method == http.MethodPost {
			if !allowedRequestHost(r) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				http.Error(w, "不允許跨來源登入", http.StatusForbidden)
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" && origin != requestOrigin(r) {
				http.Error(w, "不允許跨來源登入", http.StatusForbidden)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 4096)
			if err := r.ParseForm(); err != nil || subtle.ConstantTimeCompare([]byte(r.PostForm.Get("token")), []byte(token)) != 1 {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				_ = accessPage.Execute(w, "存取憑證不正確，請重試。")
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "multi-chain-wallet_session", Value: session, Path: "/", HttpOnly: true, Secure: strings.HasPrefix(requestOrigin(r), "https://"), SameSite: http.SameSiteStrictMode})
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if cookie, err := r.Cookie("multi-chain-wallet_session"); err == nil && subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(session)) == 1 {
			authenticated = true
		}
		if authenticated {
			authenticatedTraffic.ServeHTTP(w, r)
			return
		}
		if _, public := r.Context().Value(publicOriginKey{}).(string); public && publicReadAllowed(r) {
			if r.URL.Path == "/" || strings.HasSuffix(r.URL.Path, "/") {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				native := "ETH"
				if r.URL.Path == "/net/polygon/" {
					native = "POL"
				}
				_ = publicPage.Execute(w, map[string]any{"Native": native, "Public": true})
			} else {
				publicTraffic.ServeHTTP(w, r)
			}
			return
		}
		if r.Method == http.MethodGet && strings.Contains(r.Header.Get("Accept"), "text/html") && !strings.Contains(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
			_ = accessPage.Execute(w, "")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"登入已失效，請重新整理頁面並登入。"}`))
	})
}
