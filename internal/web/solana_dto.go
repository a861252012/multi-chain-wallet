package web

import (
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/wallet"
)

type solanaStatusResponse struct {
	Exists    bool   `json:"exists"`
	Network   string `json:"network"`
	Address   string `json:"address,omitempty"`
	CSRFToken string `json:"csrfToken"`
}

func newSolanaStatusResponse(status wallet.SolanaStatus, csrf string) *solanaStatusResponse {
	return &solanaStatusResponse{
		Exists: status.Exists, Network: status.Network, Address: status.Address, CSRFToken: csrf,
	}
}

type solanaBalanceResponse struct {
	Address  string `json:"address"`
	SOL      string `json:"sol"`
	Lamports string `json:"lamports"`
	Slot     uint64 `json:"slot"`
}

func newSolanaBalanceResponse(balance *wallet.SolanaBalance) *solanaBalanceResponse {
	if balance == nil {
		return nil
	}
	return &solanaBalanceResponse{
		Address: balance.Address, SOL: balance.SOL, Lamports: balance.Lamports, Slot: balance.Slot,
	}
}

type solanaCreateResponse struct {
	Address  string `json:"address"`
	Mnemonic string `json:"mnemonic,omitempty"`
}

func newSolanaCreateResponse(result *wallet.SolanaCreateResponse) *solanaCreateResponse {
	if result == nil {
		return nil
	}
	return &solanaCreateResponse{Address: result.Address, Mnemonic: result.Mnemonic}
}

type solanaAirdropResponse struct {
	Signature string `json:"signature"`
	State     string `json:"state"`
	Amount    string `json:"amount"`
}

func newSolanaAirdropResponse(result *wallet.SolanaAirdropResponse) *solanaAirdropResponse {
	if result == nil {
		return nil
	}
	return &solanaAirdropResponse{Signature: result.Signature, State: result.State, Amount: result.Amount}
}

type solanaQuoteResponse struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Amount    string    `json:"amount"`
	FeeSOL    string    `json:"feeSol"`
	Blockhash string    `json:"blockhash"`
	LastValid uint64    `json:"lastValidBlockHeight"`
	Expires   time.Time `json:"expiresAt"`
}

func newSolanaQuoteResponse(quote *wallet.SolanaQuote) *solanaQuoteResponse {
	if quote == nil {
		return nil
	}
	return &solanaQuoteResponse{
		ID: quote.ID, From: quote.From, To: quote.To, Amount: quote.Amount,
		FeeSOL: quote.FeeSOL, Blockhash: quote.Blockhash.String(), LastValid: quote.LastValid,
		Expires: quote.Expires,
	}
}

type solanaRecordResponse struct {
	Signature string    `json:"signature"`
	QuoteID   string    `json:"quoteId"`
	To        string    `json:"to"`
	Amount    string    `json:"amount"`
	State     string    `json:"state"`
	Finalized bool      `json:"finalized"`
	LastValid uint64    `json:"lastValidBlockHeight"`
	CreatedAt time.Time `json:"createdAt"`
}

func newSolanaRecordResponse(record *wallet.SolanaRecord) *solanaRecordResponse {
	if record == nil {
		return nil
	}
	return &solanaRecordResponse{
		Signature: record.Signature, QuoteID: record.QuoteID, To: record.To, Amount: record.Amount,
		State: record.State, Finalized: record.Finalized, LastValid: record.LastValid,
		CreatedAt: record.CreatedAt,
	}
}

func newSolanaRecordResponses(records []wallet.SolanaRecord) []solanaRecordResponse {
	if records == nil {
		return nil
	}
	result := make([]solanaRecordResponse, len(records))
	for i := range records {
		result[i] = *newSolanaRecordResponse(&records[i])
	}
	return result
}
