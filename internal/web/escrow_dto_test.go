package web

import (
	"encoding/json"
	"testing"

	"github.com/a861252012/multi-chain-wallet/internal/wallet"
)

func TestEVMEscrowReplacementDTO(t *testing.T) {
	preview := &wallet.EscrowPreview{Action: wallet.ActionEscrowRefund, OrderID: "order-001", Buyer: "buyer", Seller: "seller", Token: "token"}
	assertJSONLiteral(t, `{"action":"escrow_refund","orderId":"order-001","buyer":"buyer","seller":"seller","token":"token"}`, newEVMEscrowPreview(preview))
	quote := newEVMQuoteResponse(&wallet.QuoteResponse{Action: "speedup", Escrow: preview})
	if quote.Action != "speedup" || quote.Escrow == nil || quote.Escrow.Action != "escrow_refund" {
		t.Fatal("quote lost replacement or original action", quote)
	}
	for _, action := range []string{"escrow_fund", "escrow_release", "escrow_refund", ""} {
		outer := "speedup"
		if action == "" {
			outer = "cancel"
		}
		send := &wallet.SendResponse{Action: outer, EscrowAction: action}
		history := &wallet.HistoryResponse{Transactions: []wallet.HistoryItem{{Action: outer, EscrowAction: action}}}
		assertSameJSON(t, send, newEVMSendResponse(send))
		assertSameJSON(t, history, newEVMHistoryResponse(history))
		for _, response := range []any{newEVMSendResponse(send), newEVMHistoryResponse(history).Transactions[0]} {
			data, err := json.Marshal(response)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]any
			if err := json.Unmarshal(data, &fields); err != nil {
				t.Fatal(err)
			}
			value, present := fields["escrowAction"]
			if (action == "" && present) || (action != "" && value != action) {
				t.Fatal("optional escrow action changed on the wire", fields)
			}
		}
	}
}
