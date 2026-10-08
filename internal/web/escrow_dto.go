package web

import "github.com/a861252012/multi-chain-wallet/internal/wallet"

type evmEscrowInfo struct {
	Enabled   bool   `json:"enabled"`
	Contract  string `json:"contract"`
	Token     string `json:"token"`
	Balance   string `json:"balance"`
	Allowance string `json:"allowance"`
}

func newEVMEscrowInfo(info *wallet.EscrowInfo) *evmEscrowInfo {
	return &evmEscrowInfo{Enabled: info.Enabled, Contract: info.Contract, Token: info.Token, Balance: info.Balance, Allowance: info.Allowance}
}

type evmEscrowOrder struct {
	OrderID   string `json:"orderId"`
	Buyer     string `json:"buyer"`
	Seller    string `json:"seller"`
	Amount    string `json:"amount"`
	AmountRaw string `json:"amountRaw"`
	State     string `json:"state"`
	Block     string `json:"block"`
	Finalized bool   `json:"finalized"`
}

func newEVMEscrowOrder(order *wallet.EscrowOrder) *evmEscrowOrder {
	return &evmEscrowOrder{OrderID: order.OrderID, Buyer: order.Buyer, Seller: order.Seller, Amount: order.Amount, AmountRaw: order.AmountRaw, State: order.State, Block: order.Block, Finalized: order.Finalized}
}

type evmEscrowPreview struct {
	Action  string `json:"action"`
	OrderID string `json:"orderId"`
	Buyer   string `json:"buyer"`
	Seller  string `json:"seller"`
	Token   string `json:"token"`
}

func newEVMEscrowPreview(preview *wallet.EscrowPreview) *evmEscrowPreview {
	if preview == nil {
		return nil
	}
	return &evmEscrowPreview{Action: string(preview.Action), OrderID: preview.OrderID, Buyer: preview.Buyer, Seller: preview.Seller, Token: preview.Token}
}
