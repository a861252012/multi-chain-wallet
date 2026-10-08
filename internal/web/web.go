package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/a861252012/multi-chain-wallet/internal/wallet"
)

//go:embed templates/*.html static/*
var assets embed.FS

func New(client *chain.Client, walletService ...*wallet.Service) (http.Handler, error) {
	page, err := template.ParseFS(assets, "templates/index.html")
	if err != nil {
		return nil, err
	}
	static, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	registerObservation(mux, client)
	showcase, err := template.ParseFS(assets, "templates/showcase.html")
	if err != nil {
		return nil, err
	}
	mux.HandleFunc("GET /showcase", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = showcase.Execute(w, nil)
	})
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = page.Execute(w, map[string]any{"Native": client.NativeSymbol(), "Shared": r.Context().Value(sharedDemoKey{}) == true})
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		respond(w, map[string]string{"status": "ok", "mode": "wallet"}, nil)
	})
	mux.HandleFunc("GET /api/network", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		result, err := client.Network(ctx)
		respond(w, newEVMNetworkResponse(result), err)
	})
	mux.HandleFunc("GET /api/balance", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		result, err := client.Balance(ctx, strings.TrimSpace(r.URL.Query().Get("address")))
		respond(w, newEVMBalanceResponse(result), err)
	})
	mux.HandleFunc("GET /api/transactions/{hash}", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		result, err := client.Transaction(ctx, r.PathValue("hash"))
		respond(w, newEVMTransactionResponse(result), err)
	})
	if len(walletService) > 0 && walletService[0] != nil {
		registerWalletRoutes(mux, walletService[0])
	}
	return secureHeaders(mux), nil
}

func secureHeaders(mux http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; connect-src 'self'; img-src 'self' data:; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		mux.ServeHTTP(w, r)
	})
}

func respond(w http.ResponseWriter, result any, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		status := http.StatusBadGateway
		switch {
		case errors.Is(err, chain.ErrAddress), errors.Is(err, chain.ErrHash):
			status = http.StatusBadRequest
		case errors.Is(err, chain.ErrNotFound):
			status = http.StatusNotFound
		case errors.Is(err, chain.ErrTimeout):
			status = http.StatusGatewayTimeout
		}
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(result)
}

// NormalizeWorkspaceRedirect checks if a workspace request path is missing a required trailing slash
// and returns the canonical redirect path if needed.
func NormalizeWorkspaceRedirect(path string) (string, bool) {
	if path == "/solana" || path == "/tron" {
		return path + "/", true
	}
	if after, ok := strings.CutPrefix(path, "/accounts/"); ok {
		account, rest, hasSlash := strings.Cut(after, "/")
		if !hasSlash && account != "" {
			return path + "/", true
		}
		if slug, ok := strings.CutPrefix(rest, "net/"); ok && slug != "" && !strings.Contains(slug, "/") {
			return path + "/", true
		}
	}
	return "", false
}
