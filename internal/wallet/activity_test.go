package wallet

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
)

func TestActivityIndexDeduplicatesAndSurvivesRestart(t *testing.T) {
	s, _ := guardedFixture(t)
	hash := "0x" + strings.Repeat("a", 64)
	if n, err := s.addActivityHashes([]string{hash, hash}); err != nil || n != 1 {
		t.Fatalf("first index %d %v", n, err)
	}
	if n, err := s.addActivityHashes([]string{hash}); err != nil || n != 0 {
		t.Fatalf("duplicate index %d %v", n, err)
	}
	stat, err := os.Stat(filepath.Join(s.walletDir, "activity.json"))
	if err != nil || stat.Mode().Perm() != 0600 {
		t.Fatal("unsafe index permissions")
	}
	s.Close()
	next, err := NewService(s.client, s.walletDir, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	ids, err := next.activityHashes()
	if err != nil || len(ids) != 1 || ids[0] != hash {
		t.Fatalf("restart %v %v", ids, err)
	}
	result, err := next.Activity(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Incomplete || len(result.Totals) != 0 || result.TotalTransactions != 1 {
		t.Fatalf("unknown RPC counted as settled %+v", result)
	}
	if _, err := next.Activity(context.Background(), 2); err == nil {
		t.Fatal("out of range page accepted")
	}
}

func TestChainActivityBoundaryValidatesEnumsAndAmounts(t *testing.T) {
	valid := &chain.Activity{
		Hash: "0x" + strings.Repeat("a", 64), State: "succeeded",
		Movements: []chain.Movement{{
			Kind: "receive", Asset: "ETH", Raw: "1000000000000000001",
			Counterparty: "0x" + strings.Repeat("1", 40),
		}},
	}
	domain, err := chainActivityToDomain(valid)
	if err != nil || domain.state != activitySucceeded || len(domain.movements) != 1 || domain.movements[0].amount.String() != valid.Movements[0].Raw {
		t.Fatalf("valid chain activity conversion failed: %+v %v", domain, err)
	}
	invalidKind := *valid
	invalidKind.Movements = append([]chain.Movement{}, valid.Movements...)
	invalidKind.Movements[0].Kind = "mint"
	if _, err := chainActivityToDomain(&invalidKind); err == nil {
		t.Fatal("unknown movement kind was accepted")
	}
	invalidAmount := *valid
	invalidAmount.Movements = append([]chain.Movement{}, valid.Movements...)
	invalidAmount.Movements[0].Raw = "1.5"
	if _, err := chainActivityToDomain(&invalidAmount); err == nil {
		t.Fatal("non-integer movement amount was accepted")
	}
}
