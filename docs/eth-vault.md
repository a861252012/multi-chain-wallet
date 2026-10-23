# ETH 存入與取回

在側欄開啟「智慧合約」，就能使用「ETH 存入與取回」：把測試 ETH 存入專案的 Solidity `ETHVault` 合約，再取回目前錢包。只支援 Ethereum Sepolia，沒有利息或鎖定期，也不提供代幣、管理員代提、多簽或升級功能。

付款、放款與退款使用另一個 [PaymentEscrow 合約](payment-escrow.md)，在同頁「付款託管」分頁操作；不改變本 ETHVault 的存提規則。

## 操作方式

1. 選好要使用的錢包，確認網路為 Ethereum Sepolia，開啟側欄「智慧合約」。
2. 核對「目前錢包餘額」與「此錢包的合約餘額」。存入與取回都要支付手續費（Gas），請在錢包留一些測試 ETH；需要時可按「需要測試 ETH？前往領取」。
3. 選擇「存入合約」或「取回錢包」，確認畫面上的資金流向，再輸入 ETH 金額。取回只能使用目前錢包的合約餘額，款項會回到同一個錢包地址。
4. 按「預估費用並核對」，查看操作、錢包、合約地址、金額與費用上限。取回時，「最多扣除」只計算錢包支付的 Gas 上限，取回的 ETH 會另外回到錢包。這一步只做預覽，還不會送出交易。
5. 確認內容後，輸入錢包密碼並按「簽署並送出至 Ethereum Sepolia」。報價過期時須重新預估。
6. 查看交易紀錄與鏈上收據，再按「更新餘額與紀錄」核對兩邊餘額。取得交易 hash 代表已取得廣播追蹤資料；仍須確認收據 `status=1`，才能判定鏈上執行成功。

共用展示錢包的合約餘額可能隨其他人的操作變動。畫面會提醒這是共用錢包；需要獨立餘額時，請使用自己建立的測試錢包。公開唯讀展示需登入後，才能查看錢包的合約餘額與操作。

## 功能與限制

- `deposit()` 接收測試 ETH，記在 `msg.sender` 的存款下。
- `withdraw(uint256 amount)` 僅能提領自己的餘額，金額單位為 wei，退回呼叫者。採 Checks-Effects-Interactions 與重入鎖，轉帳失敗會回滾餘額與事件。
- `balanceOf(address)` 查詢存款。`Deposited`／`Withdrawn` 事件提供操作證據。
- 零額存提與超額提領拒絕；普通 ETH 轉帳不入帳，必須呼叫 `deposit()`。不要直接使用錢包的原生 ETH 轉帳功能存入。
- 強制轉入的 ETH 不會增加任何人的存款；合約沒有取回這類額外 ETH 的管理員功能。
- 合約帳面餘額按地址隔離。公開 demo 若共用同一個錢包地址，就共用該地址的存款；需要隔離時使用自己建立的測試錢包。
- 送出交易需要錢包密碼。報價與送出前都會檢查合約並執行 `eth_call`；模擬成功不保證之後的交易成功。簽署後先存檔再廣播，相同 quote ID 的重試沿用已保存的簽名交易。
- 活動頁核對成功、canonical 收據後，若包含 Sepolia EIP-7708 原生 ETH 系統事件，該交易本金只依系統事件計帳，包含 Vault 轉入／轉出；不再加計 `Withdrawn` 或交易 value。無系統事件的舊收據才使用伺服器指定合約的 `Withdrawn` 作為 ETH 收入證據，存入則用交易 value 並標示 `Deposited`。兩種模式都避免重複本金；詳見 [活動證據與限制](wallet-reference.md#receive-and-account-for-test-assets)。此功能不提供全鏈內部轉帳索引。

## 本機編譯與測試

先在目前終端機完成 [Go 1.26.10 工具鏈設定](go-style.md#本機工具鏈設定)，再從 repository 根目錄執行以下指令。

CI 的瀏覽器 job 使用 Node.js 22，Go 版本依 `go.mod`；容器版本固定於 Dockerfile。本機實際工具版本須隨驗收紀錄保存。編譯器固定 Solidity 0.8.37，optimizer 開啟、200 runs、EVM Cancun。npm override 將編譯工具的 `tmp` 固定到 0.2.7；不影響 Solidity 程式碼。

```sh
npm ci --ignore-scripts --prefix contracts
npm run check --prefix contracts
npm ci --ignore-scripts --prefix tests/browser
(cd tests/browser && npx playwright install chromium)
go test -race -count=1 ./...
go vet ./...
npm test --prefix tests/browser
npm run test:e2e --prefix tests/browser
```

`contracts/artifacts/` 保留可重現編譯結果。Go 測試核對來源 SHA-256；CI 重新編譯並比較產物，防止 Solidity 修改後仍測舊 bytecode。Go 的 simulated backend 在記憶體執行合約，不使用使用者 keystore 或公開 RPC。測試用 `ReentrancyAttacker` 與 `RejectingReceiver` 不需要部署到公共鏈。

驗收先執行 `check`；只有刻意修改 Solidity 來源後，才執行 `npm run compile --prefix contracts` 並審閱 artifact diff。不要在驗收前先覆寫產物而掩蓋來源與 bytecode 不一致。

| 測試入口 | 執行範圍 | 存入／取回／剩餘 ETH |
|---|---|---|
| `npm test --prefix tests/browser` | Chromium + 本機 HTTP fixtures，檢查 UI、語言、手機版及錯誤狀態 | fixture 數值，不是合約存提證據 |
| `TestE2EVaultFullLifecycle`（`tests/e2e/e2e_test.go`） | HTTP client + Go server + simulated EVM，核對收據、事件、餘額及兩筆交易各自的 history | **0.5 / 0.2 / 0.3** |
| `npm run test:e2e --prefix tests/browser` | Chromium + Go server + simulated EVM（`-race`）；送出回應即核對 action／金額，再逐筆核對 hash、history 與 UI；Go runner 另查收據、事件與 `balanceOf` | **0.05 / 0.02 / 0.03** |
| `npm run test:live --prefix tests/browser` | 公開站版本、UI 與設定的唯讀檢查 | 不送出存提交易 |

瀏覽器合約流程須另跑 `npm run test:e2e --prefix tests/browser`；CI 的 browser job 也會執行。這套測試使用本機 simulated EVM。

本機 simulated EVM 使用 Sepolia 的 chain ID；相同 chain ID、合約地址格式或交易 hash 都不構成公共 Sepolia 證據。以上本機測試也不代表新版 Demo 已發布；須另外核對線上版本與 UI。

## 啟用條件與歷史驗收

`SEPOLIA_VAULT_ADDRESS` 預設空白，未設定時面板不提供存提操作。啟動錢包不會自動部署合約；設定地址前須確認 Sepolia 上的 bytecode 與合約來源。

2026-09-19，合約 `0xDbB49ee9eC6ab924eA2D3e37Fd594b06648247bf` 在公共 Sepolia 完成部署、Sourcify 原始碼驗證，以及公開 UI 的存入與全額取回。[發布結果](evidence/release-2026-09-19/REPORT.md)與[交易收據、事件及餘額](evidence/vault-sepolia-2026-09-19/REPORT.md)保留當日證據；Etherscan 個別驗證當時尚未完成。這些紀錄不代表目前版本已重新驗收。
