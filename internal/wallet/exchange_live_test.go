package wallet

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

// Opt-in read-only integration check. No key, wallet, signature or broadcast is used.
func TestSepoliaExchangeReadOnly(t *testing.T) {
	endpoint := os.Getenv("MULTI_CHAIN_WALLET_LIVE_RPC")
	if endpoint == "" {
		t.Skip("set MULTI_CHAIN_WALLET_LIVE_RPC to run read-only Sepolia contract checks")
	}
	c, err := chain.New(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := c.CheckNetwork(ctx); err != nil {
		t.Fatal(err)
	}
	for _, address := range []string{WETHAddress, USDCAddress, RouterAddress, QuoterAddress, FactoryAddress} {
		if err := VerifyContractBytecode(ctx, c, common.HexToAddress(address)); err != nil {
			t.Fatalf("contract %s: %v", address, err)
		}
	}
	for _, address := range []string{WETHAddress, USDCAddress} {
		actualSymbol, actualDecimals, err := QueryERC20Metadata(ctx, c, common.HexToAddress(address))
		if err != nil {
			t.Fatal(err)
		}
		symbol, decimals, _ := exchangeToken(common.HexToAddress(address))
		if actualSymbol != symbol || actualDecimals != decimals {
			t.Fatalf("unexpected metadata for %s", address)
		}
	}
	factory, quoter := common.HexToAddress(FactoryAddress), common.HexToAddress(QuoterAddress)
	found := 0
	for _, fee := range []int64{100, 500, 3000, 10000} {
		data, _ := exchangeABI.Pack("getPool", common.HexToAddress(WETHAddress), common.HexToAddress(USDCAddress), big.NewInt(fee))
		raw, err := c.CallContract(ctx, ethereum.CallMsg{To: &factory, Data: data}, nil)
		if err != nil {
			t.Fatal(err)
		}
		values, err := exchangeABI.Unpack("getPool", raw)
		if err != nil {
			t.Fatal(err)
		}
		pool := values[0].(common.Address)
		if pool == (common.Address{}) {
			t.Logf("fee %d: no pool", fee)
			continue
		}
		data, _ = exchangeABI.Pack("quoteExactInputSingle", quoterParams{common.HexToAddress(WETHAddress), common.HexToAddress(USDCAddress), big.NewInt(1000000000000), big.NewInt(fee), big.NewInt(0)})
		raw, err = c.CallContract(ctx, ethereum.CallMsg{To: &quoter, Data: data}, nil)
		if err != nil {
			t.Logf("fee %d pool %s: quote unavailable", fee, pool.Hex())
			continue
		}
		quoted, err := exchangeABI.Unpack("quoteExactInputSingle", raw)
		if err != nil {
			t.Fatal(err)
		}
		amount := quoted[0].(*big.Int)
		if amount.Sign() > 0 {
			found += 1
			t.Logf("fee %d pool %s: 0.000001 WETH -> %s test USDC (read-only quote)", fee, pool.Hex(), FormatUnits(amount, 6))
		}
	}
	if found == 0 {
		t.Fatal("no live nonzero WETH/USDC quote available; this is not a broadcast test")
	}
}
