package web

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/a861252012/multi-chain-wallet/internal/wallet"
)

// Observation never creates a wallet, journal, scan cursor, quote, or signing session.
func registerObservation(mux *http.ServeMux, client *chain.Client) {
	mux.HandleFunc("GET /api/diagnostics", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		diagnostics := client.Diagnostics()
		network, err := client.Network(ctx)
		respond(w, newEVMDiagnosticsResponse(diagnostics, network, err), nil)
	})
	mux.HandleFunc("GET /api/watch/token", func(w http.ResponseWriter, r *http.Request) {
		address, err := wallet.ValidateAddress(r.URL.Query().Get("address"))
		if err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		contract, err := wallet.ValidateAddress(r.URL.Query().Get("contract"))
		if err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		symbol, decimals, err := wallet.QueryERC20Metadata(ctx, client, contract)
		if err != nil {
			respond(w, nil, err)
			return
		}
		balance, err := wallet.QueryERC20BalanceOf(ctx, client, contract, address)
		if err != nil {
			respond(w, nil, err)
			return
		}
		respond(w, &evmToken{Contract: contract.Hex(), Symbol: symbol, Decimals: decimals, Balance: wallet.FormatUnits(balance, decimals), BalanceRaw: balance.String()}, nil)
	})
	mux.HandleFunc("GET /api/watch/activity", func(w http.ResponseWriter, r *http.Request) {
		address, err := wallet.ValidateAddress(r.URL.Query().Get("address"))
		if err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 40*time.Second)
		defer cancel()
		if hash := r.URL.Query().Get("hash"); hash != "" {
			result, err := client.Activity(ctx, hash, address)
			respond(w, newEVMActivity(result), err)
			return
		}
		number, err := client.FinalizedNumber(ctx)
		if err != nil {
			respond(w, nil, err)
			return
		}
		if raw := r.URL.Query().Get("block"); raw != "" {
			number, err = strconv.ParseUint(raw, 10, 64)
			if err != nil {
				respondWallet(w, 400, nil, err)
				return
			}
		}
		block, err := client.ScanFinalizedBlock(ctx, number, address)
		if err != nil {
			respond(w, nil, err)
			return
		}
		respond(w, &evmWatchBlockResponse{
			Block: strconv.FormatUint(number, 10), Hashes: block.Hashes, Tokens: block.Tokens,
			Coverage: "指定的單一 finalized 區塊；點選交易再核對收據",
		}, nil)
	})
}

type evmWatchBlockResponse struct {
	Block    string   `json:"block"`
	Hashes   []string `json:"hashes"`
	Tokens   []string `json:"tokens"`
	Coverage string   `json:"coverage"`
}
