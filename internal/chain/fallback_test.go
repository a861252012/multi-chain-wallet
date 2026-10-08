package chain

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestFallbackRejectsWrongChain(t *testing.T) {
	var wrongCalls atomic.Int32
	var wrongProbes atomic.Int32
	wrong := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Method == "eth_chainId" {
			wrongProbes.Add(1)
		}
		if req.Method != "eth_chainId" {
			wrongCalls.Add(1)
		}
		json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x1"})
	}))
	defer wrong.Close()
	var rightCalls atomic.Int32
	right := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		result := "0xaa36a7"
		if req.Method == "eth_getTransactionCount" {
			rightCalls.Add(1)
			result = "0x3"
		}
		json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	defer right.Close()
	client, err := NewFallback([]string{wrong.URL, right.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.CheckNetwork(context.Background()); err != nil {
		t.Fatal(err)
	}
	nonce, err := client.NonceAt(context.Background(), common.Address{})
	if err != nil || nonce != 3 || rightCalls.Load() != 1 {
		t.Fatalf("fallback did not execute on testnet: %v", err)
	}
	if wrongProbes.Load() != 1 {
		t.Fatal("did not retain healthy fallback")
	}
	if wrongCalls.Load() != 0 {
		t.Fatal("forwarded operation to mainnet")
	}
}

func TestFallbackFastPathNoRedundantProbe(t *testing.T) {
	var primaryProbes atomic.Int32
	var primaryCalls atomic.Int32
	var primaryDown atomic.Bool

	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if r.Header.Get("X-Multi-Chain-Wallet-Probe") == "1" {
			primaryProbes.Add(1)
			json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0xaa36a7"})
			return
		}
		if req.Method == "eth_chainId" {
			json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0xaa36a7"})
			return
		}
		if primaryDown.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		primaryCalls.Add(1)
		json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x5"})
	}))
	defer primary.Close()

	var backupProbes atomic.Int32
	var backupCalls atomic.Int32
	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if r.Header.Get("X-Multi-Chain-Wallet-Probe") == "1" {
			backupProbes.Add(1)
			json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0xaa36a7"})
			return
		}
		if req.Method == "eth_chainId" {
			json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0xaa36a7"})
			return
		}
		backupCalls.Add(1)
		json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x6"})
	}))
	defer backup.Close()

	client, err := NewFallback([]string{primary.URL, backup.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// 1. Initial network check performs probe and verifies chain ID
	if err := client.CheckNetwork(context.Background()); err != nil {
		t.Fatal(err)
	}
	if primaryProbes.Load() != 1 {
		t.Fatalf("expected 1 initial transport probe, got %d", primaryProbes.Load())
	}

	// 2. Subsequent 5 requests MUST take the fast path with zero additional probes
	for i := range 5 {
		n, err := client.NonceAt(context.Background(), common.Address{})
		if err != nil || n != 5 {
			t.Fatalf("fast-path call %d failed: %v", i, err)
		}
	}
	if primaryProbes.Load() != 1 {
		t.Fatalf("fast-path executed redundant probe: expected 1, got %d", primaryProbes.Load())
	}
	if primaryCalls.Load() != 5 {
		t.Fatalf("expected 5 primary calls, got %d", primaryCalls.Load())
	}
	if backupProbes.Load() != 0 || backupCalls.Load() != 0 {
		t.Fatalf("backup touched prematurely: probes=%d calls=%d", backupProbes.Load(), backupCalls.Load())
	}

	// 3. Primary fails: failover must probe backup and switch
	primaryDown.Store(true)
	n2, err := client.NonceAt(context.Background(), common.Address{})
	if err != nil || n2 != 6 {
		t.Fatalf("failover call failed: %v, nonce: %d", err, n2)
	}
	if backupProbes.Load() != 1 {
		t.Fatalf("expected backup to be probed upon failover, got %d", backupProbes.Load())
	}
	if backupCalls.Load() != 1 {
		t.Fatalf("expected 1 backup call, got %d", backupCalls.Load())
	}

	// 4. Subsequent call to backup also takes fast path with zero probes
	n3, err := client.NonceAt(context.Background(), common.Address{})
	if err != nil || n3 != 6 {
		t.Fatalf("backup fast-path call failed: %v, nonce: %d", err, n3)
	}
	if backupProbes.Load() != 1 {
		t.Fatalf("backup fast path executed redundant probe: expected 1, got %d", backupProbes.Load())
	}
	if backupCalls.Load() != 2 {
		t.Fatalf("expected 2 backup calls, got %d", backupCalls.Load())
	}
}

func TestCheckNetworkCaching(t *testing.T) {
	var chainIDCalls atomic.Int32
	var headerCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Method == "eth_chainId" {
			chainIDCalls.Add(1)
			json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0xaa36a7"})
			return
		}
		if req.Method == "eth_getBlockByNumber" {
			headerCalls.Add(1)
			header := &types.Header{Number: big.NewInt(100), Difficulty: big.NewInt(0), GasLimit: 30000000, Time: 1700000000}
			json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": header})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x1"})
	}))
	defer server.Close()

	client, err := New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()
	// First high-level method call initializes and verifies the network
	h1, err := client.HeaderByNumber(ctx, nil)
	if err != nil || h1 == nil {
		t.Fatalf("first HeaderByNumber failed: %v", err)
	}
	if chainIDCalls.Load() != 1 {
		t.Fatalf("expected 1 chain ID query on initial call, got %d", chainIDCalls.Load())
	}
	if headerCalls.Load() != 1 {
		t.Fatalf("expected 1 header call, got %d", headerCalls.Load())
	}

	// Repeated high-level method calls must reuse cached network verification with zero extra eth_chainId queries
	for i := range 10 {
		h, err := client.HeaderByNumber(ctx, nil)
		if err != nil || h == nil {
			t.Fatalf("HeaderByNumber call %d failed: %v", i, err)
		}
	}
	if chainIDCalls.Load() != 1 {
		t.Fatalf("redundant chainId roundtrips in high-level calls: expected 1, got %d", chainIDCalls.Load())
	}
	if headerCalls.Load() != 11 {
		t.Fatalf("expected 11 header calls, got %d", headerCalls.Load())
	}
}

func TestFallbackCallerContextCancellationKeepsHealthy(t *testing.T) {
	var probes atomic.Int32
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if r.Header.Get("X-Multi-Chain-Wallet-Probe") == "1" || req.Method == "eth_chainId" {
			probes.Add(1)
			json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0xaa36a7"})
			return
		}
		calls.Add(1)
		// Delay to allow client context cancellation
		time.Sleep(50 * time.Millisecond)
		json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x7"})
	}))
	defer server.Close()

	client, err := NewFallback([]string{server.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// Initial verification
	if err := client.CheckNetwork(context.Background()); err != nil {
		t.Fatal(err)
	}
	initialProbes := probes.Load()

	// Normal call succeeds on fast path
	n, err := client.NonceAt(context.Background(), common.Address{})
	if err != nil || n != 7 {
		t.Fatalf("unexpected call failure: %v", err)
	}
	if probes.Load() != initialProbes {
		t.Fatalf("expected no extra probes on fast path, got %d", probes.Load())
	}

	// Cancelled call: should fail with context error but NOT poison healthy flag
	canceledCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, _ = client.NonceAt(canceledCtx, common.Address{})

	// Subsequent normal call must STILL take fast path with zero additional probes
	n2, err := client.NonceAt(context.Background(), common.Address{})
	if err != nil || n2 != 7 {
		t.Fatalf("fast path after context cancel failed: %v", err)
	}
	if probes.Load() != initialProbes {
		t.Fatalf("caller context cancellation poisoned healthy flag, triggered probes: expected %d, got %d", initialProbes, probes.Load())
	}
}

func TestConcurrentCheckNetworkCachesVerification(t *testing.T) {
	var chainIDCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Method == "eth_chainId" {
			chainIDCalls.Add(1)
			time.Sleep(10 * time.Millisecond)
			json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0xaa36a7"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x1"})
	}))
	defer server.Close()

	client, err := New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// 20 concurrent goroutines call checkNetwork simultaneously
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for range 20 {
		wg.Go(func() {
			if err := client.checkNetwork(context.Background()); err != nil {
				errs <- err
			}
		})
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("checkNetwork failed: %v", err)
	}

	initialCalls := chainIDCalls.Load()
	if initialCalls < 1 {
		t.Fatal("network was not verified")
	}
	if err := client.checkNetwork(context.Background()); err != nil {
		t.Fatal(err)
	}
	if chainIDCalls.Load() != initialCalls {
		t.Fatal("completed network verification was not cached")
	}
}

func TestCheckNetworkDeadlineDoesNotWaitForOtherVerification(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID any `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		if calls.Add(1) == 1 {
			close(entered)
		}
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0xaa36a7"})
	}))
	defer server.Close()
	defer close(release)
	client, err := New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	first := make(chan error, 1)
	go func() { first <- client.CheckNetwork(t.Context()) }()
	<-entered

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	second := make(chan error, 1)
	go func() {
		_, err := client.NonceAt(ctx, common.Address{})
		second <- err
	}()
	select {
	case err := <-second:
		if !errors.Is(err, ErrTimeout) {
			t.Fatalf("deadline error: got %v, want %v", err, ErrTimeout)
		}
	case <-time.After(time.Second):
		t.Fatal("expired request waited for an unrelated network verification")
	}
	select {
	case err := <-first:
		t.Fatalf("first verification ended before release: %v", err)
	default:
	}
}
