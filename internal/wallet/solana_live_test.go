package wallet

import (
	"context"
	"os"
	"testing"
	"time"

	sol "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/google/uuid"
)

// Explicit opt-in only. Uses a disposable Devnet-only wallet, never the runtime wallet volume.
func TestSolanaDevnetSendAcceptance(t *testing.T) {
	if os.Getenv("MULTI_CHAIN_WALLET_SOLANA_LIVE_SEND") != "1" {
		t.Skip("requires explicit Devnet faucet/send opt-in")
	}
	s, err := NewSolanaService("https://api.devnet.solana.com", t.TempDir(), 262144)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	if err := s.check(ctx); err != nil {
		t.Fatal(err)
	}
	password := uuid.NewString()
	created, err := s.Create("", password)
	if err != nil {
		t.Fatal(err)
	}
	address, err := sol.PublicKeyFromBase58(created.Address)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("disposable Devnet address: %s", address)
	signature, err := s.rpc.RequestAirdrop(ctx, address, 100000000, rpc.CommitmentFinalized)
	if err != nil {
		t.Skip("Devnet faucet unavailable/rate-limited; no signed outgoing acceptance completed")
	}
	t.Logf("faucet signature: %s", signature)
	for {
		balance, err := s.rpc.GetBalance(ctx, address, rpc.CommitmentConfirmed)
		if err == nil && balance != nil && balance.Value >= 10000000 {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("faucet confirmation timed out")
		case <-time.After(5 * time.Second):
		}
	}
	q, err := s.Quote(ctx, address.String(), "0.000001")
	if err != nil {
		t.Fatal(err)
	}
	sent, err := s.Send(ctx, q.ID, password)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("outgoing signature: %s ; https://explorer.solana.com/tx/%s?cluster=devnet", sent.Signature, sent.Signature)
	for {
		history, err := s.History(ctx)
		if err == nil && len(history) == 1 && history[0].Finalized {
			if history[0].State != "finalized" {
				t.Fatalf("transaction failed: %s", history[0].State)
			}
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("transaction not finalized before timeout; retain signature for read-only verification")
		case <-time.After(5 * time.Second):
		}
	}
	t.Log("native SOL self-transfer finalized; test lamports plus fee checked by bound quote")
}
