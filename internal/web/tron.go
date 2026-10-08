package web

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/wallet"
)

func NewTron(service *wallet.TronService, csrf string) (http.Handler, error) {
	page, err := template.ParseFS(assets, "templates/tron.html")
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = page.Execute(w, map[string]any{"Shared": r.Context().Value(sharedDemoKey{}) == true})
	})
	mux.HandleFunc("GET /api/status", localWalletFilter(csrf, func(w http.ResponseWriter, r *http.Request) {
		result, err := service.Status()
		respond(w, newTronStatusResponse(result, csrf), err)
	}))
	mux.HandleFunc("GET /api/balance", localWalletFilter(csrf, func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		result, err := service.Balance(ctx, r.URL.Query().Get("address"))
		respond(w, newTronBalanceResponse(result), err)
	}))
	mux.HandleFunc("GET /api/history", localWalletFilter(csrf, func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		result, err := service.History(ctx)
		respond(w, newTronRecordResponses(result), err)
	}))
	mux.HandleFunc("GET /api/token", localWalletFilter(csrf, func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		result, err := service.Token(ctx, r.URL.Query().Get("address"), r.URL.Query().Get("contract"))
		respond(w, newTronTokenResponse(result), err)
	}))
	for _, action := range []string{"create", "quote", "send", "retry"} {
		mux.HandleFunc("POST /api/"+action, localWalletFilter(csrf, func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 40*time.Second)
			defer cancel()
			switch action {
			case "create":
				var req struct {
					Mnemonic string `json:"mnemonic"`
					Password string `json:"password"`
				}
				if err := decodeStrictJSON(w, r, &req); err != nil {
					respondWallet(w, 400, nil, err)
					return
				}
				result, err := service.Create(req.Mnemonic, req.Password)
				if err != nil {
					respondWallet(w, 400, nil, err)
					return
				}
				respondWallet(w, 200, newTronCreateResponse(result), nil)
			case "quote":
				var req struct {
					To        string `json:"to"`
					Amount    string `json:"amount"`
					AmountRaw string `json:"amountRaw"`
					Contract  string `json:"contract"`
				}
				if err := decodeStrictJSON(w, r, &req); err != nil {
					respondWallet(w, 400, nil, err)
					return
				}
				result, err := service.Quote(ctx, req.To, req.Amount, req.Contract, req.AmountRaw)
				if err != nil {
					respondWallet(w, 400, nil, err)
					return
				}
				respondWallet(w, 200, newTronQuoteResponse(result), nil)
			case "send":
				var req struct {
					ID       string `json:"id"`
					Password string `json:"password"`
				}
				if err := decodeStrictJSON(w, r, &req); err != nil {
					respondWallet(w, 400, nil, err)
					return
				}
				if r.Context().Value(sharedDemoKey{}) == true && req.ID == "" {
					respondWallet(w, 400, nil, errors.New("請提供交易報價識別碼"))
					return
				}
				if !allowSharedPasswordAttempt(w, r, req.Password) {
					return
				}
				result, err := service.Send(ctx, req.ID, req.Password)
				if err != nil {
					respondWallet(w, 400, nil, err)
					return
				}
				respondWallet(w, 200, newTronRecordResponse(result), nil)
			case "retry":
				var req struct {
					Signature string `json:"signature"`
				}
				if err := decodeStrictJSON(w, r, &req); err != nil {
					respondWallet(w, 400, nil, err)
					return
				}
				result, err := service.Retry(ctx, req.Signature)
				if err != nil {
					respondWallet(w, 400, nil, err)
					return
				}
				respondWallet(w, 200, newTronRecordResponse(result), nil)
			}
		}))
	}
	registerKeyRoutes(mux, csrf, service)
	return secureHeaders(mux), nil
}
