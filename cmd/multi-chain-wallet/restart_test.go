package main

import (
	"encoding/json"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/a861252012/multi-chain-wallet/internal/wallet"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestAccountMaintenanceResumesWithoutPageVisit(t *testing.T) {
	if os.Getenv("WALLET_RESTART_TEST_HELPER") == "1" {
		os.Args = []string{"test-wallet"}
		if err := run(); err != nil {
			t.Fatal(err)
		}
		return
	}
	head := &types.Header{Number: big.NewInt(100), Difficulty: big.NewInt(0), GasLimit: 30000000, BaseFee: big.NewInt(1)}
	rpc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     any               `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		var result any
		switch req.Method {
		case "eth_chainId":
			result = "0xaa36a7"
		case "eth_getBlockByNumber":
			var full bool
			if len(req.Params) > 1 {
				json.Unmarshal(req.Params[1], &full)
			}
			if full {
				result = map[string]any{"hash": head.Hash(), "transactions": []any{}}
			} else {
				result = head
			}
		case "eth_getBlockReceipts":
			result = []any{}
		default:
			t.Errorf("unexpected RPC %s", req.Method)
		}
		json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	defer rpc.Close()
	dir := t.TempDir()
	client, err := chain.New(rpc.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	root, err := wallet.NewService(client, dir, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	account, err := root.AddAccount("Archived scanner", "")
	if err != nil {
		t.Fatal(err)
	}
	childDir := filepath.Join(dir, "accounts", account.ID)
	child, err := wallet.NewService(client, childDir, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer child.Close()
	if _, err := child.Create("restart-test-password"); err != nil {
		t.Fatal(err)
	}
	start := uint64(100)
	if _, err := child.ConfigureScan(true, &start); err != nil {
		t.Fatal(err)
	}
	if _, err := root.UpdateAccount(account.ID, "Archived scanner", true); err != nil {
		t.Fatal(err)
	}
	child.Close()
	root.Close()
	faucetPath := filepath.Join(t.TempDir(), "faucet.json")
	data, _ := json.Marshal(faucetConfig{AccountID: account.ID, Password: "restart-test-password"})
	if err := os.WriteFile(faucetPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	logFile, err := os.Create(filepath.Join(t.TempDir(), "server.log"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { logFile.Close() })
	// No inherited credentials, real wallet paths, or public RPC endpoints.
	startServer := func(withFaucet bool) func() {
		command := exec.Command(executable, "-test.run=^TestAccountMaintenanceResumesWithoutPageVisit$")
		command.Env = []string{"WALLET_RESTART_TEST_HELPER=1", "WALLET_DIR=" + dir, "PORT=" + strconv.Itoa(port), "SEPOLIA_RPC_URL=" + rpc.URL, "SOLANA_DEVNET_RPC_URL=" + rpc.URL, "TRON_SHASTA_RPC_URL=" + rpc.URL}
		for _, network := range networkDefinitions {
			command.Env = append(command.Env, network.envPrefix+"_RPC_URL="+rpc.URL)
		}
		if withFaucet {
			command.Env = append(command.Env, "TEST_FAUCET_CONFIG="+faucetPath)
		}
		command.Stdout, command.Stderr = logFile, logFile
		if err := command.Start(); err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { done <- command.Wait() }()
		var once sync.Once
		stop := func() {
			once.Do(func() {
				command.Process.Signal(os.Interrupt)
				select {
				case err := <-done:
					if err != nil {
						content, _ := os.ReadFile(logFile.Name())
						t.Errorf("server: %v\n%s", err, content)
					}
				case <-time.After(10 * time.Second):
					command.Process.Kill()
					<-done
					t.Error("server did not shut down")
				}
			})
		}
		t.Cleanup(stop)
		return stop
	}
	stop := startServer(false)
	httpClient := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(10 * time.Second)
	for {
		data, err := os.ReadFile(filepath.Join(childDir, "scan.json"))
		var scan wallet.ScanProgress
		if err == nil && json.Unmarshal(data, &scan) == nil && scan.Next == 101 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("archived account scan never resumed before any HTTP request")
		}
		time.Sleep(20 * time.Millisecond)
	}
	// The first request must reuse the startup service instead of acquiring its flock again.
	response, err := httpClient.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/accounts/" + account.ID + "/api/wallet/scan")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("account route failed after eager startup: %d", response.StatusCode)
	}
	// A faucet account must also reuse the loaded services on the next restart.
	stop()
	startServer(true)
	deadline = time.Now().Add(5 * time.Second)
	for {
		response, err := httpClient.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/healthz")
		if err == nil {
			response.Body.Close()
			if response.StatusCode == 200 {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("faucet account could not reuse the startup service")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
