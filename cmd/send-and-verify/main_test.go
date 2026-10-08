package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestQuoteAndSend(t *testing.T) {
	for _, tc := range []struct {
		name, wei string
		status    int
		quote     string
		args      []string
		input     string
		wantErr   bool
		sends     int
	}{
		{name: "funded quote", wei: "5000", status: 200, quote: `{"id":"bound"}`, args: []string{"--test-quote"}},
		{name: "zero guard", wei: "0", status: 400, quote: `{"error":"餘額不足以支付轉帳金額與最高 Gas 手續費"}`, args: []string{"--test-quote"}},
		{name: "RPC failure", wei: "0", status: 502, quote: `{"error":"RPC unavailable"}`, args: []string{"--test-quote"}, wantErr: true},
		{name: "unexpected zero success", wei: "0", status: 200, quote: `{"id":"bound"}`, args: []string{"--test-quote"}, wantErr: true},
		{name: "cancel", wei: "5000", status: 200, quote: `{"id":"bound"}`, args: []string{"--send"}, input: "NO\n", wantErr: true},
		{name: "send once", wei: "5000", status: 200, quote: `{"id":"bound"}`, args: []string{"--send"}, input: "SEND\n", sends: 1},
		{name: "CLI token", wei: "5000", status: 200, quote: `{"id":"bound"}`, args: []string{"--test-quote", "--token", "override"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sends, passwords := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				wantToken := "env-token"
				if tc.name == "CLI token" {
					wantToken = "override"
				}
				user, token, ok := r.BasicAuth()
				if !ok || user != "multi-chain-wallet" || token != wantToken {
					t.Errorf("unexpected auth")
				}
				if r.Method == http.MethodPost && r.Header.Get("X-Wallet-CSRF") != "csrf" {
					t.Error("missing CSRF")
				}
				switch r.URL.Path {
				case "/api/network":
					w.Write([]byte(`{"chainId":11155111}`))
				case "/api/wallet":
					w.Write([]byte(`{"exists":true,"address":"0x1111111111111111111111111111111111111111","csrfToken":"csrf"}`))
				case "/api/balance":
					json.NewEncoder(w).Encode(map[string]string{"wei": tc.wei})
				case "/api/wallet/quote":
					w.WriteHeader(tc.status)
					w.Write([]byte(tc.quote))
				case "/api/wallet/send":
					sends++
					var body struct{ QuoteID, Password string }
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.QuoteID != "bound" || body.Password != "secret" {
						t.Error("wrong signing request")
					}
					w.Write([]byte(`{"hash":"0x` + strings.Repeat("1", 64) + `","state":"submitted"}`))
				default:
					if !strings.HasPrefix(r.URL.Path, "/api/transactions/") {
						t.Errorf("unexpected path %s", r.URL.Path)
					}
					w.Write([]byte(`{"state":"succeeded"}`))
				}
			}))
			defer server.Close()
			var output bytes.Buffer
			err := run(append(tc.args, "--base-url", server.URL), "env-token", server.Client(), strings.NewReader(tc.input), &output, func() ([]byte, error) { passwords++; return []byte("secret"), nil }, func(time.Duration) {})
			if (err != nil) != tc.wantErr {
				t.Fatalf("error %v output %s", err, output.String())
			}
			if sends != tc.sends || passwords != tc.sends {
				t.Fatalf("send/password counts %d/%d", sends, passwords)
			}
		})
	}
}
func TestRejectArgumentsBeforeNetwork(t *testing.T) {
	for _, args := range [][]string{{}, {"--send", "--test-quote"}, {"--hash", "bad"}, {"--test-quote", "--base-url", "https://example.com"}, {"--test-quote", "--base-url", "http://localhost/path"}} {
		if err := run(args, "", &http.Client{}, strings.NewReader(""), &bytes.Buffer{}, nil, nil); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
func TestReceiptStates(t *testing.T) {
	for _, state := range []string{"succeeded", "reverted", "pending"} {
		t.Run(state, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "GET" || r.Header.Get("Authorization") != "" {
					t.Error("unexpected request")
				}
				json.NewEncoder(w).Encode(map[string]string{"state": state})
			}))
			defer server.Close()
			err := run([]string{"--hash", "0x" + strings.Repeat("1", 64), "--base-url", server.URL}, "", server.Client(), strings.NewReader(""), &bytes.Buffer{}, func() ([]byte, error) { return nil, errors.New("must not prompt") }, func(time.Duration) {})
			if (err == nil) != (state == "succeeded") {
				t.Fatal(err)
			}
			want := 1
			if state == "pending" {
				want = 40
			}
			if calls != want {
				t.Fatalf("calls %d", calls)
			}
		})
	}
}
func TestRedirectDoesNotForwardCredentials(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("redirect followed") }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	if err := run([]string{"--test-quote", "--base-url", server.URL}, "secret", server.Client(), strings.NewReader(""), &bytes.Buffer{}, nil, nil); err == nil {
		t.Fatal("accepted redirect")
	}
}
