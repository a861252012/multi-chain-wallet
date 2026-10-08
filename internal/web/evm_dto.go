package web

import (
	"encoding/csv"
	"io"
	"strconv"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/a861252012/multi-chain-wallet/internal/wallet"
)

type evmNetworkResponse struct {
	ChainID   int       `json:"chainId"`
	Block     string    `json:"block"`
	BlockTime time.Time `json:"blockTime"`
	CheckedAt time.Time `json:"checkedAt"`
}

type evmRPCDiagnosticsResponse struct {
	Requests          uint64 `json:"requests"`
	TransportFailures uint64 `json:"transportFailures"`
	Failovers         uint64 `json:"failovers"`
	LastRequestMS     int64  `json:"lastRequestMs"`
	ActiveEndpoint    int    `json:"activeEndpoint"`
	EndpointCount     int    `json:"endpointCount"`
}

type evmDiagnosticsResponse struct {
	ChainID      int64                     `json:"chainId"`
	Instrumented bool                      `json:"instrumented"`
	RPC          evmRPCDiagnosticsResponse `json:"rpc"`
	Network      *evmNetworkResponse       `json:"network"`
	Error        string                    `json:"error,omitempty"`
}

func newEVMDiagnosticsResponse(diagnostics chain.Diagnostics, network *chain.Network, networkErr error) *evmDiagnosticsResponse {
	result := &evmDiagnosticsResponse{
		ChainID: diagnostics.ChainID, Instrumented: diagnostics.Instrumented,
		RPC: evmRPCDiagnosticsResponse{
			Requests: diagnostics.RPC.Requests, TransportFailures: diagnostics.RPC.TransportFailures,
			Failovers: diagnostics.RPC.Failovers, LastRequestMS: diagnostics.RPC.LastRequestMS,
			ActiveEndpoint: diagnostics.RPC.ActiveEndpoint, EndpointCount: diagnostics.RPC.EndpointCount,
		},
		Network: newEVMNetworkResponse(network),
	}
	if networkErr != nil {
		result.Error = networkErr.Error()
	}
	return result
}

func newEVMNetworkResponse(network *chain.Network) *evmNetworkResponse {
	if network == nil {
		return nil
	}
	return &evmNetworkResponse{
		ChainID: network.ChainID, Block: network.Block,
		BlockTime: network.BlockTime, CheckedAt: network.CheckedAt,
	}
}

type evmBalanceResponse struct {
	Address   string    `json:"address"`
	Wei       string    `json:"wei"`
	ETH       string    `json:"eth"`
	Block     string    `json:"block"`
	CheckedAt time.Time `json:"checkedAt"`
}

func newEVMBalanceResponse(balance *chain.Balance) *evmBalanceResponse {
	if balance == nil {
		return nil
	}
	return &evmBalanceResponse{
		Address: balance.Address, Wei: balance.Wei, ETH: balance.ETH,
		Block: balance.Block, CheckedAt: balance.CheckedAt,
	}
}

type evmTransactionResponse struct {
	Finalized     bool      `json:"finalized"`
	BlockHash     string    `json:"blockHash,omitempty"`
	Hash          string    `json:"hash"`
	State         string    `json:"state"`
	Block         string    `json:"block,omitempty"`
	Confirmations string    `json:"confirmations,omitempty"`
	GasUsed       string    `json:"gasUsed,omitempty"`
	FeeETH        string    `json:"feeEth,omitempty"`
	CheckedAt     time.Time `json:"checkedAt"`
}

func newEVMTransactionResponse(transaction *chain.Transaction) *evmTransactionResponse {
	if transaction == nil {
		return nil
	}
	return &evmTransactionResponse{
		Finalized: transaction.Finalized, BlockHash: transaction.BlockHash,
		Hash: transaction.Hash, State: transaction.State, Block: transaction.Block,
		Confirmations: transaction.Confirmations, GasUsed: transaction.GasUsed,
		FeeETH: transaction.FeeETH, CheckedAt: transaction.CheckedAt,
	}
}

type evmQuoteRequest struct {
	OrderID     string `json:"orderId,omitempty"`
	Buyer       string `json:"buyer,omitempty"`
	Hash        string `json:"hash,omitempty"`
	Action      string `json:"action"`
	To          string `json:"to"`
	Amount      string `json:"amount"`
	AmountRaw   string `json:"amountRaw,omitempty"`
	Contract    string `json:"contract,omitempty"`
	TokenOut    string `json:"tokenOut,omitempty"`
	SlippageBPS int    `json:"slippageBps,omitempty"`
	PoolFee     int    `json:"poolFee,omitempty"`
}

func (r evmQuoteRequest) walletCommand() (wallet.QuoteCommand, error) {
	return wallet.ParseQuoteRequest(r.walletRequest())
}

func (r evmQuoteRequest) poolComparisonCommand() (wallet.PoolComparisonCommand, error) {
	return wallet.ParsePoolComparisonRequest(r.walletRequest())
}

func (r evmQuoteRequest) walletRequest() *wallet.QuoteRequest {
	return &wallet.QuoteRequest{
		OrderID: r.OrderID, Buyer: r.Buyer,
		Hash:        r.Hash,
		Action:      r.Action,
		To:          r.To,
		Amount:      r.Amount,
		AmountRaw:   r.AmountRaw,
		Contract:    r.Contract,
		TokenOut:    r.TokenOut,
		SlippageBPS: r.SlippageBPS,
		PoolFee:     r.PoolFee,
	}
}

type evmWalletInfo struct {
	ChainID   int64             `json:"chainId"`
	Exists    bool              `json:"exists"`
	Address   string            `json:"address"`
	Path      string            `json:"path"`
	CSRFToken string            `json:"csrfToken"`
	Exchange  map[string]string `json:"exchange"`
}

func newEVMWalletInfo(info *wallet.WalletInfo) *evmWalletInfo {
	if info == nil {
		return nil
	}
	return &evmWalletInfo{
		ChainID: info.ChainID, Exists: info.Exists, Address: info.Address,
		Path: info.Path, CSRFToken: info.CSRFToken, Exchange: info.Exchange,
	}
}

type evmCreateResponse struct {
	Address  string `json:"address"`
	Mnemonic string `json:"mnemonic"`
	Path     string `json:"path"`
}

func newEVMCreateResponse(result *wallet.CreateResponse) *evmCreateResponse {
	if result == nil {
		return nil
	}
	return &evmCreateResponse{Address: result.Address, Mnemonic: result.Mnemonic, Path: result.Path}
}

type evmImportResponse struct {
	Address string `json:"address"`
	Path    string `json:"path"`
}

func newEVMImportResponse(result *wallet.ImportResponse) *evmImportResponse {
	if result == nil {
		return nil
	}
	return &evmImportResponse{Address: result.Address, Path: result.Path}
}

type evmAccount struct {
	Name     string `json:"name"`
	Archived bool   `json:"archived"`
	ID       string `json:"id"`
	Address  string `json:"address"`
}

func newEVMAccount(account *wallet.AccountInfo) *evmAccount {
	if account == nil {
		return nil
	}
	return &evmAccount{ID: account.ID, Address: account.Address, Name: account.Name, Archived: account.Archived}
}

func newEVMAccounts(accounts []wallet.AccountInfo) []evmAccount {
	if accounts == nil {
		return nil
	}
	result := make([]evmAccount, len(accounts))
	for i := range accounts {
		result[i] = *newEVMAccount(&accounts[i])
	}
	return result
}

type evmToken struct {
	Spender      string `json:"spender,omitempty"`
	Allowance    string `json:"allowance,omitempty"`
	AllowanceRaw string `json:"allowanceRaw,omitempty"`
	Contract     string `json:"contract"`
	Symbol       string `json:"symbol"`
	Decimals     int    `json:"decimals"`
	Balance      string `json:"balance"`
	BalanceRaw   string `json:"balanceRaw"`
	Trusted      bool   `json:"trustedMetadata"`
}

func newEVMToken(info *wallet.TokenInfo) *evmToken {
	if info == nil {
		return nil
	}
	return &evmToken{
		Spender: info.Spender, Allowance: info.Allowance, AllowanceRaw: info.AllowanceRaw,
		Contract: info.Contract, Symbol: info.Symbol, Decimals: info.Decimals,
		Balance: info.Balance, BalanceRaw: info.BalanceRaw, Trusted: info.Trusted,
	}
}

type evmVaultInfo struct {
	Enabled    bool   `json:"enabled"`
	Contract   string `json:"contract"`
	Balance    string `json:"balance"`
	BalanceRaw string `json:"balanceRaw"`
}

func newEVMVaultInfo(info *wallet.VaultInfo) *evmVaultInfo {
	if info == nil {
		return nil
	}
	return &evmVaultInfo{Enabled: info.Enabled, Contract: info.Contract, Balance: info.Balance, BalanceRaw: info.BalanceRaw}
}

type evmExchangePreview struct {
	TokenIn       string `json:"tokenIn"`
	TokenOut      string `json:"tokenOut"`
	SymbolOut     string `json:"symbolOut"`
	ExpectedOut   string `json:"expectedOut"`
	MinimumOut    string `json:"minimumOut"`
	MinimumOutRaw string `json:"minimumOutRaw"`
	Router        string `json:"router,omitempty"`
	Pool          string `json:"pool,omitempty"`
	PoolFee       int    `json:"poolFee,omitempty"`
	SlippageBPS   int    `json:"slippageBps,omitempty"`
	Deadline      string `json:"deadline"`
}

func newEVMExchangePreview(preview *wallet.ExchangePreview) *evmExchangePreview {
	if preview == nil {
		return nil
	}
	return &evmExchangePreview{
		TokenIn: preview.TokenIn, TokenOut: preview.TokenOut, SymbolOut: preview.SymbolOut,
		ExpectedOut: preview.ExpectedOut, MinimumOut: preview.MinimumOut,
		MinimumOutRaw: preview.MinimumOutRaw, Router: preview.Router, Pool: preview.Pool,
		PoolFee: preview.PoolFee, SlippageBPS: preview.SlippageBPS, Deadline: preview.Deadline,
	}
}

type evmQuoteResponse struct {
	Escrow               *evmEscrowPreview   `json:"escrow,omitempty"`
	RollupFeeETH         string              `json:"rollupFeeEth,omitempty"`
	ID                   string              `json:"id"`
	Action               string              `json:"action"`
	From                 string              `json:"from"`
	To                   string              `json:"to"`
	Contract             string              `json:"contract"`
	Symbol               string              `json:"symbol"`
	Amount               string              `json:"amount"`
	AmountRaw            string              `json:"amountRaw"`
	Nonce                string              `json:"nonce"`
	GasLimit             string              `json:"gasLimit"`
	MaxFeePerGas         string              `json:"maxFeePerGas"`
	MaxPriorityFeePerGas string              `json:"maxPriorityFeePerGas"`
	MaxFeeETH            string              `json:"maxFeeEth"`
	TotalETH             string              `json:"totalEth"`
	Data                 string              `json:"data"`
	Method               string              `json:"method"`
	ExpiresAt            string              `json:"expiresAt"`
	Exchange             *evmExchangePreview `json:"exchange,omitempty"`
}

func newEVMQuoteResponse(quote *wallet.QuoteResponse) *evmQuoteResponse {
	if quote == nil {
		return nil
	}
	return &evmQuoteResponse{
		Escrow:       newEVMEscrowPreview(quote.Escrow),
		RollupFeeETH: quote.RollupFeeETH, ID: quote.ID, Action: quote.Action, From: quote.From,
		To: quote.To, Contract: quote.Contract, Symbol: quote.Symbol, Amount: quote.Amount,
		AmountRaw: quote.AmountRaw, Nonce: quote.Nonce, GasLimit: quote.GasLimit,
		MaxFeePerGas: quote.MaxFeePerGas, MaxPriorityFeePerGas: quote.MaxPriorityFeePerGas,
		MaxFeeETH: quote.MaxFeeETH, TotalETH: quote.TotalETH, Data: quote.Data,
		Method: quote.Method, ExpiresAt: quote.ExpiresAt, Exchange: newEVMExchangePreview(quote.Exchange),
	}
}

type evmSendResponse struct {
	EscrowAction string `json:"escrowAction,omitempty"`
	Reused       bool   `json:"reused,omitempty"`
	Hash         string `json:"hash"`
	State        string `json:"state"`
	To           string `json:"to"`
	Amount       string `json:"amount"`
	Symbol       string `json:"symbol"`
	Action       string `json:"action"`
	CreatedAt    string `json:"createdAt"`
}

func newEVMSendResponse(result *wallet.SendResponse) *evmSendResponse {
	if result == nil {
		return nil
	}
	return &evmSendResponse{
		EscrowAction: result.EscrowAction,
		Reused:       result.Reused, Hash: result.Hash, State: result.State, To: result.To,
		Amount: result.Amount, Symbol: result.Symbol, Action: result.Action, CreatedAt: result.CreatedAt,
	}
}

type evmHistoryItem struct {
	EscrowAction   string `json:"escrowAction,omitempty"`
	NonceConsumed  bool   `json:"nonceConsumed,omitempty"`
	OrderID        string `json:"orderId,omitempty"`
	EscrowBuyer    string `json:"escrowBuyer,omitempty"`
	EscrowContract string `json:"escrowContract,omitempty"`
	QuoteID        string `json:"quoteId"`
	ReplacedBy     string `json:"replacedBy,omitempty"`
	Finalized      bool   `json:"finalized"`
	Hash           string `json:"hash"`
	State          string `json:"state"`
	To             string `json:"to"`
	Amount         string `json:"amount"`
	Symbol         string `json:"symbol"`
	Action         string `json:"action"`
	CreatedAt      string `json:"createdAt"`
	Confirmations  string `json:"confirmations,omitempty"`
	FeeETH         string `json:"feeEth,omitempty"`
	Error          string `json:"error,omitempty"`
}

type evmHistoryResponse struct {
	CanCreateTransaction bool             `json:"canCreateTransaction,omitempty"`
	Transactions         []evmHistoryItem `json:"transactions"`
	RefreshError         string           `json:"refreshError,omitempty"`
}

func newEVMHistoryResponse(history *wallet.HistoryResponse) *evmHistoryResponse {
	if history == nil {
		return nil
	}
	result := &evmHistoryResponse{RefreshError: history.RefreshError, CanCreateTransaction: history.CanCreateTransaction}
	if history.Transactions != nil {
		result.Transactions = make([]evmHistoryItem, len(history.Transactions))
		for i, item := range history.Transactions {
			result.Transactions[i] = evmHistoryItem{
				EscrowAction:  item.EscrowAction,
				NonceConsumed: item.NonceConsumed,
				OrderID:       item.OrderID, EscrowBuyer: item.EscrowBuyer, EscrowContract: item.EscrowContract,
				QuoteID: item.QuoteID, ReplacedBy: item.ReplacedBy, Finalized: item.Finalized,
				Hash: item.Hash, State: item.State, To: item.To, Amount: item.Amount,
				Symbol: item.Symbol, Action: item.Action, CreatedAt: item.CreatedAt,
				Confirmations: item.Confirmations, FeeETH: item.FeeETH, Error: item.Error,
			}
		}
	}
	return result
}

type evmPoolQuote struct {
	Fee       int    `json:"fee"`
	Pool      string `json:"pool,omitempty"`
	Output    string `json:"output,omitempty"`
	OutputRaw string `json:"outputRaw,omitempty"`
	Error     string `json:"error,omitempty"`
}

type evmPoolComparison struct {
	Pools     []evmPoolQuote `json:"pools"`
	BestFee   int            `json:"bestFee"`
	Symbol    string         `json:"symbol"`
	CheckedAt time.Time      `json:"checkedAt"`
}

func newEVMPoolComparison(comparison *wallet.PoolComparison) *evmPoolComparison {
	if comparison == nil {
		return nil
	}
	result := &evmPoolComparison{BestFee: comparison.BestFee, Symbol: comparison.Symbol, CheckedAt: comparison.CheckedAt}
	if comparison.Pools != nil {
		result.Pools = make([]evmPoolQuote, len(comparison.Pools))
		for i, pool := range comparison.Pools {
			result.Pools[i] = evmPoolQuote{Fee: pool.Fee, Pool: pool.Pool, Output: pool.Output, OutputRaw: pool.OutputRaw, Error: pool.Error}
		}
	}
	return result
}

type evmMovement struct {
	Kind         string `json:"kind"`
	Asset        string `json:"asset"`
	Raw          string `json:"raw"`
	Counterparty string `json:"counterparty"`
	Evidence     string `json:"evidence"`
}

type evmActivity struct {
	Hash      string        `json:"hash"`
	State     string        `json:"state"`
	Block     string        `json:"block,omitempty"`
	BlockHash string        `json:"blockHash,omitempty"`
	BlockTime string        `json:"blockTime,omitempty"`
	CheckedAt time.Time     `json:"checkedAt"`
	Movements []evmMovement `json:"movements"`
	Error     string        `json:"error,omitempty"`
}

func newEVMActivity(activity *chain.Activity) *evmActivity {
	if activity == nil {
		return nil
	}
	result := &evmActivity{
		Hash: activity.Hash, State: activity.State, Block: activity.Block,
		BlockHash: activity.BlockHash, BlockTime: activity.BlockTime,
		CheckedAt: activity.CheckedAt, Error: activity.Error,
	}
	if activity.Movements != nil {
		result.Movements = make([]evmMovement, len(activity.Movements))
		for i, movement := range activity.Movements {
			result.Movements[i] = evmMovement{
				Kind: movement.Kind, Asset: movement.Asset, Raw: movement.Raw,
				Counterparty: movement.Counterparty, Evidence: movement.Evidence,
			}
		}
	}
	return result
}

type evmActivityTotal struct {
	Asset       string `json:"asset"`
	ReceivedRaw string `json:"receivedRaw"`
	SentRaw     string `json:"sentRaw"`
	FeeRaw      string `json:"feeRaw"`
	NetRaw      string `json:"netRaw"`
}

type evmActivityResponse struct {
	ChainID           int64              `json:"chainId"`
	Transactions      []*evmActivity     `json:"transactions"`
	Totals            []evmActivityTotal `json:"totals"`
	Page              int                `json:"page"`
	Pages             int                `json:"pages"`
	TotalTransactions int                `json:"totalTransactions"`
	Incomplete        bool               `json:"incomplete"`
}

func newEVMActivityResponse(activity *wallet.ActivityResponse) *evmActivityResponse {
	if activity == nil {
		return nil
	}
	result := &evmActivityResponse{
		ChainID: activity.ChainID, Page: activity.Page, Pages: activity.Pages,
		TotalTransactions: activity.TotalTransactions, Incomplete: activity.Incomplete,
	}
	if activity.Transactions != nil {
		result.Transactions = make([]*evmActivity, len(activity.Transactions))
		for i, transaction := range activity.Transactions {
			result.Transactions[i] = newEVMActivity(transaction)
		}
	}
	if activity.Totals != nil {
		result.Totals = make([]evmActivityTotal, len(activity.Totals))
		for i, total := range activity.Totals {
			result.Totals[i] = evmActivityTotal{
				Asset: total.Asset, ReceivedRaw: total.ReceivedRaw, SentRaw: total.SentRaw,
				FeeRaw: total.FeeRaw, NetRaw: total.NetRaw,
			}
		}
	}
	return result
}

func writeEVMActivityCSV(w io.Writer, response *evmActivityResponse) error {
	writer := csv.NewWriter(w)
	chainID := response.ChainID
	if chainID == 0 {
		chainID = chain.SepoliaID
	}
	if err := writer.Write([]string{"chain_id", "hash", "state", "block", "block_time", "kind", "asset", "amount_raw", "counterparty", "evidence"}); err != nil {
		return err
	}
	for _, transaction := range response.Transactions {
		if len(transaction.Movements) == 0 {
			if err := writer.Write([]string{strconv.FormatInt(chainID, 10), transaction.Hash, transaction.State, transaction.Block, transaction.BlockTime, "", "", "", "", ""}); err != nil {
				return err
			}
		}
		for _, movement := range transaction.Movements {
			if err := writer.Write([]string{
				strconv.FormatInt(chainID, 10), transaction.Hash, transaction.State,
				transaction.Block, transaction.BlockTime, movement.Kind, movement.Asset,
				movement.Raw, movement.Counterparty, movement.Evidence,
			}); err != nil {
				return err
			}
		}
	}
	writer.Flush()
	return writer.Error()
}

type evmSyncResponse struct {
	From  uint64 `json:"from"`
	To    uint64 `json:"to"`
	Added int    `json:"added"`
}

func newEVMSyncResponse(result *wallet.SyncResponse) *evmSyncResponse {
	if result == nil {
		return nil
	}
	return &evmSyncResponse{From: result.From, To: result.To, Added: result.Added}
}

type evmScanProgress struct {
	Enabled   bool      `json:"enabled"`
	Start     uint64    `json:"start"`
	Next      uint64    `json:"next"`
	Finalized uint64    `json:"finalized"`
	Tokens    []string  `json:"tokens"`
	Error     string    `json:"error,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func newEVMScanProgress(progress *wallet.ScanProgress) *evmScanProgress {
	if progress == nil {
		return nil
	}
	return &evmScanProgress{
		Enabled: progress.Enabled, Start: progress.Start, Next: progress.Next,
		Finalized: progress.Finalized, Tokens: progress.Tokens, Error: progress.Error,
		UpdatedAt: progress.UpdatedAt,
	}
}
