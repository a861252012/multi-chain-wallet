package chain

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

func TestSepoliaActivityReadOnly(t *testing.T) {
	endpoint := os.Getenv("MULTI_CHAIN_WALLET_LIVE_RPC")
	if endpoint == "" {
		t.Skip("opt-in read-only Sepolia check")
	}
	c, err := New(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := c.CheckNetwork(ctx); err != nil {
		t.Fatal(err)
	}
	head, err := c.rpc.HeaderByNumber(ctx, nil)
	if err != nil {
		t.Fatal("cannot obtain header")
	}
	from := new(big.Int).Sub(head.Number, big.NewInt(999))
	if from.Sign() < 0 {
		from = big.NewInt(0)
	}
	logs, err := c.rpc.FilterLogs(ctx, ethereum.FilterQuery{FromBlock: from, ToBlock: head.Number, Addresses: []common.Address{common.HexToAddress(SepoliaWETH)}, Topics: [][]common.Hash{{depositTopic, withdrawalTopic}}})
	if err != nil {
		t.Fatal("cannot query WETH events")
	}
	if len(logs) == 0 {
		t.Skip("no recent WETH event available for read-only receipt sample")
	}
	log := logs[len(logs)-1]
	if len(log.Topics) != 2 {
		t.Fatal("unexpected WETH topics")
	}
	owner := common.BytesToAddress(log.Topics[1].Bytes())
	result, err := c.Activity(ctx, log.TxHash.Hex(), owner)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "succeeded" || len(result.Movements) == 0 {
		t.Fatal("canonical WETH activity unavailable")
	}
	t.Logf("public receipt sample %s: %s, %d evidence movements; no transaction sent", result.Hash, result.State, len(result.Movements))
	// Verify that the scanner can decode current Sepolia full blocks. The address is a public fixture, not a user wallet.
	ids, start, end, err := c.DiscoverActivity(ctx, common.HexToAddress("0x1111111111111111111111111111111111111111"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if end-start > 19 {
		t.Fatal("scan exceeded bound")
	}
	t.Logf("read-only scan %d..%d: %d related hashes", start, end, len(ids))
}
