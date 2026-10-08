package wallet

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/ethereum/go-ethereum/common"
)

type ActivityTotal struct {
	Asset       string `json:"asset"`
	ReceivedRaw string `json:"receivedRaw"`
	SentRaw     string `json:"sentRaw"`
	FeeRaw      string `json:"feeRaw"`
	NetRaw      string `json:"netRaw"`
}
type ActivityResponse struct {
	ChainID           int64             `json:"chainId"`
	Transactions      []*chain.Activity `json:"transactions"`
	Totals            []ActivityTotal   `json:"totals"`
	Page              int               `json:"page"`
	Pages             int               `json:"pages"`
	TotalTransactions int               `json:"totalTransactions"`
	Incomplete        bool              `json:"incomplete"`
}

type SyncResponse struct {
	From  uint64 `json:"from"`
	To    uint64 `json:"to"`
	Added int    `json:"added"`
}

// Bound both decoding and persistence; never discard transaction evidence to fit.
const maxActivityHashes = 1000
const maxActivityIndexBytes = 128 * 1024
const activityActiveKeepCount = 200

var errActivityIndexFull = errors.New("收支索引已達 1000 筆上限；請先由管理者備份並處理索引")

// Derived only from durable activity files. A service owns its wallet directory;
// external repairs require closing and reopening the service.
type activityIndex struct {
	generation uint64
	ids        []string
	seen       map[string]bool
}

// readActivityFile 讀取單一收支索引或封存檔，並驗證雙重硬邊界與雜湊格式。
func readActivityFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxActivityIndexBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxActivityIndexBytes {
		return nil, errors.New("收支索引檔超過大小上限")
	}
	var ids []string
	if json.Unmarshal(data, &ids) != nil {
		return nil, errors.New("收支索引檔格式錯誤")
	}
	if ids == nil {
		ids = []string{}
	}
	if len(ids) > maxActivityHashes {
		return nil, errActivityIndexFull
	}
	for _, id := range ids {
		if _, err := ParseTransactionHash(id); err != nil {
			return nil, errors.New("收支索引檔雜湊錯誤")
		}
	}
	return ids, nil
}

// Validated hashes are 66 bytes each; 1,000 entries fit below the file size limit.
func splitActivityArchiveChunks(hashes []string) [][]string {
	var chunks [][]string
	for len(hashes) > 0 {
		n := min(len(hashes), maxActivityHashes)
		chunks = append(chunks, hashes[:n])
		hashes = hashes[n:]
	}
	return chunks
}

func (s *Service) activityArchivePaths() ([]string, error) {
	entries, err := os.ReadDir(s.walletDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var paths []string
	// ReadDir 按檔名排序；只篩選檔名，避免將錢包目錄視為 Glob 模式。
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "activity-archive-") && strings.HasSuffix(name, ".json") {
			paths = append(paths, filepath.Join(s.walletDir, name))
		}
	}
	return paths, nil
}

// nextActivityArchiveName keeps activity archives separate from transaction journals.
func (s *Service) nextActivityArchiveName(existing []string) string {
	now := time.Now().UTC().UnixNano()
	if now <= s.lastActivityArchiveNano {
		now = s.lastActivityArchiveNano + 1
	}
	for _, p := range existing {
		base := filepath.Base(p)
		var ts int64
		if _, err := fmt.Sscanf(base, "activity-archive-%d.json", &ts); err == nil {
			if ts >= now {
				now = ts + 1
			}
		}
	}
	s.lastActivityArchiveNano = now
	return filepath.Join(s.walletDir, fmt.Sprintf("activity-archive-%020d.json", now))
}

// Capture both sides of a compaction under the writer lock. Archives are immutable,
// so their contents can be read after releasing the lock without losing hashes.
func (s *Service) activitySnapshot() ([]string, []string, uint64, error) {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	paths, err := s.activityArchivePaths()
	if err != nil {
		return nil, nil, 0, err
	}
	activeIDs, err := readActivityFile(filepath.Join(s.walletDir, "activity.json"))
	return paths, activeIDs, s.activityGeneration, err
}

// Rebuild once after activity writes or restart, not for every page. Serialize
// rebuilds without holding the trading lock during archive reads.
func (s *Service) loadActivityIndex() (*activityIndex, error) {
	s.activityMu.Lock()
	defer s.activityMu.Unlock()
	s.sendMu.Lock()
	generation := s.activityGeneration
	s.sendMu.Unlock()
	if s.activityIndex != nil && s.activityIndex.generation == generation {
		return s.activityIndex, nil
	}
	paths, activeIDs, generation, err := s.activitySnapshot()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var all []string
	for _, path := range paths {
		ids, err := readActivityFile(path)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			if !seen[id] {
				seen[id] = true
				all = append(all, id)
			}
		}
	}
	for _, id := range activeIDs {
		if !seen[id] {
			seen[id] = true
			all = append(all, id)
		}
	}
	index := &activityIndex{generation: generation, ids: all, seen: seen}
	s.activityIndex = index
	return index, nil
}

func (s *Service) activityHashes() ([]string, error) {
	index, err := s.loadActivityIndex()
	if err != nil {
		return nil, err
	}
	return slices.Clone(index.ids), nil
}

// addActivityHashes writes archives before shortening the active index so a crash cannot lose hashes.
func (s *Service) addActivityHashes(ids []string) (int, error) {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	var validIDs []string
	for _, id := range ids {
		parsed, err := ParseTransactionHash(id)
		if err != nil {
			return 0, err
		}
		validIDs = append(validIDs, string(parsed))
	}
	if len(validIDs) == 0 {
		return 0, nil
	}

	archivePaths, err := s.activityArchivePaths()
	if err != nil {
		return 0, err
	}
	seen := map[string]bool{}
	for _, path := range archivePaths {
		archivedIDs, err := readActivityFile(path)
		if err != nil {
			return 0, err
		}
		for _, id := range archivedIDs {
			seen[id] = true
		}
	}

	activePath := filepath.Join(s.walletDir, "activity.json")
	activeIDs, err := readActivityFile(activePath)
	if err != nil {
		return 0, err
	}

	var deduplicatedActive []string
	for _, id := range activeIDs {
		if !seen[id] {
			seen[id] = true
			deduplicatedActive = append(deduplicatedActive, id)
		}
	}

	count := 0
	var toAppend []string
	for _, id := range validIDs {
		if !seen[id] {
			seen[id] = true
			toAppend = append(toAppend, id)
			count += 1
		}
	}
	if count == 0 {
		return 0, nil
	}

	combined := append(deduplicatedActive, toAppend...)
	data, err := json.Marshal(combined)
	if err != nil {
		return 0, err
	}
	// Even a failed write may have published archive chunks. The next reader must
	// rebuild from disk rather than serve an index that predates a partial commit.
	s.activityGeneration++

	if len(combined) <= maxActivityHashes {
		if err := atomicWriteFile(activePath, data, 0600); err != nil {
			return 0, err
		}
		return count, nil
	}

	toArchive := combined[:len(combined)-activityActiveKeepCount]
	toKeep := combined[len(combined)-activityActiveKeepCount:]

	chunks := splitActivityArchiveChunks(toArchive)
	writeArchive := s.writeActivityArchive
	if writeArchive == nil {
		writeArchive = atomicWriteFile
	}
	for _, chunk := range chunks {
		chunkData, err := json.Marshal(chunk)
		if err != nil {
			return 0, err
		}
		archivePath := s.nextActivityArchiveName(archivePaths)
		if err := writeArchive(archivePath, chunkData, 0600); err != nil {
			return 0, err
		}
	}

	keepData, err := json.Marshal(toKeep)
	if err != nil {
		return 0, err
	}
	if err := atomicWriteFile(activePath, keepData, 0600); err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Service) ImportActivity(ctx context.Context, hash string) (*chain.Activity, error) {
	address, err := s.keystore.Address()
	if err != nil {
		return nil, err
	}
	result, err := s.client.Activity(ctx, hash, common.HexToAddress(address), common.HexToAddress(s.VaultAddress()))
	if err != nil {
		return nil, err
	}
	domain, err := chainActivityToDomain(result)
	if err != nil {
		return nil, err
	}
	if domain.state != activitySucceeded && domain.state != activityReverted {
		return nil, errors.New("交易尚未取得有效的鏈上收據，請稍後再匯入")
	}
	if _, err := s.addActivityHashes([]string{string(domain.hash)}); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) SyncActivity(ctx context.Context, from uint64, contracts []string) (*SyncResponse, error) {
	address, err := s.keystore.Address()
	if err != nil {
		return nil, err
	}
	if len(contracts) > 20 {
		return nil, errors.New("每次同步最多 20 個代幣合約")
	}
	addresses := []common.Address{}
	for _, contract := range contracts {
		parsed, err := ValidateAddress(contract)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, parsed)
	}
	ids, start, end, err := s.client.DiscoverActivity(ctx, common.HexToAddress(address), from, addresses...)
	if err != nil {
		return nil, err
	}
	count, err := s.addActivityHashes(ids)
	if err != nil {
		return nil, err
	}
	return &SyncResponse{start, end, count}, nil
}

func (s *Service) Activity(ctx context.Context, page int) (*ActivityResponse, error) {
	if page < 1 {
		return nil, errors.New("頁碼必須大於 0")
	}
	address, err := s.keystore.Address()
	if err != nil {
		return nil, err
	}
	activity, err := s.loadActivityIndex()
	if err != nil {
		return nil, err
	}
	// Merge only journal additions; never copy or walk the complete activity index
	// merely to serve one page, and never mutate a published index.
	seen := map[string]bool{}
	var extra []string
	history := s.journal.ListHistory()
	for i := len(history) - 1; i >= 0; i -= 1 {
		item := history[i]
		if !activity.seen[item.Hash] && !seen[item.Hash] {
			seen[item.Hash] = true
			extra = append(extra, item.Hash)
		}
	}
	// Pagination follows index insertion order, not block time; each row shows its verified block time.
	total := len(activity.ids) + len(extra)
	pages := (total + 19) / 20
	if pages == 0 {
		pages = 1
	}
	if page > pages {
		return nil, errors.New("頁碼超出範圍")
	}
	begin := (page - 1) * 20
	end := min(begin+20, total)
	response := &ActivityResponse{ChainID: s.client.ChainID(), Page: page, Pages: pages, TotalTransactions: total, Transactions: make([]*chain.Activity, end-begin), Totals: []ActivityTotal{}}
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for i := begin; i < end; i += 1 {
		position := total - 1 - i
		var hash string
		if position < len(activity.ids) {
			hash = activity.ids[position]
		} else {
			hash = extra[position-len(activity.ids)]
		}
		index := i - begin
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			item, err := s.client.Activity(ctx, hash, common.HexToAddress(address), common.HexToAddress(s.VaultAddress()))
			if err != nil {
				item = &chain.Activity{Hash: hash, State: "unverified", Movements: []chain.Movement{}, Error: err.Error()}
			}
			response.Transactions[index] = item
		})
	}
	wg.Wait()
	type totals struct{ received, sent, fee *big.Int }
	sums := map[string]*totals{}
	for _, tx := range response.Transactions {
		domain, err := chainActivityToDomain(tx)
		if err != nil {
			return nil, err
		}
		if domain.state != activitySucceeded && domain.state != activityReverted {
			response.Incomplete = true
			continue
		}
		for _, move := range domain.movements {
			sum := sums[move.asset]
			if sum == nil {
				sum = &totals{big.NewInt(0), big.NewInt(0), big.NewInt(0)}
				sums[move.asset] = sum
			}
			switch move.kind {
			case activityReceive:
				sum.received.Add(sum.received, move.amount)
			case activitySend:
				sum.sent.Add(sum.sent, move.amount)
			case activityFee:
				sum.fee.Add(sum.fee, move.amount)
			}
		}
	}
	for asset, sum := range sums {
		net := new(big.Int).Sub(sum.received, sum.sent)
		net.Sub(net, sum.fee)
		response.Totals = append(response.Totals, ActivityTotal{asset, sum.received.String(), sum.sent.String(), sum.fee.String(), net.String()})
	}
	slices.SortFunc(response.Totals, func(a, b ActivityTotal) int { return cmp.Compare(a.Asset, b.Asset) })
	return response, nil
}
