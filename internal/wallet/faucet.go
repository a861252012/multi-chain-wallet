package wallet

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	sol "github.com/gagliardetto/solana-go"
)

// TestFaucet uses the existing account services so signing, nonce locks and journals remain shared.
type TestFaucet struct {
	mu            sync.Mutex
	Root          *Service
	Sources       map[int64]*Service
	Password      string
	TronSource    *TronService
	TronRecipient *TronService
	TronPassword  string
}

func (f *TestFaucet) ClaimTRX(ctx context.Context) (*TronRecord, error) {
	if !f.mu.TryLock() {
		return nil, errors.New("另一筆測試幣申請正在處理，請稍後再試")
	}
	defer f.mu.Unlock()
	if f.TronSource == nil || f.TronPassword == "" {
		return nil, errors.New("尚未設定 Shasta 測試幣發放帳戶，請使用外部水龍頭")
	}
	status, err := f.TronRecipient.Status()
	if err != nil {
		return nil, err
	}
	address := status.Address
	if address == "" {
		return nil, ErrWalletNotFound
	}
	f.TronSource.mu.Lock()
	var previous *TronRecord
	recent := 0
	for _, record := range f.TronSource.records {
		if time.Since(record.CreatedAt) >= time.Hour {
			continue
		}
		if record.State == "reverted" || record.State == "execution_failed" || record.State == "expired_unconfirmed" {
			continue
		}
		recent += 1
		if string(record.To) == address && record.Contract == "" && record.Amount == "5" {
			copy := tronRecordResponse(record)
			previous = &copy
		}
	}
	f.TronSource.mu.Unlock()
	if previous != nil {
		previous.Reused = true
		return previous, nil
	}
	if recent >= 20 {
		return nil, errors.New("本小時已達測試幣發放上限")
	}
	if _, err := f.TronSource.History(ctx); err != nil {
		return nil, err
	}
	quote, err := f.TronSource.Quote(ctx, address, "5", "", "")
	if err != nil {
		if errors.Is(err, ErrInsufficientFunds) {
			return nil, errors.New("Shasta 測試幣庫存不足，請先補充發放帳戶或使用外部水龍頭")
		}
		return nil, err
	}
	fee, ok := new(big.Rat).SetString(quote.FeeTRX)
	if !ok || fee.Sign() < 0 || fee.Cmp(big.NewRat(2, 1)) > 0 {
		return nil, errors.New("Shasta 發放手續費超過 2 TRX，請稍後再試")
	}
	return f.TronSource.Send(ctx, quote.ID, f.TronPassword)
}

func (f *TestFaucet) Claim(ctx context.Context, chainID int64, asset, address string) (*SendResponse, error) {
	if !f.mu.TryLock() {
		return nil, errors.New("另一筆測試幣申請正在處理，請稍後再試")
	}
	defer f.mu.Unlock()
	amount := ""
	switch chainID {
	case chain.SepoliaID:
		amount = "0.001"
	case chain.BaseSepoliaID, chain.OptimismSepoliaID, chain.ArbitrumSepoliaID:
		amount = "0.0001"
	case chain.PolygonAmoyID:
		amount = "0.1"
	default:
		return nil, ErrWrongChain
	}
	source := f.Sources[chainID]
	if source == nil || f.Password == "" {
		return nil, errors.New("尚未設定此網路的測試幣發放帳戶，請使用外部水龍頭")
	}
	if source.client.ChainID() != chainID {
		return nil, ErrWrongChain
	}
	req := &QuoteRequest{Action: "eth", To: address, Amount: amount}
	if asset == "usdc" && chainID == chain.SepoliaID {
		req.Action, req.Contract, req.Amount = "transfer", USDCAddress, "0.01"
	} else if asset != "native" {
		return nil, errors.New("此網路尚未提供這種測試幣")
	}
	accounts, err := f.Root.Accounts()
	if err != nil {
		return nil, err
	}
	owned := false
	for _, account := range accounts {
		if account.Address != "" && strings.EqualFold(account.Address, address) {
			owned = true
		}
	}
	if !owned {
		return nil, errors.New("僅能領取至此專案已建立的本機帳戶")
	}
	sender, err := source.keystore.Address()
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(sender, address) {
		return nil, errors.New("目前選用的是測試幣發放帳戶；請切換至你的測試錢包再領取")
	}
	// The persisted journal also covers unknown broadcasts and archived entries across restarts.
	recent := 0
	for _, tx := range source.journal.ListHistory() {
		created, err := time.Parse(time.RFC3339, tx.CreatedAt)
		if err != nil || time.Since(created) >= time.Hour {
			continue
		}
		if tx.State == "reverted" || tx.State == "execution_failed" || tx.State == "expired_unconfirmed" {
			continue
		}
		recent += 1
		if strings.EqualFold(tx.To, address) && tx.Action == req.Action && tx.Amount == req.Amount && (asset == "native" || tx.Symbol == "USDC") {
			return &SendResponse{Reused: true, Hash: tx.Hash, State: tx.State, To: tx.To, Amount: tx.Amount, Symbol: tx.Symbol, Action: tx.Action, CreatedAt: tx.CreatedAt}, nil
		}
	}
	if recent >= 20 {
		return nil, errors.New("此網路本小時已達發放上限，請稍後再試")
	}
	if _, err := source.History(ctx); err != nil {
		return nil, err
	}
	quote, err := source.Quote(ctx, req)
	if err != nil {
		if errors.Is(err, ErrInsufficientFunds) {
			return nil, errors.New("測試幣發放帳戶庫存不足，請先補充或使用外部水龍頭")
		}
		return nil, err
	}
	fee, ok := new(big.Rat).SetString(quote.MaxFeeETH)
	if ok && quote.RollupFeeETH != "" {
		rollup, valid := new(big.Rat).SetString(quote.RollupFeeETH)
		if !valid || rollup.Sign() < 0 {
			return nil, errors.New("無法核對測試幣發放手續費")
		}
		fee.Add(fee, rollup)
	}
	limit := "0.0001"
	if chainID == chain.PolygonAmoyID {
		limit = "0.1"
	} else if asset == "usdc" || req.Action == "transfer" {
		limit = "0.0005"
	}
	max, _ := new(big.Rat).SetString(limit)
	if !ok || fee.Sign() < 0 || fee.Cmp(max) > 0 {
		return nil, errors.New("目前預估手續費超過測試幣發放上限，請稍後再試")
	}
	return source.Send(ctx, quote.ID, f.Password)
}

func (s *SolanaService) RequestTestSOL(ctx context.Context) (*SolanaAirdropResponse, error) {
	status, err := s.Status()
	if err != nil {
		return nil, err
	}
	address := status.Address
	if address == "" {
		return nil, ErrWalletNotFound
	}
	if err := s.check(ctx); err != nil {
		return nil, err
	}
	key, err := sol.PublicKeyFromBase58(address)
	if err != nil {
		return nil, err
	}
	signature, err := s.rpc.RequestAirdrop(ctx, key, 10000000, "confirmed")
	if err != nil {
		return nil, errors.New("Devnet 空投未能確認：可能限流、庫存不足或連線逾時。請先查詢餘額，稍後再試或使用外部水龍頭")
	}
	return &SolanaAirdropResponse{Signature: signature.String(), State: "submitted", Amount: "0.01"}, nil
}
