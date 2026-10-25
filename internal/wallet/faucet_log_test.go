package wallet

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSolanaFaucetDiagnosticStages(t *testing.T) {
	for _, mode := range []string{"preflight", "http502", "rpc", "timeout", "success"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					ID     any
					Method string
				}
				json.NewDecoder(r.Body).Decode(&req)
				var result any
				if req.Method == "getGenesisHash" {
					result = SolanaDevnetGenesis
					if mode == "preflight" {
						result = "wrong-network"
					}
				} else if req.Method == "requestAirdrop" {
					calls.Add(1)
					switch mode {
					case "http502":
						w.WriteHeader(502)
						w.Write([]byte("secret-provider-token"))
						return
					case "rpc":
						json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": map[string]any{"code": -32005, "message": "secret-provider-token"}})
						return
					case "timeout":
						<-r.Context().Done()
						return
					}
					result = strings.Repeat("1", 64)
				} else {
					t.Errorf("unexpected RPC %s", req.Method)
				}
				json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
			}))
			defer server.Close()
			service, err := NewSolanaService(server.URL, t.TempDir(), 2)
			if err != nil {
				t.Fatal(err)
			}
			defer service.Close()
			if _, err := service.Create("", "test-password-123"); err != nil {
				t.Fatal(err)
			}
			var logs bytes.Buffer
			ctx := WithFaucetLogger(context.Background(), slog.New(slog.NewJSONHandler(&logs, nil)).With("request_id", "fixture-request"))
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			result, err := service.RequestTestSOL(ctx)
			output := logs.String()
			if strings.Contains(output, "secret-provider-token") || strings.Contains(output, server.URL) {
				t.Fatal("sensitive upstream data logged")
			}
			if !strings.Contains(output, "fixture-request") {
				t.Fatal("missing correlation")
			}
			if mode == "preflight" {
				if err == nil || calls.Load() != 0 || !strings.Contains(output, `"airdrop_attempted":false`) || !strings.Contains(output, "network_check") {
					t.Fatalf("preflight: %v %s", err, output)
				}
				return
			}
			if calls.Load() != 1 {
				t.Fatalf("unexpected airdrop count %d", calls.Load())
			}
			if !strings.Contains(output, "solana_airdrop_call_started") {
				t.Fatal(output)
			}
			if mode == "success" {
				if err != nil || result == nil || !strings.Contains(output, "solana_airdrop_submitted") || !strings.Contains(output, result.Signature) {
					t.Fatalf("%v %s", err, output)
				}
			} else {
				if err == nil || !strings.Contains(output, "solana_airdrop_result_unknown") {
					t.Fatalf("%v %s", err, output)
				}
				if strings.Contains(err.Error(), "secret-provider-token") || strings.Contains(err.Error(), server.URL) {
					t.Fatal("sensitive upstream data returned")
				}
				events := make(map[string]map[string]any)
				decoder := json.NewDecoder(strings.NewReader(output))
				for decoder.More() {
					var event map[string]any
					if err := decoder.Decode(&event); err != nil {
						t.Fatal(err)
					}
					events[event["msg"].(string)] = event
				}
				unknown := events["solana_airdrop_result_unknown"]
				failed := events["solana_faucet_failed"]
				for _, field := range []string{"error_type", "error_class", "upstream_http_status", "upstream_rpc_code"} {
					if failed[field] != unknown[field] {
						t.Fatalf("inconsistent %s: unknown=%v failed=%v", field, unknown, failed)
					}
				}
				expected := map[string]string{"http502": `"upstream_http_status":502`, "rpc": `"upstream_rpc_code":-32005`, "timeout": `"error_class":"deadline_exceeded"`}[mode]
				if !strings.Contains(output, expected) {
					t.Fatal(output)
				}
			}
		})
	}
}
