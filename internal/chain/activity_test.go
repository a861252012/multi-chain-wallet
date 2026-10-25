package chain

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestReceiptMovements(t *testing.T) {
	owner := common.HexToAddress("0x1111111111111111111111111111111111111111")
	other := common.HexToAddress("0x2222222222222222222222222222222222222222")
	tx := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(SepoliaID), To: &other, Value: big.NewInt(100), Gas: 21000})
	receipt := &types.Receipt{TxHash: tx.Hash(), BlockHash: common.HexToHash("0x01"), BlockNumber: big.NewInt(10), GasUsed: 21000, EffectiveGasPrice: big.NewInt(2), Status: 1, Logs: []*types.Log{}}
	movements, err := receiptMovements(tx, receipt, owner, owner, SepoliaID)
	if err != nil {
		t.Fatal(err)
	}
	if len(movements) != 2 || movements[0].kind != movementFee || movements[0].raw.Cmp(big.NewInt(42000)) != 0 || movements[1].kind != movementSend || movements[1].raw.Cmp(big.NewInt(100)) != 0 {
		t.Fatalf("wrong outgoing ledger %+v", movements)
	}
	receipt.Status = 0
	movements, err = receiptMovements(tx, receipt, owner, owner, SepoliaID)
	if err != nil || len(movements) != 1 || movements[0].kind != movementFee {
		t.Fatalf("revert moved principal %+v %v", movements, err)
	}
	receipt.Status = 1
	movements, err = receiptMovements(tx, receipt, owner, other, SepoliaID)
	if err != nil || len(movements) != 1 || movements[0].kind != movementReceive {
		t.Fatalf("receiver paid sender gas %+v %v", movements, err)
	}
	self := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(SepoliaID), To: &owner, Value: big.NewInt(100), Gas: 21000})
	movements, err = receiptMovements(self, receipt, owner, owner, SepoliaID)
	if err != nil || len(movements) != 3 {
		t.Fatalf("self transfer missing offset %+v %v", movements, err)
	}
}

func TestReceiptTokenAndWETHMovementEvidence(t *testing.T) {
	owner := common.HexToAddress("0x1111111111111111111111111111111111111111")
	other := common.HexToAddress("0x2222222222222222222222222222222222222222")
	weth := common.HexToAddress(SepoliaWETH)
	tx := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(SepoliaID), To: &weth, Value: big.NewInt(0), Gas: 80000})
	r := &types.Receipt{TxHash: tx.Hash(), BlockHash: common.HexToHash("0x01"), BlockNumber: big.NewInt(10), GasUsed: 30000, EffectiveGasPrice: big.NewInt(1), Status: 1}
	log := &types.Log{Address: other, TxHash: r.TxHash, BlockHash: r.BlockHash, BlockNumber: 10, Index: 3, Data: common.LeftPadBytes(big.NewInt(100).Bytes(), 32), Topics: []common.Hash{transferTopic, common.BytesToHash(other.Bytes()), common.BytesToHash(owner.Bytes())}}
	r.Logs = []*types.Log{log}
	moves, err := receiptMovements(tx, r, other, owner, SepoliaID)
	if err != nil || len(moves) != 1 || moves[0].asset != other.Hex() || moves[0].kind != movementReceive {
		t.Fatalf("token transfer %+v %v", moves, err)
	}
	log.Topics = append(log.Topics, common.Hash{})
	moves, err = receiptMovements(tx, r, other, owner, SepoliaID)
	if err != nil || len(moves) != 0 {
		t.Fatalf("ERC721 counted as ERC20 %+v %v", moves, err)
	}
	log.Address = weth
	log.Topics = []common.Hash{withdrawalTopic, common.BytesToHash(owner.Bytes())}
	moves, err = receiptMovements(tx, r, owner, owner, SepoliaID)
	if err != nil || len(moves) != 3 || moves[1].kind != movementSend || moves[1].asset != weth.Hex() || moves[2].asset != "ETH" || moves[2].kind != movementReceive {
		t.Fatalf("withdrawal %+v %v", moves, err)
	}
	log.Address = other
	moves, err = receiptMovements(tx, r, other, owner, SepoliaID)
	if err != nil || len(moves) != 0 {
		t.Fatalf("forged WETH event counted %+v %v", moves, err)
	}
	log.Removed = true
	if _, err := receiptMovements(tx, r, owner, owner, SepoliaID); err == nil {
		t.Fatal("removed log accepted")
	}
	log.Removed = false
	r.Logs = append(r.Logs, log)
	if _, err := receiptMovements(tx, r, owner, owner, SepoliaID); err == nil {
		t.Fatal("duplicate log accepted")
	}
}

func TestActivityCanonicalReceiptAndOwnership(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	sender := crypto.PubkeyToAddress(key.PublicKey)
	recipient := common.HexToAddress("0x2222222222222222222222222222222222222222")
	tx, err := types.SignTx(types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(SepoliaID), Nonce: 0, To: &recipient, Value: big.NewInt(100), Gas: 21000, GasFeeCap: big.NewInt(2), GasTipCap: big.NewInt(1)}), types.LatestSignerForChainID(big.NewInt(SepoliaID)), key)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"received", "unrelated", "reorg", "reverted", "pending"} {
		t.Run(kind, func(t *testing.T) {
			header := &types.Header{Number: big.NewInt(10), Difficulty: big.NewInt(0), Time: 100, GasLimit: 30000000}
			receipt := &types.Receipt{TxHash: tx.Hash(), BlockHash: header.Hash(), BlockNumber: big.NewInt(10), Status: 1, GasUsed: 21000, EffectiveGasPrice: big.NewInt(2), Logs: []*types.Log{}}
			if kind == "reorg" {
				receipt.BlockHash = common.HexToHash("0xbad")
			}
			if kind == "reverted" {
				receipt.Status = 0
			}
			c := rpcClient(t, func(method string, _ json.RawMessage) any {
				switch method {
				case "eth_chainId":
					return "0xaa36a7"
				case "eth_getTransactionByHash":
					data, _ := tx.MarshalJSON()
					var value map[string]any
					json.Unmarshal(data, &value)
					if kind != "pending" {
						value["blockNumber"] = "0xa"
						value["blockHash"] = header.Hash().Hex()
						value["transactionIndex"] = "0x0"
					}
					return value
				case "eth_getBlockByNumber":
					return header
				case "eth_getTransactionReceipt":
					return receipt
				default:
					t.Errorf("unexpected %s", method)
					return nil
				}
			})
			owner := recipient
			if kind == "unrelated" {
				owner = common.HexToAddress("0x3333333333333333333333333333333333333333")
			}
			if kind == "reverted" {
				owner = sender
			}
			activity, err := c.Activity(context.Background(), tx.Hash().Hex(), owner)
			if kind == "unrelated" {
				if err != ErrUnrelated {
					t.Fatalf("unrelated %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "received":
				if activity.State != "succeeded" || len(activity.Movements) != 1 || activity.Movements[0].Raw != "100" {
					t.Fatalf("bad receipt %+v", activity)
				}
			case "reorg", "pending":
				if len(activity.Movements) != 0 {
					t.Fatal("unconfirmed movements counted")
				}
			case "reverted":
				if activity.State != "reverted" || len(activity.Movements) != 1 || activity.Movements[0].Kind != "fee" {
					t.Fatal("reverted principal counted")
				}
			}
		})
	}
}

func TestDiscoveryScopesLogsAndDeduplicates(t *testing.T) {
	owner := common.HexToAddress("0x1111111111111111111111111111111111111111")
	token := common.HexToAddress("0x2222222222222222222222222222222222222222")
	hash := common.HexToHash("0xabcd")
	blocks := 0
	c := rpcClient(t, func(method string, params json.RawMessage) any {
		switch method {
		case "eth_chainId":
			return "0xaa36a7"
		case "eth_getBlockByNumber":
			var args []json.RawMessage
			json.Unmarshal(params, &args)
			var include bool
			json.Unmarshal(args[1], &include)
			header := &types.Header{Number: big.NewInt(2), Difficulty: big.NewInt(0), GasLimit: 30000000, UncleHash: types.EmptyUncleHash, TxHash: types.EmptyTxsHash}
			if !include {
				return header
			}
			blocks += 1
			raw, _ := header.MarshalJSON()
			var value map[string]any
			json.Unmarshal(raw, &value)
			value["transactions"] = []any{}
			value["uncles"] = []any{}
			return value
		case "eth_getLogs":
			var args []struct {
				Address []common.Address `json:"address"`
				Topics  [][]common.Hash  `json:"topics"`
			}
			if err := json.Unmarshal(params, &args); err != nil {
				t.Fatal(err)
			}
			if len(args) != 1 || len(args[0].Address) != 3 || len(args[0].Topics) != 3 || args[0].Topics[2][0] != common.BytesToHash(owner.Bytes()) {
				t.Fatalf("unscoped log request %s", params)
			}
			log := types.Log{Address: token, TxHash: hash, BlockNumber: 2, Data: common.LeftPadBytes(big.NewInt(1).Bytes(), 32), Topics: []common.Hash{transferTopic, {}, common.BytesToHash(owner.Bytes())}}
			return []types.Log{log, log}
		default:
			t.Errorf("unexpected RPC %s", method)
			return nil
		}
	})
	ids, from, to, err := c.DiscoverActivity(context.Background(), owner, 1, token, common.HexToAddress(SepoliaWETH))
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != hash.Hex() || from != 1 || to != 2 || blocks != 2 {
		t.Fatalf("incorrect discovery %v %d %d %d", ids, from, to, blocks)
	}
}

func TestPolygonMovementsUsePOL(t *testing.T) {
	owner := common.HexToAddress("0x1111111111111111111111111111111111111111")
	tx := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(80002), To: &owner, Value: big.NewInt(1), Gas: 21000})
	receipt := &types.Receipt{GasUsed: 21000, EffectiveGasPrice: big.NewInt(2), Status: 1}
	moves, err := receiptMovements(tx, receipt, owner, owner, 80002)
	if err != nil || len(moves) != 3 {
		t.Fatal(moves, err)
	}
	for _, move := range moves {
		if move.asset != "POL" {
			t.Fatal("mislabelled native POL", move)
		}
	}
	for _, id := range []int64{1, 137} {
		if c, err := NewNetwork(id, []string{"http://127.0.0.1:1"}); err == nil {
			c.Close()
			t.Fatal("accepted mainnet", id)
		}
	}
}
