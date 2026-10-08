package wallet

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math/big"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// replacementQuote preserves the original intent for speed-up, or replaces it with a zero-value self-transfer.
func (s *Service) replacementQuote(ctx context.Context, command QuoteCommand) (*BoundQuote, error) {
	record := s.journal.FindByHash(string(command.Hash))
	if record == nil {
		return nil, chain.ErrNotFound
	}
	if s.journal.NonceMined(record.Nonce) {
		return nil, errors.New("此 Nonce 已有收錄交易，請先更新紀錄")
	}
	raw, err := hexutil.Decode(record.SignedRaw)
	if err != nil {
		return nil, err
	}
	var tx types.Transaction
	if tx.UnmarshalBinary(raw) != nil || tx.Hash().Hex() != string(record.Hash) || tx.ChainId().Cmp(big.NewInt(s.client.ChainID())) != 0 || tx.Nonce() != record.Nonce || tx.Type() != types.DynamicFeeTxType || len(tx.AccessList()) != 0 || tx.To() == nil {
		return nil, errors.New("無法替代這筆交易")
	}
	address, err := s.keystore.Address()
	if err != nil {
		return nil, err
	}
	from := common.HexToAddress(address)
	sender, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), &tx)
	if err != nil || sender != from {
		return nil, errors.New("原交易簽名地址不符")
	}
	if err := restoreEscrowJournal(record, &tx); err != nil {
		return nil, err
	}
	nonce, err := s.client.NonceAt(ctx, from)
	if err != nil {
		return nil, err
	}
	if nonce > tx.Nonce() {
		return nil, ErrNonceMismatch
	}
	head, err := s.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}
	if head.BaseFee == nil {
		return nil, errors.New("無法取得基本費用")
	}
	tip, err := s.client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, err
	}
	fee := new(big.Int)
	for _, item := range s.journal.NonceRecords(tx.Nonce()) {
		b, err := hexutil.Decode(item.SignedRaw)
		var prior types.Transaction
		if err != nil || prior.UnmarshalBinary(b) != nil {
			return nil, errors.New("交易日誌格式錯誤")
		}
		// A 20% local policy, not an inclusion guarantee or a protocol requirement.
		bumpTip := new(big.Int).Div(new(big.Int).Add(new(big.Int).Mul(prior.GasTipCap(), big.NewInt(120)), big.NewInt(99)), big.NewInt(100))
		bumpFee := new(big.Int).Div(new(big.Int).Add(new(big.Int).Mul(prior.GasFeeCap(), big.NewInt(120)), big.NewInt(99)), big.NewInt(100))
		if bumpTip.Sign() == 0 {
			bumpTip.SetInt64(1)
		}
		if tip.Cmp(bumpTip) < 0 {
			tip = bumpTip
		}
		if fee.Cmp(bumpFee) < 0 {
			fee = bumpFee
		}
	}
	suggested := new(big.Int).Add(new(big.Int).Mul(head.BaseFee, big.NewInt(2)), tip)
	if fee.Cmp(suggested) < 0 {
		fee = suggested
	}
	to, value, data := *tx.To(), tx.Value(), tx.Data()
	recipient, amount, symbol := record.To, record.Amount, record.Symbol
	amountRaw, ok := new(big.Int).SetString(record.AmountRaw, 10)
	if !ok {
		return nil, errors.New("原交易金額格式錯誤")
	}
	if command.Action == ActionCancel {
		to, value, data = from, big.NewInt(0), nil
		recipient, amount, symbol, amountRaw = EVMAddress(from.Hex()), "0", s.client.NativeSymbol(), big.NewInt(0)
	}
	decimals := 18
	replacementERC20 := false
	var escrow *EscrowPreview
	if command.Action == ActionSpeedup {
		if record.EscrowAction != "" {
			payload, err := s.prepareEscrowSpeedup(ctx, from, record, &tx)
			if err != nil {
				return nil, err
			}
			escrow = payload.Preview
			recipient, amountRaw = EVMAddress(payload.To.Hex()), payload.Amount
			amount, symbol, decimals = FormatUnits(payload.Amount, 6), "USDC", 6
		} else if _, decodeErr := DecodeERC20Calldata(data, 18); decodeErr == nil {
			symbol, decimals, err = QueryERC20Metadata(ctx, s.client, to)
			if err != nil {
				return nil, err
			}
			decoded, err := DecodeERC20Calldata(data, decimals)
			if err != nil {
				return nil, err
			}
			recipient, amount, amountRaw = EVMAddress(decoded.Target.Hex()), decoded.FormattedAmount, decoded.RawAmount
			replacementERC20 = true
			if err := SimulateERC20Call(ctx, s.client, from, to, data); err != nil {
				return nil, err
			}
		}
	}
	gas, err := s.client.EstimateGas(ctx, ethereum.CallMsg{From: from, To: &to, Value: value, Data: data, GasFeeCap: fee, GasTipCap: tip})
	if err != nil {
		return nil, err
	}
	if gas < 21000 || gas > head.GasLimit {
		return nil, errors.New("Gas 預估值無效")
	}
	gas += gas / 5
	if gas > head.GasLimit {
		gas = head.GasLimit
	}
	cost := new(big.Int).Mul(new(big.Int).SetUint64(gas), fee)
	total := new(big.Int).Add(value, cost)
	balance, err := s.client.BalanceAt(ctx, from, nil)
	if err != nil {
		return nil, err
	}
	if balance.Cmp(total) < 0 {
		return nil, ErrInsufficientFunds
	}
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return nil, err
	}
	contract := common.Address{}
	if command.Action == ActionSpeedup && len(data) > 0 {
		contract = to
	}
	return &BoundQuote{Escrow: escrow, ReplacementERC20: replacementERC20, Decimals: decimals, Contract: contract, ID: QuoteID(hex.EncodeToString(id)), Action: command.Action, ReplacementHash: record.Hash, ReplacementCount: len(s.journal.NonceRecords(tx.Nonce())), From: from, To: common.HexToAddress(string(recipient)), TxTo: to, TxValue: value, Amount: amount, AmountRaw: amountRaw, Symbol: symbol, Nonce: tx.Nonce(), Data: data, GasLimit: gas, MaxFeePerGas: fee, MaxPriorityFeePerGas: tip, TotalETHWei: total, TotalETH: FormatUnits(total, 18), MaxFeeETH: FormatUnits(cost, 18), Method: string(command.Action), CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().Add(120 * time.Second)}, nil
}
