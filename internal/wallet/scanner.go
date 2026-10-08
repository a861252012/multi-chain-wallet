package wallet

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/ethereum/go-ethereum/common"
)

type ScanProgress struct {
	Enabled   bool      `json:"enabled"`
	Start     uint64    `json:"start"`
	Next      uint64    `json:"next"`
	Finalized uint64    `json:"finalized"`
	Tokens    []string  `json:"tokens"`
	Error     string    `json:"error,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type scanProgressDisk struct {
	ExplicitStart bool      `json:"explicitStart,omitempty"`
	Enabled       bool      `json:"enabled"`
	Start         uint64    `json:"start"`
	Next          uint64    `json:"next"`
	Finalized     uint64    `json:"finalized"`
	Tokens        []string  `json:"tokens"`
	Error         string    `json:"error,omitempty"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type scanState struct {
	ExplicitStart bool
	Enabled       bool
	Start         uint64
	Next          uint64
	Finalized     uint64
	Tokens        []EVMAddress
	Error         string
	UpdatedAt     time.Time
}

func scanProgressResponse(state *scanState) *ScanProgress {
	var tokens []string
	if state.Tokens != nil {
		tokens = make([]string, len(state.Tokens))
	}
	for i, token := range state.Tokens {
		tokens[i] = string(token)
	}
	return &ScanProgress{Enabled: state.Enabled, Start: state.Start, Next: state.Next, Finalized: state.Finalized, Tokens: tokens, Error: state.Error, UpdatedAt: state.UpdatedAt}
}

func scanProgressFromDisk(stored scanProgressDisk) (*scanState, error) {
	if len(stored.Tokens) > 200 {
		return nil, errors.New("同步進度檔代幣數量超過上限")
	}
	for _, token := range stored.Tokens {
		if !common.IsHexAddress(token) || common.HexToAddress(token) == (common.Address{}) {
			return nil, errors.New("同步進度檔包含無效代幣地址")
		}
	}
	var tokens []EVMAddress
	if stored.Tokens != nil {
		tokens = make([]EVMAddress, len(stored.Tokens))
	}
	for i, token := range stored.Tokens {
		tokens[i] = EVMAddress(token)
	}
	return &scanState{
		ExplicitStart: stored.ExplicitStart,
		Enabled:       stored.Enabled, Start: stored.Start, Next: stored.Next, Finalized: stored.Finalized,
		Tokens: tokens, Error: stored.Error, UpdatedAt: stored.UpdatedAt,
	}, nil
}

func scanProgressToDisk(state *scanState) scanProgressDisk {
	var tokens []string
	if state.Tokens != nil {
		tokens = make([]string, len(state.Tokens))
		for i, token := range state.Tokens {
			tokens[i] = string(token)
		}
	}
	return scanProgressDisk{
		ExplicitStart: state.ExplicitStart,
		Enabled:       state.Enabled, Start: state.Start, Next: state.Next, Finalized: state.Finalized,
		Tokens: tokens, Error: state.Error, UpdatedAt: state.UpdatedAt,
	}
}

func (s *Service) scanProgress() (*scanState, error) {
	data, err := os.ReadFile(filepath.Join(s.walletDir, "scan.json"))
	if os.IsNotExist(err) {
		return &scanState{Tokens: []EVMAddress{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var stored scanProgressDisk
	if json.Unmarshal(data, &stored) != nil {
		return nil, errors.New("同步進度檔格式錯誤")
	}
	return scanProgressFromDisk(stored)
}

func (s *Service) ScanProgress() (*ScanProgress, error) {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	state, err := s.scanProgress()
	if err != nil {
		return nil, err
	}
	return scanProgressResponse(state), nil
}

func (s *Service) ConfigureScan(enabled bool, start *uint64) (*ScanProgress, error) {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	s.scanGeneration += 1
	s.scanBackoffUntil = time.Time{}
	s.scanBackoffDuration = 0
	state, err := s.scanProgress()
	if err != nil {
		return nil, err
	}
	if start != nil {
		state.ExplicitStart = true
		state.Start = *start
		state.Next = *start
	}
	state.Enabled = enabled
	state.Error = ""
	data, err := json.Marshal(scanProgressToDisk(state))
	if err != nil {
		return nil, err
	}
	if err := atomicWriteFile(filepath.Join(s.walletDir, "scan.json"), data, 0600); err != nil {
		return nil, err
	}
	return scanProgressResponse(state), nil
}

func (s *Service) ScanOnce(ctx context.Context) error {
	s.scanRunMu.Lock()
	defer s.scanRunMu.Unlock()
	s.scanMu.Lock()
	state, err := s.scanProgress()
	if err != nil || !state.Enabled {
		s.scanMu.Unlock()
		return err
	}
	address, err := s.keystore.Address()
	if err != nil {
		s.scanMu.Unlock()
		return err
	}
	generation := s.scanGeneration
	s.scanMu.Unlock()

	final, err := s.client.FinalizedNumber(ctx)
	var block *chain.ScannedBlock
	if err == nil {
		state.Finalized = final
		// Legacy files without explicitStart retain the default recent-block behavior.
		if state.Next == 0 && !state.ExplicitStart {
			state.Next = final
			if final > 19 {
				state.Next = final - 19
			}
			state.Start = state.Next
		}
		if state.Next <= final {
			block, err = s.client.ScanFinalizedBlock(ctx, state.Next, common.HexToAddress(address))
		}
	}

	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	// A changed setting owns the progress file. Ignore results from the old scan.
	if generation != s.scanGeneration {
		return nil
	}
	if err == nil && block != nil {
		_, err = s.addActivityHashes(block.Hashes)
	}
	if err == nil && block != nil {
		known := map[EVMAddress]bool{}
		for _, token := range state.Tokens {
			known[token] = true
		}
		for _, raw := range block.Tokens {
			token, parseErr := ParseEVMAddress(raw)
			if parseErr != nil {
				return parseErr
			}
			if !known[token] && len(state.Tokens) < 200 {
				state.Tokens = append(state.Tokens, token)
				known[token] = true
			}
		}
		state.Next += 1
	}
	state.Error = ""
	if err != nil {
		state.Error = err.Error()
	}
	state.UpdatedAt = time.Now().UTC()
	data, marshalErr := json.Marshal(scanProgressToDisk(state))
	if marshalErr != nil {
		return marshalErr
	}
	if saveErr := atomicWriteFile(filepath.Join(s.walletDir, "scan.json"), data, 0600); saveErr != nil {
		return saveErr
	}
	return err
}

// checkScanBackoff 檢查目前是否處於掃描錯誤退避期間，並回傳當前配置世代。
func (s *Service) checkScanBackoff(now time.Time) (uint64, bool) {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	return s.scanGeneration, now.Before(s.scanBackoffUntil)
}

// recordScanError 依指定配置世代記錄掃描執行結果；若世代已變更則忽略舊結果。
func (s *Service) recordScanError(generation uint64, err error, now time.Time) {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	if generation != s.scanGeneration {
		return
	}
	if err == nil {
		s.scanBackoffDuration = 0
		s.scanBackoffUntil = time.Time{}
		return
	}
	base := 4 * time.Second
	if s.scanBackoffInitial > 0 {
		base = s.scanBackoffInitial
	}
	if s.scanBackoffDuration == 0 {
		s.scanBackoffDuration = base
	} else {
		s.scanBackoffDuration = min(s.scanBackoffDuration*2, 60*time.Second)
	}
	s.scanBackoffUntil = now.Add(s.scanBackoffDuration)
}

// RunMaintenance is cancelled and joined by the server before closing the wallet.
func (s *Service) RunMaintenance(ctx context.Context) {
	tickInterval := 2 * time.Second
	if s.maintenanceTickInterval > 0 {
		tickInterval = s.maintenanceTickInterval
	}
	historyInterval := 15 * time.Second
	if s.maintenanceHistoryInterval > 0 {
		historyInterval = s.maintenanceHistoryInterval
	}
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()
	nextHistory := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !s.keystore.Exists() {
				continue
			}
			now := time.Now()
			if !now.Before(nextHistory) {
				check, cancel := context.WithTimeout(ctx, 10*time.Second)
				_, _ = s.History(check)
				cancel()
				nextHistory = time.Now().Add(historyInterval)
			}
			generation, skip := s.checkScanBackoff(time.Now())
			if skip {
				continue
			}
			check, cancel := context.WithTimeout(ctx, 40*time.Second)
			err := s.ScanOnce(check)
			cancel()
			if ctx.Err() != nil {
				return
			}
			s.recordScanError(generation, err, time.Now())
		}
	}
}
