package wallet

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

var (
	ErrWalletExists                 = errors.New("錢包已存在，無法覆寫")
	ErrWalletNotFound               = errors.New("尚未建立或匯入錢包")
	ErrInvalidPassword              = errors.New("密碼長度必須介於 12 至 128 字元")
	ErrPasswordMismatch             = errors.New("密碼錯誤，無法解密金鑰")
	ErrInvalidMnemonic              = errors.New("助記詞格式不正確或校驗失敗")
	ErrInvalidAddress               = errors.New("地址格式不正確，請輸入 0x 開頭的 40 位十六進位地址")
	ErrZeroAddress                  = errors.New("不可使用零地址")
	ErrMalformedChecksum            = errors.New("地址混合大小寫校驗和不正確")
	ErrWrongChain                   = errors.New("RPC 連到其他網路，已停止操作；Multi-Chain Wallet 僅允許支援的測試網")
	ErrQuoteNotFound                = errors.New("找不到指定的報價或報價已過期")
	ErrQuoteExpired                 = errors.New("報價已過期，請重新建立報價")
	ErrQuoteStorageFull             = errors.New("報價數量已達上限 (256)，請稍後重試")
	ErrTxInFlight                   = errors.New("已有處理中或廣播結果未確認的交易，請等待該交易確認後再操作")
	ErrNonceMismatch                = errors.New("鏈上 Nonce 已變更，請重新建立報價")
	ErrInsufficientFunds            = errors.New("餘額不足以支付轉帳金額與最高 Gas 手續費")
	ErrNotContract                  = errors.New("指定合約地址在目前測試網上沒有 bytecode")
	ErrApprovalRace                 = errors.New("既有授權額度大於 0，為避免 ERC20 approve race pattern，必須先將授權額度歸零（approve 0）後才能設定新的非零額度")
	ErrUnlimitedAllowanceNotAllowed = errors.New("不支援無上限授權，請輸入明確的授權額度")
	ErrSimulationFailed             = errors.New("交易模擬執行失敗（eth_call 未通過）")
	ErrJournalFull                  = errors.New("交易日誌數量已達上限 (1000)，為保全歷史紀錄已拒絕新交易")
	ErrDecimalsTooLarge             = errors.New("代幣小數位數超出上限 36")
	ErrSymbolTooLong                = errors.New("代幣符號長度超出上限 32 字元")
	ErrTooManyScryptRequests        = errors.New("系統密碼運算繁忙，請稍後重試")
)

// SendRejectedError means this quote has no durable transaction and was not broadcast.
// It preserves the cause for HTTP status mapping and existing callers.
type SendRejectedError struct{ Err error }

func (e *SendRejectedError) Error() string { return e.Err.Error() }
func (e *SendRejectedError) Unwrap() error { return e.Err }

type QuoteID string

var opaqueIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)
var transactionHashPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)

func ParseQuoteID(value string) (QuoteID, error) {
	if !opaqueIDPattern.MatchString(value) {
		return "", fmt.Errorf("交易報價 ID 格式錯誤")
	}
	return QuoteID(value), nil
}

func ParseLegacyQuoteID(value string) (QuoteID, error) {
	if value == "" || len(value) > 128 {
		return "", errors.New("交易日誌的報價 ID 格式錯誤")
	}
	return QuoteID(value), nil
}

type EVMAddress string

func ParseEVMAddress(value string) (EVMAddress, error) {
	address, err := ValidateAddress(value)
	if err != nil {
		return "", err
	}
	return EVMAddress(address.Hex()), nil
}

type TransactionHash string

func ParseTransactionHash(value string) (TransactionHash, error) {
	if !transactionHashPattern.MatchString(value) {
		return "", errors.New("交易雜湊格式錯誤")
	}
	return TransactionHash(value), nil
}

type QuoteCommand struct {
	OrderID     OrderReference
	Buyer       EVMAddress
	Hash        TransactionHash
	Action      TransactionAction
	To          EVMAddress
	Amount      string
	AmountRaw   string
	Contract    EVMAddress
	TokenOut    EVMAddress
	SlippageBPS int
	PoolFee     int
}

func ParseQuoteRequest(request *QuoteRequest) (QuoteCommand, error) {
	if request == nil {
		return QuoteCommand{}, errors.New("缺少交易報價內容")
	}
	action, err := ParseTransactionAction(request.Action)
	if err != nil {
		return QuoteCommand{}, err
	}
	var hash TransactionHash
	var to, contract, tokenOut EVMAddress
	if action == ActionSpeedup || action == ActionCancel {
		hash, err = ParseTransactionHash(request.Hash)
		if err != nil {
			return QuoteCommand{}, errors.New("欲替代的交易雜湊格式錯誤")
		}
	} else {
		if request.To != "" {
			to, err = ParseEVMAddress(request.To)
			if err != nil {
				return QuoteCommand{}, err
			}
		} else {
			return QuoteCommand{}, ErrInvalidAddress
		}
		if request.Contract == "" {
			switch action {
			case ActionTransfer:
				return QuoteCommand{}, errors.New("代幣轉帳必須指定 contract 合約地址")
			case ActionApprove:
				return QuoteCommand{}, errors.New("代幣授權必須指定 contract 合約地址")
			}
		}
		if request.Contract != "" {
			if action == ActionVaultDeposit || action == ActionVaultWithdraw {
				return QuoteCommand{}, errors.New("合約地址由伺服器設定，無法在操作時變更")
			}
			contract, err = ParseEVMAddress(request.Contract)
			if err != nil {
				return QuoteCommand{}, err
			}
		}
		if request.TokenOut != "" {
			tokenOut, err = ParseEVMAddress(request.TokenOut)
			if err != nil {
				return QuoteCommand{}, err
			}
		}
	}
	var orderID OrderReference
	var buyer EVMAddress
	if isEscrowAction(action) {
		if request.Contract != "" || request.TokenOut != "" || request.AmountRaw != "" {
			return QuoteCommand{}, errors.New("託管合約與代幣由伺服器設定")
		}
		orderID, err = ParseOrderReference(request.OrderID)
		if err != nil {
			return QuoteCommand{}, err
		}
		if request.Buyer != "" {
			buyer, err = ParseEVMAddress(request.Buyer)
			if err != nil {
				return QuoteCommand{}, err
			}
		}
	} else if request.OrderID != "" || request.Buyer != "" {
		return QuoteCommand{}, errors.New("此操作不接受訂單欄位")
	}
	return QuoteCommand{
		OrderID: orderID, Buyer: buyer,
		Hash: hash, Action: action, To: to, Amount: request.Amount,
		AmountRaw: request.AmountRaw, Contract: contract, TokenOut: tokenOut,
		SlippageBPS: request.SlippageBPS, PoolFee: request.PoolFee,
	}, nil
}

type PoolComparisonCommand struct {
	Contract EVMAddress
	TokenOut EVMAddress
	Amount   string
}

func ParsePoolComparisonRequest(request *QuoteRequest) (PoolComparisonCommand, error) {
	if request == nil {
		return PoolComparisonCommand{}, errors.New("缺少流動池比較內容")
	}
	contract, err := ParseEVMAddress(request.Contract)
	if err != nil {
		return PoolComparisonCommand{}, err
	}
	tokenOut, err := ParseEVMAddress(request.TokenOut)
	if err != nil {
		return PoolComparisonCommand{}, err
	}
	if contract == tokenOut {
		return PoolComparisonCommand{}, errors.New("兌換資產不可相同")
	}
	return PoolComparisonCommand{Contract: contract, TokenOut: tokenOut, Amount: request.Amount}, nil
}

type SolanaStatus struct {
	Exists  bool
	Network string
	Address string
}

type SolanaBalance struct {
	Address  string
	SOL      string
	Lamports string
	Slot     uint64
}

type SolanaCreateResponse struct {
	Address  string
	Mnemonic string
}

type SolanaAirdropResponse struct {
	Signature string
	State     string
	Amount    string
}

type TronStatus struct {
	Exists  bool
	Network string
	Address string
}

type TronBalance struct {
	Address   string
	TRX       string
	Active    bool
	Bandwidth int64
	Energy    int64
}

type TronCreateResponse struct {
	Address  string
	Mnemonic string
}

type TransactionAction string

const (
	ActionETH           TransactionAction = "eth"
	ActionTransfer      TransactionAction = "transfer"
	ActionApprove       TransactionAction = "approve"
	ActionWrap          TransactionAction = "wrap"
	ActionUnwrap        TransactionAction = "unwrap"
	ActionSwap          TransactionAction = "swap"
	ActionSpeedup       TransactionAction = "speedup"
	ActionCancel        TransactionAction = "cancel"
	ActionEscrowFund    TransactionAction = "escrow_fund"
	ActionEscrowRelease TransactionAction = "escrow_release"
	ActionEscrowRefund  TransactionAction = "escrow_refund"
	ActionVaultDeposit  TransactionAction = "vault_deposit"
	ActionVaultWithdraw TransactionAction = "vault_withdraw"
)

func ParseTransactionAction(value string) (TransactionAction, error) {
	action := TransactionAction(value)
	switch action {
	case ActionETH, ActionTransfer, ActionApprove, ActionWrap, ActionUnwrap, ActionSwap, ActionSpeedup, ActionCancel, ActionVaultDeposit, ActionVaultWithdraw, ActionEscrowFund, ActionEscrowRelease, ActionEscrowRefund:
		return action, nil
	default:
		return "", fmt.Errorf("不支援的交易操作 %q", value)
	}
}

type JournalState string

const (
	JournalPending            JournalState = "pending"
	JournalSubmitted          JournalState = "submitted"
	JournalBroadcastUnknown   JournalState = "broadcast_unknown"
	JournalReceiptUnavailable JournalState = "receipt_unavailable"
	JournalSucceeded          JournalState = "succeeded"
	JournalReverted           JournalState = "reverted"
	JournalReorgDetected      JournalState = "reorg_detected"
	JournalReplaced           JournalState = "replaced"
)

func ParseJournalState(value string) (JournalState, error) {
	state := JournalState(value)
	switch state {
	case JournalPending, JournalSubmitted, JournalBroadcastUnknown, JournalReceiptUnavailable,
		JournalSucceeded, JournalReverted, JournalReorgDetected, JournalReplaced:
		return state, nil
	default:
		return "", fmt.Errorf("不支援的交易狀態 %q", value)
	}
}

type WalletInfo struct {
	ChainID   int64             `json:"chainId"`
	Exists    bool              `json:"exists"`
	Address   string            `json:"address"`
	Path      string            `json:"path"`
	CSRFToken string            `json:"csrfToken"`
	Exchange  map[string]string `json:"exchange"`
}

type CreateResponse struct {
	Address  string `json:"address"`
	Mnemonic string `json:"mnemonic"`
	Path     string `json:"path"`
}

type ImportResponse struct {
	Address string `json:"address"`
	Path    string `json:"path"`
}

type TokenInfo struct {
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

type QuoteRequest struct {
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

type QuoteResponse struct {
	Escrow               *EscrowPreview   `json:"escrow,omitempty"`
	RollupFeeETH         string           `json:"rollupFeeEth,omitempty"`
	ID                   string           `json:"id"`
	Action               string           `json:"action"`
	From                 string           `json:"from"`
	To                   string           `json:"to"`
	Contract             string           `json:"contract"`
	Symbol               string           `json:"symbol"`
	Amount               string           `json:"amount"`
	AmountRaw            string           `json:"amountRaw"`
	Nonce                string           `json:"nonce"`
	GasLimit             string           `json:"gasLimit"`
	MaxFeePerGas         string           `json:"maxFeePerGas"`
	MaxPriorityFeePerGas string           `json:"maxPriorityFeePerGas"`
	MaxFeeETH            string           `json:"maxFeeEth"`
	TotalETH             string           `json:"totalEth"`
	Data                 string           `json:"data"`
	Method               string           `json:"method"`
	ExpiresAt            string           `json:"expiresAt"`
	Exchange             *ExchangePreview `json:"exchange,omitempty"`
}

type SendResponse struct {
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

type HistoryItem struct {
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

type HistoryResponse struct {
	CanCreateTransaction bool          `json:"canCreateTransaction,omitempty"`
	Transactions         []HistoryItem `json:"transactions"`
	RefreshError         string        `json:"refreshError,omitempty"`
}

type JournalRecord struct {
	OrderID OrderReference
	// These fields are derived from SignedRaw when loading or appending.
	EscrowAction   TransactionAction
	EscrowBuyer    EVMAddress
	EscrowContract EVMAddress
	Finalized      bool
	Hash           TransactionHash
	QuoteID        QuoteID
	State          JournalState
	To             EVMAddress
	Amount         string
	AmountRaw      string
	Symbol         string
	Action         TransactionAction
	Nonce          uint64
	SignedRaw      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Confirmations  string
	FeeETH         string
	Error          string
	Version        uint64
}
