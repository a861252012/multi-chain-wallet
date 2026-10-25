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
- 影響鏈別判斷的參數明確傳入；例如 `receiptMovements` 必須指定 `chainID`，不以可變參數隱含預設網路。
- 使用 `min` 內建函式，以及標準函式庫的 `slices.SortFunc`／`SortStableFunc`、`slices.Clone`、`maps.Copy`、`errors.AsType`、`strings.SplitSeq` 與 `WaitGroup.Go`。
- API 契約包含欄位名稱、`omitempty`、時間格式，以及 nil 與空陣列的差別。
- EVM journal 相容缺少 `To`／`Action` 的舊紀錄，也接受舊 opaque quote ID。交易原始位元組與 hash 先持久化再廣播；並行更新使用 CAS 版本檢查與既有鎖順序。
- 跨鏈 journal 的業務型別沒有 JSON tag，disk DTO 獨立於公開回應型別。TRON 使用 `slices.Equal` 比較業務紀錄。

## 工具鏈安全更新

本專案建置與完整測試使用 **Go 1.26.10**。Docker 開發、發佈建置、CI 漏洞掃描與部署 smoke probe 固定相同的 `golang:1.26.10-alpine` 多平台 digest；browser CI 從 `go.mod` 取得版本。

Go 1.26.10 與 1.27.3 都包含 GO-2026-6809 的憑證信任修補，但工具鏈安全版本與專案相容性是兩項檢查。目前測試用的 `cockroachdb/swiss` 依賴 Go runtime internals，其 build constraint 排除 Go 1.27；因此本專案完整測試尚不支援 1.27.3。主機預設可以保留已修補的 1.27.3，執行此專案時選用下列 1.26.10 環境。

### 本機工具鏈設定

從 repository 根目錄，在要執行專案指令的終端機先執行：

```sh
wallet_go_root=$(GOTOOLCHAIN=go1.26.10 go env GOROOT) &&
export PATH="$wallet_go_root/bin:$PATH" GOTOOLCHAIN=local &&
go version
```

輸出必須為 `go version go1.26.10` 加上主機的 OS／架構。首次使用可能下載官方工具鏈；下載失敗時先停止後續指令。設定只作用於目前 shell 與子程序，開新終端機後須重做，不使用 `go env -w` 改全域設定。

這個 PATH 同時讓 `go`、`gofmt`、Air 的 build command 與 npm 啟動的 Go 子程序使用 1.26.10。瀏覽器 fixture 會設定 `GOTOOLCHAIN=local`，因此不能只設定 `GOTOOLCHAIN=go1.26.10` 而保留原 PATH。已安裝 Air 且已有本機 `.air.toml` 時，可在上述環境執行 `air`；標準啟動方式仍見 [Docker 操作指南](wallet-reference.md#run-with-docker)。

### 本機驗證指令

完成上述設定後執行：

```sh
go test -race -count=1 ./...
go vet ./...
test -z "$(gofmt -l cmd internal contracts tests)"
go fix -diff ./...
```

`go fix -diff` 預期沒有輸出；有差異須先檢查。瀏覽器測試的 Node／Chromium 準備及指令見 [ETH 合約測試](eth-vault.md#本機編譯與測試)。掃描工具已安裝時，在相同環境執行 `govulncheck ./cmd/... ./internal/...`；CI 固定使用 `govulncheck@v1.8.0`。掃描通過不取代編譯與行為回歸，也不代表已部署的執行檔同步更新。

安全更新時同步更新 `go.mod`、兩份 Dockerfile、`.github/workflows/verify.yml` 與 `tests/deployment/smoke.py` 的版本及 digest，並以容器工具鏈完成 race、vet、格式、go fix、govulncheck、瀏覽器與部署 smoke 回歸。本機較新的工具鏈掃描不能取代發佈工具鏈及實際執行檔掃描。只升級編譯器不會修補既有執行檔，必須重建與部署，再核對執行中的版本／映像 digest。

參考：[Go 安全公告](https://pkg.go.dev/vuln/GO-2026-6809)。

## 驗證與限制

`go test ./...` 包含 DTO literal JSON 契約、nil/empty、儲存格式、舊 journal、簽名重載、原始 bytes 重送、SIGKILL 復原及並行狀態更新測試。`TestLayerBoundaries` 檢查依賴方向與 web DTO 外部欄位型別；不涵蓋每個 handler 的實際資料流。

CI 執行 `go test -race -count=1`、`go vet`、`gofmt` 與 `go fix -diff`，另外執行瀏覽器 fixture；CLI 測試已包含在 Go 測試內。Go 測試使用離線容器和暫存錢包；通過代表本機/mock 契約與復原驗證，不代表真實鏈上廣播、主網安全或正式環境效能。

驗收工具統一使用 Go：`go run ./cmd/send-and-verify --test-quote` 與
`go run ./cmd/verify-onchain-evidence`，測試納入 `go test ./...`。
後者預設從 repo 根目錄讀取 `internal/web/static/onchain-evidence.json`；
在其他目錄執行編譯後的 CLI 時，使用 `--manifest /absolute/path/to/onchain-evidence.json`。
`scripts/send_and_verify.sh` 僅保留為 Go CLI 的相容入口，不依賴 Python。
