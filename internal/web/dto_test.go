package web

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/a861252012/multi-chain-wallet/internal/wallet"
)

func TestEVMActivityCSVUsesRawEvidence(t *testing.T) {
	response := &wallet.ActivityResponse{Transactions: []*chain.Activity{{
		Hash: "0x" + strings.Repeat("a", 64), State: "succeeded", Block: "10",
		Movements: []chain.Movement{{
			Kind: "receive", Asset: "ETH", Raw: "1000000000000000001",
			Counterparty: "0x" + strings.Repeat("1", 40), Evidence: "transaction.value",
		}},
	}}}
	var output bytes.Buffer
	if err := writeEVMActivityCSV(&output, newEVMActivityResponse(response)); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(&output).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[1][0] != "11155111" || rows[1][7] != "1000000000000000001" || rows[1][9] != "transaction.value" {
		t.Fatalf("lossy CSV %v", rows)
	}
}

// TestEVMActivityCSVPreservesMultipleMovementsPerTransaction 驗證同一交易內的多筆資產異動產生多列合法業務邏輯，不按交易雜湊粗暴去重。
func TestEVMActivityCSVPreservesMultipleMovementsPerTransaction(t *testing.T) {
	txHash := "0x" + strings.Repeat("b", 64)
	response := &wallet.ActivityResponse{
		ChainID: 11155111,
		Transactions: []*chain.Activity{
			{
				Hash:  txHash,
				State: "succeeded",
				Block: "100",
				Movements: []chain.Movement{
					{Kind: "send", Asset: "ETH", Raw: "1000000000000000000", Counterparty: "0x" + strings.Repeat("1", 40), Evidence: "transfer"},
					{Kind: "fee", Asset: "ETH", Raw: "21000000000000", Counterparty: "", Evidence: "gas"},
				},
			},
		},
	}
	var output bytes.Buffer
	if err := writeEVMActivityCSV(&output, newEVMActivityResponse(response)); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(&output).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	// 標題列 + 2 筆異動 = 3 列
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows (header + 2 movements), got %d: %v", len(rows), rows)
	}
	// 兩列異動皆具有相同的交易雜湊，但異動種類與金額不同
	if rows[1][1] != txHash || rows[2][1] != txHash {
		t.Fatalf("mismatched tx hash in rows: %s, %s", rows[1][1], rows[2][1])
	}
	if rows[1][5] != "send" || rows[1][7] != "1000000000000000000" {
		t.Fatalf("first row movement mismatch: %v", rows[1])
	}
	if rows[2][5] != "fee" || rows[2][7] != "21000000000000" {
		t.Fatalf("second row movement mismatch: %v", rows[2])
	}
}

func TestEVMDTOsPreserveJSONContract(t *testing.T) {
	now := time.Date(2026, time.September, 16, 1, 2, 3, 0, time.UTC)
	activity := &chain.Activity{
		Hash: "0x1234", State: "succeeded", Block: "12", BlockHash: "0xabcd",
		BlockTime: "2026-09-16T01:02:03Z", CheckedAt: now,
		Movements: []chain.Movement{{Kind: "receive", Asset: "ETH", Raw: "10", Counterparty: "0xbeef", Evidence: "value"}},
	}
	history := &wallet.HistoryResponse{
		Transactions: []wallet.HistoryItem{{
			QuoteID: "quote", Finalized: true, Hash: "0x1234", State: "succeeded",
			To: "0x1111", Amount: "1", Symbol: "ETH", Action: "eth", CreatedAt: "2026-09-16T01:02:03Z",
		}},
	}
	quote := &wallet.QuoteResponse{
		ID: "quote", Action: "swap", From: "0x1111", To: "0x2222", Contract: "0x3333",
		Symbol: "WETH", Amount: "1", AmountRaw: "1000000000000000000", Nonce: "7",
		GasLimit: "21000", MaxFeePerGas: "2", MaxPriorityFeePerGas: "1", MaxFeeETH: "0.1",
		TotalETH: "1.1", Data: "0x", Method: "exactInputSingle", ExpiresAt: "2026-09-16T01:04:03Z",
		Exchange: &wallet.ExchangePreview{
			TokenIn: "WETH", TokenOut: "USDC", SymbolOut: "USDC", ExpectedOut: "10",
			MinimumOut: "9.9", MinimumOutRaw: "9900000", Deadline: "2026-09-16T01:04:03Z",
		},
	}

	tests := []struct {
		name string
		want any
		got  any
	}{
		{
			name: "wallet status",
			want: &wallet.WalletInfo{ChainID: 11155111, Exists: true, Address: "0x1111", Path: "m/44'/60'/0'/0/0", CSRFToken: "csrf", Exchange: map[string]string{"weth": "0x2222"}},
			got:  newEVMWalletInfo(&wallet.WalletInfo{ChainID: 11155111, Exists: true, Address: "0x1111", Path: "m/44'/60'/0'/0/0", CSRFToken: "csrf", Exchange: map[string]string{"weth": "0x2222"}}),
		},
		{
			name: "token omitted allowance",
			want: &wallet.TokenInfo{Contract: "0x2222", Symbol: "USDC", Decimals: 6, Balance: "1", BalanceRaw: "1000000", Trusted: true},
			got:  newEVMToken(&wallet.TokenInfo{Contract: "0x2222", Symbol: "USDC", Decimals: 6, Balance: "1", BalanceRaw: "1000000", Trusted: true}),
		},
		{
			name: "pool comparison",
			want: &wallet.PoolComparison{Pools: []wallet.PoolQuote{{Fee: 500}, {Fee: 3000, Pool: "0x4444", Output: "10", OutputRaw: "10000000"}}, BestFee: 3000, Symbol: "USDC", CheckedAt: now},
			got:  newEVMPoolComparison(&wallet.PoolComparison{Pools: []wallet.PoolQuote{{Fee: 500}, {Fee: 3000, Pool: "0x4444", Output: "10", OutputRaw: "10000000"}}, BestFee: 3000, Symbol: "USDC", CheckedAt: now}),
		},
		{name: "activity", want: activity, got: newEVMActivity(activity)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertSameJSON(t, test.want, test.got)
		})
	}

	assertJSONLiteral(t, `{
		"id":"quote","action":"swap","from":"0x1111","to":"0x2222","contract":"0x3333",
		"symbol":"WETH","amount":"1","amountRaw":"1000000000000000000","nonce":"7",
		"gasLimit":"21000","maxFeePerGas":"2","maxPriorityFeePerGas":"1","maxFeeEth":"0.1",
		"totalEth":"1.1","data":"0x","method":"exactInputSingle","expiresAt":"2026-09-16T01:04:03Z",
		"exchange":{"tokenIn":"WETH","tokenOut":"USDC","symbolOut":"USDC","expectedOut":"10","minimumOut":"9.9","minimumOutRaw":"9900000","deadline":"2026-09-16T01:04:03Z"}
	}`, newEVMQuoteResponse(quote))
	assertJSONLiteral(t, `{
		"hash":"0x1234","state":"submitted","to":"0x2222","amount":"1","symbol":"ETH","action":"eth","createdAt":"2026-09-16T01:02:03Z"
	}`, newEVMSendResponse(&wallet.SendResponse{
		Hash: "0x1234", State: "submitted", To: "0x2222", Amount: "1",
		Symbol: "ETH", Action: "eth", CreatedAt: "2026-09-16T01:02:03Z",
	}))
	assertJSONLiteral(t, `{
		"transactions":[{"quoteId":"quote","finalized":true,"hash":"0x1234","state":"succeeded","to":"0x1111","amount":"1","symbol":"ETH","action":"eth","createdAt":"2026-09-16T01:02:03Z"}]
	}`, newEVMHistoryResponse(history))
	assertJSONLiteral(t, `{"transactions":null}`, newEVMHistoryResponse(&wallet.HistoryResponse{}))
	assertJSONLiteral(t, `{"transactions":[]}`, newEVMHistoryResponse(&wallet.HistoryResponse{Transactions: []wallet.HistoryItem{}}))
	assertJSONLiteral(t, `{
		"chainId":11155111,"transactions":[],"totals":[],"page":1,"pages":1,"totalTransactions":0,"incomplete":false
	}`, newEVMActivityResponse(&wallet.ActivityResponse{
		ChainID: 11155111, Transactions: []*chain.Activity{}, Totals: []wallet.ActivityTotal{}, Page: 1, Pages: 1,
	}))
	assertJSONLiteral(t, `{
		"enabled":false,"start":0,"next":0,"finalized":0,"tokens":[],"updatedAt":"2026-09-16T01:02:03Z"
	}`, newEVMScanProgress(&wallet.ScanProgress{Tokens: []string{}, UpdatedAt: now}))
}

func TestCrossChainDTOsPreserveJSONContract(t *testing.T) {
	now := time.Date(2026, time.September, 16, 1, 2, 3, 0, time.UTC)

	assertSameJSON(t,
		map[string]any{"exists": false, "network": "Solana Devnet", "csrfToken": "csrf"},
		newSolanaStatusResponse(wallet.SolanaStatus{Network: "Solana Devnet"}, "csrf"),
	)
	assertSameJSON(t,
		map[string]any{"address": "sol-address", "mnemonic": "words"},
		newSolanaCreateResponse(&wallet.SolanaCreateResponse{Address: "sol-address", Mnemonic: "words"}),
	)
	assertSameJSON(t,
		map[string]any{"exists": true, "network": "TRON Shasta", "address": "TAddress", "csrfToken": "csrf"},
		newTronStatusResponse(wallet.TronStatus{Exists: true, Network: "TRON Shasta", Address: "TAddress"}, "csrf"),
	)
	assertSameJSON(t,
		map[string]any{"address": "TAddress"},
		newTronCreateResponse(&wallet.TronCreateResponse{Address: "TAddress"}),
	)

	solanaQuote := &wallet.SolanaQuote{ID: "quote", From: "from", To: "to", Amount: "1", FeeSOL: "0.000005", LastValid: 99, Expires: now}
	assertJSONLiteral(t, `{
		"id":"quote","from":"from","to":"to","amount":"1","feeSol":"0.000005",
		"blockhash":"11111111111111111111111111111111","lastValidBlockHeight":99,"expiresAt":"2026-09-16T01:02:03Z"
	}`, newSolanaQuoteResponse(solanaQuote))
	solanaRecord := &wallet.SolanaRecord{Signature: "sig", QuoteID: "quote", To: "to", Amount: "1", State: "submitted", LastValid: 99, CreatedAt: now}
	assertJSONLiteral(t, `{
		"signature":"sig","quoteId":"quote","to":"to","amount":"1","state":"submitted",
		"finalized":false,"lastValidBlockHeight":99,"createdAt":"2026-09-16T01:02:03Z"
	}`, newSolanaRecordResponse(solanaRecord))
	assertJSONLiteral(t, `null`, newSolanaRecordResponses(nil))
	assertJSONLiteral(t, `[]`, newSolanaRecordResponses([]wallet.SolanaRecord{}))

	tronQuote := &wallet.TronQuote{ID: "quote", From: "from", To: "to", Symbol: "TRX", Amount: "1", FeeTRX: "0.1", FeeLimitTRX: "1", Expires: now}
	assertJSONLiteral(t, `{
		"id":"quote","from":"from","to":"to","symbol":"TRX","amount":"1","feeTrx":"0.1",
		"feeLimitTrx":"1","energy":0,"bandwidth":0,"expiresAt":"2026-09-16T01:02:03Z"
	}`, newTronQuoteResponse(tronQuote))
	tronRecord := &wallet.TronRecord{Signature: "sig", QuoteID: "quote", From: "from", To: "to", Symbol: "TRX", Amount: "1", State: "submitted", CreatedAt: now, Expires: now}
	assertJSONLiteral(t, `{
		"signature":"sig","quoteId":"quote","from":"from","to":"to","symbol":"TRX","amount":"1",
		"state":"submitted","finalized":false,"createdAt":"2026-09-16T01:02:03Z","expiresAt":"2026-09-16T01:02:03Z"
	}`, newTronRecordResponse(tronRecord))
	assertJSONLiteral(t, `null`, newTronRecordResponses(nil))
	assertJSONLiteral(t, `[]`, newTronRecordResponses([]wallet.TronRecord{}))
	tronToken := &wallet.TronToken{Contract: "token", Symbol: "USDT", Decimals: 6, Balance: "1"}
	assertSameJSON(t, tronToken, newTronTokenResponse(tronToken))
}

func TestEVMQuoteRequestParsesIntoDomainCommand(t *testing.T) {
	const address = "0x1111111111111111111111111111111111111111"
	request := evmQuoteRequest{Action: "eth", To: address, Amount: "1"}
	command, err := request.walletCommand()
	if err != nil {
		t.Fatal(err)
	}
	if command.Action != wallet.ActionETH || command.To != address || command.Amount != "1" {
		t.Fatalf("unexpected command: %+v", command)
	}

	request.Action = "not-an-action"
	if _, err := request.walletCommand(); err == nil {
		t.Fatal("invalid action crossed the HTTP to business boundary")
	}
	request.Action = "eth"
	request.To = "invalid"
	if _, err := request.walletCommand(); err == nil {
		t.Fatal("invalid address crossed the HTTP to business boundary")
	}

	poolRequest := evmQuoteRequest{
		Amount:   "1",
		Contract: "0x2222222222222222222222222222222222222222",
		TokenOut: "0x3333333333333333333333333333333333333333",
	}
	poolCommand, err := poolRequest.poolComparisonCommand()
	if err != nil {
		t.Fatal(err)
	}
	if string(poolCommand.Contract) != poolRequest.Contract || string(poolCommand.TokenOut) != poolRequest.TokenOut || poolCommand.Amount != "1" {
		t.Fatalf("unexpected pool command: %+v", poolCommand)
	}
	poolRequest.Action = "swap"
	if _, err := poolRequest.walletCommand(); !errors.Is(err, wallet.ErrInvalidAddress) {
		t.Fatalf("swap without recipient: %v", err)
	}
	poolRequest.TokenOut = "invalid"
	if _, err := poolRequest.poolComparisonCommand(); err == nil {
		t.Fatal("invalid pool token crossed the HTTP to business boundary")
	}
}

func TestCoreEVMDTOGoldens(t *testing.T) {
	now := time.Date(2026, time.September, 16, 1, 2, 3, 0, time.UTC)
	assertJSONLiteral(t, `{
		"chainId":11155111,"block":"12","blockTime":"2026-09-16T01:02:03Z","checkedAt":"2026-09-16T01:02:03Z"
	}`, newEVMNetworkResponse(&chain.Network{ChainID: 11155111, Block: "12", BlockTime: now, CheckedAt: now}))
	assertJSONLiteral(t, `{
		"address":"0x1111","wei":"1000000000000000000","eth":"1","block":"12","checkedAt":"2026-09-16T01:02:03Z"
	}`, newEVMBalanceResponse(&chain.Balance{Address: "0x1111", Wei: "1000000000000000000", ETH: "1", Block: "12", CheckedAt: now}))
	assertJSONLiteral(t, `{
		"finalized":false,"hash":"0x1234","state":"pending","checkedAt":"2026-09-16T01:02:03Z"
	}`, newEVMTransactionResponse(&chain.Transaction{Hash: "0x1234", State: "pending", CheckedAt: now}))
	assertJSONLiteral(t, `{
		"chainId":11155111,"instrumented":false,
		"rpc":{"requests":0,"transportFailures":0,"failovers":0,"lastRequestMs":0,"activeEndpoint":0,"endpointCount":0},
		"network":null,"error":"rpc unavailable"
	}`, newEVMDiagnosticsResponse(chain.Diagnostics{ChainID: 11155111}, nil, errors.New("rpc unavailable")))
}

func assertSameJSON(t *testing.T, want, got any) {
	t.Helper()
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	gotJSON, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var wantValue, gotValue any
	if err := json.Unmarshal(wantJSON, &wantValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(gotJSON, &gotValue); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(wantValue, gotValue) {
		t.Fatalf("JSON mismatch\nwant: %s\n got: %s", wantJSON, gotJSON)
	}
}

func assertJSONLiteral(t *testing.T, want string, got any) {
	t.Helper()
	var wantValue any
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatal(err)
	}
	assertSameJSON(t, wantValue, got)
}

func TestEVMHistorySeparatesConsumedNonceFromPaymentResult(t *testing.T) {
	result := newEVMHistoryResponse(&wallet.HistoryResponse{
		CanCreateTransaction: true,
		Transactions:         []wallet.HistoryItem{{Hash: "original", State: "broadcast_unknown", NonceConsumed: true}},
	})
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		CanCreateTransaction bool `json:"canCreateTransaction"`
		Transactions         []struct {
			State         string `json:"state"`
			NonceConsumed bool   `json:"nonceConsumed"`
			Finalized     bool   `json:"finalized"`
			ReplacedBy    string `json:"replacedBy"`
		} `json:"transactions"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if !decoded.CanCreateTransaction || len(decoded.Transactions) != 1 {
		t.Fatalf("missing eligibility: %s", raw)
	}
	item := decoded.Transactions[0]
	if !item.NonceConsumed || item.State != "broadcast_unknown" || item.Finalized || item.ReplacedBy != "" {
		t.Fatalf("fabricated payment outcome: %s", raw)
	}
}
