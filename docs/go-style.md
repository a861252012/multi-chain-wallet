# Go 分層與風格

專案名稱、既有資料相容性與部署切換規範見 [project-naming.md](project-naming.md)。

這份文件記錄程式的分層與 Go 寫法；型別邊界和分支寫法參考 [Ardan Labs service 的固定版本](https://github.com/ardanlabs/service/tree/4d02671018b8b6138ceaaa7c40bd66e6d912bcbb)。

## 套用範圍

| 範圍 | 責任與可檢查依據 |
|---|---|
| `cmd/multi-chain-wallet` | `loadConfig` 集中環境設定解析；main 組裝服務、管理生命週期，不放交易規則。設定測試涵蓋預設值、無效值與 faucet 檔案權限。 |
| `internal/web` | HTTP、驗證存取來源、嚴格解碼、狀態碼與 primitive DTO。EVM、Solana、TRON、faucet、observe、diagnostics 都先轉成回應 DTO；不直接編碼交易日誌。分層說明見 `architecture.md`。 |
| `internal/wallet` | 交易動作、金額、帳戶與地址驗證、簽名、持久化順序與重送政策。`QuoteCommand`、`QuoteID`、`TransactionHash`、交易狀態及鏈別識別型別用於業務運算與查找。 |
| 儲存邊界 | EVM journal、Solana/TRON journal、scan 各有獨立 disk DTO 與具名轉換。service 持有業務紀錄；讀檔先解析，開啟 journal 仍驗證原始簽名交易。activity 索引只保存已驗證公開 hash。 |
| `internal/chain` | RPC adapter，primitive wire 值先解析為地址、hash、整數及狀態，再供查詢與交易邏輯使用。不可依賴 wallet 或 web。 |
| `internal/wallet/erc4337` | ABI、hash、簽章使用具名型別；JSON codec 在 RPC 邊界轉換 primitive 欄位。解析失敗時 receiver 保持原值。 |
| 前端、scripts、測試、CI | 交易驗收 CLI 使用 Go；瀏覽器測試用 JavaScript，部署工具與測試也使用 Python。 |

複合 EVM 請求在 web → wallet 轉成 command；單值 service 入口在服務邊界解析地址、金額、報價或交易 ID。`QuoteRequest` 與公開 read model 供既有呼叫使用；HTTP 與持久化檔案各有自己的 DTO。加密 keystore 與備份沿用原有格式，`json.RawMessage` 保存完整加密檔案。

## 可讀性與相容性

- 分支使用早期返回及有限狀態的 `switch`。
- 使用 Go 1.26 的 `min`、`slices.SortFunc`／`SortStableFunc`、`slices.Clone`、`maps.Copy`、`errors.AsType`、`strings.SplitSeq` 與 `WaitGroup.Go`。
- API 契約包含欄位名稱、`omitempty`、時間格式，以及 nil 與空陣列的差別。
- EVM journal 相容缺少 `To`／`Action` 的舊紀錄，也接受舊 opaque quote ID。交易原始位元組與 hash 先持久化再廣播；並行更新使用 CAS 版本檢查與既有鎖順序。
- 跨鏈 journal 的業務型別沒有 JSON tag，disk DTO 獨立於公開回應型別。TRON 使用 `slices.Equal` 比較業務紀錄。

## 驗證與限制

`go test ./...` 包含 DTO literal JSON 契約、nil/empty、儲存格式、舊 journal、簽名重載、原始 bytes 重送、SIGKILL 復原及並行狀態更新測試。`TestLayerBoundaries` 檢查依賴方向與 web DTO 外部欄位型別；不涵蓋每個 handler 的實際資料流。

CI 執行 `go test -race -count=1`、`go vet`、`gofmt` 與 `go fix -diff`，另外執行瀏覽器 fixture；CLI 測試已包含在 Go 測試內。Go 測試使用離線容器和暫存錢包；通過代表本機/mock 契約與復原驗證，不代表真實鏈上廣播、主網安全或正式環境效能。

驗收工具統一使用 Go：`go run ./cmd/send-and-verify --test-quote` 與
`go run ./cmd/verify-onchain-evidence`，測試納入 `go test ./...`。
後者預設從 repo 根目錄讀取 `internal/web/static/onchain-evidence.json`；
在其他目錄執行編譯後的 CLI 時，使用 `--manifest /absolute/path/to/onchain-evidence.json`。
`scripts/send_and_verify.sh` 僅保留為 Go CLI 的相容入口，不依賴 Python。
