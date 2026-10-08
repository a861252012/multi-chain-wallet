package e2e

import (
	"context"
	"encoding/json"
	"math/big"
	"sync"
	"testing"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

func TestSimulatedRPCBlockTags(t *testing.T) {
	ctx := context.Background()
	sim := simulated.NewBackend(types.GenesisAlloc{})
	defer sim.Close()
	sim.Commit()
	sim.Commit()
	var mu sync.Mutex
	server := startSimulatedRPC(t, sim, &mu)
	client, err := rpc.DialContext(ctx, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	for _, tc := range []struct {
		tag    string
		number rpc.BlockNumber
	}{
		{"latest", rpc.LatestBlockNumber}, {"pending", rpc.PendingBlockNumber},
		{"earliest", rpc.EarliestBlockNumber}, {"safe", rpc.SafeBlockNumber},
		{"finalized", rpc.FinalizedBlockNumber}, {"0x1", 1},
	} {
		t.Run(tc.tag, func(t *testing.T) {
			expected, err := sim.Client().HeaderByNumber(ctx, big.NewInt(tc.number.Int64()))
			if err != nil {
				t.Fatal(err)
			}
			var actual *types.Header
			if err := client.CallContext(ctx, &actual, "eth_getBlockByNumber", tc.tag, false); err != nil {
				t.Fatal(err)
			}
			if actual == nil || actual.Hash() != expected.Hash() {
				t.Fatalf("tag %s did not return backend header", tc.tag)
			}
		})
	}
	for _, tc := range []struct {
		name string
		args []any
	}{
		{"unknown tag", []any{"unknown", false}}, {"malformed hex", []any{"0xgg", false}},
		{"leading zero", []any{"0x01", false}}, {"overflow", []any{"0x8000000000000000", false}},
		{"null tag", []any{nil, false}}, {"number tag", []any{1, false}},
		{"object tag", []any{map[string]string{"blockNumber": "0x1"}, false}},
		{"missing params", nil}, {"missing flag", []any{"latest"}},
		{"null flag", []any{"latest", nil}}, {"string flag", []any{"latest", "false"}},
		{"extra param", []any{"latest", false, false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var actual json.RawMessage
			if err := client.CallContext(ctx, &actual, "eth_getBlockByNumber", tc.args...); err == nil {
				t.Fatalf("invalid params returned %s", actual)
			}
		})
	}
}

func TestSimulatedRPCReportsActualFinality(t *testing.T) {
	ctx := context.Background()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	address := crypto.PubkeyToAddress(key.PublicKey)
	sim := simulated.NewBackend(types.GenesisAlloc{
		address: {Balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil)},
	}, func(_ *node.Config, config *ethconfig.Config) {
		config.Genesis.Config.ChainID = big.NewInt(chain.SepoliaID)
	})
	defer sim.Close()
	tx, err := types.SignTx(types.NewTransaction(0, address, big.NewInt(1), 21000, big.NewInt(2_000_000_000), nil), types.LatestSignerForChainID(big.NewInt(chain.SepoliaID)), key)
	if err != nil {
		t.Fatal(err)
	}
	if err := sim.Client().SendTransaction(ctx, tx); err != nil {
		t.Fatal(err)
	}
	sim.Commit()
	receipt, err := sim.Client().TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		t.Fatal(err)
	}
	final, err := sim.Client().HeaderByNumber(ctx, big.NewInt(rpc.FinalizedBlockNumber.Int64()))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.BlockNumber.Uint64() != 1 || final.Number.Uint64() != 0 {
		t.Fatalf("fixture must separate mined and finalized: receipt=%s finalized=%s", receipt.BlockNumber, final.Number)
	}
	var mu sync.Mutex
	server := startSimulatedRPC(t, sim, &mu)
	client, err := chain.NewNetwork(chain.SepoliaID, []string{server.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	observed, err := client.Transaction(ctx, tx.Hash().Hex())
	if err != nil {
		t.Fatal(err)
	}
	if observed.State != "succeeded" || observed.Finalized {
		t.Fatalf("mined transaction finalized prematurely: %+v", observed)
	}
	t.Logf("receipt=%s backend_finalized=%s application_finalized=%t", receipt.BlockNumber, final.Number, observed.Finalized)
	// Advance the real simulated beacon through its first 32-block epoch.
	for range 31 {
		sim.Commit()
	}
	final, err = sim.Client().HeaderByNumber(ctx, big.NewInt(rpc.FinalizedBlockNumber.Int64()))
	if err != nil {
		t.Fatal(err)
	}
	if final.Number.Cmp(receipt.BlockNumber) < 0 {
		t.Fatalf("backend did not finalize receipt: %s", final.Number)
	}
	observed, err = client.Transaction(ctx, tx.Hash().Hex())
	if err != nil {
		t.Fatal(err)
	}
	if observed.State != "succeeded" || !observed.Finalized {
		t.Fatalf("backend-finalized transaction not recognized: %+v", observed)
	}
	t.Logf("receipt=%s backend_finalized=%s application_finalized=%t", receipt.BlockNumber, final.Number, observed.Finalized)
}
