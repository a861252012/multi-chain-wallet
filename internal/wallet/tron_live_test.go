package wallet

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestTronShastaReadOnly(t *testing.T) {
	if os.Getenv("MULTI_CHAIN_WALLET_LIVE_TRON") != "1" {
		t.Skip("opt-in read-only Shasta verification")
	}
	s, err := NewTronService("https://api.shasta.trongrid.io", "", t.TempDir(), 2)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Public derivation vector address. No account is created, funded, unlocked or signed.
	balance, err := s.Balance(ctx, "TUEZSdKsoDHQMeZwihtdoBiN46zxhGWYdH")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Shasta genesis verified; public-address balance/resources=%v; no signing or broadcast", balance)
}
