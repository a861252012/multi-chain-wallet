package wallet

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/ethereum/go-ethereum/common"
)

type PoolQuote struct {
	Fee       int    `json:"fee"`
	Pool      string `json:"pool,omitempty"`
	Output    string `json:"output,omitempty"`
	OutputRaw string `json:"outputRaw,omitempty"`
	Error     string `json:"error,omitempty"`
}
type PoolComparison struct {
	Pools     []PoolQuote `json:"pools"`
	BestFee   int         `json:"bestFee"`
	Symbol    string      `json:"symbol"`
	CheckedAt time.Time   `json:"checkedAt"`
}

func (s *Service) ComparePools(ctx context.Context, req *QuoteRequest) (*PoolComparison, error) {
	command, err := ParsePoolComparisonRequest(req)
	if err != nil {
		return nil, err
	}
	return s.ComparePoolsCommand(ctx, command)
}

func (s *Service) ComparePoolsCommand(ctx context.Context, command PoolComparisonCommand) (*PoolComparison, error) {
	if s.client.ChainID() != chain.SepoliaID {
		return nil, errors.New("此網路尚未配置兌換合約")
	}
	in := common.HexToAddress(string(command.Contract))
	out := common.HexToAddress(string(command.TokenOut))
	if in == out {
		return nil, errors.New("兌換資產不可相同")
	}
	_, decimals, err := exchangeToken(in)
	if err != nil {
		return nil, err
	}
	symbol, outDecimals, err := exchangeToken(out)
	if err != nil {
		return nil, err
	}
	amount, err := ParseUnits(command.Amount, decimals)
	if err != nil {
		return nil, err
	}
	if amount.Sign() <= 0 {
		return nil, errors.New("兌換數量必須大於 0")
	}
	for _, address := range []common.Address{in, out, common.HexToAddress(FactoryAddress), common.HexToAddress(QuoterAddress)} {
		if err := VerifyContractBytecode(ctx, s.client, address); err != nil {
			return nil, err
		}
	}
	result := &PoolComparison{Pools: make([]PoolQuote, 4), Symbol: symbol, CheckedAt: time.Now().UTC()}
	var group sync.WaitGroup
	for i, fee := range []int{100, 500, 3000, 10000} {
		group.Add(1)
		go func(idx, feeTier int) {
			defer group.Done()
			row := PoolQuote{Fee: feeTier}
			expected, pool, err := quotePool(ctx, s.client, in, out, amount, feeTier)
			if err != nil {
				row.Error = err.Error()
			} else if expected.Sign() <= 0 {
				row.Error = "此池無可用報價"
			} else {
				row.Pool = pool.Hex()
				row.Output = FormatUnits(expected, outDecimals)
				row.OutputRaw = expected.String()
			}
			result.Pools[idx] = row
		}(i, fee)
	}
	group.Wait()
	best := new(big.Int)
	for _, pool := range result.Pools {
		if value, ok := new(big.Int).SetString(pool.OutputRaw, 10); ok && value.Cmp(best) > 0 {
			best = value
			result.BestFee = pool.Fee
		}
	}
	return result, nil
}
