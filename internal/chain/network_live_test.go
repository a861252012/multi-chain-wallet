package chain

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestAdditionalNetworksReadOnly(t *testing.T) {
	if os.Getenv("MULTI_CHAIN_WALLET_LIVE_NETWORKS") != "1" {
		t.Skip("opt-in read-only network verification")
	}
	for _, network := range []struct {
		id  int64
		url string
	}{{80002, "https://polygon-amoy.drpc.org"}, {84532, "https://sepolia.base.org"}, {11155420, "https://sepolia.optimism.io"}} {
		t.Run(big.NewInt(network.id).String(), func(t *testing.T) {
			c, err := NewNetwork(network.id, []string{network.url})
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			defer cancel()
			state, err := c.Network(ctx)
			if err != nil {
				t.Fatal(err)
			}
			address := common.HexToAddress("0x219465ECA5EB591b7587E4B1Aa97F34971206Ab9")
			balance, err := c.Balance(ctx, address.Hex())
			if err != nil {
				t.Fatal(err)
			}
			tx := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(network.id), To: &address, Gas: 21000, GasFeeCap: big.NewInt(1000000000), GasTipCap: big.NewInt(1000000), Value: big.NewInt(1000)})
			fee, err := c.RollupFee(ctx, tx)
			if err != nil {
				t.Fatal(err)
			}
			final, err := c.FinalizedNumber(ctx)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("chain=%d block=%s finalized=%d balanceWei=%s rollupFeeEstimateWei=%s; no signing or broadcast", network.id, state.Block, final, balance.Wei, fee)
		})
	}
}
