# 一鍵領取測試幣：2026-09-15 驗收

## 使用流程

開啟錢包 → 選擇測試網路與收款帳戶 → 按「領取測試幣」→ 查看交易連結與更新後的餘額。收款不需要輸入錢包密碼。EVM 支援目前本機帳戶清單內的地址；發放帳戶本身不接受自領。

這是有庫存的本機測試幣發放工具。EVM/TRON 在後端以專用測試帳戶簽署、先寫入日誌，再廣播到測試鏈。它不能創造無限 ETH/POL/TRX，庫存用完仍須補充。Solana 則直接使用官方 Devnet RPC 的 requestAirdrop。

## 真實交易證據

Sepolia 與 TRON 透過實際產品按鈕操作。其餘三個 EVM 網路透過相同後端領幣 API 驗收；不是所有網路都重做一次完整瀏覽器點擊。

| 測試網路 | 發放量 | 查核結果 | 交易 |
| --- | --- | --- | --- |
| sepolia | 0.001 ETH | succeeded | [0x600f2556487f3be85558e34e3abe51274e162912566bec78d129553f918f676e](https://sepolia.etherscan.io/tx/0x600f2556487f3be85558e34e3abe51274e162912566bec78d129553f918f676e) |
| base | 0.0001 ETH | succeeded | [0xbf38289b223d52687dcefcb86055e7dff697dacb14d38884adeb4f40ac6ebfc5](https://sepolia.basescan.org/tx/0xbf38289b223d52687dcefcb86055e7dff697dacb14d38884adeb4f40ac6ebfc5) |
| optimism | 0.0001 ETH | succeeded | [0x32b908efd9aab3b1af5cc1529573660aa1646b9e1f02d5f1ed115848ac03ec9c](https://testnet-explorer.optimism.io/tx/0x32b908efd9aab3b1af5cc1529573660aa1646b9e1f02d5f1ed115848ac03ec9c) |
| arbitrum | 0.0001 ETH | succeeded | [0xb39a1c0fa2a21c3e183b788e6c25bf8e4d3ad1d4754bb4bbb3f9d5431c869b95](https://sepolia.arbiscan.io/tx/0xb39a1c0fa2a21c3e183b788e6c25bf8e4d3ad1d4754bb4bbb3f9d5431c869b95) |
| tron | 5 TRX | finalized | [b63626979ec658cf906ae40009055df78db134acd41cf4d7d2e85f8758e1d7b2](https://shasta.tronscan.org/#/transaction/b63626979ec658cf906ae40009055df78db134acd41cf4d7d2e85f8758e1d7b2) |

Sepolia 畫面餘額由 0.05 增加到 0.051 ETH。TRON 畫面由 1976.0945 增加到 1981.0945 TRX，並取得 Shasta 固化收據。EVM 的 succeeded 僅表示取得成功收據，不等於此刻皆已 finalized。

Shasta 發放帳戶先由既有測試錢包轉入 20 TRX，資金補充交易為 `ea97910effb4980597b70d72c5b1292438895f1e6b4021549918020f771f0898`；隨後發放 5 TRX。這是測試庫存準備，不應算成額外一筆使用者領取。

## 已實作的邊界

- 後端固定網路與金額，拒絕 caller 自訂發放金額。沒有 mainnet 選項。
- 重用既有 wallet Service、Nonce 保護與 durable journal，不另外建立同帳戶的並行簽名實例。
- 只向本機錢包發放。EVM 原生幣及 Sepolia USDC、TRON 5 TRX 皆會查找近一小時同額交易，找到時回傳原雜湊並標示 reused，不再廣播。檢查包含供應帳戶的手動同額轉帳，及未知廣播結果。
- 近一小時每網路最多 20 筆供應帳戶交易；計入其其他操作。這是本機演示限額，不是公開商用 faucet 的配額系統。
- EVM 預估手續費（含 Base/OP 預留費）最多 0.0001 ETH；Amoy 最多 0.1 POL；TRON 原生幣發放最多預估 2 TRX。OP 額外費用仍是估算，不能宣稱協議層硬上限。
- POST 套用既有 localhost Host、Origin、CSRF 與嚴格 JSON 檢查。設定檔要求普通檔案與 0600 權限，預設未啟用發放。
- Solana 每分鐘最多嘗試一次，重啟會重設此本機冷卻；上游額度獨立存在。逾時不代表沒有發出空投，因此介面要求先核對餘額。

## 尚未完成或受外部供應限制

- Polygon Amoy 支援直接發放 0.1 POL 的程式路徑，但供應帳戶沒有 POL；尚無此路徑的真實領幣成功證據。
- Solana 本次實際按鈕請求未取得可確認的成功空投結果，餘額仍為 0。UI 顯示未確認、可能限流／供應不足／連線逾時，而非已到帳。
- Sepolia USDC 固定發放 0.01，使用既有 ERC-20 簽名路徑；下方五筆新交易驗收僅涵蓋原生幣發放。
- 不會自動操作第三方登入、接受條款或繞過 CAPTCHA 補庫存。公開 clone 不含測試幣、密碼或私鑰。

## 驗證方法

隔離容器使用唯讀原始碼、唯讀模組快取、獨立 build cache，沒有掛載 runtime wallet volume，也沒有外網：

```sh
docker run --rm --network none \
  -v "$PWD:/app:ro" \
  -v multi-chain-wallet_go_modules:/go/pkg/mod:ro \
  -v /private/tmp/multi-chain-wallet-development-cache:/tmp/review-cache \
  -e GOCACHE=/tmp/review-cache multi-chain-wallet-app \
  sh -c 'go test -race -count=1 ./... && go vet ./...'
```

新增測試驗證非允許網路／外部收款人／自訂金額被拒絕、過高手續費不廣播、簽署金額與鏈 ID 綁定、廣播回應遺失後重啟不重送，以及 Solana genesis 檢查與固定空投額度、TRON 原生幣發放與未知廣播去重。

瀏覽器回歸使用本機 mock API，禁止外部連線，涵蓋領幣成功、SOL 失敗訊息、TRON 庫存不足和既有流程；不把 mock 測試算成真實收據。當時使用 bundled Playwright 執行 `tests/browser/browser.cjs`；目前的安裝與重跑方式見[瀏覽器測試說明](wallet-reference.md#browser-and-continuous-verification)。

最終 Go 測試與靜態分析 Exit Code 0：

```text
?   github.com/a861252012/multi-chain-wallet/cmd/multi-chain-wallet [no test files]
ok  github.com/a861252012/multi-chain-wallet/internal/chain 1.111s
ok  github.com/a861252012/multi-chain-wallet/internal/wallet 7.028s
ok  github.com/a861252012/multi-chain-wallet/internal/web 1.075s
```

瀏覽器 mock 回歸也已通過領幣成功、SOL 未取得空投、TRON 庫存不足及既有錢包流程。Go Race Detector 結果僅涵蓋已執行的測試路徑，不代表沒有所有業務競態。

另於容器重啟後再次按下 Sepolia 領幣按鈕，UI 顯示「本小時已有同額發放紀錄」，連結仍為 `0x600f2556487f3be85558e34e3abe51274e162912566bec78d129553f918f676e`，餘額維持 0.051 ETH。
