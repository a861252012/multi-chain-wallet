package chain

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestVaultReceiptEvidence(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	owner := crypto.PubkeyToAddress(key.PublicKey)
	vault := common.HexToAddress("0x5555555555555555555555555555555555555555")
	other := common.HexToAddress("0x6666666666666666666666666666666666666666")
	for _, scenario := range []string{"withdraw", "deposit", "unconfigured", "fake-emitter", "wrong-account", "reverted", "reorg", "removed", "malformed", "native-deposit", "native-withdraw", "native-unconfigured", "native-deposit-extra"} {
		t.Run(scenario, func(t *testing.T) {
			value := big.NewInt(0)
			topic := vaultWithdrawnTopic
			if scenario == "deposit" || strings.HasPrefix(scenario, "native-deposit") {
				value = big.NewInt(100)
				topic = vaultDepositedTopic
			}
			tx, err := types.SignTx(types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(SepoliaID), To: &vault, Value: value, Gas: 60000, GasFeeCap: big.NewInt(2), GasTipCap: big.NewInt(1)}), types.LatestSignerForChainID(big.NewInt(SepoliaID)), key)
			if err != nil {
				t.Fatal(err)
			}
			header := &types.Header{Number: big.NewInt(10), Difficulty: big.NewInt(0), Time: 100, GasLimit: 30000000}
			log := &types.Log{Address: vault, Topics: []common.Hash{topic, common.BytesToHash(owner.Bytes())}, Data: common.LeftPadBytes(big.NewInt(100).Bytes(), 32), TxHash: tx.Hash(), BlockHash: header.Hash(), BlockNumber: 10}
			receipt := &types.Receipt{TxHash: tx.Hash(), BlockHash: header.Hash(), BlockNumber: big.NewInt(10), Status: 1, GasUsed: 50000, EffectiveGasPrice: big.NewInt(2), Logs: []*types.Log{log}}
			if strings.HasPrefix(scenario, "native-") {
				if value.Sign() > 0 {
					log.Index = 1
					receipt.Logs = []*types.Log{nativeTestLog(receipt, ethTransferSystemAddress, owner, vault, 100, 0), log}
				} else {
					receipt.Logs = append(receipt.Logs, nativeTestLog(receipt, ethTransferSystemAddress, vault, owner, 100, 1))
				}
				if scenario == "native-deposit-extra" {
					receipt.Logs = append(receipt.Logs, nativeTestLog(receipt, ethTransferSystemAddress, owner, other, 7, 2))
				}
			}
			configured := vault
			switch scenario {
			case "unconfigured", "native-unconfigured":
				configured = common.Address{}
			case "fake-emitter":
				log.Address = other
			case "wrong-account":
				log.Topics[1] = common.BytesToHash(other.Bytes())
			case "reverted":
				receipt.Status = 0
			case "reorg":
				receipt.BlockHash = common.HexToHash("0xbad")
			case "removed":
				log.Removed = true
			case "malformed":
				log.Data = []byte{1}
			}
			client := rpcClient(t, func(method string, _ json.RawMessage) any {
				switch method {
				case "eth_chainId":
					return "0xaa36a7"
				case "eth_getBlockByNumber":
					return header
				case "eth_getTransactionReceipt":
					return receipt
				case "eth_getTransactionByHash":
					data, _ := tx.MarshalJSON()
					var result map[string]any
					json.Unmarshal(data, &result)
					result["blockNumber"] = "0xa"
					result["blockHash"] = header.Hash().Hex()
					result["transactionIndex"] = "0x0"
					return result
				default:
					t.Errorf("unexpected %s", method)
					return nil
				}
			})
			activity, err := client.Activity(context.Background(), tx.Hash().Hex(), owner, configured)
			if scenario == "removed" {
				if err == nil {
					t.Fatal("removed log accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			principal := 0
			for _, move := range activity.Movements {
				if move.Kind == "fee" {
					continue
				}
				principal++
				if strings.HasPrefix(scenario, "native-") && (!strings.Contains(move.Evidence, ":EIP-7708") || strings.Contains(move.Evidence, "Deposited") || strings.Contains(move.Evidence, "Withdrawn")) {
					t.Fatalf("native evidence duplicated or incorrectly annotated: %+v", move)
				}
				if scenario == "native-deposit-extra" && move.Counterparty == other.Hex() && move.Asset == "ETH" && move.Kind == "send" && move.Raw == "7" {
					continue
				}
				if move.Raw != "100" || move.Asset != "ETH" {
					t.Fatal(move)
				}
				if scenario == "withdraw" && (move.Kind != "receive" || !strings.Contains(move.Evidence, "Withdrawn")) {
					t.Fatal(move)
				}
				if scenario == "deposit" && (move.Kind != "send" || !strings.Contains(move.Evidence, "Deposited")) {
					t.Fatal(move)
				}
			}
			want := 0
			if scenario == "withdraw" || scenario == "deposit" || strings.HasPrefix(scenario, "native-") {
				want = 1
			}
			if scenario == "native-deposit-extra" {
				want = 2
			}
			if principal != want {
				t.Fatalf("principal movements %d want %d", principal, want)
			}
		})
	}
}
