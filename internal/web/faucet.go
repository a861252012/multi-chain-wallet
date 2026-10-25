package web

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/wallet"
	"github.com/google/uuid"
)

func NewFaucet(faucet *wallet.TestFaucet, solana *wallet.SolanaService, csrf string) http.Handler {
	mux := http.NewServeMux()
	var solMu sync.Mutex
	var lastSOL time.Time
	mux.HandleFunc("POST /api/faucet/tron", localWalletFilter(csrf, func(w http.ResponseWriter, r *http.Request) {
		var req struct{}
		if err := decodeStrictJSON(w, r, &req); err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 40*time.Second)
		defer cancel()
		result, err := faucet.ClaimTRX(ctx)
		if err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		respondWallet(w, 200, newTronRecordResponse(result), nil)
	}))
	mux.HandleFunc("GET /api/faucet/tron/{hash}", localWalletFilter(csrf, func(w http.ResponseWriter, r *http.Request) {
		if faucet.TronSource == nil {
			http.NotFound(w, r)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		history, err := faucet.TronSource.History(ctx)
		if err != nil {
			respondWallet(w, 502, nil, err)
			return
		}
		for _, record := range history {
			if record.Signature == r.PathValue("hash") {
				respondWallet(w, 200, newTronRecordResponse(&record), nil)
				return
			}
		}
		http.NotFound(w, r)
	}))
	mux.HandleFunc("GET /api/faucet", localWalletFilter(csrf, func(w http.ResponseWriter, r *http.Request) {
		respondWallet(w, 200, map[string]any{"enabled": len(faucet.Sources) > 0, "tronEnabled": faucet.TronSource != nil && faucet.TronRecipient != nil && faucet.TronPassword != "", "csrfToken": csrf}, nil)
	}))
	mux.HandleFunc("POST /api/faucet", localWalletFilter(csrf, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ChainID int64  `json:"chainId"`
			Asset   string `json:"asset"`
			Address string `json:"address"`
		}
		if err := decodeStrictJSON(w, r, &req); err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()
		result, err := faucet.Claim(ctx, req.ChainID, req.Asset, req.Address)
		if err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		respondWallet(w, 200, newEVMSendResponse(result), nil)
	}))
	mux.HandleFunc("POST /api/faucet/solana", localWalletFilter(csrf, func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.NewString()
		logger := slog.Default().With("component", "solana_faucet", "request_id", requestID)
		r = r.WithContext(wallet.WithFaucetLogger(r.Context(), logger))
		w.Header().Set("X-Request-ID", requestID)
		started := time.Now()
		responseStatus := http.StatusInternalServerError
		logger.InfoContext(r.Context(), "solana_faucet_received")
		defer func() {
			logger.InfoContext(r.Context(), "solana_faucet_response", "http_status", responseStatus, "duration_ms", time.Since(started).Milliseconds())
		}()

		var req struct{}
		if err := decodeStrictJSON(w, r, &req); err != nil {
			responseStatus = 400
			respondWallet(w, 400, nil, err)
			return
		}
		solMu.Lock()
		if time.Since(lastSOL) < time.Minute {
			solMu.Unlock()
			responseStatus = 429
			respondWallet(w, 429, nil, errors.New("請至少等待一分鐘再申請 Devnet 空投"))
			return
		}
		lastSOL = time.Now()
		solMu.Unlock()
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		result, err := solana.RequestTestSOL(ctx)
		if err != nil {
			responseStatus = 502
			respondWallet(w, 502, nil, err)
			return
		}
		responseStatus = 200
		respondWallet(w, 200, newSolanaAirdropResponse(result), nil)
	}))
	return secureHeaders(mux)
}
