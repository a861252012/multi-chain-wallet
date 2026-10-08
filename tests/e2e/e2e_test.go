package e2e

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/a861252012/multi-chain-wallet/internal/wallet"
	"github.com/a861252012/multi-chain-wallet/internal/web"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

type rpcRequest struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

func startSimulatedRPC(t *testing.T, sim *simulated.Backend, mu *sync.Mutex) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mu.Lock()
		defer mu.Unlock()

		ctx := r.Context()
		var res any
		var rpcErr error

		switch req.Method {
		case "eth_chainId":
			res = "0xaa36a7" // Sepolia 11155111
		case "eth_blockNumber":
			head, err := sim.Client().HeaderByNumber(ctx, nil)
			if err != nil {
				rpcErr = err
			} else {
				res = hexutil.EncodeBig(head.Number)
			}
		case "eth_getBlockByNumber":
			var params []json.RawMessage
			if err := json.Unmarshal(req.Params, &params); err != nil || len(params) != 2 {
				rpcErr = errors.New("eth_getBlockByNumber requires a block tag and transaction flag")
				break
			}
			var number rpc.BlockNumber
			if err := json.Unmarshal(params[0], &number); err != nil {
				rpcErr = fmt.Errorf("invalid block tag: %w", err)
				break
			}
			if string(params[1]) != "true" && string(params[1]) != "false" {
				rpcErr = errors.New("eth_getBlockByNumber requires a boolean transaction flag")
				break
			}
			head, err := sim.Client().HeaderByNumber(ctx, big.NewInt(number.Int64()))
			if err != nil {
				rpcErr = err
			} else {
				res = head
			}
		case "eth_getCode":
			var params []string
			_ = json.Unmarshal(req.Params, &params)
			if len(params) > 0 {
				addr := common.HexToAddress(params[0])
				code, err := sim.Client().CodeAt(ctx, addr, nil)
				if err != nil {
					rpcErr = err
				} else {
					res = hexutil.Encode(code)
				}
			}
		case "eth_call":
			var params []json.RawMessage
			_ = json.Unmarshal(req.Params, &params)
			if len(params) > 0 {
				var callMsg struct {
					From  string `json:"from"`
					To    string `json:"to"`
					Gas   string `json:"gas"`
					Price string `json:"gasPrice"`
					Value string `json:"value"`
					Data  string `json:"data"`
					Input string `json:"input"`
				}
				_ = json.Unmarshal(params[0], &callMsg)
				var toAddr *common.Address
				if callMsg.To != "" {
					a := common.HexToAddress(callMsg.To)
					toAddr = &a
				}
				var val *big.Int
				if callMsg.Value != "" {
					val, _ = hexutil.DecodeBig(callMsg.Value)
				}
				rawData := callMsg.Data
				if rawData == "" {
					rawData = callMsg.Input
				}
				var data []byte
				if rawData != "" {
					data, _ = hexutil.Decode(rawData)
				}
				msg := ethereum.CallMsg{
					From:  common.HexToAddress(callMsg.From),
					To:    toAddr,
					Value: val,
					Data:  data,
				}
				var block *big.Int
				if len(params) > 1 {
					var tag rpc.BlockNumber
					if err := json.Unmarshal(params[1], &tag); err != nil {
						rpcErr = err
						break
					}
					if tag != rpc.LatestBlockNumber {
						block = big.NewInt(tag.Int64())
					}
				}
				output, err := sim.Client().CallContract(ctx, msg, block)
				if err != nil {
					rpcErr = err
				} else {
					res = hexutil.Encode(output)
				}
			}
		case "eth_estimateGas":
			var params []json.RawMessage
			_ = json.Unmarshal(req.Params, &params)
			if len(params) > 0 {
				var callMsg struct {
					From  string `json:"from"`
					To    string `json:"to"`
					Value string `json:"value"`
					Data  string `json:"data"`
					Input string `json:"input"`
				}
				_ = json.Unmarshal(params[0], &callMsg)
				var toAddr *common.Address
				if callMsg.To != "" {
					a := common.HexToAddress(callMsg.To)
					toAddr = &a
				}
				var val *big.Int
				if callMsg.Value != "" {
					val, _ = hexutil.DecodeBig(callMsg.Value)
				}
				rawData := callMsg.Data
				if rawData == "" {
					rawData = callMsg.Input
				}
				var data []byte
				if rawData != "" {
					data, _ = hexutil.Decode(rawData)
				}
				msg := ethereum.CallMsg{
					From:  common.HexToAddress(callMsg.From),
					To:    toAddr,
					Value: val,
					Data:  data,
				}
				gas, err := sim.Client().EstimateGas(ctx, msg)
				if err != nil {
					rpcErr = err
				} else {
					res = hexutil.EncodeUint64(gas)
				}
			}
		case "eth_gasPrice", "eth_maxPriorityFeePerGas":
			res = "0x3b9aca00" // 1 Gwei
		case "eth_getTransactionCount":
			var params []string
			_ = json.Unmarshal(req.Params, &params)
			if len(params) > 0 {
				addr := common.HexToAddress(params[0])
				nonce, err := sim.Client().PendingNonceAt(ctx, addr)
				if err != nil {
					rpcErr = err
				} else {
					res = hexutil.EncodeUint64(nonce)
				}
			}
		case "eth_getBalance":
			var params []string
			_ = json.Unmarshal(req.Params, &params)
			if len(params) > 0 {
				addr := common.HexToAddress(params[0])
				bal, err := sim.Client().BalanceAt(ctx, addr, nil)
				if err != nil {
					rpcErr = err
				} else {
					res = hexutil.EncodeBig(bal)
				}
			}
		case "eth_sendRawTransaction":
			var params []string
			_ = json.Unmarshal(req.Params, &params)
			if len(params) > 0 {
				raw, err := hexutil.Decode(params[0])
				if err != nil {
					rpcErr = err
				} else {
					tx := new(types.Transaction)
					if err := tx.UnmarshalBinary(raw); err != nil {
						rpcErr = err
					} else {
						if err := sim.Client().SendTransaction(ctx, tx); err != nil {
							rpcErr = err
						} else {
							sim.Commit()
							res = tx.Hash().Hex()
						}
					}
				}
			}
		case "eth_getTransactionReceipt":
			var params []string
			_ = json.Unmarshal(req.Params, &params)
			if len(params) > 0 {
				h := common.HexToHash(params[0])
				receipt, err := sim.Client().TransactionReceipt(ctx, h)
				if err != nil || receipt == nil {
					res = nil
				} else {
					if receipt.EffectiveGasPrice == nil {
						receipt.EffectiveGasPrice = big.NewInt(1000000000)
					}
					res = receipt
				}
			}
		case "eth_getTransactionByHash":
			var params []string
			_ = json.Unmarshal(req.Params, &params)
			if len(params) > 0 {
				h := common.HexToHash(params[0])
				tx, _, err := sim.Client().TransactionByHash(ctx, h)
				if err != nil {
					res = nil
				} else {
					res = tx
				}
			}
		default:
			res = "0x0"
		}

		w.Header().Set("Content-Type", "application/json")
		if rpcErr != nil {
			errPayload := map[string]any{
				"code":    -32000,
				"message": rpcErr.Error(),
			}
			var rpcCodeErr rpc.Error
			if errors.As(rpcErr, &rpcCodeErr) {
				errPayload["code"] = rpcCodeErr.ErrorCode()
			}
			var dataErr rpc.DataError
			if errors.As(rpcErr, &dataErr) {
				data := dataErr.ErrorData()
				if b, ok := data.([]byte); ok {
					errPayload["data"] = hexutil.Encode(b)
				} else {
					errPayload["data"] = data
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"error":   errPayload,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  res,
		})
	}))
	t.Cleanup(server.Close)
	return server
}

func deployContract(t *testing.T, sim *simulated.Backend, transactor *bind.TransactOpts, artifactPath string, args ...any) (common.Address, abi.ABI) {
	t.Helper()
	raw, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	var artifact struct {
		ABI        json.RawMessage `json:"abi"`
		Bytecode   string          `json:"bytecode"`
		SourceFile string          `json:"sourceFile"`
		SourceHash string          `json:"sourceHash"`
	}
	if err := json.Unmarshal(raw, &artifact); err != nil {
		t.Fatalf("unmarshal artifact: %v", err)
	}

	sourcePath := filepath.Join(filepath.Dir(artifactPath), "..", artifact.SourceFile)
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source file %s: %v", sourcePath, err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(source)) != artifact.SourceHash {
		t.Fatal("stale contract artifact; source hash mismatch")
	}

	parsedABI, err := abi.JSON(strings.NewReader(string(artifact.ABI)))
	if err != nil {
		t.Fatalf("parse ABI: %v", err)
	}

	addr, tx, _, err := bind.DeployContract(transactor, parsedABI, common.FromHex(artifact.Bytecode), sim.Client(), args...)
	if err != nil {
		t.Fatalf("deploy contract: %v", err)
	}
	sim.Commit()

	receipt, err := sim.Client().TransactionReceipt(context.Background(), tx.Hash())
	if err != nil || receipt.Status != 1 {
		t.Fatalf("deployment receipt failed: %v", err)
	}
	return addr, parsedABI
}

func TestE2EVaultFullLifecycle(t *testing.T) {
	t.Log("[ENV] Local HTTP + Go + simulated EVM; deposit 0.5 ETH, withdraw 0.2 ETH, remaining 0.3 ETH; no public Sepolia RPC")
	var simMu sync.Mutex
	deployerKey, _ := crypto.GenerateKey()
	deployerTransactor, _ := bind.NewKeyedTransactorWithChainID(deployerKey, big.NewInt(chain.SepoliaID))

	// Allocate 100 ETH to deployer
	sim := simulated.NewBackend(types.GenesisAlloc{
		deployerTransactor.From: {Balance: new(big.Int).Mul(big.NewInt(100), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))},
	}, func(nodeConf *node.Config, ethConf *ethconfig.Config) {
		ethConf.Genesis.Config.ChainID = big.NewInt(chain.SepoliaID)
	})
	defer sim.Close()

	// 1. Deploy compiled ETHVault contract
	artifactPath := filepath.Join("..", "..", "contracts", "artifacts", "ETHVault.json")
	vaultAddr, vaultABI := deployContract(t, sim, deployerTransactor, artifactPath)
	t.Logf("ETHVault deployed on local simulated EVM at %s", vaultAddr.Hex())

	// 2. Start simulated JSON-RPC server
	rpcServer := startSimulatedRPC(t, sim, &simMu)

	// 3. Initialize Go chain client connected to simulated RPC
	chainClient, err := chain.NewNetwork(chain.SepoliaID, []string{rpcServer.URL})
	if err != nil {
		t.Fatalf("chain.NewNetwork: %v", err)
	}
	defer chainClient.Close()

	// 4. Initialize Go wallet service configured with the deployed vault address
	walletDir := t.TempDir()
	walletService, err := wallet.NewService(chainClient, walletDir, 2, 1)
	if err != nil {
		t.Fatalf("wallet.NewService: %v", err)
	}
	defer walletService.Close()

	if err := walletService.SetVaultAddress(vaultAddr.Hex()); err != nil {
		t.Fatalf("SetVaultAddress: %v", err)
	}

	// 5. Start Go web server
	webHandler, err := web.New(chainClient, walletService)
	if err != nil {
		t.Fatalf("web.New: %v", err)
	}
	webServer := httptest.NewServer(webHandler)
	defer webServer.Close()

	httpClient := &http.Client{Timeout: 10 * time.Second}
	password := "StrongE2ETestPass123!"

	// Helper to send HTTP requests to web server
	doJSON := func(method, path string, body any, csrf string) (*http.Response, map[string]any) {
		var reqBody *bytes.Buffer
		if body != nil {
			b, _ := json.Marshal(body)
			reqBody = bytes.NewBuffer(b)
		} else {
			reqBody = bytes.NewBuffer(nil)
		}
		req, err := http.NewRequest(method, webServer.URL+path, reqBody)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Host = "127.0.0.1"
		if csrf != "" {
			req.Header.Set("X-Wallet-CSRF", csrf)
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		defer resp.Body.Close()
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return resp, out
	}

	// Step A: Check wallet status
	resp, statusData := doJSON("GET", "/api/wallet/status", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/wallet/status: code %d", resp.StatusCode)
	}
	csrfToken := statusData["csrfToken"].(string)

	// Step B: Create wallet
	resp, createData := doJSON("POST", "/api/wallet/create", map[string]string{"password": password}, csrfToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/wallet/create: code %d, err %v", resp.StatusCode, createData["error"])
	}
	userWalletAddr := common.HexToAddress(createData["address"].(string))
	t.Logf("Created test wallet address: %s", userWalletAddr.Hex())

	// Fund test wallet with 10 ETH from deployer
	simMu.Lock()
	deployerNonce, err := sim.Client().PendingNonceAt(context.Background(), deployerTransactor.From)
	if err != nil {
		t.Fatal(err)
	}
	fundTx, err := types.SignTx(
		types.NewTx(&types.DynamicFeeTx{
			ChainID:   big.NewInt(chain.SepoliaID),
			Nonce:     deployerNonce,
			GasTipCap: big.NewInt(1000000000),
			GasFeeCap: big.NewInt(2000000000),
			Gas:       21000,
			To:        &userWalletAddr,
			Value:     new(big.Int).Mul(big.NewInt(10), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)),
		}),
		types.LatestSignerForChainID(big.NewInt(chain.SepoliaID)),
		deployerKey,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := sim.Client().SendTransaction(context.Background(), fundTx); err != nil {
		t.Fatal(err)
	}
	sim.Commit()
	simMu.Unlock()

	// Step C: Check initial vault status via web API
	resp, vaultData := doJSON("GET", "/api/wallet/vault", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/wallet/vault: code %d", resp.StatusCode)
	}
	if vaultData["enabled"] != true || vaultData["contract"] != vaultAddr.Hex() || vaultData["balanceRaw"] != "0" {
		t.Fatalf("initial vault status mismatch: %#v", vaultData)
	}

	// Step D: Local simulated EVM deposit (0.5 ETH)
	depositAmount := "0.5"
	resp, quoteData := doJSON("POST", "/api/wallet/quote", map[string]string{
		"action": "vault_deposit",
		"amount": depositAmount,
		"to":     userWalletAddr.Hex(),
	}, csrfToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("deposit quote failed: code %d, err: %v", resp.StatusCode, quoteData["error"])
	}
	quoteID := quoteData["id"].(string)

	resp, sendData := doJSON("POST", "/api/wallet/send", map[string]string{
		"quoteId":  quoteID,
		"password": password,
	}, csrfToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("send deposit failed: code %d, err: %v", resp.StatusCode, sendData["error"])
	}
	depositTxHash := common.HexToHash(sendData["hash"].(string))
	t.Logf("Deposit transaction broadcast: %s", depositTxHash.Hex())

	// Verify the local simulated EVM execution receipt
	simMu.Lock()
	receipt, err := sim.Client().TransactionReceipt(context.Background(), depositTxHash)
	simMu.Unlock()
	if err != nil || receipt == nil || receipt.Status != 1 {
		t.Fatalf("deposit receipt status not 1: %v", err)
	}
	if len(receipt.Logs) != 1 {
		t.Fatalf("deposit expected one vault event, got %d", len(receipt.Logs))
	}

	// Verify ETHVault Deposited event emitted
	depositedTopic := crypto.Keccak256Hash([]byte("Deposited(address,uint256)"))
	foundDepositEvent := false
	for _, l := range receipt.Logs {
		if l.Address == vaultAddr && len(l.Topics) == 2 && l.Topics[0] == depositedTopic {
			accountInEvent := common.BytesToAddress(l.Topics[1].Bytes())
			if accountInEvent != userWalletAddr {
				t.Fatalf("event account mismatch: got %s, want %s", accountInEvent.Hex(), userWalletAddr.Hex())
			}
			amountInEvent := new(big.Int).SetBytes(l.Data)
			expectedDepositWei, _ := new(big.Int).SetString("500000000000000000", 10)
			if amountInEvent.Cmp(expectedDepositWei) != 0 {
				t.Fatalf("event amount mismatch: got %s, want %s", amountInEvent, expectedDepositWei)
			}
			foundDepositEvent = true
		}
	}
	if !foundDepositEvent {
		t.Fatal("Deposited event not emitted")
	}

	// Verify smart contract state directly on EVM
	simMu.Lock()
	boundVault := bind.NewBoundContract(vaultAddr, vaultABI, sim.Client(), sim.Client(), sim.Client())
	var balResult []any
	err = boundVault.Call(&bind.CallOpts{Context: context.Background()}, &balResult, "balanceOf", userWalletAddr)
	simMu.Unlock()
	if err != nil {
		t.Fatalf("call balanceOf: %v", err)
	}
	contractVaultBal := balResult[0].(*big.Int)
	expectedDepositWei, _ := new(big.Int).SetString("500000000000000000", 10)
	if contractVaultBal.Cmp(expectedDepositWei) != 0 {
		t.Fatalf("contract on-chain balance mismatch: got %s, want %s", contractVaultBal, expectedDepositWei)
	}

	// Verify web API returns updated vault balance
	resp, vaultData = doJSON("GET", "/api/wallet/vault", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/wallet/vault after deposit: code %d", resp.StatusCode)
	}
	if vaultData["balanceRaw"] != "500000000000000000" || vaultData["balance"] != "0.5" {
		t.Fatalf("vaultData after deposit mismatch: %#v", vaultData)
	}

	// Refresh wallet history to transition transaction to succeeded
	resp, historyData := doJSON("GET", "/api/wallet/history", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/wallet/history after deposit: code %d", resp.StatusCode)
	}
	assertHistory := func(data map[string]any, hash common.Hash, action, amount string) {
		t.Helper()
		if refreshError, _ := data["refreshError"].(string); refreshError != "" {
			t.Fatalf("history refresh failed: %s", refreshError)
		}
		transactions, ok := data["transactions"].([]any)
		if !ok {
			t.Fatal("history transactions missing")
		}
		for _, value := range transactions {
			transaction, ok := value.(map[string]any)
			if !ok || transaction["hash"] != hash.Hex() {
				continue
			}
			if transaction["action"] != action || transaction["amount"] != amount || transaction["state"] != "succeeded" {
				t.Fatalf("history mismatch for %s: action=%v amount=%v state=%v", hash.Hex(), transaction["action"], transaction["amount"], transaction["state"])
			}
			return
		}
		t.Fatalf("history missing transaction %s", hash.Hex())
	}
	assertHistory(historyData, depositTxHash, "vault_deposit", depositAmount)

	// Step E: Local simulated EVM withdrawal (0.2 ETH)
	withdrawAmount := "0.2"
	resp, withQuoteData := doJSON("POST", "/api/wallet/quote", map[string]string{
		"action": "vault_withdraw",
		"amount": withdrawAmount,
		"to":     userWalletAddr.Hex(),
	}, csrfToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("withdraw quote failed: code %d, err: %v", resp.StatusCode, withQuoteData["error"])
	}
	withQuoteID := withQuoteData["id"].(string)

	resp, withSendData := doJSON("POST", "/api/wallet/send", map[string]string{
		"quoteId":  withQuoteID,
		"password": password,
	}, csrfToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("send withdraw failed: code %d, err: %v", resp.StatusCode, withSendData["error"])
	}
	withTxHash := common.HexToHash(withSendData["hash"].(string))
	t.Logf("Withdraw transaction broadcast: %s", withTxHash.Hex())

	// Verify the local simulated EVM execution receipt
	simMu.Lock()
	withReceipt, err := sim.Client().TransactionReceipt(context.Background(), withTxHash)
	simMu.Unlock()
	if err != nil || withReceipt == nil || withReceipt.Status != 1 {
		t.Fatalf("withdraw receipt status not 1: %v", err)
	}
	if len(withReceipt.Logs) != 1 {
		t.Fatalf("withdraw expected one vault event, got %d", len(withReceipt.Logs))
	}

	// Verify ETHVault Withdrawn event emitted
	withdrawnTopic := crypto.Keccak256Hash([]byte("Withdrawn(address,uint256)"))
	foundWithdrawEvent := false
	for _, l := range withReceipt.Logs {
		if l.Address == vaultAddr && len(l.Topics) == 2 && l.Topics[0] == withdrawnTopic {
			accountInEvent := common.BytesToAddress(l.Topics[1].Bytes())
			if accountInEvent != userWalletAddr {
				t.Fatalf("withdraw event account mismatch: got %s, want %s", accountInEvent.Hex(), userWalletAddr.Hex())
			}
			amountInEvent := new(big.Int).SetBytes(l.Data)
			expectedWithdrawWei, _ := new(big.Int).SetString("200000000000000000", 10)
			if amountInEvent.Cmp(expectedWithdrawWei) != 0 {
				t.Fatalf("withdraw event amount mismatch: got %s, want %s", amountInEvent, expectedWithdrawWei)
			}
			foundWithdrawEvent = true
		}
	}
	if !foundWithdrawEvent {
		t.Fatal("Withdrawn event not emitted")
	}

	// Verify smart contract state directly on EVM: remaining balance = 0.3 ETH
	simMu.Lock()
	balResult = nil
	err = boundVault.Call(&bind.CallOpts{Context: context.Background()}, &balResult, "balanceOf", userWalletAddr)
	simMu.Unlock()
	if err != nil {
		t.Fatalf("call balanceOf: %v", err)
	}
	contractVaultBal = balResult[0].(*big.Int)
	expectedRemainingWei, _ := new(big.Int).SetString("300000000000000000", 10)
	if contractVaultBal.Cmp(expectedRemainingWei) != 0 {
		t.Fatalf("remaining balance mismatch: got %s, want %s", contractVaultBal, expectedRemainingWei)
	}

	// Verify web API returns remaining balance 0.3 ETH
	resp, vaultData = doJSON("GET", "/api/wallet/vault", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/wallet/vault after withdraw: code %d", resp.StatusCode)
	}
	if vaultData["balanceRaw"] != "300000000000000000" || vaultData["balance"] != "0.3" {
		t.Fatalf("vaultData after withdraw mismatch: %#v", vaultData)
	}

	// Refresh wallet history to finalize withdraw transaction in journal
	resp, historyData = doJSON("GET", "/api/wallet/history", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/wallet/history after withdraw: code %d", resp.StatusCode)
	}
	assertHistory(historyData, depositTxHash, "vault_deposit", depositAmount)
	assertHistory(historyData, withTxHash, "vault_withdraw", withdrawAmount)
	t.Log("[PASS] Local simulated EVM: both transaction hashes have successful receipts, matching events, balances and history")

	// Step F: Adversarial / Edge Cases:
	// 1. Over-withdrawal (trying to withdraw 0.4 ETH when only 0.3 ETH in vault)
	resp, overQuoteData := doJSON("POST", "/api/wallet/quote", map[string]string{
		"action": "vault_withdraw",
		"amount": "0.4",
		"to":     userWalletAddr.Hex(),
	}, csrfToken)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("over-withdrawal quote should fail, got %d", resp.StatusCode)
	}
	if overQuoteData["error"] != "合約餘額不足，請減少取回金額" {
		t.Fatalf("unexpected over-withdrawal error: %v", overQuoteData["error"])
	}
	t.Logf("Over-withdrawal correctly rejected: %v", overQuoteData["error"])

	// 2. Zero-amount deposit
	resp, zeroDepositData := doJSON("POST", "/api/wallet/quote", map[string]string{
		"action": "vault_deposit",
		"amount": "0",
		"to":     userWalletAddr.Hex(),
	}, csrfToken)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("zero deposit should fail, got %d", resp.StatusCode)
	}
	if zeroDepositData["error"] != "存入金額必須大於 0" {
		t.Fatalf("unexpected zero deposit error: %v", zeroDepositData["error"])
	}
	t.Logf("Zero deposit correctly rejected: %v", zeroDepositData["error"])

	// 3. Zero-amount withdrawal
	resp, zeroWithdrawData := doJSON("POST", "/api/wallet/quote", map[string]string{
		"action": "vault_withdraw",
		"amount": "0",
		"to":     userWalletAddr.Hex(),
	}, csrfToken)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("zero withdrawal should fail, got %d", resp.StatusCode)
	}
	if zeroWithdrawData["error"] != "取回金額必須大於 0" {
		t.Fatalf("unexpected zero withdrawal error: %v", zeroWithdrawData["error"])
	}
	t.Logf("Zero withdrawal correctly rejected: %v", zeroWithdrawData["error"])

	// 4. Wrong target address (cannot deposit to someone else's vault account)
	resp, wrongTargetData := doJSON("POST", "/api/wallet/quote", map[string]string{
		"action": "vault_deposit",
		"amount": "0.1",
		"to":     deployerTransactor.From.Hex(),
	}, csrfToken)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("wrong target quote should fail, got %d", resp.StatusCode)
	}
	if wrongTargetData["error"] != "合約操作的錢包地址不符，請重新預估" {
		t.Fatalf("unexpected wrong target error: %v", wrongTargetData["error"])
	}
	t.Logf("Wrong target address correctly rejected: %v", wrongTargetData["error"])

	// 5. Client cannot specify contract address for vault operations
	resp, clientContractData := doJSON("POST", "/api/wallet/quote", map[string]string{
		"action":   "vault_deposit",
		"amount":   "0.1",
		"to":       userWalletAddr.Hex(),
		"contract": deployerTransactor.From.Hex(),
	}, csrfToken)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("client contract specification should fail, got %d", resp.StatusCode)
	}
	if clientContractData["error"] != "合約地址由伺服器設定，無法在操作時變更" {
		t.Fatalf("unexpected client contract error: %v", clientContractData["error"])
	}
	t.Logf("Client contract address correctly rejected: %v", clientContractData["error"])

	// 6. CSRF defense: unauthorized request rejected with 403
	resp, _ = doJSON("POST", "/api/wallet/quote", map[string]string{
		"action": "vault_deposit",
		"amount": "0.1",
		"to":     userWalletAddr.Hex(),
	}, "invalid-csrf-token")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("invalid CSRF request should return 403, got %d", resp.StatusCode)
	}
	t.Log("[PASS] Invalid CSRF correctly rejected with 403 Forbidden")
}

func TestE2EVaultBrowser(t *testing.T) {
	t.Log("[ENV] Local Chromium + Go + simulated EVM; deposit 0.05 ETH, withdraw 0.02 ETH, remaining 0.03 ETH")
	required := os.Getenv("RUN_BROWSER_E2E") == "1"
	if testing.Short() {
		if required {
			t.Fatal("required browser test cannot run in short mode")
		}
		t.Skip("skipping browser test in short mode")
	}
	nodeBin, err := exec.LookPath("node")
	if err != nil {
		if required {
			t.Fatal("required browser test needs node in PATH")
		}
		t.Skip("node executable not found in PATH")
	}
	playwrightDir := filepath.Join("..", "browser", "node_modules", "playwright")
	if _, err := os.Stat(playwrightDir); err != nil {
		if required {
			t.Fatalf("required browser test needs playwright: %v", err)
		}
		t.Skip("playwright not installed in tests/browser")
	}

	var simMu sync.Mutex
	deployerKey, _ := crypto.GenerateKey()
	deployerTransactor, _ := bind.NewKeyedTransactorWithChainID(deployerKey, big.NewInt(chain.SepoliaID))

	sim := simulated.NewBackend(types.GenesisAlloc{
		deployerTransactor.From: {Balance: new(big.Int).Mul(big.NewInt(100), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))},
	}, func(nodeConf *node.Config, ethConf *ethconfig.Config) {
		ethConf.Genesis.Config.ChainID = big.NewInt(chain.SepoliaID)
	})
	defer sim.Close()

	artifactPath := filepath.Join("..", "..", "contracts", "artifacts", "ETHVault.json")
	vaultAddr, vaultABI := deployContract(t, sim, deployerTransactor, artifactPath)

	rpcServer := startSimulatedRPC(t, sim, &simMu)
	chainClient, err := chain.NewNetwork(chain.SepoliaID, []string{rpcServer.URL})
	if err != nil {
		t.Fatalf("chain.NewNetwork: %v", err)
	}
	defer chainClient.Close()

	walletDir := t.TempDir()
	walletService, err := wallet.NewService(chainClient, walletDir, 2, 1)
	if err != nil {
		t.Fatalf("wallet.NewService: %v", err)
	}
	defer walletService.Close()

	if err := walletService.SetVaultAddress(vaultAddr.Hex()); err != nil {
		t.Fatalf("SetVaultAddress: %v", err)
	}

	password := "StrongE2ETestPass123!"
	createRes, err := walletService.Create(password)
	if err != nil {
		t.Fatalf("Create wallet: %v", err)
	}
	userWalletAddr := common.HexToAddress(createRes.Address)

	// Fund wallet with 10 ETH
	simMu.Lock()
	deployerNonce, err := sim.Client().PendingNonceAt(context.Background(), deployerTransactor.From)
	if err != nil {
		t.Fatal(err)
	}
	fundTx, err := types.SignTx(
		types.NewTx(&types.DynamicFeeTx{
			ChainID:   big.NewInt(chain.SepoliaID),
			Nonce:     deployerNonce,
			GasTipCap: big.NewInt(1000000000),
			GasFeeCap: big.NewInt(2000000000),
			Gas:       21000,
			To:        &userWalletAddr,
			Value:     new(big.Int).Mul(big.NewInt(10), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)),
		}),
		types.LatestSignerForChainID(big.NewInt(chain.SepoliaID)),
		deployerKey,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := sim.Client().SendTransaction(context.Background(), fundTx); err != nil {
		t.Fatal(err)
	}
	sim.Commit()
	simMu.Unlock()

	webHandler, err := web.New(chainClient, walletService)
	if err != nil {
		t.Fatalf("web.New: %v", err)
	}
	webServer := httptest.NewServer(webHandler)
	defer webServer.Close()

	scriptPath := filepath.Join("..", "browser", "e2e.cjs")
	cmd := exec.Command(nodeBin, scriptPath)
	cmd.Env = append(os.Environ(),
		"E2E_BACKEND=simulated",
		"E2E_BASE_URL="+webServer.URL,
		"E2E_PASSWORD="+password,
		"E2E_VAULT="+vaultAddr.Hex(),
	)
	out, err := cmd.CombinedOutput()
	t.Log(string(out))
	if err != nil {
		t.Fatalf("browser e2e failed: %v", err)
	}

	// Check the browser's transactions independently of DOM text and API responses.
	history, err := walletService.History(context.Background())
	if err != nil {
		t.Fatalf("browser transaction history: %v", err)
	}
	if history.RefreshError != "" || len(history.Transactions) != 2 {
		t.Fatalf("expected two verified browser transactions: count=%d refreshError=%s", len(history.Transactions), history.RefreshError)
	}
	expected := map[string]struct {
		amount string
		wei    int64
		event  string
	}{
		"vault_deposit":  {"0.05", 50_000_000_000_000_000, "Deposited"},
		"vault_withdraw": {"0.02", 20_000_000_000_000_000, "Withdrawn"},
	}
	simMu.Lock()
	defer simMu.Unlock()
	for _, transaction := range history.Transactions {
		want, ok := expected[transaction.Action]
		if !ok || transaction.Amount != want.amount || transaction.State != "succeeded" {
			t.Fatalf("unexpected browser transaction: action=%s amount=%s state=%s", transaction.Action, transaction.Amount, transaction.State)
		}
		delete(expected, transaction.Action)
		receipt, err := sim.Client().TransactionReceipt(context.Background(), common.HexToHash(transaction.Hash))
		if err != nil || receipt == nil || receipt.Status != types.ReceiptStatusSuccessful {
			t.Fatalf("browser %s receipt failed: %v", transaction.Action, err)
		}
		if len(receipt.Logs) != 1 {
			t.Fatalf("browser %s expected one vault event, got %d", transaction.Action, len(receipt.Logs))
		}
		event := receipt.Logs[0]
		if event.Address != vaultAddr || len(event.Topics) != 2 || event.Topics[0] != vaultABI.Events[want.event].ID ||
			common.BytesToAddress(event.Topics[1].Bytes()) != userWalletAddr || new(big.Int).SetBytes(event.Data).Cmp(big.NewInt(want.wei)) != 0 {
			t.Fatalf("browser %s event does not match its contract, account and amount", transaction.Action)
		}
	}
	boundVault := bind.NewBoundContract(vaultAddr, vaultABI, sim.Client(), sim.Client(), sim.Client())
	var balance []any
	if err := boundVault.Call(&bind.CallOpts{Context: context.Background()}, &balance, "balanceOf", userWalletAddr); err != nil {
		t.Fatalf("browser final vault balance: %v", err)
	}
	if len(balance) != 1 || balance[0].(*big.Int).Cmp(big.NewInt(30_000_000_000_000_000)) != 0 {
		t.Fatal("browser final vault balance must be 0.03 ETH on the local simulated EVM")
	}
	t.Log("[PASS] Independent local EVM verification: browser deposit/withdraw receipts status=1, matching events and balanceOf=0.03 ETH")
}

func TestE2EVaultRevertDataPropagation(t *testing.T) {
	t.Log("[ENV] Revert data propagation from local simulated EVM; no public Sepolia RPC")
	var simMu sync.Mutex
	deployerKey, _ := crypto.GenerateKey()
	deployerTransactor, _ := bind.NewKeyedTransactorWithChainID(deployerKey, big.NewInt(chain.SepoliaID))

	sim := simulated.NewBackend(types.GenesisAlloc{
		deployerTransactor.From: {Balance: new(big.Int).Mul(big.NewInt(100), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))},
	}, func(nodeConf *node.Config, ethConf *ethconfig.Config) {
		ethConf.Genesis.Config.ChainID = big.NewInt(chain.SepoliaID)
	})
	defer sim.Close()

	artifactPath := filepath.Join("..", "..", "contracts", "artifacts", "ETHVault.json")
	vaultAddr, vaultABI := deployContract(t, sim, deployerTransactor, artifactPath)

	rpcServer := startSimulatedRPC(t, sim, &simMu)

	chainClient, err := chain.NewNetwork(chain.SepoliaID, []string{rpcServer.URL})
	if err != nil {
		t.Fatalf("chain.NewNetwork: %v", err)
	}
	defer chainClient.Close()

	ctx := context.Background()
	userAddr := deployerTransactor.From

	// 1. Zero-amount deposit revert (ZeroDeposit -> 0x56316e87)
	depData, err := vaultABI.Pack("deposit")
	if err != nil {
		t.Fatal(err)
	}
	err = wallet.SimulateVaultCall(ctx, chainClient, userAddr, vaultAddr, big.NewInt(0), depData)
	if err == nil || err.Error() != "存入金額必須大於 0" {
		t.Fatalf("expected ZeroDeposit revert decoded to '存入金額必須大於 0', got: %v", err)
	}

	// 2. Zero-amount withdrawal revert (ZeroWithdraw -> 0xb8cb6219)
	withdrawZeroData, err := vaultABI.Pack("withdraw", big.NewInt(0))
	if err != nil {
		t.Fatal(err)
	}
	err = wallet.SimulateVaultCall(ctx, chainClient, userAddr, vaultAddr, big.NewInt(0), withdrawZeroData)
	if err == nil || err.Error() != "取回金額必須大於 0" {
		t.Fatalf("expected ZeroWithdraw revert decoded to '取回金額必須大於 0', got: %v", err)
	}

	// 3. Insufficient balance withdrawal revert (InsufficientBalance -> 0xcf479181)
	withdrawOverData, err := vaultABI.Pack("withdraw", new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18)))
	if err != nil {
		t.Fatal(err)
	}
	err = wallet.SimulateVaultCall(ctx, chainClient, userAddr, vaultAddr, big.NewInt(0), withdrawOverData)
	if err == nil || err.Error() != "合約餘額不足，請減少取回金額" {
		t.Fatalf("expected InsufficientBalance revert decoded to '合約餘額不足，請減少取回金額', got: %v", err)
	}
}
