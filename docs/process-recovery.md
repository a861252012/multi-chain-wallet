# 程序中止後的交易恢復

這項驗證使用 `TestProcessKillRestartReusesRaw`，聚焦一個故障邊界：本機 Mock RPC 已收到簽名交易，但還沒回應時，簽署程序遭 `SIGKILL` 終止。

## 驗證內容

1. 子程序在父程序建立的暫存目錄匯入公開測試助記詞，透過正常 Quote／Send 流程簽署。
2. Mock RPC 記下收到的 bytes，立即終止整個子程序；不執行 Service.Close 或正常清理。
3. 父程序確認退出原因確實為 SIGKILL，再用新的 Service 開啟同一目錄，驗證程序鎖已釋放。
4. journal 必須保有一筆 `pending` 交易，簽名 bytes 與 Mock 收到的內容完全相同。
5. 尚未解決的 nonce 必須阻擋新報價；以相同 quote ID 呼叫 Send 不需密碼、不重新簽署，也不再廣播。
6. 明確呼叫 Retry 後，Mock 只收到一次相同 bytes，hash 不變，狀態更新為 `submitted`；重新載入 journal 後仍保有此結果。

## 隔離重跑

使用已有的專案映像及 Go modules volume：

```sh
docker run --rm --network none \
  -v "$PWD:/app:ro" \
  -v multi-chain-wallet_go_modules:/go/pkg/mod:ro \
  -e GOCACHE=/tmp/cache multi-chain-wallet-app \
  go test -race -count=1 -v ./internal/wallet \
  -run '^TestProcessKillRestartReusesRaw$'
```

容器不掛載錢包資料、不連公開 RPC；測試金鑰與 journal 全部位於可丟棄的暫存目錄。測試設有 20 秒子程序期限，超時不視為成功故障注入。一般 `go test ./...` 也會執行此案例。

## 能證明與不能證明的事

這是作業系統程序中止及磁碟重新載入測試，比單純 Close／NewService 多覆蓋未正常關閉的情境。Mock 收到 bytes 不代表真實節點已接受交易；`submitted` 也不代表鏈上成功。

未涵蓋主機斷電、檔案系統損壞、廣播請求送出前的中止、真實節點遺失回應或區塊重組。既有收據驗收仍以 [歷史紀錄](onchain-acceptance-2026-09-15.md) 為準。

## 本次執行紀錄（2026-09-16）

- 單獨執行 `TestProcessKillRestartReusesRaw` 並開啟 Race Detector：PASS。
- 無外部網路、唯讀原始碼、未掛載執行中錢包的容器內，`go test -race -count=1 ./...`：chain、wallet、wallet/erc4337、web 全部通過；cmd/multi-chain-wallet 沒有測試檔。
- 既有 Playwright Mock 套件：PASS，包含三種語系、兩種主題、375／768／1024／1440 px 版面，以及 EVM／Solana／TRON 操作流程。
- 同一隔離容器內的 `go vet ./...` 通過。
- `git diff --check` 通過；新測試檔符合 gofmt。

以上是本機驗證，沒有執行真實鏈廣播、部署或遠端 CI。
