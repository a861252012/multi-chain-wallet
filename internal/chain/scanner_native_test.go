package chain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestScanFinalizedBlockNativeTransferCandidates(t *testing.T) {
	owner := common.HexToAddress("0x1111111111111111111111111111111111111111")
	other := common.HexToAddress("0x2222222222222222222222222222222222222222")
	contract := common.HexToAddress("0x3333333333333333333333333333333333333333")
	system := common.HexToAddress("0xfffffffffffffffffffffffffffffffffffffffe")
	token := common.HexToAddress(SepoliaUSDC)
	hash := common.HexToHash("0x1234")
	header := &types.Header{Number: big.NewInt(10), Difficulty: big.NewInt(0), Time: 100, GasLimit: 30000000}
	for _, scenario := range []struct {
		name       string
		chainID    int64
		wantHash   bool
		wantTokens []string
		wantError  bool
	}{
		{name: "internal-receive", chainID: SepoliaID, wantHash: true},
		{name: "internal-send", chainID: SepoliaID, wantHash: true},
		{name: "unrelated", chainID: SepoliaID},
		{name: "erc20", chainID: SepoliaID, wantHash: true, wantTokens: []string{token.Hex()}},
		{name: "mixed-native-token", chainID: SepoliaID, wantHash: true, wantTokens: []string{token.Hex()}},
		{name: "other-chain", chainID: BaseSepoliaID, wantHash: true, wantTokens: []string{system.Hex()}},
		{name: "reverted", chainID: SepoliaID},
		{name: "removed", chainID: SepoliaID, wantError: true},
		{name: "foreign-block", chainID: SepoliaID, wantError: true},
		{name: "duplicate-index", chainID: SepoliaID, wantError: true},
		{name: "malformed-native-data", chainID: SepoliaID, wantError: true},
		{name: "malformed-native-padding", chainID: SepoliaID, wantError: true},
		{name: "zero-native-value", chainID: SepoliaID, wantError: true},
		{name: "native-self-transfer", chainID: SepoliaID, wantError: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			log := &types.Log{
				Address: system, TxHash: hash, BlockHash: header.Hash(), BlockNumber: 10, Index: 7,
				Topics: []common.Hash{transferTopic, common.BytesToHash(contract.Bytes()), common.BytesToHash(owner.Bytes())},
				Data:   common.LeftPadBytes(big.NewInt(100).Bytes(), 32),
			}
			receipt := &types.Receipt{TxHash: hash, BlockHash: header.Hash(), BlockNumber: big.NewInt(10), Status: 1, Logs: []*types.Log{log}}
			switch scenario.name {
			case "internal-send":
				log.Topics[1], log.Topics[2] = log.Topics[2], log.Topics[1]
			case "unrelated":
				log.Topics[2] = common.BytesToHash(other.Bytes())
			case "erc20":
				log.Address = token
			case "mixed-native-token":
				tokenLog := *log
				tokenLog.Address, tokenLog.Index = token, 8
				receipt.Logs = append(receipt.Logs, &tokenLog)
			case "reverted":
				receipt.Status = 0
			case "removed":
				log.Removed = true
			case "foreign-block":
				log.BlockHash = common.HexToHash("0xbad")
			case "duplicate-index":
				receipt.Logs = append(receipt.Logs, log)
			case "malformed-native-data":
				log.Data = []byte{1}
			case "malformed-native-padding":
				log.Topics[1][0] = 1
			case "zero-native-value":
				log.Data = make([]byte, 32)
			case "native-self-transfer":
				log.Topics[1] = log.Topics[2]
			}
			client := rpcClient(t, func(method string, params json.RawMessage) any {
				switch method {
				case "eth_chainId":
					return fmt.Sprintf("0x%x", scenario.chainID)
				case "eth_getBlockByNumber":
					var args []json.RawMessage
					if err := json.Unmarshal(params, &args); err != nil || len(args) != 2 {
						t.Fatalf("invalid block request %s: %v", params, err)
					}
					var full bool
					if err := json.Unmarshal(args[1], &full); err != nil {
						t.Fatal(err)
					}
					if full {
						to := contract.Hex()
						return rpcBlock{Hash: header.Hash().Hex(), Transactions: []rpcTransaction{{Hash: hash.Hex(), From: other.Hex(), To: &to}}}
					}
					return header
				case "eth_getBlockReceipts":
					return []*types.Receipt{receipt}
				default:
					t.Errorf("unexpected RPC %s", method)
					return nil
				}
			})
			client.chainID = scenario.chainID
			result, err := client.ScanFinalizedBlock(context.Background(), 10, owner)
			if scenario.wantError {
				if !errors.Is(err, ErrUnavailable) {
					t.Fatalf("invalid native evidence accepted: result=%+v err=%v", result, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var wantHashes []string
			if scenario.wantHash {
				wantHashes = []string{hash.Hex()}
			}
			if !slices.Equal(result.Hashes, wantHashes) || !slices.Equal(result.Tokens, scenario.wantTokens) {
				t.Fatalf("wrong candidates: hashes=%v tokens=%v; want hashes=%v tokens=%v", result.Hashes, result.Tokens, wantHashes, scenario.wantTokens)
			}
		})
	}
}
