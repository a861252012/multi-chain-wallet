package main

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	aa "github.com/a861252012/multi-chain-wallet/internal/wallet/erc4337"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func TestSavedOperationCannotChangeIntent(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	signer, err := aa.NewPrivateKeySigner(key)
	if err != nil {
		t.Fatal(err)
	}
	defer signer.Wipe()
	owner := signer.Address()
	sender := common.HexToAddress("0x1234")
	args := append(common.LeftPadBytes(owner.Bytes(), 32), make([]byte, 32)...)
	initCode := append(factory.Bytes(), append(crypto.Keccak256([]byte("createAccount(address,uint256)"))[:4], args...)...)
	b := aa.NewBuilder(entryPoint, chainID).SetSender(sender).SetInitCode(initCode)
	if _, err = b.SetExecuteCallData(owner, big.NewInt(1), nil); err != nil {
		t.Fatal(err)
	}
	op, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	op.Signature, err = aa.SignUserOpWithEthPrefix(signer, op, entryPoint, chainID)
	if err != nil {
		t.Fatal(err)
	}
	if err = validateOperation(op, sender, owner, args); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*aa.UserOperation){
		"different nonce":    func(o *aa.UserOperation) { o.Nonce = big.NewInt(1) },
		"different calldata": func(o *aa.UserOperation) { o.CallData[40] ^= 1 },
		"different factory":  func(o *aa.UserOperation) { o.InitCode[0] ^= 1 },
		"different sender":   func(o *aa.UserOperation) { o.Sender = owner },
		"bad signature":      func(o *aa.UserOperation) { o.Signature[0] ^= 1 },
	} {
		t.Run(name, func(t *testing.T) {
			changed := op.Clone()
			mutate(changed)
			if validateOperation(changed, sender, owner, args) == nil {
				t.Fatal("accepted modified operation")
			}
		})
	}
}

func TestWriteNewNeverOverwritesSignedOperation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "operation.json")
	if err := writeNew(path, []byte("original")); err != nil {
		t.Fatal(err)
	}
	if err := writeNew(path, []byte("replacement")); err == nil {
		t.Fatal("overwrote operation")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "original" {
		t.Fatalf("saved operation changed: %q %v", got, err)
	}
}

func TestReceiptEvidence(t *testing.T) {
	for _, scenario := range []string{"success", "bundler failure", "wrong hash", "wrong nonce", "tx revert", "wrong block", "missing event", "event failure", "wrong factory", "already deployed", "wrong owner", "wrong entrypoint", "wrong balance"} {
		t.Run(scenario, func(t *testing.T) {
			sender := common.HexToAddress("0x1234")
			owner := common.HexToAddress("0x5678")
			hash := common.HexToHash("0x1111")
			header := &types.Header{Number: big.NewInt(10), Difficulty: big.NewInt(0), GasLimit: 30_000_000, Extra: []byte{}, BaseFee: big.NewInt(1)}
			eventData := make([]byte, 128)
			eventData[63] = 1
			event := &types.Log{Address: entryPoint, Topics: []common.Hash{crypto.Keccak256Hash([]byte("UserOperationEvent(bytes32,address,address,uint256,bool,uint256,uint256)")), hash, common.BytesToHash(sender.Bytes()), {}}, Data: eventData}
			deployment := &types.Log{Address: entryPoint, Topics: []common.Hash{crypto.Keccak256Hash([]byte("AccountDeployed(bytes32,address,address,address)")), hash, common.BytesToHash(sender.Bytes())}, Data: append(common.LeftPadBytes(factory.Bytes(), 32), make([]byte, 32)...)}
			txr := &types.Receipt{Status: 1, BlockNumber: big.NewInt(10), BlockHash: header.Hash(), TxHash: common.HexToHash("0x2222"), Logs: []*types.Log{deployment, event}, CumulativeGasUsed: 100000, GasUsed: 100000}
			r := &aa.UserOperationReceipt{UserOpHash: hash, EntryPoint: entryPoint, Sender: sender, Nonce: big.NewInt(0), Success: true, Receipt: txr}
			switch scenario {
			case "bundler failure":
				r.Success = false
			case "wrong hash":
				r.UserOpHash = common.Hash{}
			case "wrong nonce":
				r.Nonce = big.NewInt(1)
			case "tx revert":
				txr.Status = 0
			case "wrong block":
				txr.BlockHash = common.Hash{}
			case "missing event":
				txr.Logs = nil
			case "event failure":
				event.Data[63] = 0
			case "wrong factory":
				deployment.Data[31] ^= 1
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				var call struct {
					ID     json.RawMessage   `json:"id"`
					Method string            `json:"method"`
					Params []json.RawMessage `json:"params"`
				}
				if err := json.NewDecoder(req.Body).Decode(&call); err != nil {
					t.Error(err)
					return
				}
				var result any
				switch call.Method {
				case "eth_getTransactionReceipt":
					result = txr
				case "eth_getBlockByNumber":
					result = header
				case "eth_getCode":
					result = "0x6000"
					if string(call.Params[1]) == `"0x9"` && scenario != "already deployed" {
						result = "0x"
					}
				case "eth_call":
					var arg map[string]string
					json.Unmarshal(call.Params[0], &arg)
					target := owner
					if arg["input"] == hexutil.Encode(crypto.Keccak256([]byte("entryPoint()"))[:4]) {
						target = entryPoint
						if scenario == "wrong entrypoint" {
							target = owner
						}
					} else if scenario == "wrong owner" {
						target = sender
					}
					result = hexutil.Encode(common.LeftPadBytes(target.Bytes(), 32))
				case "eth_getBalance":
					result = "0x0"
					if string(call.Params[1]) == `"0xa"` {
						result = "0x1"
						if scenario == "wrong balance" {
							result = "0x2"
						}
					}
				default:
					t.Errorf("unexpected method %s", call.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": call.ID, "result": result})
			}))
			defer server.Close()
			client, err := ethclient.Dial(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			if scenario == "success" {
				partial := *r.Receipt
				partial.Status = 0 // Bundler responses can omit status.
				r.Receipt = &partial
			}
			err = verifyReceipt(context.Background(), client, &aa.UserOperation{Sender: sender, Nonce: big.NewInt(0)}, owner, hash, r)
			if scenario == "success" && (err != nil || r.Receipt.Status != 1) {
				t.Fatalf("verified node receipt not preserved: %v", err)
			}
			if scenario != "success" && err == nil {
				t.Fatal("accepted invalid evidence")
			}
		})
	}
}

func TestConfirmationRPCMustBeExplicitAndIndependent(t *testing.T) {
	for _, pair := range [][2]string{
		{"https://rpc.example", ""}, {"https://rpc.example", "https://RPC.EXAMPLE./other"},
		{"http://127.0.0.1:1234", "http://localhost:5678"}, {"file:///rpc", "https://other.example"},
	} {
		if validateConfirmationRPC(pair[0], pair[1]) == nil {
			t.Fatalf("accepted %v", pair)
		}
	}
	if err := validateConfirmationRPC("https://a.example", "https://b.example"); err != nil {
		t.Fatal(err)
	}
}

func TestConfirmedSenderRejectsSubstitutionAndMalformedResponses(t *testing.T) {
	owner := common.HexToAddress("0x1234")
	expected := common.HexToAddress("0x5678")
	args := append(common.LeftPadBytes(owner.Bytes(), 32), make([]byte, 32)...)
	encoded := common.LeftPadBytes(expected.Bytes(), 32)
	server := func(result []byte, network string) *ethclient.Client {
		t.Helper()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var call struct {
				ID     json.RawMessage
				Method string
				Params []json.RawMessage
			}
			if err := json.NewDecoder(r.Body).Decode(&call); err != nil {
				t.Error(err)
				return
			}
			var value any = network
			if call.Method == "eth_call" {
				var msg map[string]string
				json.Unmarshal(call.Params[0], &msg)
				if common.HexToAddress(msg["to"]) != factory || msg["input"] != hexutil.Encode(append(crypto.Keccak256([]byte("getAddress(address,uint256)"))[:4], args...)) {
					t.Error("wrong factory query")
				}
				value = hexutil.Encode(result)
			} else if call.Method != "eth_chainId" {
				t.Errorf("unexpected %s", call.Method)
			}
			json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": call.ID, "result": value})
		}))
		t.Cleanup(srv.Close)
		c, err := ethclient.Dial(srv.URL)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(c.Close)
		return c
	}
	good := server(encoded, "0xaa36a7")
	if got, err := confirmedSender(context.Background(), good, server(encoded, "0xaa36a7"), args); err != nil || got != expected {
		t.Fatalf("agreement: %s %v", got, err)
	}
	padding := append([]byte{}, encoded...)
	padding[0] = 1
	for _, tc := range []struct {
		data  []byte
		chain string
	}{
		{common.LeftPadBytes(owner.Bytes(), 32), "0xaa36a7"}, {make([]byte, 32), "0xaa36a7"},
		{encoded[1:], "0xaa36a7"}, {append(encoded, 0), "0xaa36a7"}, {padding, "0xaa36a7"}, {encoded, "0x1"},
	} {
		bad := server(tc.data, tc.chain)
		for _, pair := range [][2]*ethclient.Client{{bad, good}, {good, bad}} {
			if got, err := confirmedSender(context.Background(), pair[0], pair[1], args); err == nil || got != (common.Address{}) {
				t.Fatal("unsafe funding address escaped")
			}
		}
	}
}
