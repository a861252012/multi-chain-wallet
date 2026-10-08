package wallet

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// Service manages keys, quotes, transactions, and the journal.
type Service struct {
	historyOffset              int
	scanMu                     sync.Mutex
	scanRunMu                  sync.Mutex
	catalog                    *Service
	client                     *chain.Client
	keystore                   *KeystoreManager
	quotes                     *QuoteStore
	journal                    *JournalManager
	csrfToken                  string
	sendMu                     sync.Mutex
	activityMu                 sync.Mutex
	activityGeneration         uint64         // guarded by sendMu
	activityIndex              *activityIndex // guarded by activityMu; published indexes are immutable
	historyMu                  sync.Mutex
	walletDir                  string
	vaultAddress               string
	escrowAddress              string
	escrowToken                string
	lockFile                   *os.File
	storageFault               atomic.Bool
	lastActivityArchiveNano    int64
	writeActivityArchive       func(string, []byte, os.FileMode) error
	scanBackoffUntil           time.Time
	scanBackoffDuration        time.Duration
	scanGeneration             uint64
	maintenanceTickInterval    time.Duration
	maintenanceHistoryInterval time.Duration
	scanBackoffInitial         time.Duration
}

func NewService(client *chain.Client, walletDir string, scryptParams ...int) (*Service, error) {
	if walletDir == "" {
		walletDir = "./data/wallet"
	}

	var n, p int
	if len(scryptParams) >= 2 {
		n = scryptParams[0]
		p = scryptParams[1]
	}

	if err := os.MkdirAll(walletDir, 0700); err != nil {
		return nil, err
	}
	if err := os.Chmod(walletDir, 0700); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(filepath.Join(walletDir, ".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		return nil, errors.New("錢包資料夾正由另一個程序使用")
	}
	success := false
	defer func() {
		if !success {
			lock.Close()
		}
	}()
	km := NewKeystoreManager(walletDir, n, p)
	jm, err := NewJournalManager(walletDir, client.ChainID())
	if err != nil {
		return nil, err
	}

	csrfBytes := make([]byte, 32)
	if _, err := rand.Read(csrfBytes); err != nil {
		return nil, err
	}

	success = true
	return &Service{
		client:    client,
		keystore:  km,
		quotes:    NewQuoteStore(),
		journal:   jm,
		csrfToken: hex.EncodeToString(csrfBytes),
		walletDir: walletDir,
		lockFile:  lock,
	}, nil
}

func (s *Service) Close() error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	s.historyMu.Lock()
	defer s.historyMu.Unlock()
	if s.lockFile == nil {
		return nil
	}
	err := s.lockFile.Close()
	s.lockFile = nil
	return err
}

func (s *Service) CSRFToken() string {
	return s.csrfToken
}

// ChainID returns the connected client's chain ID.
func (s *Service) ChainID() int64 {
	return s.client.ChainID()
}

func sendResponseFromRecord(record *JournalRecord, state JournalState) *SendResponse {
	return &SendResponse{
		EscrowAction: string(record.EscrowAction),
		Hash:         string(record.Hash),
		State:        string(state),
		To:           string(record.To),
		Amount:       record.Amount,
		Symbol:       record.Symbol,
		Action:       string(record.Action),
		CreatedAt:    record.CreatedAt.Format(time.RFC3339),
	}
}

func (s *Service) Status() (*WalletInfo, error) {
	addr, err := s.keystore.Address()
	if err != nil && !errors.Is(err, ErrWalletNotFound) {
		return nil, err
	}
	info := &WalletInfo{
		ChainID:   s.client.ChainID(),
		Exists:    err == nil,
		Address:   addr,
		Path:      "m/44'/60'/0'/0/0",
		CSRFToken: s.csrfToken,
		Exchange: map[string]string{
			"weth":   common.HexToAddress(WETHAddress).Hex(),
			"usdc":   common.HexToAddress(USDCAddress).Hex(),
			"router": common.HexToAddress(RouterAddress).Hex(),
		},
	}
	if s.client.ChainID() != chain.SepoliaID {
		info.Exchange = map[string]string{}
	}
	return info, nil
}

func (s *Service) Create(password string) (*CreateResponse, error) {
	return s.keystore.Create(password)
}

func (s *Service) Import(mnemonic, password string) (*ImportResponse, error) {
	return s.keystore.Import(mnemonic, password)
}

func (s *Service) Backup(password string) (json.RawMessage, error) {
	return s.keystore.Backup(password)
}

func (s *Service) ImportKeystore(data json.RawMessage, password, newPassword string) (*ImportResponse, error) {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	return s.keystore.ImportKeystore(data, password, newPassword)
}

func (s *Service) ChangePassword(password, newPassword string) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	return s.keystore.ChangePassword(password, newPassword)
}

func (s *Service) Token(ctx context.Context, contract, spender string) (*TokenInfo, error) {
	if !s.keystore.Exists() {
		return nil, ErrWalletNotFound
	}
	addrStr, err := s.keystore.Address()
	if err != nil {
		return nil, err
	}
	ownerAddr := common.HexToAddress(addrStr)

	contractAddr, err := ValidateAddress(contract)
	if err != nil {
		return nil, err
	}

	sym, dec, err := QueryERC20Metadata(ctx, s.client, contractAddr)
	if err != nil {
		return nil, err
	}

	bal, err := QueryERC20BalanceOf(ctx, s.client, contractAddr, ownerAddr)
	if err != nil {
		return nil, err
	}

	info := &TokenInfo{
		Contract:   contractAddr.Hex(),
		Symbol:     sym,
		Decimals:   dec,
		Balance:    FormatUnits(bal, dec),
		BalanceRaw: bal.String(),
	}
	trustedSymbol, trustedDecimals, trusted := trustedEVMToken(s.client.ChainID(), contractAddr)
	if trusted {
		if sym != trustedSymbol || dec != trustedDecimals {
			return nil, errors.New("RPC 回傳的代幣資料與內建登錄不符")
		}
		info.Trusted = true
	}
	if spender != "" {
		address, err := ValidateAddress(spender)
		if err != nil {
			return nil, err
		}
		allowance, err := QueryERC20Allowance(ctx, s.client, contractAddr, ownerAddr, address)
		if err != nil {
			return nil, err
		}
		info.Spender, info.Allowance, info.AllowanceRaw = address.Hex(), FormatUnits(allowance, dec), allowance.String()
	}
	return info, nil
}

func (s *Service) Quote(ctx context.Context, req *QuoteRequest) (*QuoteResponse, error) {
	command, err := ParseQuoteRequest(req)
	if err != nil {
		return nil, err
	}
	return s.QuoteCommand(ctx, command)
}

func (s *Service) QuoteCommand(ctx context.Context, command QuoteCommand) (*QuoteResponse, error) {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	if s.storageFault.Load() {
		return nil, errors.New("交易儲存發生錯誤，請修復磁碟後重啟錢包")
	}
	if !s.keystore.Exists() {
		return nil, ErrWalletNotFound
	}

	if command.Action == ActionSpeedup || command.Action == ActionCancel {
		bound, err := s.replacementQuote(ctx, command)
		if err != nil {
			return nil, err
		}
		if err := s.rollupFee(ctx, bound, true); err != nil {
			return nil, err
		}
		if err := s.quotes.Add(bound); err != nil {
			return nil, err
		}
		return bound.ToResponse(), nil
	}

	if s.client.ChainID() != chain.SepoliaID {
		switch command.Action {
		case ActionWrap, ActionUnwrap, ActionSwap:
			return nil, errors.New("此網路支援原生幣與 ERC-20 收付款；兌換目前只配置 Ethereum Sepolia")
		case ActionVaultDeposit, ActionVaultWithdraw:
			return nil, errors.New("此合約目前僅支援 Ethereum Sepolia 測試網")
		}
	}

	if command.Action == ActionVaultDeposit || command.Action == ActionVaultWithdraw {
		if s.vaultAddress == "" {
			return nil, errors.New("尚未設定合約地址，暫時無法操作")
		}
		command.Contract = EVMAddress(s.vaultAddress)
	}
	if isEscrowAction(command.Action) {
		if s.client.ChainID() != chain.SepoliaID || s.escrowAddress == "" {
			return nil, errors.New("此環境尚未開放付款託管")
		}
		command.Contract, command.TokenOut = EVMAddress(s.escrowAddress), EVMAddress(s.escrowToken)
	}
	// Single outstanding tx constraint: block new quotes while a transaction is in flight
	finalized, err := s.checkInFlight(ctx)
	if err != nil {
		return nil, err
	}

	addrStr, err := s.keystore.Address()
	if err != nil {
		return nil, err
	}
	fromAddr := common.HexToAddress(addrStr)

	bound, err := CreateQuote(ctx, s.client, fromAddr, command)
	if err != nil {
		return nil, err
	}
	if finalized != nil && bound.Nonce < finalized.nonce {
		return nil, ErrNonceMismatch
	}

	if command.Action == ActionETH {
		bound.Symbol = s.client.NativeSymbol()
	}
	if err := s.rollupFee(ctx, bound, true); err != nil {
		return nil, err
	}
	if err := s.quotes.Add(bound); err != nil {
		return nil, err
	}

	return bound.ToResponse(), nil
}

func (s *Service) Send(ctx context.Context, quoteID, password string) (result *SendResponse, err error) {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	if s.storageFault.Load() {
		return nil, errors.New("交易儲存發生錯誤，請修復磁碟後重啟錢包")
	}
	// Double-click / retry of same quoteID: return existing record without re-signing
	if existing := s.journal.FindByQuoteID(quoteID); existing != nil {
		return sendResponseFromRecord(existing, existing.State), nil
	}

	// Only failures before durable preparation are safe for a caller to abandon.
	// Storage faults and retries of an existing quote are handled above.
	prepared := false
	defer func() {
		if err != nil && !prepared {
			err = &SendRejectedError{Err: err}
		}
	}()

	quote, err := s.quotes.Get(quoteID)
	if err != nil {
		return nil, err
	}

	if quote.ReplacementHash == "" {
		finalized, err := s.checkInFlight(ctx)
		if err != nil {
			return nil, err
		}
		if finalized != nil && quote.Nonce < finalized.nonce {
			return nil, ErrNonceMismatch
		}
	}

	// Decrypt keystore
	key, err := s.keystore.DecryptKey(password)
	if err != nil {
		return nil, err
	}
	defer wipePrivateKey(key.PrivateKey)

	if key.Address != quote.From {
		return nil, errors.New("金鑰地址與報價不符")
	}
	if !time.Now().Before(quote.ExpiresAt) {
		return nil, ErrQuoteExpired
	}
	// Verify chain & nonce
	if err := s.client.CheckNetwork(ctx); err != nil {
		return nil, err
	}

	currentNonce, err := s.client.PendingNonceAt(ctx, quote.From)
	if quote.ReplacementHash != "" {
		if len(s.journal.NonceRecords(quote.Nonce)) != quote.ReplacementCount {
			return nil, ErrQuoteExpired
		}
		currentNonce, err = s.client.NonceAt(ctx, quote.From)
		if err == nil && (currentNonce > quote.Nonce || s.journal.NonceMined(quote.Nonce)) {
			return nil, ErrNonceMismatch
		}
		currentNonce = quote.Nonce
	}
	if err != nil {
		return nil, err
	}
	if currentNonce != quote.Nonce {
		return nil, ErrNonceMismatch
	}

	if err := s.rollupFee(ctx, quote, false); err != nil {
		return nil, err
	}

	// Verify ETH balance
	currentBal, err := s.client.BalanceAt(ctx, quote.From, nil)
	if err != nil {
		return nil, err
	}
	if currentBal.Cmp(quote.TotalETHWei) < 0 {
		return nil, ErrInsufficientFunds
	}

	// If token transfer, verify token balance
	if quote.Action == "transfer" {
		tokBal, err := QueryERC20BalanceOf(ctx, s.client, quote.Contract, quote.From)
		if err != nil {
			return nil, err
		}
		if tokBal.Cmp(quote.AmountRaw) < 0 {
			return nil, errors.New("代幣餘額不足以支付轉帳金額")
		}
	}

	head, err := s.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}
	if head.BaseFee == nil || head.BaseFee.Cmp(quote.MaxFeePerGas) > 0 {
		return nil, errors.New("目前基本費用超過報價上限，請重新預估")
	}
	if quote.Action == "approve" {
		allowance, err := QueryERC20Allowance(ctx, s.client, quote.Contract, quote.From, quote.To)
		if err != nil {
			return nil, err
		}
		if allowance.Sign() > 0 && quote.AmountRaw.Sign() > 0 {
			return nil, ErrApprovalRace
		}
	}
	if isEscrowAction(quote.Action) || quote.Escrow != nil {
		if s.client.ChainID() != chain.SepoliaID || s.escrowAddress == "" || quote.Escrow == nil || !isEscrowAction(quote.Escrow.Action) || (quote.Action != ActionSpeedup && quote.Action != quote.Escrow.Action) || quote.Contract != common.HexToAddress(s.escrowAddress) || quote.Escrow.Token != s.escrowToken {
			return nil, errors.New("託管設定已變更，請重新預估")
		}
		if err := verifyEscrow(ctx, s.client, quote.Contract, common.HexToAddress(s.escrowToken)); err != nil {
			return nil, err
		}
		if err := simulateEscrow(ctx, s.client, quote.From, quote.TxTo, quote.Data); err != nil {
			return nil, err
		}
	}
	switch quote.Action {
	case "wrap", "unwrap", "swap":
		if err := RecheckExchange(ctx, s.client, quote); err != nil {
			return nil, err
		}
	case ActionVaultDeposit, ActionVaultWithdraw:
		if s.client.ChainID() != chain.SepoliaID || s.vaultAddress == "" || quote.Contract != common.HexToAddress(s.vaultAddress) {
			return nil, errors.New("合約設定已變更，請重新預估")
		}
		if err := VerifyContractBytecode(ctx, s.client, quote.Contract); err != nil {
			return nil, err
		}
		if quote.Action == ActionVaultWithdraw {
			vaultBal, err := QueryVaultBalanceOf(ctx, s.client, quote.Contract, quote.From)
			if err != nil {
				return nil, err
			}
			if vaultBal.Cmp(quote.AmountRaw) < 0 {
				return nil, errors.New("合約餘額不足，請減少取回金額")
			}
		}
		if err := SimulateVaultCall(ctx, s.client, quote.From, quote.TxTo, quote.TxValue, quote.Data); err != nil {
			return nil, err
		}
	}
	if quote.Action == "transfer" || quote.Action == "approve" || quote.ReplacementERC20 {
		symbol, decimals, err := QueryERC20Metadata(ctx, s.client, quote.Contract)
		if err != nil {
			return nil, err
		}
		if symbol != quote.Symbol || decimals != quote.Decimals {
			return nil, errors.New("代幣資料已變更，請重新預估")
		}
		if err := SimulateERC20Call(ctx, s.client, quote.From, quote.TxTo, quote.Data); err != nil {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, chain.ErrTimeout
	}
	if !time.Now().Before(quote.ExpiresAt) {
		return nil, ErrQuoteExpired
	}
	if quote.ReplacementHash != "" || isEscrowAction(quote.Action) {
		gas, err := s.client.EstimateGas(ctx, ethereum.CallMsg{From: quote.From, To: &quote.TxTo, Value: quote.TxValue, Data: quote.Data, GasFeeCap: quote.MaxFeePerGas, GasTipCap: quote.MaxPriorityFeePerGas})
		if err != nil {
			return nil, err
		}
		if gas > quote.GasLimit {
			return nil, errors.New("交易需要更多 Gas，請重新預估")
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, chain.ErrTimeout
	}
	if !time.Now().Before(quote.ExpiresAt) {
		return nil, ErrQuoteExpired
	}
	// Build EIP-1559 DynamicFeeTx
	dynamicTx := &types.DynamicFeeTx{
		ChainID:   big.NewInt(s.client.ChainID()),
		Nonce:     quote.Nonce,
		GasTipCap: quote.MaxPriorityFeePerGas,
		GasFeeCap: quote.MaxFeePerGas,
		Gas:       quote.GasLimit,
		To:        &quote.TxTo,
		Value:     quote.TxValue,
		Data:      quote.Data,
	}
	tx := types.NewTx(dynamicTx)
	signer := types.LatestSignerForChainID(big.NewInt(s.client.ChainID()))
	signedTx, err := types.SignTx(tx, signer, key.PrivateKey)
	if err != nil {
		return nil, err
	}

	txHash := signedTx.Hash().Hex()
	rawBytes, err := signedTx.MarshalBinary()
	if err != nil {
		return nil, err
	}
	signedRawHex := hexutil.Encode(rawBytes)

	now := time.Now().UTC()
	record := &JournalRecord{
		Hash:      TransactionHash(txHash),
		QuoteID:   quote.ID,
		State:     "pending",
		To:        EVMAddress(quote.To.Hex()),
		Amount:    quote.Amount,
		AmountRaw: quote.AmountRaw.String(),
		Symbol:    quote.Symbol,
		Action:    quote.Action,
		Nonce:     quote.Nonce,
		SignedRaw: signedRawHex,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if quote.Escrow != nil {
		record.OrderID = OrderReference(quote.Escrow.OrderID)
	}

	// PREPARE / SIGN / BROADCAST:
	// Must persist atomically before any RPC broadcast! No send if persistence fails!
	prepared = true // A persistence failure may have already written the record.
	baseVersion, err := s.journal.AppendAtomic(record)
	if err != nil {
		if !errors.Is(err, ErrJournalFull) {
			s.storageFault.Store(true)
		}
		return nil, err
	}

	// Broadcast to RPC
	broadcastErr := s.client.SendTransaction(ctx, signedTx)
	newState := JournalSubmitted
	var errStr string
	if broadcastErr != nil {
		newState = JournalBroadcastUnknown
		errStr = broadcastErr.Error()
	}

	updated, err := s.journal.UpdateStateAtomicIfVersion(txHash, baseVersion, string(newState), "", "", errStr)
	if err != nil {
		s.storageFault.Store(true)
		return nil, err
	}
	s.quotes.Remove(quote.ID)
	if !updated {
		latest := s.journal.FindByHash(txHash)
		if latest != nil {
			return sendResponseFromRecord(latest, latest.State), nil
		}
	}
	return sendResponseFromRecord(record, newState), nil
}

func (s *Service) Retry(ctx context.Context, hash string) (*SendResponse, error) {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	if s.storageFault.Load() {
		return nil, errors.New("交易儲存發生錯誤，請修復磁碟後重啟錢包")
	}

	record := s.journal.FindByHash(hash)
	if record == nil {
		return nil, chain.ErrNotFound
	}

	if s.journal.NonceMined(record.Nonce) {
		if record.State != "succeeded" && record.State != "reverted" {
			record.State = "replaced"
		}
		return sendResponseFromRecord(record, record.State), nil
	}

	baseVersion := record.Version

	rawBytes, err := hexutil.Decode(record.SignedRaw)
	if err != nil {
		return nil, err
	}

	var tx types.Transaction
	if err := tx.UnmarshalBinary(rawBytes); err != nil {
		return nil, err
	}

	// Sepolia check
	if tx.Hash().Hex() != string(record.Hash) {
		return nil, errors.New("儲存的交易雜湊不符")
	}
	if tx.ChainId().Cmp(big.NewInt(s.client.ChainID())) != 0 {
		return nil, ErrWrongChain
	}

	broadcastErr := s.client.SendTransaction(ctx, &tx)
	newState := JournalSubmitted
	var errStr string
	if broadcastErr != nil {
		newState = JournalBroadcastUnknown
		errStr = broadcastErr.Error()
	}

	updated, err := s.journal.UpdateStateAtomicIfVersion(string(record.Hash), baseVersion, string(newState), "", "", errStr)
	if err != nil {
		s.storageFault.Store(true)
		return nil, err
	}
	if !updated {
		latest := s.journal.FindByHash(string(record.Hash))
		if latest != nil {
			return sendResponseFromRecord(latest, latest.State), nil
		}
	}

	return sendResponseFromRecord(record, newState), nil
}

func (s *Service) History(ctx context.Context) (*HistoryResponse, error) {
	s.historyMu.Lock()
	defer s.historyMu.Unlock()
	// Refresh recent and outstanding records without treating RPC errors as transaction failure.
	refreshError := ""
	items := s.journal.RefreshItems()
	start := s.historyOffset
	for i := 0; i < len(items); i += 1 {
		index := (start + i) % len(items)
		item := items[index]
		if ctx.Err() != nil {
			refreshError = "部分交易未能完成鏈上查核；以下保留本機最後紀錄，請稍後更新。"
			break
		}
		txInfo, err := s.client.Transaction(ctx, item.Hash)
		s.historyOffset = (index + 1) % len(items)
		if ctx.Err() != nil {
			refreshError = "部分交易未能完成鏈上查核；以下保留本機最後紀錄，請稍後更新。"
			break
		}
		if err == nil && txInfo != nil {
			switch txInfo.State {
			case "succeeded", "reverted", "reorg_detected", "pending", "receipt_unavailable":
				if _, err := s.journal.UpdateStateAtomicIfVersion(item.Hash, item.Version, txInfo.State, txInfo.Confirmations, txInfo.FeeETH, "", txInfo.Finalized); err != nil {
					s.storageFault.Store(true)
					return nil, err
				}
			}
		}
		if errors.Is(err, chain.ErrNotFound) {
			record := s.journal.FindByHash(item.Hash)
			if record != nil && (record.State == "succeeded" || record.State == "reverted") {
				if _, err := s.journal.UpdateStateAtomicIfVersion(item.Hash, item.Version, "broadcast_unknown", "", "", ""); err != nil {
					s.storageFault.Store(true)
					return nil, err
				}
			}
		}
		if err != nil {
			refreshError = "部分交易未能完成鏈上查核；以下保留本機最後紀錄，請稍後更新。"
		}
	}

	if ctx.Err() != nil && refreshError == "" {
		refreshError = "部分交易未能完成鏈上查核；以下保留本機最後紀錄，請稍後更新。"
	}

	if _, err := s.journal.ArchiveFinalized(); err != nil {
		s.storageFault.Store(true)
		return nil, err
	}
	var finalized *finalizedNonce
	if s.journal.HasInFlightTx() {
		finalized = s.finalizedNonce(ctx)
	}
	transactions, canCreate := s.journal.listHistory(finalized)
	return &HistoryResponse{
		Transactions:         transactions,
		RefreshError:         refreshError,
		CanCreateTransaction: finalized != nil && canCreate && !s.storageFault.Load(),
	}, nil
}

// Linked networks share one keystore manager, while quotes and transaction storage remain isolated.
func NewLinkedService(client *chain.Client, walletDir string, primary *Service) (*Service, error) {
	service, err := NewService(client, walletDir)
	if err != nil {
		return nil, err
	}
	service.keystore = primary.keystore
	service.catalog = primary.catalog
	if service.catalog == nil {
		service.catalog = primary
	}
	return service, nil
}
