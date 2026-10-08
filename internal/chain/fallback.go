package chain

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// NewFallback retries transport failures with identical request bytes. Each endpoint is checked before use.
func NewFallback(endpoints []string) (*Client, error) { return NewNetwork(SepoliaID, endpoints) }

func NewNetwork(chainID int64, endpoints []string) (*Client, error) {
	if chainID != SepoliaID && chainID != ArbitrumSepoliaID && chainID != BaseSepoliaID && chainID != OptimismSepoliaID && chainID != PolygonAmoyID {
		return nil, ErrNetwork
	}
	if len(endpoints) == 0 || len(endpoints) > 4 {
		return nil, ErrUnavailable
	}
	targets := make([]*url.URL, len(endpoints))
	for i, endpoint := range endpoints {
		u, err := url.Parse(endpoint)
		if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
			return nil, ErrUnavailable
		}
		targets[i] = u
	}
	transport := &fallbackTransport{targets: targets, base: http.DefaultTransport, chainID: chainID}
	c, err := rpc.DialOptions(context.Background(), endpoints[0], rpc.WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		return nil, ErrUnavailable
	}
	return &Client{rpc: ethclient.NewClient(c), chainID: chainID, transport: transport}, nil
}

type fallbackTransport struct {
	requests  atomic.Uint64
	failures  atomic.Uint64
	failovers atomic.Uint64
	lastMS    atomic.Int64
	active    atomic.Uint32
	healthy   atomic.Bool
	chainID   int64
	targets   []*url.URL
	base      http.RoundTripper
}

func (t *fallbackTransport) probeEndpoint(req *http.Request, endpoint *url.URL) bool {
	ctx, cancel := context.WithTimeout(req.Context(), 4*time.Second)
	defer cancel()
	probe := req.Clone(ctx)
	probe.URL = endpoint
	probe.Host = endpoint.Host
	probe.Header.Del("Authorization")
	probe.Header.Set("X-Multi-Chain-Wallet-Probe", "1")
	if endpoint.User != nil {
		password, _ := endpoint.User.Password()
		probe.SetBasicAuth(endpoint.User.Username(), password)
	}
	probe.Body = io.NopCloser(bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}`))
	probe.ContentLength = -1
	response, err := t.base.RoundTrip(probe)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	var result struct {
		Result string `json:"result"`
	}
	data, readErr := io.ReadAll(io.LimitReader(response.Body, 4096))
	if readErr != nil || response.StatusCode != http.StatusOK || json.Unmarshal(data, &result) != nil {
		return false
	}
	id, ok := new(big.Int).SetString(result.Result, 0)
	if !ok || id.Cmp(big.NewInt(t.chainID)) != 0 {
		return false
	}
	return true
}

func (t *fallbackTransport) sendAttempt(req *http.Request, endpoint *url.URL, payload []byte) (*http.Response, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(req.Context(), 8*time.Second)
	attempt := req.Clone(ctx)
	attempt.URL = endpoint
	attempt.Host = endpoint.Host
	attempt.Header.Del("Authorization")
	if endpoint.User != nil {
		password, _ := endpoint.User.Password()
		attempt.SetBasicAuth(endpoint.User.Username(), password)
	}
	attempt.Body = io.NopCloser(bytes.NewReader(payload))
	attempt.ContentLength = int64(len(payload))
	response, err := t.base.RoundTrip(attempt)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500 {
		response.Body.Close()
		cancel()
		return nil, nil, ErrUnavailable
	}
	return response, cancel, nil
}

func (t *fallbackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	started := time.Now()
	success := false
	t.requests.Add(1)
	defer func() {
		t.lastMS.Store(time.Since(started).Milliseconds())
		if !success {
			t.failures.Add(1)
		}
	}()
	payload, err := io.ReadAll(io.LimitReader(req.Body, 1024*1024+1))
	req.Body.Close()
	if err != nil || len(payload) > 1024*1024 {
		return nil, ErrUnavailable
	}

	var startOffset int
	if t.healthy.Load() {
		activeIdx := int(t.active.Load()) % len(t.targets)
		endpoint := t.targets[activeIdx]
		if req.Context().Err() != nil {
			return nil, req.Context().Err()
		}
		resp, cancel, sendErr := t.sendAttempt(req, endpoint, payload)
		if sendErr == nil {
			success = true
			resp.Body = &cancelBody{ReadCloser: resp.Body, cancel: cancel}
			return resp, nil
		}
		if req.Context().Err() != nil {
			return nil, req.Context().Err()
		}
		t.healthy.Store(false)
		startOffset = 1
	}

	start := int(t.active.Load())
	for offset := 0; offset < len(t.targets); offset++ {
		index := (start + startOffset + offset) % len(t.targets)
		endpoint := t.targets[index]
		if req.Context().Err() != nil {
			return nil, req.Context().Err()
		}
		if !t.probeEndpoint(req, endpoint) {
			continue
		}
		resp, cancel, sendErr := t.sendAttempt(req, endpoint, payload)
		if sendErr != nil {
			continue
		}
		success = true
		if index != start {
			if t.active.CompareAndSwap(uint32(start), uint32(index)) {
				t.failovers.Add(1)
			}
		} else {
			t.active.Store(uint32(index))
		}
		t.healthy.Store(true)
		resp.Body = &cancelBody{ReadCloser: resp.Body, cancel: cancel}
		return resp, nil
	}
	return nil, ErrUnavailable
}

type cancelBody struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b *cancelBody) Close() error { err := b.ReadCloser.Close(); b.cancel(); return err }
