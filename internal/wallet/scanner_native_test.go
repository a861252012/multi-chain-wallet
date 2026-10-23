package wallet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestScannerFiltersLegacyNativeEmitterOnLoad(t *testing.T) {
	system := "0xfffffffffffffffffffffffffffffffffffffffe"
	for _, chainID := range []int64{chain.SepoliaID, chain.BaseSepoliaID} {
		t.Run(fmt.Sprint(chainID), func(t *testing.T) {
			client := mockRPC(t, func(method string, _ json.RawMessage) any {
				t.Errorf("unexpected RPC during stored progress read: %s", method)
				return nil
			}, chainID)
			dir := t.TempDir()
			svc, err := NewService(client, dir, 2, 1)
			if err != nil {
				t.Fatal(err)
			}
			defer svc.Close()
			stored := scanProgressDisk{Enabled: true, Start: 10, Next: 11, Finalized: 12, Tokens: []string{chain.SepoliaUSDC, system, chain.SepoliaWETH}}
			original, err := json.Marshal(stored)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "scan.json")
			if err := os.WriteFile(path, original, 0600); err != nil {
				t.Fatal(err)
			}
			want := stored.Tokens
			if chainID == chain.SepoliaID {
				want = []string{chain.SepoliaUSDC, chain.SepoliaWETH}
			}
			progress, err := svc.ScanProgress()
			if err != nil || !slices.Equal(progress.Tokens, want) || progress.Next != 11 || !progress.Enabled {
				t.Fatalf("stored token migration changed unrelated state: %+v, %v; want tokens %v", progress, err, want)
			}
			readBack, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(readBack, original) {
				t.Fatalf("read-only progress read rewrote the stored file: %s %v", readBack, err)
			}
			if _, err := svc.ConfigureScan(false, nil); err != nil {
				t.Fatal(err)
			}
			persisted, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var saved scanProgressDisk
			if err := json.Unmarshal(persisted, &saved); err != nil || !slices.Equal(saved.Tokens, want) || saved.Next != 11 || saved.Enabled {
				t.Fatalf("normal write did not preserve cleaned progress: %+v %v", saved, err)
			}
		})
	}
}

func TestScannerPersistsNativeHashWithoutTokenCandidate(t *testing.T) {
	header := &types.Header{Number: big.NewInt(10), Difficulty: big.NewInt(0), Time: 100, GasLimit: 30000000}
	hash := common.HexToHash("0x1234")
	other := common.HexToAddress("0x2222222222222222222222222222222222222222")
	contract := common.HexToAddress("0x3333333333333333333333333333333333333333")
	var owner common.Address
	client := mockRPC(t, func(method string, params json.RawMessage) any {
		switch method {
		case "eth_chainId":
			return "0xaa36a7"
		case "eth_getBlockByNumber":
			var args []any
			if err := json.Unmarshal(params, &args); err != nil {
				return err
			}
			if len(args) == 2 && args[1] == true {
				return map[string]any{"hash": header.Hash(), "transactions": []any{map[string]any{"hash": hash, "from": other.Hex(), "to": contract.Hex()}}}
			}
			return header
		case "eth_getBlockReceipts":
			native := &types.Log{
				Address: common.HexToAddress("0xfffffffffffffffffffffffffffffffffffffffe"),
				TxHash:  hash, BlockHash: header.Hash(), BlockNumber: 10, Index: 7,
				Topics: []common.Hash{crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")), common.BytesToHash(contract.Bytes()), common.BytesToHash(owner.Bytes())},
				Data:   common.LeftPadBytes(big.NewInt(100).Bytes(), 32),
			}
			token := *native
			token.Address, token.Index = common.HexToAddress(chain.SepoliaUSDC), 8
			return []*types.Receipt{{TxHash: hash, BlockHash: header.Hash(), BlockNumber: big.NewInt(10), Status: 1, Logs: []*types.Log{native, &token}}}
		default:
			t.Errorf("unexpected RPC %s", method)
			return nil
		}
	})
	dir := t.TempDir()
	svc, err := NewService(client, dir, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	created, err := svc.Create("test-password-123")
	if err != nil {
		t.Fatal(err)
	}
	owner = common.HexToAddress(created.Address)
	start := uint64(10)
	if _, err := svc.ConfigureScan(true, &start); err != nil {
		t.Fatal(err)
	}
	if err := svc.ScanOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	progress, err := svc.ScanProgress()
	if err != nil || progress.Next != 11 || !slices.Equal(progress.Tokens, []string{chain.SepoliaUSDC}) {
		t.Fatalf("wrong progress after mixed native/token scan: %+v %v", progress, err)
	}
	storedHashes, err := readActivityFile(filepath.Join(dir, "activity.json"))
	if err != nil || !slices.Equal(storedHashes, []string{hash.Hex()}) {
		t.Fatalf("native candidate hash lost or duplicated: %v %v", storedHashes, err)
	}
}
