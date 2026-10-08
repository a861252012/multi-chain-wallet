package wallet

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"regexp"
	"strings"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var escrowABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(`[
 {"type":"function","name":"token","inputs":[],"outputs":[{"type":"address"}],"stateMutability":"view"},
 {"type":"function","name":"orders","inputs":[{"type":"address"},{"type":"bytes32"}],"outputs":[{"type":"address"},{"type":"uint256"},{"type":"uint8"}],"stateMutability":"view"},
 {"type":"function","name":"fund","inputs":[{"type":"bytes32"},{"type":"address"},{"type":"uint256"}],"outputs":[],"stateMutability":"nonpayable"},
 {"type":"function","name":"release","inputs":[{"type":"address"},{"type":"bytes32"}],"outputs":[],"stateMutability":"nonpayable"},
 {"type":"function","name":"refund","inputs":[{"type":"address"},{"type":"bytes32"}],"outputs":[],"stateMutability":"nonpayable"}
 ]`))
	if err != nil {
		panic(err)
	}
	return a
}()

type OrderReference string

var orderReferencePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
var orderKeyPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)

func ParseOrderReference(value string) (OrderReference, error) {
	if !orderReferencePattern.MatchString(value) && !orderKeyPattern.MatchString(value) {
		return "", errors.New("訂單編號請使用 1–64 個英文字母、數字、連字號或底線")
	}
	return OrderReference(value), nil
}

// Old journals retain the signed order key even when the original reference was lost.
func (reference OrderReference) key() common.Hash {
	if orderKeyPattern.MatchString(string(reference)) {
		return common.HexToHash(string(reference))
	}
	return crypto.Keccak256Hash([]byte(reference))
}

type escrowCall struct {
	action   TransactionAction
	buyer    common.Address
	orderKey common.Hash
	seller   common.Address
	amount   *big.Int
}

// A replacement's outer action is speedup. Recover its original operation
// from canonical signed calldata, including old journals without order metadata.
func decodeEscrowTransaction(tx *types.Transaction) (*escrowCall, error) {
	data := tx.Data()
	if len(data) < 4 {
		return nil, nil
	}
	method, err := escrowABI.MethodById(data[:4])
	if err != nil {
		return nil, nil
	}
	action := map[string]TransactionAction{"fund": ActionEscrowFund, "release": ActionEscrowRelease, "refund": ActionEscrowRefund}[method.Name]
	if action == "" {
		return nil, nil
	}
	if tx.To() == nil || tx.Value().Sign() != 0 {
		return nil, errors.New("交易日誌的託管操作與簽名不符")
	}
	values, err := method.Inputs.Unpack(data[4:])
	if err != nil {
		return nil, errors.New("交易日誌的訂單資料不完整")
	}
	encoded, err := method.Inputs.Pack(values...)
	if err != nil || !bytes.Equal(encoded, data[4:]) {
		return nil, errors.New("交易日誌的訂單資料不符")
	}
	sender, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
	if err != nil {
		return nil, errors.New("無法驗證交易日誌的簽名")
	}
	call := &escrowCall{action: action}
	if action == ActionEscrowFund {
		call.buyer, call.orderKey = sender, values[0].([32]byte)
		call.seller, call.amount = values[1].(common.Address), values[2].(*big.Int)
	} else {
		call.buyer, call.orderKey = values[0].(common.Address), values[1].([32]byte)
	}
	return call, nil
}

// Persist only the escrow reference. Derive the original action, buyer and
// contract from signed bytes, never from unverified disk metadata.
func restoreEscrowJournal(record *JournalRecord, tx *types.Transaction) error {
	record.EscrowAction, record.EscrowBuyer, record.EscrowContract = "", "", ""
	// Disk action is a display field. Decode first so removing the reference
	// or changing that field cannot bypass escrow checks during a speedup.
	call, err := decodeEscrowTransaction(tx)
	if err != nil {
		return err
	}
	if call == nil {
		if isEscrowAction(record.Action) || record.OrderID != "" {
			return errors.New("交易日誌的託管操作與簽名不符")
		}
		return nil
	}
	if record.Action != "" && record.Action != ActionSpeedup && record.Action != call.action {
		return errors.New("交易日誌的託管操作與簽名不符")
	}
	if record.OrderID == "" {
		record.OrderID = OrderReference(call.orderKey.Hex())
	} else if _, err := ParseOrderReference(string(record.OrderID)); err != nil || record.OrderID.key() != call.orderKey {
		return errors.New("交易日誌的訂單編號與簽名不符")
	}
	record.EscrowAction = call.action
	record.EscrowBuyer = EVMAddress(call.buyer.Hex())
	record.EscrowContract = EVMAddress(tx.To().Hex())
	return nil
}

func isEscrowAction(action TransactionAction) bool {
	return action == ActionEscrowFund || action == ActionEscrowRelease || action == ActionEscrowRefund
}

type EscrowInfo struct {
	Enabled   bool
	Contract  string
	Token     string
	Balance   string
	Allowance string
}

type EscrowOrder struct {
	OrderID   string
	Buyer     string
	Seller    string
	Amount    string
	AmountRaw string
	State     string
	Block     string
	Finalized bool
}

type EscrowPreview struct {
	Action  TransactionAction
	OrderID string
	Buyer   string
	Seller  string
	Token   string
}

// SetEscrow configures a single trusted token before requests start. Main uses Circle's Sepolia USDC.
func (s *Service) SetEscrow(address, token string) error {
	if s.client.ChainID() != chain.SepoliaID || address == "" && token == "" {
		s.escrowAddress, s.escrowToken = "", ""
		return nil
	}
	contract, err := ValidateAddress(address)
	if err != nil {
		return err
	}
	paymentToken, err := ValidateAddress(token)
	if err != nil {
		return err
	}
	if contract == paymentToken {
		return errors.New("託管合約與代幣地址不可相同")
	}
	s.escrowAddress, s.escrowToken = contract.Hex(), paymentToken.Hex()
	return nil
}

func verifyEscrow(ctx context.Context, caller ChainCaller, contract, token common.Address) error {
	if err := VerifyContractBytecode(ctx, caller, contract); err != nil {
		return err
	}
	data, err := escrowABI.Pack("token")
	if err != nil {
		return err
	}
	result, err := caller.CallContract(ctx, ethereum.CallMsg{To: &contract, Data: data}, nil)
	if err != nil {
		return err
	}
	values, err := escrowABI.Unpack("token", result)
	if err != nil || len(values) != 1 || values[0].(common.Address) != token {
		return errors.New("託管合約的付款代幣與設定不符")
	}
	symbol, decimals, err := QueryERC20Metadata(ctx, caller, token)
	if err != nil {
		return err
	}
	if symbol != "USDC" || decimals != 6 {
		return errors.New("託管僅支援設定的 6 位小數測試 USDC")
	}
	return nil
}

func (s *Service) EscrowStatus(ctx context.Context) (*EscrowInfo, error) {
	if s.escrowAddress == "" || s.client.ChainID() != chain.SepoliaID {
		return &EscrowInfo{}, nil
	}
	contract, token := common.HexToAddress(s.escrowAddress), common.HexToAddress(s.escrowToken)
	if err := verifyEscrow(ctx, s.client, contract, token); err != nil {
		return nil, err
	}
	owner, err := s.keystore.Address()
	if err != nil {
		return nil, err
	}
	balance, err := QueryERC20BalanceOf(ctx, s.client, token, common.HexToAddress(owner))
	if err != nil {
		return nil, err
	}
	allowance, err := QueryERC20Allowance(ctx, s.client, token, common.HexToAddress(owner), contract)
	if err != nil {
		return nil, err
	}
	return &EscrowInfo{Enabled: true, Contract: contract.Hex(), Token: token.Hex(), Balance: FormatUnits(balance, 6), Allowance: FormatUnits(allowance, 6)}, nil
}

func queryEscrowOrder(ctx context.Context, caller ChainCaller, contract, buyer common.Address, reference OrderReference, block *big.Int) (*EscrowOrder, error) {
	data, err := escrowABI.Pack("orders", buyer, reference.key())
	if err != nil {
		return nil, err
	}
	result, err := caller.CallContract(ctx, ethereum.CallMsg{To: &contract, Data: data}, block)
	if err != nil {
		return nil, err
	}
	values, err := escrowABI.Unpack("orders", result)
	if err != nil || len(values) != 3 {
		return nil, errors.New("無法讀取訂單狀態")
	}
	state := values[2].(uint8)
	if state > 3 {
		return nil, errors.New("未知的訂單狀態")
	}
	amount := values[1].(*big.Int)
	return &EscrowOrder{OrderID: string(reference), Buyer: buyer.Hex(), Seller: values[0].(common.Address).Hex(), Amount: FormatUnits(amount, 6), AmountRaw: amount.String(), State: []string{"none", "funded", "released", "refunded"}[state]}, nil
}

func (s *Service) EscrowOrder(ctx context.Context, buyer EVMAddress, reference OrderReference) (*EscrowOrder, error) {
	if s.escrowAddress == "" || s.client.ChainID() != chain.SepoliaID {
		return nil, errors.New("此環境尚未開放付款託管")
	}
	if _, err := ParseOrderReference(string(reference)); err != nil {
		return nil, err
	}
	address, err := ValidateAddress(string(buyer))
	if err != nil {
		return nil, err
	}
	contract := common.HexToAddress(s.escrowAddress)
	if err := verifyEscrow(ctx, s.client, contract, common.HexToAddress(s.escrowToken)); err != nil {
		return nil, err
	}
	head, err := s.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}
	order, err := queryEscrowOrder(ctx, s.client, contract, address, reference, head.Number)
	if err != nil {
		return nil, err
	}
	canonical, err := s.client.HeaderByNumber(ctx, head.Number)
	if err != nil {
		return nil, err
	}
	if canonical.Hash() != head.Hash() {
		return nil, errors.New("區塊已變更，請重新查詢訂單")
	}
	order.Block = head.Number.String()
	// If finality cannot be read, retain the known inclusion state without claiming finality.
	finalized, finalErr := s.client.HeaderByNumber(ctx, big.NewInt(-3))
	if finalErr == nil && finalized != nil {
		if finalized.Number.Cmp(head.Number) >= 0 {
			order.Finalized = true
		} else {
			// Finality usually trails latest. Check this order's monotonic state,
			// rather than waiting for the entire latest snapshot to be finalized.
			settled, err := queryEscrowOrder(ctx, s.client, contract, address, reference, finalized.Number)
			order.Finalized = err == nil && settled.State == order.State && settled.Seller == order.Seller && settled.AmountRaw == order.AmountRaw
		}
	}
	return order, nil
}

type escrowPayload struct {
	To      common.Address
	Data    []byte
	Amount  *big.Int
	Preview *EscrowPreview
	Method  string
}

func prepareEscrow(ctx context.Context, provider ChainQuoteProvider, from common.Address, command QuoteCommand) (*escrowPayload, error) {
	if provider.ChainID() != chain.SepoliaID || command.Contract == "" || command.TokenOut == "" {
		return nil, errors.New("此環境尚未開放付款託管")
	}
	reference, err := ParseOrderReference(string(command.OrderID))
	if err != nil {
		return nil, err
	}
	contract, token := common.HexToAddress(string(command.Contract)), common.HexToAddress(string(command.TokenOut))
	if err := verifyEscrow(ctx, provider, contract, token); err != nil {
		return nil, err
	}
	buyer := common.HexToAddress(string(command.Buyer))
	if command.Action == ActionEscrowFund {
		if command.Buyer != "" && buyer != from {
			return nil, errors.New("付款人必須是目前錢包")
		}
		buyer = from
	}
	if buyer == (common.Address{}) {
		return nil, ErrInvalidAddress
	}
	order, err := queryEscrowOrder(ctx, provider, contract, buyer, reference, nil)
	if err != nil {
		return nil, err
	}
	seller := common.HexToAddress(order.Seller)
	amount, ok := new(big.Int).SetString(order.AmountRaw, 10)
	if !ok {
		return nil, errors.New("無法讀取訂單金額")
	}
	method := ""
	var data []byte
	switch command.Action {
	case ActionEscrowFund:
		if order.State != "none" {
			return nil, errors.New("此付款人的訂單編號已使用，請查詢原訂單")
		}
		seller = common.HexToAddress(string(command.To))
		if seller == from || seller == contract || seller == (common.Address{}) {
			return nil, errors.New("收款人須為另一個錢包地址")
		}
		amount, err = ParseUnits(command.Amount, 6)
		if err != nil {
			return nil, err
		}
		if amount.Sign() <= 0 {
			return nil, errors.New("付款金額必須大於 0")
		}
		allowance, err := QueryERC20Allowance(ctx, provider, token, from, contract)
		if err != nil {
			return nil, err
		}
		if allowance.Cmp(amount) < 0 {
			return nil, errors.New("授權額度不足，請先完成本次金額的 USDC 授權")
		}
		method = "fund"
		data, err = escrowABI.Pack(method, reference.key(), seller, amount)
	case ActionEscrowRelease, ActionEscrowRefund:
		if order.State != "funded" {
			return nil, errors.New("訂單未在託管中，請更新狀態；已完成的訂單不能重複操作")
		}
		if command.Action == ActionEscrowRelease {
			if from != buyer {
				return nil, errors.New("只有付款人可以放款給收款人")
			}
			method = "release"
		} else {
			if from != seller {
				return nil, errors.New("只有收款人可以退款給原付款人")
			}
			method = "refund"
		}
		data, err = escrowABI.Pack(method, buyer, reference.key())
	default:
		return nil, errors.New("不支援的託管操作")
	}
	if err != nil {
		return nil, err
	}
	if err := simulateEscrow(ctx, provider, from, contract, data); err != nil {
		return nil, err
	}
	to := seller
	if command.Action == ActionEscrowRefund {
		to = buyer
	}
	return &escrowPayload{To: to, Data: data, Amount: amount, Method: method, Preview: &EscrowPreview{Action: command.Action, OrderID: string(reference), Buyer: buyer.Hex(), Seller: seller.Hex(), Token: token.Hex()}}, nil
}

func (s *Service) prepareEscrowSpeedup(ctx context.Context, from common.Address, record *JournalRecord, tx *types.Transaction) (*escrowPayload, error) {
	if s.escrowAddress == "" || tx.To() == nil || *tx.To() != common.HexToAddress(s.escrowAddress) {
		return nil, errors.New("託管設定已變更，請重新預估")
	}
	call, err := decodeEscrowTransaction(tx)
	if err != nil {
		return nil, err
	}
	if call == nil {
		return nil, errors.New("原交易不是託管操作")
	}
	command := QuoteCommand{
		Action: call.action, OrderID: record.OrderID, Buyer: EVMAddress(call.buyer.Hex()),
		Contract: EVMAddress(s.escrowAddress), TokenOut: EVMAddress(s.escrowToken),
	}
	if call.action == ActionEscrowFund {
		command.To, command.Amount = EVMAddress(call.seller.Hex()), FormatUnits(call.amount, 6)
	}
	// Funding details come from signed calldata; settlement details come from
	// the on-chain order. Never use the journal's display amount or recipient.
	payload, err := prepareEscrow(ctx, s.client, from, command)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(payload.Data, tx.Data()) {
		return nil, errors.New("原託管交易的簽名內容與訂單不符")
	}
	return payload, nil
}

func simulateEscrow(ctx context.Context, caller ChainCaller, from, contract common.Address, data []byte) error {
	_, err := caller.CallContract(ctx, ethereum.CallMsg{From: from, To: &contract, Data: data}, nil)
	if err == nil {
		return nil
	}
	if errors.Is(err, chain.ErrUnavailable) || errors.Is(err, chain.ErrTimeout) || errors.Is(err, chain.ErrNetwork) {
		return err
	}
	return errors.New("託管交易預先檢查未通過，請更新訂單、餘額與授權後重試；交易尚未送出")
}
