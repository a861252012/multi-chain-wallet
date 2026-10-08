package web

import (
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/wallet"
)

type tronStatusResponse struct {
	Exists    bool   `json:"exists"`
	Network   string `json:"network"`
	Address   string `json:"address,omitempty"`
	CSRFToken string `json:"csrfToken"`
}

func newTronStatusResponse(status wallet.TronStatus, csrf string) *tronStatusResponse {
	return &tronStatusResponse{
		Exists: status.Exists, Network: status.Network, Address: status.Address, CSRFToken: csrf,
	}
}

type tronBalanceResponse struct {
	Address   string `json:"address"`
	TRX       string `json:"trx"`
	Active    bool   `json:"active"`
	Bandwidth int64  `json:"bandwidth"`
	Energy    int64  `json:"energy"`
}

func newTronBalanceResponse(balance *wallet.TronBalance) *tronBalanceResponse {
	if balance == nil {
		return nil
	}
	return &tronBalanceResponse{
		Address: balance.Address, TRX: balance.TRX, Active: balance.Active,
		Bandwidth: balance.Bandwidth, Energy: balance.Energy,
	}
}

type tronCreateResponse struct {
	Address  string `json:"address"`
	Mnemonic string `json:"mnemonic,omitempty"`
}

func newTronCreateResponse(result *wallet.TronCreateResponse) *tronCreateResponse {
	if result == nil {
		return nil
	}
	return &tronCreateResponse{Address: result.Address, Mnemonic: result.Mnemonic}
}

type tronTokenResponse struct {
	Contract string `json:"contract"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
	Balance  string `json:"balance"`
}

func newTronTokenResponse(token *wallet.TronToken) *tronTokenResponse {
	if token == nil {
		return nil
	}
	return &tronTokenResponse{Contract: token.Contract, Symbol: token.Symbol, Decimals: token.Decimals, Balance: token.Balance}
}

type tronQuoteResponse struct {
	ID          string    `json:"id"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	Contract    string    `json:"contract,omitempty"`
	Symbol      string    `json:"symbol"`
	Amount      string    `json:"amount"`
	AmountRaw   string    `json:"amountRaw,omitempty"`
	Decimals    int       `json:"decimals,omitempty"`
	FeeTRX      string    `json:"feeTrx"`
	FeeLimitTRX string    `json:"feeLimitTrx"`
	Energy      int64     `json:"energy"`
	Bandwidth   int64     `json:"bandwidth"`
	Expires     time.Time `json:"expiresAt"`
}

func newTronQuoteResponse(quote *wallet.TronQuote) *tronQuoteResponse {
	if quote == nil {
		return nil
	}
	return &tronQuoteResponse{
		ID: quote.ID, From: quote.From, To: quote.To, Contract: quote.Contract,
		Symbol: quote.Symbol, Amount: quote.Amount, AmountRaw: quote.AmountRaw,
		Decimals: quote.Decimals, FeeTRX: quote.FeeTRX, FeeLimitTRX: quote.FeeLimitTRX,
		Energy: quote.Energy, Bandwidth: quote.Bandwidth, Expires: quote.Expires,
	}
}

type tronRecordResponse struct {
	Reused             bool      `json:"reused,omitempty"`
	Signature          string    `json:"signature"`
	QuoteID            string    `json:"quoteId"`
	From               string    `json:"from"`
	To                 string    `json:"to"`
	Contract           string    `json:"contract,omitempty"`
	Symbol             string    `json:"symbol"`
	Amount             string    `json:"amount"`
	State              string    `json:"state"`
	Finalized          bool      `json:"finalized"`
	FeeTRX             string    `json:"feeTrx,omitempty"`
	ExpiryCheckedBlock string    `json:"expiryCheckedBlock,omitempty"`
	ExpiryCheckedAt    int64     `json:"expiryCheckedAt,omitempty"`
	Result             string    `json:"result,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
	Expires            time.Time `json:"expiresAt"`
}

func newTronRecordResponse(record *wallet.TronRecord) *tronRecordResponse {
	if record == nil {
		return nil
	}
	return &tronRecordResponse{
		Reused: record.Reused, Signature: record.Signature, QuoteID: record.QuoteID,
		From: record.From, To: record.To, Contract: record.Contract, Symbol: record.Symbol,
		Amount: record.Amount, State: record.State, Finalized: record.Finalized,
		FeeTRX: record.FeeTRX, ExpiryCheckedBlock: record.ExpiryCheckedBlock,
		ExpiryCheckedAt: record.ExpiryCheckedAt, Result: record.Result,
		CreatedAt: record.CreatedAt, Expires: record.Expires,
	}
}

func newTronRecordResponses(records []wallet.TronRecord) []tronRecordResponse {
	if records == nil {
		return nil
	}
	result := make([]tronRecordResponse, len(records))
	for i := range records {
		result[i] = *newTronRecordResponse(&records[i])
	}
	return result
}
