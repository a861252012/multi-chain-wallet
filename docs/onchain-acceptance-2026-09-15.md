# 真實測試鏈發送驗收 — 2026-09-15

這次完成 13 筆成功交易，涵蓋五個測試網。以下每筆均由 Multi-Chain Wallet 的報價、簽署及送出 API 執行，並另外透過公開 RPC 重新查核。這不代表所有網路、所有功能均完成端到端驗收。

## 成功交易

| 網路 | 操作 | 輸入金額 | 區塊瀏覽器 | 區塊 |
| --- | --- | --- | --- | --- |
| sepolia | approve-usdc | 0.01 USDC | [0x084ec2d1a2…](https://sepolia.etherscan.io/tx/0x084ec2d1a201f285def032dca19399fa629a038f9c136c948e93d271b5d8ae45) | 11708692 |
| sepolia | approve-weth | 0.00001 WETH | [0x7c80aa176b…](https://sepolia.etherscan.io/tx/0x7c80aa176b941ac48a186a963fc44dd4b4243959a40d2fce7a50a83bfbd950dc) | 11708629 |
| arbitrum | arbitrum | 0.000001 ETH | [0x69487db1e1…](https://sepolia.arbiscan.io/tx/0x69487db1e192608afd8e5f3bc1fcbc6cb5c9782cf6aa96e00b8d87a9817d2d0c) | 309087875 |
| base | base | 0.000001 ETH | [0x68d5168ec6…](https://sepolia.basescan.org/tx/0x68d5168ec616cb7590c238ee6d8e7a60072367619fcb1336bde1c9014aa70642) | 46846024 |
| sepolia | native | 0.000001 ETH | [0xfd2a5f420e…](https://sepolia.etherscan.io/tx/0xfd2a5f420ea560f213668a5e1f76169778f10104ee3fe4a8f16c072016d732c1) | 11708610 |
| optimism | optimism | 0.000001 ETH | [0x398f945097…](https://testnet-explorer.optimism.io/tx/0x398f945097aa6a64b12baf1a6871c6ea9e7585e7674fb5159bd58a3306b50943) | 48828817 |
| sepolia | swap-back | 0.01 USDC | [0xa58bde5e8c…](https://sepolia.etherscan.io/tx/0xa58bde5e8c37779c7f388dfc7e1766b4b078bfa18dbdb67df62af7887ab95b62) | 11708696 |
| sepolia | swap | 0.00001 WETH | [0x20b10d728e…](https://sepolia.etherscan.io/tx/0x20b10d728e69e9d7bdfb790357f7136ce828647c6cb292573d8217a207b2576c) | 11708638 |
| sepolia | unwrap | 0.00000034849160253 WETH | [0x8f29fe8a27…](https://sepolia.etherscan.io/tx/0x8f29fe8a27b6660317c2ccf0e21dbebae56ac078feef01387aaaf398c226b5dd) | 11708709 |
| sepolia | usdc-transfer | 0.01 USDC | [0x637b5b454a…](https://sepolia.etherscan.io/tx/0x637b5b454a4713a18d887c03079b953486b93dfefa091aa993359fff83af0b5a) | 11708659 |
| sepolia | wrap | 0.0001 ETH | [0x294e8d9a73…](https://sepolia.etherscan.io/tx/0x294e8d9a73e3f7cb5520b6e3e6e7064103d0a28e0ccc3d3cd8f3b30143129c26) | 11708617 |
| tron | native-to-receiver | 0.001 TRX | [c54d2865a210…](https://shasta.tronscan.org/#/transaction/c54d2865a210e2b3a1086b8bd287ace389da56d43530f00761e3126a6b9641e8) | 68400024 |
| tron | token | 0.01 USDT | [b877853662c2…](https://shasta.tronscan.org/#/transaction/b877853662c288a399a6857417158e75088ebfd567a1c32209d71a3554fcad2a) | 68400054 |

Sepolia 已實際執行 ETH→WETH→USDC，以及 USDC→WETH→ETH。反向兌換收據實際收到 348491602530 wei WETH，隨後解包相同數量；未把預估輸出當成實收。兩條流程透過產品 API 執行，瀏覽器引導流程另以 Mock 測試驗證，不能混稱為整套瀏覽器實鏈驗收。

TRON 的 USDT 為 Shasta 水龍頭提供的測試 TRC-20（`TG3XXyExBkPp9nzdajDZsozEu4BkaSJozs`），不宣稱為 Tether 官方發行資產。TRX 與 TRC-20 的 Solidity-node 收據均已確認；新收款帳戶的 TRX 交易含啟用費，實際總費用 1.1 TRX。測試 USDT 交易費為 2.8045 TRX。

## 驗收時發現並修正

TRON 節點拒絕原生 TRX 自轉帳。現於報價前回報清楚錯誤，並移除小額按鈕自動填入自己地址的行為。該次失敗嘗試的簽名紀錄保留，沒有列入上述成功交易。

當交易過期且 Solidity-node 的區塊時間已超過交易有效期、仍查無收據時，狀態改為 `expired_unconfirmed` 並保存觀測區塊，允許後續交易。只看本機時間、RPC 失敗或查無交易都不會單獨解除阻塞。紀錄不刪除，後續收據仍可更新；此機制信任設定的 RPC，沒有多節點共識保證。

## 可重跑的驗證

在專案根目錄執行：

```sh
docker run --rm --network none \
  -v "$PWD:/app:ro" \
  -v multi-chain-wallet_go_modules:/go/pkg/mod:ro \
  -v /private/tmp/multi-chain-wallet-development-cache:/tmp/review-cache \
  -e GOCACHE=/tmp/review-cache multi-chain-wallet-app \
  sh -c 'go test -race -count=1 ./... && go vet ./...'
go test ./cmd/send-and-verify ./cmd/verify-onchain-evidence
go run ./cmd/verify-onchain-evidence
```

Go 執行結果（Exit 0）：

```text
?    github.com/a861252012/multi-chain-wallet/cmd/multi-chain-wallet [no test files]
ok   github.com/a861252012/multi-chain-wallet/internal/chain 1.148s
ok   github.com/a861252012/multi-chain-wallet/internal/wallet 6.118s
ok   github.com/a861252012/multi-chain-wallet/internal/web 1.071s
```

Go 容器使用唯讀原始碼、無網路及獨立測試 cache，未掛載執行中錢包資料。Race Detector 僅針對執行到的記憶體存取，不證明不存在邏輯競態。

當時瀏覽器驗證使用本機 bundled Playwright 執行 `tests/browser/browser.cjs`。目前可依 [瀏覽器測試說明](wallet-reference.md#browser-and-continuous-verification) 安裝專案依賴並重跑。

Exit 0；涵蓋五個 EVM 網路、雙向兌換引導、回應遺失復原、Solana/TRON UI、TRC-20 查詢及手機版。全部 API 為本機 Mock，未簽署真實交易。

Python 單元測試：12 tests，Exit 0。

唯讀鏈上工具結果：`PASS: 13 receipts independently rechecked; no signing or broadcast.` 公開快照在 `internal/web/static/onchain-evidence.json`；EVM 檢查網路、簽署 calldata/金額、成功收據、canonical 區塊及 swap 輸出事件下限。TRON 檢查網路、solidified 收據、發送者及合約／原生收款者，未獨立解碼 TRC-20 收款者與金額。每次結果受 RPC 當下可用性影響；首次成功收據不等同 finalized。

作品頁 `/showcase` 已在重啟後透過瀏覽器確認：13 個交易連結、五個測試網、帳戶選單及未完成項目正常顯示，並檢查了交易區塊版面。

## 帳戶與入金

驗收使用專用測試帳戶，EVM 地址為 `0x57b44407cb4445f743e694da808B60654926C6ea`。Repository 不含驗收帳戶的憑證或加密備份。

Google 水龍頭取得 0.05 Sepolia ETH；TRON 官方連結水龍頭取得測試 TRX／USDT。Base、OP、Arbitrum 分別以 0.001 Sepolia ETH 經測試網橋入金；這是驗收準備，不宣稱產品已提供跨鏈橋 UI。Arbitrum 入金使用額外一次性測試工具，其後 Arbitrum 原生轉帳使用產品 API。

## 尚未完成

以下是當日狀態。Polygon Amoy 與 Solana Devnet 的原生幣發送已於 [2026-09-16 補齊](onchain-acceptance-2026-09-16.md)。

- Polygon Amoy：地址已建立、發送實作及 Mock 已驗證，但當時尚未取得 POL 並完成實際廣播。
- Solana Devnet：地址已建立；公開 RPC 水龍頭回覆限流。當時尚未完成入金與實際廣播。
- 其他 EVM 網路的 DEX、Solana SPL／DEX、TRON DEX 未實作。
- 所有功能並非都已實鏈故障演練；交易加速／取消等仍以既有測試涵蓋為主。
- 這份驗收不含 mainnet、真實資金、獨立安全稽核或已執行的 GitHub CI。

2026-09-16 工具遷移：上述 Python 測試數量是當時的歷史紀錄。
現行驗收 CLI 與測試已移至 `cmd/send-and-verify`、`cmd/verify-onchain-evidence`，
統一由 `go test ./...` 執行；不再需要 Python 或 curl。
