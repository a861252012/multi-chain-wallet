package wallet

import (
	"errors"
	"math/big"
	"strings"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/ethereum/go-ethereum/common"
)

type activityState string

const (
	activityPending            activityState = "pending"
	activityReceiptUnavailable activityState = "receipt_unavailable"
	activityReorgDetected      activityState = "reorg_detected"
	activitySucceeded          activityState = "succeeded"
	activityReverted           activityState = "reverted"
	activityUnverified         activityState = "unverified"
)

type activityMovementKind string

const (
	activityReceive activityMovementKind = "receive"
	activitySend    activityMovementKind = "send"
	activityFee     activityMovementKind = "fee"
)

type activityMovement struct {
	kind         activityMovementKind
	asset        string
	amount       *big.Int
	counterparty common.Address
}

type walletActivity struct {
	hash      TransactionHash
	state     activityState
	movements []activityMovement
}

func chainActivityToDomain(value *chain.Activity) (walletActivity, error) {
	if value == nil {
		return walletActivity{}, errors.New("收支資料不存在")
	}
	hash, err := ParseTransactionHash(value.Hash)
	if err != nil {
		return walletActivity{}, errors.New("收支交易雜湊格式錯誤")
	}
	state := activityState(value.State)
	switch state {
	case activityPending, activityReceiptUnavailable, activityReorgDetected, activitySucceeded, activityReverted, activityUnverified:
	default:
		return walletActivity{}, errors.New("收支交易狀態格式錯誤")
	}
	result := walletActivity{hash: hash, state: state, movements: make([]activityMovement, len(value.Movements))}
	for i, movement := range value.Movements {
		kind := activityMovementKind(movement.Kind)
		switch kind {
		case activityReceive, activitySend, activityFee:
		default:
			return walletActivity{}, errors.New("收支類型格式錯誤")
		}
		if movement.Asset == "" || (strings.HasPrefix(movement.Asset, "0x") && !common.IsHexAddress(movement.Asset)) {
			return walletActivity{}, errors.New("收支資產格式錯誤")
		}
		amount, ok := new(big.Int).SetString(movement.Raw, 10)
		if !ok || amount.Sign() < 0 {
			return walletActivity{}, errors.New("收支金額格式錯誤")
		}
		if !common.IsHexAddress(movement.Counterparty) {
			return walletActivity{}, errors.New("收支對手地址格式錯誤")
		}
		result.movements[i] = activityMovement{
			kind: kind, asset: movement.Asset, amount: amount,
			counterparty: common.HexToAddress(movement.Counterparty),
		}
	}
	return result, nil
}
