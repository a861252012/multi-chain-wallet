package wallet

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// Addresses are the Ethereum Sepolia deployments published by Uniswap and Circle.
const WETHAddress = chain.SepoliaWETH
const USDCAddress = chain.SepoliaUSDC
const RouterAddress = "0x3bFA4769FB09eefC5a80d6E87c3B9C650f7Ae48E"
const QuoterAddress = "0xEd1f6473345F45b75F8179591dd5bA1888cf2FB3"
const FactoryAddress = "0x0227628f3F023bb0B980b67D528571c95c6DaC1c"

var exchangeABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(`[
 {"type":"function","name":"deposit","inputs":[],"outputs":[],"stateMutability":"payable"},
 {"type":"function","name":"withdraw","inputs":[{"name":"wad","type":"uint256"}],"outputs":[],"stateMutability":"nonpayable"},
 {"type":"function","name":"getPool","inputs":[{"name":"a","type":"address"},{"name":"b","type":"address"},{"name":"fee","type":"uint24"}],"outputs":[{"name":"pool","type":"address"}],"stateMutability":"view"},
 {"type":"function","name":"quoteExactInputSingle","inputs":[{"name":"params","type":"tuple","components":[{"name":"tokenIn","type":"address"},{"name":"tokenOut","type":"address"},{"name":"amountIn","type":"uint256"},{"name":"fee","type":"uint24"},{"name":"sqrtPriceLimitX96","type":"uint160"}]}],"outputs":[{"name":"amountOut","type":"uint256"},{"name":"sqrtPriceX96After","type":"uint160"},{"name":"initializedTicksCrossed","type":"uint32"},{"name":"gasEstimate","type":"uint256"}],"stateMutability":"nonpayable"},
 {"type":"function","name":"exactInputSingle","inputs":[{"name":"params","type":"tuple","components":[{"name":"tokenIn","type":"address"},{"name":"tokenOut","type":"address"},{"name":"fee","type":"uint24"},{"name":"recipient","type":"address"},{"name":"amountIn","type":"uint256"},{"name":"amountOutMinimum","type":"uint256"},{"name":"sqrtPriceLimitX96","type":"uint160"}]}],"outputs":[{"name":"amountOut","type":"uint256"}],"stateMutability":"payable"},
 {"type":"function","name":"multicall","inputs":[{"name":"deadline","type":"uint256"},{"name":"data","type":"bytes[]"}],"outputs":[{"name":"results","type":"bytes[]"}],"stateMutability":"payable"}
 ]`))
	if err != nil {
		panic(err)
	}
	return a
}()

type ExchangePreview struct {
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

type exchangePayload struct {
	TxTo     common.Address
	Contract common.Address
	Value    *big.Int
	Data     []byte
	Method   string
	Symbol   string
	Decimals int
	Preview  *ExchangePreview
}

type swapParams struct {
	TokenIn           common.Address
	TokenOut          common.Address
	Fee               *big.Int
	Recipient         common.Address
	AmountIn          *big.Int
	AmountOutMinimum  *big.Int
	SqrtPriceLimitX96 *big.Int
}

type quoterParams struct {
	TokenIn           common.Address
	TokenOut          common.Address
	AmountIn          *big.Int
	Fee               *big.Int
	SqrtPriceLimitX96 *big.Int
}

func exchangeToken(address common.Address) (string, int, error) {
	switch address {
	case common.HexToAddress(WETHAddress):
		return "WETH", 18, nil
	case common.HexToAddress(USDCAddress):
		return "USDC", 6, nil
	default:
		return "", 0, errors.New("兌換只支援 Sepolia 官方 WETH 與 Circle 測試 USDC")
	}
}

// PrepareExchange builds only allowlisted calls; the client never supplies arbitrary calldata or a router.
func PrepareExchange(ctx context.Context, caller ChainCaller, owner common.Address, command QuoteCommand) (*exchangePayload, error) {
	weth := common.HexToAddress(WETHAddress)
	p := &exchangePayload{TxTo: weth, Contract: weth, Value: big.NewInt(0), Symbol: "WETH", Decimals: 18}
	preview := &ExchangePreview{Deadline: time.Now().UTC().Add(120 * time.Second).Format(time.RFC3339)}
	p.Preview = preview
	if command.Action == ActionWrap || command.Action == ActionUnwrap {
		if command.Contract != "" && !strings.EqualFold(string(command.Contract), WETHAddress) {
			return nil, errors.New("包裝與解包只能使用指定的 Sepolia WETH")
		}
		if command.TokenOut != "" || command.PoolFee != 0 || command.SlippageBPS != 0 {
			return nil, errors.New("包裝與解包不接受交易池或滑價參數")
		}
		amount, err := ParseUnits(command.Amount, 18)
		if err != nil {
			return nil, err
		}
		if amount.Sign() <= 0 {
			return nil, errors.New("兌換數量必須大於 0")
		}
		if err := VerifyContractBytecode(ctx, caller, weth); err != nil {
			return nil, err
		}
		preview.TokenIn, preview.TokenOut, preview.SymbolOut = "ETH", weth.Hex(), "WETH"
		if command.Action == ActionWrap {
			p.Symbol, p.Method, p.Value = "ETH", "deposit", amount
			p.Data, err = exchangeABI.Pack("deposit")
		} else {
			p.Method = "withdraw"
			p.Data, err = exchangeABI.Pack("withdraw", amount)
			preview.TokenIn, preview.TokenOut, preview.SymbolOut = weth.Hex(), "ETH", "ETH"
			balance, err := QueryERC20BalanceOf(ctx, caller, weth, owner)
			if err != nil {
				return nil, err
			}
			if balance.Cmp(amount) < 0 {
				return nil, errors.New("WETH 餘額不足")
			}
		}
		if err != nil {
			return nil, err
		}
		preview.ExpectedOut, preview.MinimumOut, preview.MinimumOutRaw = FormatUnits(amount, 18), FormatUnits(amount, 18), amount.String()
	} else if command.Action == ActionSwap {
		in := common.HexToAddress(string(command.Contract))
		out := common.HexToAddress(string(command.TokenOut))
		if in == out {
			return nil, errors.New("兌換的兩種資產不可相同")
		}
		symbol, decimals, err := exchangeToken(in)
		if err != nil {
			return nil, err
		}
		p.Symbol, p.Decimals = symbol, decimals
		outSymbol, outDecimals, err := exchangeToken(out)
		if err != nil {
			return nil, err
		}
		if command.SlippageBPS < 1 || command.SlippageBPS > 500 {
			return nil, errors.New("滑價必須介於 0.01% 至 5%")
		}
		if command.PoolFee != 100 && command.PoolFee != 500 && command.PoolFee != 3000 && command.PoolFee != 10000 {
			return nil, errors.New("請選擇有效的 Uniswap V3 費率")
		}
		amount, err := ParseUnits(command.Amount, p.Decimals)
		if err != nil {
			return nil, err
		}
		if amount.Sign() <= 0 {
			return nil, errors.New("兌換數量必須大於 0")
		}
		router, quoter, factory := common.HexToAddress(RouterAddress), common.HexToAddress(QuoterAddress), common.HexToAddress(FactoryAddress)
		for _, address := range []common.Address{in, out, router, quoter, factory} {
			if err := VerifyContractBytecode(ctx, caller, address); err != nil {
				return nil, err
			}
		}
		balance, err := QueryERC20BalanceOf(ctx, caller, in, owner)
		if err != nil {
			return nil, err
		}
		if balance.Cmp(amount) < 0 {
			return nil, errors.New("兌換來源代幣餘額不足")
		}
		allowance, err := QueryERC20Allowance(ctx, caller, in, owner, router)
		if err != nil {
			return nil, err
		}
		if allowance.Cmp(amount) < 0 {
			return nil, errors.New("請先授權 Router 本次兌換所需數量，等待授權交易成功，再重新報價")
		}
		fee := big.NewInt(int64(command.PoolFee))
		expected, pool, err := quotePool(ctx, caller, in, out, amount, command.PoolFee)
		if err != nil {
			return nil, err
		}

		minimum := new(big.Int).Div(new(big.Int).Mul(expected, big.NewInt(int64(10000-command.SlippageBPS))), big.NewInt(10000))
		if minimum.Sign() <= 0 {
			return nil, errors.New("可收到的數量太小或交易池沒有足夠流動性")
		}
		callData, err := exchangeABI.Pack("exactInputSingle", swapParams{in, out, fee, owner, amount, minimum, big.NewInt(0)})
		if err != nil {
			return nil, err
		}
		deadline, _ := time.Parse(time.RFC3339, preview.Deadline)
		p.Data, err = exchangeABI.Pack("multicall", big.NewInt(deadline.Unix()), [][]byte{callData})
		if err != nil {
			return nil, err
		}
		p.TxTo, p.Contract, p.Method = router, in, "multicall(deadline, exactInputSingle)"
		preview.TokenIn, preview.TokenOut, preview.SymbolOut = in.Hex(), out.Hex(), outSymbol
		preview.ExpectedOut, preview.MinimumOut, preview.MinimumOutRaw = FormatUnits(expected, outDecimals), FormatUnits(minimum, outDecimals), minimum.String()
		preview.Router, preview.Pool, preview.PoolFee, preview.SlippageBPS = router.Hex(), pool.Hex(), command.PoolFee, command.SlippageBPS
	} else {
		return nil, errors.New("未知的兌換操作")
	}
	if err := simulateExchange(ctx, caller, owner, p.TxTo, p.Value, p.Data, string(command.Action), preview); err != nil {
		return nil, err
	}
	return p, nil
}

func simulateExchange(ctx context.Context, caller ChainCaller, from, to common.Address, value *big.Int, data []byte, action string, preview *ExchangePreview) error {
	result, err := caller.CallContract(ctx, ethereum.CallMsg{From: from, To: &to, Value: value, Data: data}, nil)
	if err != nil {
		return ErrSimulationFailed
	}
	if action != "swap" {
		if len(result) != 0 {
			return ErrSimulationFailed
		}
		return nil
	}
	values, err := exchangeABI.Unpack("multicall", result)
	if err != nil || len(values) != 1 {
		return ErrSimulationFailed
	}
	results, ok := values[0].([][]byte)
	if !ok || len(results) != 1 || len(results[0]) != 32 {
		return ErrSimulationFailed
	}
	min, ok := new(big.Int).SetString(preview.MinimumOutRaw, 10)
	if !ok || new(big.Int).SetBytes(results[0]).Cmp(min) < 0 {
		return ErrSimulationFailed
	}
	return nil
}

func RecheckExchange(ctx context.Context, caller ChainCaller, q *BoundQuote) error {
	if q.Exchange == nil || q.To != q.From {
		return errors.New("兌換報價資料不完整")
	}
	if q.Action == "unwrap" || q.Action == "swap" {
		balance, err := QueryERC20BalanceOf(ctx, caller, q.Contract, q.From)
		if err != nil {
			return err
		}
		if balance.Cmp(q.AmountRaw) < 0 {
			return errors.New("兌換來源代幣餘額已不足")
		}
	}
	if q.Action == "swap" {
		allowance, err := QueryERC20Allowance(ctx, caller, q.Contract, q.From, q.TxTo)
		if err != nil {
			return err
		}
		if allowance.Cmp(q.AmountRaw) < 0 {
			return errors.New("兌換所需授權額度已不足")
		}
	}
	return simulateExchange(ctx, caller, q.From, q.TxTo, q.TxValue, q.Data, string(q.Action), q.Exchange)
}

func quotePool(ctx context.Context, caller ChainCaller, in, out common.Address, amount *big.Int, poolFee int) (*big.Int, common.Address, error) {
	factory, quoter := common.HexToAddress(FactoryAddress), common.HexToAddress(QuoterAddress)
	poolData, err := exchangeABI.Pack("getPool", in, out, big.NewInt(int64(poolFee)))
	if err != nil {
		return nil, common.Address{}, err
	}
	poolRaw, err := caller.CallContract(ctx, ethereum.CallMsg{To: &factory, Data: poolData}, nil)
	if err != nil {
		return nil, common.Address{}, err
	}
	poolValues, err := exchangeABI.Unpack("getPool", poolRaw)
	if err != nil || len(poolValues) != 1 {
		return nil, common.Address{}, errors.New("無法解析交易池地址")
	}
	pool := poolValues[0].(common.Address)
	if pool == (common.Address{}) {
		return nil, common.Address{}, errors.New("此費率沒有 WETH/USDC 交易池，請選擇其他費率")
	}
	quoteData, err := exchangeABI.Pack("quoteExactInputSingle", quoterParams{in, out, amount, big.NewInt(int64(poolFee)), big.NewInt(0)})
	if err != nil {
		return nil, common.Address{}, err
	}
	quoteRaw, err := caller.CallContract(ctx, ethereum.CallMsg{To: &quoter, Data: quoteData}, nil)
	if err != nil {
		return nil, common.Address{}, errors.New("鏈上報價失敗；RPC 或此交易池的流動性目前無法完成兌換")
	}
	values, err := exchangeABI.Unpack("quoteExactInputSingle", quoteRaw)
	if err != nil || len(values) != 4 {
		return nil, common.Address{}, errors.New("兌換報價格式錯誤")
	}
	expected := values[0].(*big.Int)
	return expected, pool, nil
}
