# 測試 USDC 付款託管

這個功能用來練習付款與退款，只使用 Sepolia 測試 USDC。款項先由合約保管，再由付款人放款給收款人，或由收款人退回原付款人。

雙方都不操作，款項就會留在合約。本功能不會自動退款，也不處理交易糾紛。

## 使用流程

1. 準備兩個測試錢包，分別用來付款和收款。兩邊都要有測試 ETH 付手續費；付款錢包還要有測試 USDC。
2. 到「智慧合約 → 付款託管 → 建立付款」，填入訂單編號、收款人地址和金額。
3. 按「核對付款資料」。需要授權時，依畫面確認。**授權只設定合約可扣款的金額，不會付款。** 如果已有舊授權，畫面可能會先請你取消舊授權。
4. 授權成功後，再按「核對付款資料」送出付款。等訂單顯示「款項由合約保管」，才代表合約已收到款項。
5. 付款人確認對方已完成約定後，可放款給收款人。需要退款時，由收款人操作，全部款項會退回原付款人，不能改退到別的地址。
6. 收款人可用付款人地址和訂單編號查詢訂單。重新整理會保留同一分頁的草稿，付款結果仍以重新查到的訂單狀態為準。

每一步都要確認金額、手續費和密碼；取消確認視窗不會送出交易。公開展示錢包由多人共用，其他人也能操作其中的款項。

訂單編號限 1–64 個英文字母、數字、`-` 或 `_`，區分大小寫。同一付款人付過款的編號不能再用；不同付款人可以使用相同編號。

## 業務規則與信任邊界

| 操作 | 誰能做 | 前置狀態 | 完成後 | 資金流向 |
|---|---|---|---|---|
| fund | 付款人本人 | None | Funded | 付款人 → 合約 |
| release | 原付款人 | Funded | Released | 合約 → 指定收款人 |
| refund | 指定收款人 | Funded | Refunded | 合約 → 原付款人 |

放款（`Released`）或退款（`Refunded`）成功後，這筆訂單就不能再操作。合約只檢查誰有權操作，不知道商品是否送達；付款人要自行確認對方已完成約定，再放款。不支援部分放款、部分退款、管理員提領或合約升級。

`PaymentEscrow` 建構時固定一種 ERC-20 代幣。應用程式只在 Sepolia 啟用，正式設定使用既有登錄的 Circle 測試 USDC；設定的託管合約仍須由操作者核對來源與部署參數。查詢及送出前核對合約 bytecode、token 地址與 USDC metadata。主網不在本功能範圍。

訂單鍵為 `(buyer, keccak256(UTF-8 reference))`，因此另一個地址不能搶占付款人的訂單編號。金額使用整數與 6 位小數轉換，不用浮點數。合約使用 OpenZeppelin 5.4.0 `SafeERC20` 與 `ReentrancyGuard`，狀態先更新、轉帳失敗時整筆回滾；實際收到／付出的金額必須與訂單相符，不支援轉帳扣稅或 rebasing 代幣。

`totalLocked` 為尚未結清的付款總額。直接轉入的額外代幣不會建立訂單，也沒有管理員可將它領走；請使用應用程式付款流程。依賴函式庫與測試通過不代表已完成獨立安全審計。

## Go 與 HTTP 整合

沿用 `web → wallet → chain` 邊界，不建立另一套簽名或交易儲存系統。

- `GET /api/wallet/escrow`：啟用狀態、固定合約與代幣、目前錢包的餘額與授權。
- `GET /api/wallet/escrow/order?buyer=...&orderId=...`：指定付款人與編號的訂單、查詢區塊及 finality；不是全鏈訂單索引。
- `POST /api/wallet/quote`：`escrow_fund`、`escrow_release`、`escrow_refund`。HTTP primitive DTO 轉成 `QuoteCommand`，客戶端不能指定託管合約或代幣。放款／退款金額由合約訂單決定，不能由請求覆寫。
- `POST /api/wallet/send`：沿用既有密碼、CSRF、來源檢查、報價期限、nonce、費用上限、先存檔再廣播及 quote ID 重試規則。新交易送出前再次模擬，拒絕已被另一方結清的舊報價。
- 相同 quote ID 重試回傳既有交易，不重新簽名；RPC 內部重試可重送相同 raw bytes。重啟後從 durable journal 找回原 hash，再依收據恢復狀態。
- 加速交易沿用原訂單的 calldata、收款目標、金額與 nonce。交易日誌保留原付款、放款或退款動作；取消交易仍是零額自轉。
- 「查看訂單」可從交易紀錄重新開啟。重啟後，錢包會從已簽交易還原訂單；舊紀錄缺少原編號時，仍可用鏈上訂單鍵查詢。
- 付款遇到回應中斷或 HTML 502 時，畫面顯示「交易結果待確認」。先用原 quote ID 查詢；重試也沿用該 ID，不會直接產生第二筆付款。
- 訂單狀態會連同查詢區塊再次核對。只有 finalized 區塊中的狀態、收款人和金額一致，才標示最終確認；查詢失敗不當成零額或未付款。

## 設定與公鏈紀錄

`SEPOLIA_ESCROW_ADDRESS` 未設定時，UI 不開放付款操作。程式更新不會部署合約；公開站的設定與付款收據要分開核對。

2026-09-22 公共 Sepolia 已完成付款、全額退款與放款，交易與餘額見[驗收紀錄](evidence/escrow-sepolia-2026-09-22/README.md)。這是當日結果。`EscrowTestToken` 僅供本機測試，不是 Circle USDC。

## 測試入口

```sh
npm ci --ignore-scripts --prefix contracts
npm run check --prefix contracts
go test -race -count=1 ./contracts
go test -race -count=1 -run 'TestEscrow' ./internal/wallet ./internal/web ./cmd/multi-chain-wallet
go test -race -count=1 -run '^TestE2EEscrow(Payment|OrderFinality)' ./tests/e2e
npm ci --ignore-scripts --prefix tests/browser
(cd tests/browser && npx playwright install chromium)
npm run test:e2e --prefix tests/browser
npm test --prefix tests/browser
go test -race -count=1 ./...
go vet ./...
go fix -diff ./...
gofmt -l cmd internal contracts tests
```

合約來源刻意修改後才執行 `npm run compile --prefix contracts` 並檢查 artifact diff；驗收使用 `check`。Go 單元／整合測試使用 `t.TempDir()`、生成的測試金鑰與記憶體 EVM。瀏覽器 E2E 使用 loopback Go server，啟動器強制 `-race` 並要求依賴存在。測試不讀取使用者錢包，不依賴公共 RPC，也不使用 production／共用資料。

| 測試 | 驗證內容 |
|---|---|
| 合約 | 權限、buyer namespace、重複付款／結清、零額、自付、token false／revert／扣稅回滾、重入鎖；固定 seed 的 100 次操作序列逐次核對三方餘額與未結清負債 |
| Go／HTTP | 設定、帳戶繼承、網路限制、DTO 欄位、CSRF／Origin、精度、無授權、錯誤密碼、競爭結清、廣播回應遺失後關閉並重開 service 與 journal |
| Chromium＋Go＋EVM | 授權不付款、5 USDC 付款及全額退款、3.25 USDC 付款及放款、錯誤收款人、重複編號、重整草稿、RPC 畫面失敗、遺失 HTTP 回應後重試、角色按鈕、取消確認、手機及鍵盤 |
| 獨立鏈上斷言 | Go runner 逐筆核對 receipt status、合約地址、event topics、order ID、雙方地址與金額，再核對 token balances／totalLocked；不是只看 DOM 或 API 字串 |

本機 EVM 雖使用 Sepolia chain ID，仍不是公共 Sepolia。測試數值與 hash 不可列為公開鏈驗收；實際執行結果以當次輸出為準。

## 介面

付款與查詢訂單分開；查到訂單後顯示狀態、金額與目前帳戶可用的放款或退款操作。確認視窗列出付款人、收款人、訂單、金額、資金去向與 Gas。按鈕只顯示目前角色可用的操作，權限仍由後端與合約檢查。
