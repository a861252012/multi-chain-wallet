package web

import (
	"context"
	"net/http"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/wallet"
)

func registerEscrowRoutes(mux *http.ServeMux, service *wallet.Service, filter func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("GET /api/wallet/escrow", filter(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		info, err := service.EscrowStatus(ctx)
		if err != nil {
			respondWallet(w, http.StatusBadGateway, nil, err)
			return
		}
		respondWallet(w, http.StatusOK, newEVMEscrowInfo(info), nil)
	}))
	mux.HandleFunc("GET /api/wallet/escrow/order", filter(func(w http.ResponseWriter, r *http.Request) {
		buyer, err := wallet.ParseEVMAddress(r.URL.Query().Get("buyer"))
		if err != nil {
			respondWallet(w, http.StatusBadRequest, nil, err)
			return
		}
		reference, err := wallet.ParseOrderReference(r.URL.Query().Get("orderId"))
		if err != nil {
			respondWallet(w, http.StatusBadRequest, nil, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		order, err := service.EscrowOrder(ctx, buyer, reference)
		if err != nil {
			respondWallet(w, http.StatusBadGateway, nil, err)
			return
		}
		respondWallet(w, http.StatusOK, newEVMEscrowOrder(order), nil)
	}))
}
