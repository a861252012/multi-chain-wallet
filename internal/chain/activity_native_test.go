package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestSepoliaNativeTransferRecordedReceipt(t *testing.T) {
	// Public, finalized receipt that previously produced a fake SYSTEM_ADDRESS token.
	data, err := os.ReadFile("testdata/sepolia-native-7708.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Tx      *types.Transaction `json:"tx"`
		Receipt *types.Receipt     `json:"receipt"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	owner, err := types.Sender(types.LatestSignerForChainID(big.NewInt(SepoliaID)), fixture.Tx)
	if err != nil || owner != common.HexToAddress("0x11A0130EecDF4648efed6a0E3A8901Bf9E44B293") || fixture.Tx.Hash() != fixture.Receipt.TxHash {
		t.Fatalf("invalid public fixture: %v", err)
	}
	moves, err := receiptMovements(fixture.Tx, fixture.Receipt, owner, owner, SepoliaID)
	if err != nil {
		t.Fatal(err)
	}
	if len(moves) != 2 {
		t.Fatalf("native receipt must have fee + one principal, got %+v", moves)
	}
	if moves[0].kind != movementFee || moves[0].asset != "ETH" || moves[0].raw.String() != "357000" || moves[1].kind != movementSend || moves[1].asset != "ETH" || moves[1].raw.String() != "1000000000000" || moves[1].counterparty != *fixture.Tx.To() {
		t.Fatalf("incorrect native receipt ledger: %+v", moves)
	}
	net := new(big.Int).Neg(new(big.Int).Add(moves[0].raw, moves[1].raw))
	if net.String() != "-1000000357000" {
		t.Fatal(net)
	}
}

func TestActivityNativeTransferActivation(t *testing.T) {
	// Official Sepolia Glamsterdam activation (EIP-7773), not a guessed block height.
	const activation = uint64(1791294816)
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	owner := crypto.PubkeyToAddress(key.PublicKey)
	other := common.HexToAddress("0x2222222222222222222222222222222222222222")
	for _, scenario := range []struct {
		name      string
		timestamp uint64
		withLog   bool
		wantError bool
	}{
		{name: "before-no-log", timestamp: activation - 1},
		{name: "at-no-log", timestamp: activation, wantError: true},
		{name: "after-no-log", timestamp: activation + 1, wantError: true},
		{name: "at-valid-log", timestamp: activation, withLog: true},
		{name: "after-valid-log", timestamp: activation + 1, withLog: true},
		{name: "create-before", timestamp: activation - 1},
		{name: "create-after-no-log", timestamp: activation + 1, wantError: true},
		{name: "create-after-valid-log", timestamp: activation + 1, withLog: true},
		{name: "zero", timestamp: activation + 1},
		{name: "self", timestamp: activation + 1},
		{name: "reverted", timestamp: activation + 1},
		{name: "other-chain", timestamp: activation + 1},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			chainID, to, value, wantMoves := int64(SepoliaID), &other, int64(100), 2
			if strings.HasPrefix(scenario.name, "create-") {
				to = nil
			}
			switch scenario.name {
			case "zero":
				value, wantMoves = 0, 1
			case "self":
				to, wantMoves = &owner, 3
			case "reverted":
				wantMoves = 1
			case "other-chain":
				chainID = PolygonAmoyID
			}
			tx, err := types.SignTx(types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(chainID), To: to, Value: big.NewInt(value), Gas: 60000, GasFeeCap: big.NewInt(2), GasTipCap: big.NewInt(1)}), types.LatestSignerForChainID(big.NewInt(chainID)), key)
			if err != nil {
				t.Fatal(err)
			}
			header := &types.Header{Number: big.NewInt(10), Difficulty: big.NewInt(0), Time: scenario.timestamp, GasLimit: 30000000}
			r := nativeTestReceipt(tx)
			r.BlockHash = header.Hash()
			r.Logs = []*types.Log{}
			if to == nil {
				r.ContractAddress = other
			}
			if scenario.name == "reverted" {
				r.Status = types.ReceiptStatusFailed
			}
			if scenario.withLog {
				r.Logs = []*types.Log{nativeTestLog(r, common.HexToAddress("0xfffffffffffffffffffffffffffffffffffffffe"), owner, other, value, 66)}
			}
			client := rpcClient(t, func(method string, _ json.RawMessage) any {
				switch method {
				case "eth_chainId":
					return fmt.Sprintf("0x%x", chainID)
				case "eth_getTransactionByHash":
					data, _ := tx.MarshalJSON()
					var result map[string]any
					if err := json.Unmarshal(data, &result); err != nil {
						t.Fatal(err)
					}
					result["blockNumber"], result["blockHash"], result["transactionIndex"] = "0xa", header.Hash().Hex(), "0x0"
					return result
				case "eth_getTransactionReceipt":
					return r
				case "eth_getBlockByNumber":
					return header
				default:
					t.Errorf("unexpected RPC: %s", method)
					return nil
				}
			})
			client.chainID = chainID
			activity, err := client.Activity(context.Background(), tx.Hash().Hex(), owner)
			if scenario.wantError {
				if err != ErrUnavailable || activity != nil {
					t.Fatalf("missing required native log accepted: %+v, %v", activity, err)
				}
				return
			}
			if err != nil || activity == nil || len(activity.Movements) != wantMoves {
				t.Fatalf("unexpected activation result: %+v, %v", activity, err)
			}
		})
	}
}

func nativeTestReceipt(tx *types.Transaction) *types.Receipt {
	return &types.Receipt{TxHash: tx.Hash(), BlockHash: common.HexToHash("0x1234"), BlockNumber: big.NewInt(10), GasUsed: 21000, EffectiveGasPrice: big.NewInt(2), Status: types.ReceiptStatusSuccessful}
}

func nativeTestLog(r *types.Receipt, emitter, from, to common.Address, amount int64, index uint) *types.Log {
	return &types.Log{Address: emitter, TxHash: r.TxHash, BlockHash: r.BlockHash, BlockNumber: r.BlockNumber.Uint64(), Index: index, Topics: []common.Hash{transferTopic, common.BytesToHash(from.Bytes()), common.BytesToHash(to.Bytes())}, Data: common.LeftPadBytes(big.NewInt(amount).Bytes(), 32)}
}

func requireNativeLedger(t *testing.T, moves []movementRecord, want []Movement) {
	t.Helper()
	if len(moves) != len(want) {
		t.Fatalf("movements = %+v, want %+v", movementsToAPI(moves), want)
	}
	for i, got := range movementsToAPI(moves) {
		if got.Kind != want[i].Kind || got.Asset != want[i].Asset || got.Raw != want[i].Raw || got.Counterparty != want[i].Counterparty || got.Evidence == "" {
			t.Fatalf("movement %d = %+v, want %+v", i, got, want[i])
		}
	}
}

func TestNativeTransferEvidence(t *testing.T) {
	owner := common.HexToAddress("0x1111111111111111111111111111111111111111")
	other := common.HexToAddress("0x2222222222222222222222222222222222222222")
	token := common.HexToAddress("0x3333333333333333333333333333333333333333")
	weth := common.HexToAddress(SepoliaWETH)
	system := common.HexToAddress("0xfffffffffffffffffffffffffffffffffffffffe")
	fee := Movement{Kind: "fee", Asset: "ETH", Raw: "42000", Counterparty: (common.Address{}).Hex()}
	leg := func(kind, asset, raw string, counterparty common.Address) Movement {
		return Movement{Kind: kind, Asset: asset, Raw: raw, Counterparty: counterparty.Hex()}
	}
	for _, scenario := range []string{"incoming", "unrelated", "internal-repeat", "native-and-token", "zero", "self", "self-with-internal", "creation", "zero-recipient", "wrap", "unwrap", "unrelated-system-mode", "reverted", "other-chain"} {
		t.Run(scenario, func(t *testing.T) {
			from, account, to, value := owner, owner, &other, int64(100)
			chainID := int64(SepoliaID)
			switch scenario {
			case "incoming":
				account = other
			case "unrelated":
				account = token
			case "internal-repeat":
				from, to, value = other, &token, 0
			case "zero", "unwrap", "unrelated-system-mode":
				value = 0
			case "self", "self-with-internal":
				to = &owner
			case "creation":
				to = nil
			case "zero-recipient":
				zero := common.Address{}
				to = &zero
			case "other-chain":
				chainID = 84532
			}
			if scenario == "wrap" || scenario == "unwrap" || scenario == "unrelated-system-mode" {
				to = &weth
			}
			tx := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(chainID), To: to, Value: big.NewInt(value), Gas: 90000})
			r := nativeTestReceipt(tx)
			r.ContractAddress = other
			target := r.ContractAddress
			if to != nil {
				target = *to
			}
			r.Logs = []*types.Log{nativeTestLog(r, system, from, target, value, 66)}
			want := []Movement{fee, leg("send", "ETH", "100", target)}
			switch scenario {
			case "incoming":
				want = []Movement{leg("receive", "ETH", "100", from)}
			case "unrelated":
				want = nil
			case "internal-repeat":
				r.Logs = []*types.Log{nativeTestLog(r, system, token, owner, 7, 66), nativeTestLog(r, system, token, owner, 7, 67)}
				want = []Movement{leg("receive", "ETH", "7", token), leg("receive", "ETH", "7", token)}
			case "native-and-token":
				r.Logs = append(r.Logs, nativeTestLog(r, token, owner, other, 100, 67), nativeTestLog(r, token, owner, other, 100, 68))
				want = append(want, leg("send", token.Hex(), "100", other), leg("send", token.Hex(), "100", other))
			case "zero":
				r.Logs, want = nil, []Movement{fee}
			case "self":
				r.Logs = nil
				want = []Movement{fee, leg("send", "ETH", "100", owner), leg("receive", "ETH", "100", owner)}
			case "self-with-internal":
				r.Logs = []*types.Log{nativeTestLog(r, system, owner, other, 7, 66)}
				want = []Movement{fee, leg("send", "ETH", "7", other)}
			case "wrap":
				deposit := nativeTestLog(r, weth, owner, owner, 100, 67)
				deposit.Topics = []common.Hash{depositTopic, common.BytesToHash(owner.Bytes())}
				r.Logs = append(r.Logs, deposit)
				want = append(want, leg("receive", weth.Hex(), "100", weth))
			case "unwrap", "unrelated-system-mode":
				withdraw := nativeTestLog(r, weth, owner, owner, 100, 66)
				withdraw.Topics = []common.Hash{withdrawalTopic, common.BytesToHash(owner.Bytes())}
				r.Logs = []*types.Log{withdraw, nativeTestLog(r, system, weth, owner, 100, 67)}
				want = []Movement{fee, leg("send", weth.Hex(), "100", weth), leg("receive", "ETH", "100", weth)}
				if scenario == "unrelated-system-mode" {
					r.Logs[1] = nativeTestLog(r, system, other, token, 100, 67)
					want = want[:2]
				}
			case "reverted":
				r.Status, r.Logs, want = 0, nil, []Movement{fee}
			case "other-chain":
				want = append(want, leg("send", system.Hex(), "100", target))
			}
			moves, err := receiptMovements(tx, r, from, account, chainID)
			if err != nil {
				t.Fatal(err)
			}
			requireNativeLedger(t, moves, want)
			if len(r.Logs) > 0 && chainID == SepoliaID {
				for _, move := range moves {
					if move.asset == "ETH" && move.kind != movementFee && !strings.Contains(move.evidence, ":EIP-7708") {
						t.Fatalf("native principal used fallback: %+v", move)
					}
				}
			}
		})
	}
}

func TestNativeTransferInvalidEvidence(t *testing.T) {
	owner := common.HexToAddress("0x1111111111111111111111111111111111111111")
	other := common.HexToAddress("0x2222222222222222222222222222222222222222")
	system := common.HexToAddress("0xfffffffffffffffffffffffffffffffffffffffe")
	for _, scenario := range []string{"data", "topics", "padding", "zero", "self", "removed", "tx-hash", "block-hash", "block-number", "duplicate", "order", "top-level-amount", "top-level-from", "top-level-to", "top-level-missing", "creation-address", "reverted"} {
		t.Run(scenario, func(t *testing.T) {
			to := &other
			if scenario == "creation-address" {
				to = nil
			}
			tx := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(SepoliaID), To: to, Value: big.NewInt(100), Gas: 21000})
			r := nativeTestReceipt(tx)
			l := nativeTestLog(r, system, owner, other, 100, 66)
			r.Logs = []*types.Log{l}
			switch scenario {
			case "data":
				l.Data = []byte{1}
			case "topics":
				l.Topics = l.Topics[:2]
			case "padding":
				l.Topics[1][0] = 1
			case "zero":
				l.Data = make([]byte, 32)
			case "self":
				l.Topics[2] = l.Topics[1]
			case "removed":
				l.Removed = true
			case "tx-hash":
				l.TxHash = common.Hash{}
			case "block-hash":
				l.BlockHash = common.Hash{}
			case "block-number":
				l.BlockNumber++
			case "duplicate":
				r.Logs = append(r.Logs, l)
			case "order":
				r.Logs = append(r.Logs, nativeTestLog(r, system, owner, other, 7, 65))
			case "top-level-amount":
				l.Data = common.LeftPadBytes(big.NewInt(7).Bytes(), 32)
			case "top-level-from":
				l.Topics[1] = common.Hash{}
			case "top-level-to":
				l.Topics[2] = common.Hash{}
			case "top-level-missing":
				r.Logs = append([]*types.Log{nativeTestLog(r, other, owner, other, 100, 65)}, l)
			case "reverted":
				r.Status = 0
			}
			if moves, err := receiptMovements(tx, r, owner, owner, SepoliaID); err != ErrUnavailable || len(moves) != 0 {
				t.Fatalf("invalid evidence published: %+v, %v", moves, err)
			}
		})
	}
}
