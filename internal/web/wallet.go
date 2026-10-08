package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/a861252012/multi-chain-wallet/internal/wallet"
)

func isValidHost(host string) bool {
	h, _, err := net.SplitHostPort(host)
	if err != nil {
		h = host
	}
	h = strings.Trim(h, "[]")
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}

func decodeStrictJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	ct, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || ct != "application/json" {
		return errors.New("Content-Type 必須是 application/json")
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return errors.New("請求資料格式錯誤或含有未定義欄位")
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("請求結尾含有多餘資料")
	}
	return nil
}

func registerWalletRoutes(mux *http.ServeMux, ws *wallet.Service) {
	if ws == nil {
		return
	}

	walletFilter := func(handler http.HandlerFunc) http.HandlerFunc { return localWalletFilter(ws.CSRFToken(), handler) }
	registerEscrowRoutes(mux, ws, walletFilter)

	mux.HandleFunc("POST /api/wallet/exchange/pools", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		var req evmQuoteRequest
		if err := decodeStrictJSON(w, r, &req); err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		command, err := req.poolComparisonCommand()
		if err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		result, err := ws.ComparePoolsCommand(ctx, command)
		if err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		respondWallet(w, 200, newEVMPoolComparison(result), nil)
	}))
	// GET /api/wallet and GET /api/wallet/status
	statusHandler := walletFilter(func(w http.ResponseWriter, r *http.Request) {
		info, err := ws.Status()
		if err != nil {
			respondWallet(w, http.StatusInternalServerError, nil, err)
			return
		}
		respondWallet(w, http.StatusOK, newEVMWalletInfo(info), nil)
	})
	mux.HandleFunc("GET /api/wallet", statusHandler)
	mux.HandleFunc("GET /api/wallet/status", statusHandler)

	mux.HandleFunc("GET /api/wallet/vault", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		vaultInfo, err := ws.VaultStatus(ctx)
		if err != nil {
			status := http.StatusInternalServerError
			switch {
			case errors.Is(err, chain.ErrUnavailable):
				status = http.StatusBadGateway
			case errors.Is(err, chain.ErrTimeout):
				status = http.StatusGatewayTimeout
			}
			respondWallet(w, status, nil, err)
			return
		}
		respondWallet(w, http.StatusOK, newEVMVaultInfo(vaultInfo), nil)
	}))

	mux.HandleFunc("POST /api/wallet/import-keystore", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Keystore    json.RawMessage `json:"keystore"`
			Password    string          `json:"password"`
			NewPassword string          `json:"newPassword"`
		}
		if err := decodeStrictJSON(w, r, &req); err != nil {
			respondWallet(w, http.StatusBadRequest, nil, err)
			return
		}
		res, err := ws.ImportKeystore(req.Keystore, req.Password, req.NewPassword)
		if err != nil {
			respondWallet(w, http.StatusBadRequest, nil, err)
			return
		}
		respondWallet(w, http.StatusOK, newEVMImportResponse(res), nil)
	}))
	mux.HandleFunc("POST /api/wallet/password", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Password    string `json:"password"`
			NewPassword string `json:"newPassword"`
		}
		if err := decodeStrictJSON(w, r, &req); err != nil {
			respondWallet(w, http.StatusBadRequest, nil, err)
			return
		}
		if err := ws.ChangePassword(req.Password, req.NewPassword); err != nil {
			respondWallet(w, http.StatusBadRequest, nil, err)
			return
		}
		respondWallet(w, http.StatusOK, &walletMutationResponse{Changed: true}, nil)
	}))

	mux.HandleFunc("GET /api/wallet/accounts", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		result, err := ws.Accounts()
		if err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		respondWallet(w, 200, newEVMAccounts(result), nil)
	}))
	mux.HandleFunc("POST /api/wallet/accounts", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name     string `json:"name"`
			Password string `json:"password"`
		}
		if err := decodeStrictJSON(w, r, &req); err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		if r.Context().Value(sharedDemoKey{}) == true {
			if err := wallet.ValidatePassword(req.Password); err != nil {
				respondWallet(w, 400, nil, err)
				return
			}
		}
		if !allowSharedPasswordAttempt(w, r, req.Password) {
			return
		}
		result, err := ws.AddAccount(req.Name, req.Password)
		if err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		respondWallet(w, 200, newEVMAccount(result), nil)
	}))
	mux.HandleFunc("POST /api/wallet/accounts/update", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Archived bool   `json:"archived"`
		}
		if err := decodeStrictJSON(w, r, &req); err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		result, err := ws.UpdateAccount(req.ID, req.Name, req.Archived)
		if err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		respondWallet(w, 200, newEVMAccount(result), nil)
	}))
	// POST /api/wallet/create
	mux.HandleFunc("POST /api/wallet/create", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Password string `json:"password"`
		}
		if err := decodeStrictJSON(w, r, &req); err != nil {
			respondWallet(w, http.StatusBadRequest, nil, err)
			return
		}
		res, err := ws.Create(req.Password)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, wallet.ErrWalletExists) {
				status = http.StatusConflict
			} else if errors.Is(err, wallet.ErrTooManyScryptRequests) {
				status = http.StatusServiceUnavailable
			}
			respondWallet(w, status, nil, err)
			return
		}
		respondWallet(w, http.StatusOK, newEVMCreateResponse(res), nil)
	}))

	// POST /api/wallet/import
	mux.HandleFunc("POST /api/wallet/import", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Password string `json:"password"`
			Mnemonic string `json:"mnemonic"`
		}
		if err := decodeStrictJSON(w, r, &req); err != nil {
			respondWallet(w, http.StatusBadRequest, nil, err)
			return
		}
		res, err := ws.Import(req.Mnemonic, req.Password)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, wallet.ErrWalletExists) {
				status = http.StatusConflict
			} else if errors.Is(err, wallet.ErrTooManyScryptRequests) {
				status = http.StatusServiceUnavailable
			}
			respondWallet(w, status, nil, err)
			return
		}
		respondWallet(w, http.StatusOK, newEVMImportResponse(res), nil)
	}))

	// POST /api/wallet/backup
	mux.HandleFunc("POST /api/wallet/backup", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Password string `json:"password"`
		}
		if err := decodeStrictJSON(w, r, &req); err != nil {
			respondWallet(w, http.StatusBadRequest, nil, err)
			return
		}
		if !allowSharedPasswordAttempt(w, r, req.Password) {
			return
		}
		res, err := ws.Backup(req.Password)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, wallet.ErrPasswordMismatch) {
				status = http.StatusUnauthorized
			} else if errors.Is(err, wallet.ErrWalletNotFound) {
				status = http.StatusNotFound
			} else if errors.Is(err, wallet.ErrTooManyScryptRequests) {
				status = http.StatusServiceUnavailable
			}
			respondWallet(w, status, nil, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(res)
	}))

	handleTokenQuery := func(w http.ResponseWriter, r *http.Request, contract, spender string) {
		contract = strings.TrimSpace(contract)
		spender = strings.TrimSpace(spender)
		if contract == "" {
			respondWallet(w, http.StatusBadRequest, nil, errors.New("缺少 contract 參數"))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		info, err := ws.Token(ctx, contract, spender)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, wallet.ErrWalletNotFound) {
				status = http.StatusNotFound
			} else if errors.Is(err, chain.ErrUnavailable) {
				status = http.StatusBadGateway
			} else if errors.Is(err, chain.ErrTimeout) {
				status = http.StatusGatewayTimeout
			}
			respondWallet(w, status, nil, err)
			return
		}
		respondWallet(w, http.StatusOK, newEVMToken(info), nil)
	}

	// GET /api/wallet/token
	mux.HandleFunc("GET /api/wallet/token", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		contract := r.URL.Query().Get("contract")
		spender := r.URL.Query().Get("spender")
		handleTokenQuery(w, r, contract, spender)
	}))

	// POST /api/wallet/token
	mux.HandleFunc("POST /api/wallet/token", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Contract string `json:"contract"`
			Spender  string `json:"spender"`
		}
		if err := decodeStrictJSON(w, r, &req); err != nil {
			respondWallet(w, http.StatusBadRequest, nil, err)
			return
		}
		handleTokenQuery(w, r, req.Contract, req.Spender)
	}))

	// POST /api/wallet/quote
	mux.HandleFunc("POST /api/wallet/quote", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		var req evmQuoteRequest
		if err := decodeStrictJSON(w, r, &req); err != nil {
			respondWallet(w, http.StatusBadRequest, nil, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 40*time.Second)
		defer cancel()
		command, err := req.walletCommand()
		if err != nil {
			respondWallet(w, http.StatusBadRequest, nil, err)
			return
		}
		quote, err := ws.QuoteCommand(ctx, command)
		if err != nil {
			status := http.StatusBadRequest
			switch {
			case errors.Is(err, wallet.ErrTxInFlight), errors.Is(err, wallet.ErrApprovalRace):
				status = http.StatusConflict
			case errors.Is(err, wallet.ErrQuoteStorageFull):
				status = http.StatusServiceUnavailable
			case errors.Is(err, chain.ErrUnavailable):
				status = http.StatusBadGateway
			case errors.Is(err, chain.ErrTimeout):
				status = http.StatusGatewayTimeout
			}
			respondWallet(w, status, nil, err)
			return
		}
		respondWallet(w, http.StatusOK, newEVMQuoteResponse(quote), nil)
	}))

	// POST /api/wallet/send
	mux.HandleFunc("POST /api/wallet/send", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			QuoteID  string `json:"quoteId"`
			Password string `json:"password"`
		}
		if err := decodeStrictJSON(w, r, &req); err != nil {
			respondWallet(w, http.StatusBadRequest, nil, err)
			return
		}
		if r.Context().Value(sharedDemoKey{}) == true && req.QuoteID == "" {
			respondWallet(w, 400, nil, errors.New("請提供交易報價識別碼"))
			return
		}
		if !allowSharedPasswordAttempt(w, r, req.Password) {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 40*time.Second)
		defer cancel()
		res, err := ws.Send(ctx, req.QuoteID, req.Password)
		if err != nil {
			status := http.StatusBadRequest
			switch {
			case errors.Is(err, wallet.ErrPasswordMismatch):
				status = http.StatusUnauthorized
			case errors.Is(err, wallet.ErrTxInFlight), errors.Is(err, wallet.ErrNonceMismatch):
				status = http.StatusConflict
			case errors.Is(err, wallet.ErrJournalFull), errors.Is(err, wallet.ErrTooManyScryptRequests):
				status = http.StatusServiceUnavailable
			case errors.Is(err, chain.ErrUnavailable):
				status = http.StatusBadGateway
			case errors.Is(err, chain.ErrTimeout):
				status = http.StatusGatewayTimeout
			}
			respondWallet(w, status, nil, err)
			return
		}
		respondWallet(w, http.StatusOK, newEVMSendResponse(res), nil)
	}))

	// POST /api/wallet/retry
	mux.HandleFunc("POST /api/wallet/retry", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Hash string `json:"hash"`
		}
		if err := decodeStrictJSON(w, r, &req); err != nil {
			respondWallet(w, http.StatusBadRequest, nil, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		res, err := ws.Retry(ctx, req.Hash)
		if err != nil {
			status := http.StatusBadRequest
			switch {
			case errors.Is(err, chain.ErrNotFound):
				status = http.StatusNotFound
			case errors.Is(err, chain.ErrUnavailable):
				status = http.StatusBadGateway
			case errors.Is(err, chain.ErrTimeout):
				status = http.StatusGatewayTimeout
			}
			respondWallet(w, status, nil, err)
			return
		}
		respondWallet(w, http.StatusOK, newEVMSendResponse(res), nil)
	}))

	mux.HandleFunc("POST /api/wallet/activity/import", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Hash string `json:"hash"`
		}
		if err := decodeStrictJSON(w, r, &req); err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		result, err := ws.ImportActivity(ctx, req.Hash)
		if err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		respondWallet(w, 200, newEVMActivity(result), nil)
	}))
	mux.HandleFunc("POST /api/wallet/activity/sync", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			From      uint64   `json:"from"`
			Contracts []string `json:"contracts,omitempty"`
		}
		if err := decodeStrictJSON(w, r, &req); err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()
		result, err := ws.SyncActivity(ctx, req.From, req.Contracts)
		if err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		respondWallet(w, 200, newEVMSyncResponse(result), nil)
	}))
	mux.HandleFunc("GET /api/wallet/activity", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		page := 1
		if raw := r.URL.Query().Get("page"); raw != "" {
			var err error
			page, err = strconv.Atoi(raw)
			if err != nil || page < 1 {
				respondWallet(w, 400, nil, errors.New("頁碼格式錯誤"))
				return
			}
		}
		ctx, cancel := context.WithTimeout(r.Context(), 40*time.Second)
		defer cancel()
		result, err := ws.Activity(ctx, page)
		if err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		if r.URL.Query().Get("format") == "csv" {
			slug := chain.NetworkSlug(ws.ChainID())
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			w.Header().Set("Content-Disposition", "attachment; filename=multi-chain-wallet-"+slug+"-activity.csv")
			_ = writeEVMActivityCSV(w, newEVMActivityResponse(result))
			return
		}
		respondWallet(w, 200, newEVMActivityResponse(result), nil)
	}))

	mux.HandleFunc("GET /api/wallet/scan", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		state, err := ws.ScanProgress()
		if err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		respondWallet(w, 200, newEVMScanProgress(state), nil)
	}))
	mux.HandleFunc("POST /api/wallet/scan", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Enabled bool    `json:"enabled"`
			Start   *uint64 `json:"start"`
		}
		if err := decodeStrictJSON(w, r, &req); err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		state, err := ws.ConfigureScan(req.Enabled, req.Start)
		if err != nil {
			respondWallet(w, 400, nil, err)
			return
		}
		respondWallet(w, 200, newEVMScanProgress(state), nil)
	}))
	// GET /api/wallet/history
	mux.HandleFunc("GET /api/wallet/history", walletFilter(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		history, err := ws.History(ctx)
		if err != nil {
			respondWallet(w, http.StatusInternalServerError, nil, err)
			return
		}
		respondWallet(w, http.StatusOK, newEVMHistoryResponse(history), nil)
	}))
}

func respondWallet(w http.ResponseWriter, status int, result any, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		body := make(map[string]string)
		if _, ok := errors.AsType[*wallet.SendRejectedError](err); ok {
			body["code"] = "send_rejected"
		}
		if _, ok := errors.AsType[*os.PathError](err); ok {
			err = errors.New("本機錢包儲存失敗，請檢查資料磁碟與權限")
			status = http.StatusInternalServerError
		}
		body["error"] = err.Error()
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
		return
	}
	w.WriteHeader(status)
	if result != nil {
		_ = json.NewEncoder(w).Encode(result)
	}
}

func localWalletFilter(csrfToken string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !allowedRequestHost(r) {
			respondWallet(w, http.StatusBadRequest, nil, errors.New("無效的 Host 標頭，僅允許本機存取"))
			return
		}

		// Reject cross-origin
		if sfs := r.Header.Get("Sec-Fetch-Site"); sfs == "cross-site" {
			respondWallet(w, http.StatusForbidden, nil, errors.New("跨來源請求已被拒絕"))
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			if origin != requestOrigin(r) {
				respondWallet(w, http.StatusForbidden, nil, errors.New("跨來源請求已被拒絕"))
				return
			}
		}

		// CSRF protection for POST requests
		if r.Method == http.MethodPost {
			csrf := r.Header.Get("X-Wallet-CSRF")
			if csrf == "" || csrf != csrfToken {
				respondWallet(w, http.StatusForbidden, nil, errors.New("缺少或無效的 CSRF Token (X-Wallet-CSRF)"))
				return
			}
		}

		handler(w, r)
	}
}
