package wallet

import (
	"context"
	"testing"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
)

func TestEscrowConfigurationAndAccountInheritance(t *testing.T) {
	client, err := chain.New("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	service, err := NewService(client, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	status, err := service.EscrowStatus(context.Background())
	if err != nil || status.Enabled {
		t.Fatalf("disabled status=%+v err=%v", status, err)
	}
	contract := "0x1111111111111111111111111111111111111111"
	token := "0x2222222222222222222222222222222222222222"
	for _, args := range [][2]string{{contract, ""}, {"", token}, {contract, contract}, {"not-address", token}} {
		if service.SetEscrow(args[0], args[1]) == nil {
			t.Fatal("accepted invalid config", args)
		}
	}
	if err := service.SetEscrow(contract, token); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create("TestEscrowPassword123"); err != nil {
		t.Fatal(err)
	}
	account, err := service.AddAccount("Recipient", "TestEscrowPassword123")
	if err != nil {
		t.Fatal(err)
	}
	child, err := NewAccountService(client, service, account.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer child.Close()
	if child.escrowAddress != contract || child.escrowToken != token {
		t.Fatal("account lost escrow config")
	}
	other, err := chain.NewNetwork(84532, []string{"http://127.0.0.1:1"})
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	otherService, err := NewService(other, t.TempDir(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer otherService.Close()
	if err := otherService.SetEscrow(contract, token); err != nil {
		t.Fatal(err)
	}
	if otherService.escrowAddress != "" || otherService.escrowToken != "" {
		t.Fatal("enabled on wrong network")
	}
}

func TestEscrowQuoteRequestBoundary(t *testing.T) {
	to := "0x1111111111111111111111111111111111111111"
	for _, action := range []string{"escrow_fund", "escrow_release", "escrow_refund"} {
		request := QuoteRequest{Action: action, To: to, OrderID: "order-001", Buyer: to, Amount: "1.25"}
		parsed, err := ParseQuoteRequest(&request)
		if err != nil || parsed.OrderID != "order-001" || parsed.Buyer != EVMAddress(to) {
			t.Fatal("valid escrow request", err)
		}
		for _, field := range []string{"contract", "token", "amountRaw", "reference", "buyer"} {
			bad := request
			switch field {
			case "contract":
				bad.Contract = to
			case "token":
				bad.TokenOut = to
			case "amountRaw":
				bad.AmountRaw = "1250000"
			case "reference":
				bad.OrderID = "a b"
			case "buyer":
				bad.Buyer = "not-address"
			}
			if _, err := ParseQuoteRequest(&bad); err == nil {
				t.Fatalf("accepted %s for %s", field, action)
			}
		}
	}
	if _, err := ParseQuoteRequest(&QuoteRequest{Action: "eth", To: to, Amount: "1", OrderID: "order-001"}); err == nil {
		t.Fatal("ordinary action accepted order fields")
	}
}
