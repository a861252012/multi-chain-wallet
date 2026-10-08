package wallet

import (
	"context"
	"errors"
	"math/big"
	"strings"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

const ethVaultABIJSON = `[
	{"inputs":[],"name":"deposit","outputs":[],"stateMutability":"payable","type":"function"},
	{"inputs":[{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"withdraw","outputs":[],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[{"internalType":"address","name":"account","type":"address"}],"name":"balanceOf","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}
]`

var ethVaultABI = func() abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(ethVaultABIJSON))
	if err != nil {
		panic("failed to parse ethVaultABI: " + err.Error())
	}
	return parsed
}()

// VaultInfo describes the current vault configuration and balance for an account.
type VaultInfo struct {
	Enabled    bool   `json:"enabled"`
	Contract   string `json:"contract"`
	Balance    string `json:"balance"`
	BalanceRaw string `json:"balanceRaw"`
}

// VaultAddress returns the configured vault address for the service.
func (s *Service) VaultAddress() string {
	if s.client.ChainID() != chain.SepoliaID {
		return ""
	}
	return s.vaultAddress
}

// SetVaultAddress configures Sepolia before the service begins serving requests.
func (s *Service) SetVaultAddress(address string) error {
	if s.client.ChainID() != chain.SepoliaID {
		s.vaultAddress = ""
		return nil
	}
	address = strings.TrimSpace(address)
	if address == "" {
		s.vaultAddress = ""
		return nil
	}
	validated, err := ValidateAddress(address)
	if err != nil {
		return err
	}
	s.vaultAddress = validated.Hex()
	return nil
}

// VaultStatus returns the vault status and balance for the active wallet address.
func (s *Service) VaultStatus(ctx context.Context) (*VaultInfo, error) {
	if s.client.ChainID() != chain.SepoliaID || s.vaultAddress == "" {
		return &VaultInfo{}, nil
	}

	addrStr, err := s.keystore.Address()
	if err != nil {
		return nil, err
	}
	account := common.HexToAddress(addrStr)
	contract := common.HexToAddress(s.vaultAddress)

	rawBal, err := QueryVaultBalanceOf(ctx, s.client, contract, account)
	if err != nil {
		return nil, err
	}

	return &VaultInfo{
		Enabled:    true,
		Contract:   s.vaultAddress,
		Balance:    FormatUnits(rawBal, 18),
		BalanceRaw: rawBal.String(),
	}, nil
}

// QueryVaultBalanceOf queries the balanceOf(account) for the vault contract.
func QueryVaultBalanceOf(ctx context.Context, caller ChainCaller, contract, account common.Address) (*big.Int, error) {
	if err := VerifyContractBytecode(ctx, caller, contract); err != nil {
		return nil, err
	}
	callData, err := ethVaultABI.Pack("balanceOf", account)
	if err != nil {
		return nil, err
	}
	res, err := caller.CallContract(ctx, ethereum.CallMsg{To: &contract, Data: callData}, nil)
	if err != nil {
		return nil, err
	}
	var balance *big.Int
	if err := ethVaultABI.UnpackIntoInterface(&balance, "balanceOf", res); err != nil {
		return nil, errors.New("無法讀取合約回傳的餘額")
	}
	return balance, nil
}

func decodeVaultError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, chain.ErrTimeout) || errors.Is(err, chain.ErrUnavailable) || errors.Is(err, chain.ErrNetwork) {
		return err
	}
	var rawData string
	var dataErr rpc.DataError
	if errors.As(err, &dataErr) {
		switch v := dataErr.ErrorData().(type) {
		case string:
			rawData = v
		case []byte:
			rawData = hexutil.Encode(v)
		case map[string]any:
			rawData, _ = v["data"].(string)
		}
	}
	if rawData == "" {
		rawData = strings.TrimPrefix(err.Error(), "execution reverted: ")
	}
	data, decodeErr := hexutil.Decode(rawData)
	if decodeErr != nil || len(data) < 4 {
		return errors.New("合約預先檢查未通過，交易尚未送出")
	}
	switch hexutil.Encode(data[:4]) {
	case "0x56316e87":
		return errors.New("存入金額必須大於 0")
	case "0xb8cb6219":
		return errors.New("取回金額必須大於 0")
	case "0xcf479181":
		return errors.New("合約餘額不足，請減少取回金額")
	case "0x37ed32e8":
		return errors.New("拒絕重入呼叫")
	case "0x90b8ec18":
		return errors.New("合約轉帳失敗")
	default:
		return errors.New("合約預先檢查未通過，交易尚未送出")
	}
}

// SimulateVaultCall checks a call against the current state; inclusion can still fail.
func SimulateVaultCall(ctx context.Context, caller ChainCaller, from, to common.Address, value *big.Int, data []byte) error {
	msg := ethereum.CallMsg{
		From:  from,
		To:    &to,
		Value: value,
		Data:  data,
	}
	_, err := caller.CallContract(ctx, msg, nil)
	if err != nil {
		return decodeVaultError(err)
	}
	return nil
}
