# Multi-Chain Wallet

[English](README.md) · [简体中文](README.zh-CN.md) · [正體中文（台灣）](README.zh-TW.md) · [繁體中文（香港）](README.zh-HK.md)

**[線上 Demo（僅供測試鏈使用）](https://wallet.tedlin.fyi/)**

這是用 Go 開發的多鏈錢包，支援 EVM、Solana 與 TRON。Demo 使用測試網。重點是交易結果不明時怎麼處理：先保存簽署內容再廣播、用 quote ID 避免重複簽名，重啟後繼續查收據。

![錢包介面，使用本機測試資料](docs/images/wallet-overview.png)

*介面預覽使用本機測試資料。*

## 本機啟動與驗證

[本機啟動](docs/wallet-reference.md#run-with-docker) · [架構](docs/architecture.md) · [交易復原驗證](docs/demo-script.md) · [Go 風格與檢查](docs/go-style.md) · [驗收證據](docs/evidence/README.md)

## 主要功能

- 建立、還原及管理本機錢包，支援加密備份。
- 發送原生幣與代幣、預覽手續費，設定或撤銷 ERC-20 授權額度。
- 在 Ethereum Sepolia 包裝／解包 WETH，透過 Uniswap V3 兌換 WETH 與測試 USDC。
- 依交易收據整理 EVM 收支紀錄，支援 CSV 匯出。

## 交易處理

交易簽署後先存檔，再送到 RPC。同一帳戶、網路與 quote ID 的重試會沿用已保存的交易。廣播結果不明時，會依各鏈的有效期限規則重送原始簽署資料。

EVM 收據是否成功、所在區塊是否仍在主鏈，以及是否 finalized，分開核對。程序重啟後，錢包會讀回交易日誌，繼續追蹤已保存的交易。

## 支援網路

| 網路 | 功能 |
|---|---|
| Ethereum Sepolia | 原生幣／ERC-20 轉帳與授權、WETH 包裝、Uniswap V3 兌換 |
| Arbitrum、Base、OP Sepolia · Polygon Amoy | 原生幣／ERC-20 轉帳與授權 |
| Solana Devnet | SOL 轉帳，使用獨立錢包 |
| TRON Shasta | TRX／TRC-20 轉帳，使用獨立錢包 |

## Solidity 測試 ETH 存款箱

Solidity 存款／提領合約搭配 Sepolia 面板，包含每個地址獨立餘額與重入防護。[2026-09-19 驗收紀錄](docs/evidence/vault-sepolia-2026-09-19/REPORT.md)記載公共 Sepolia 部署、Sourcify 原始碼驗證及存提結果；Etherscan 個別驗證在當時尚未完成。未設定合約地址的環境仍停用操作。

另有[測試 USDC 付款託管](docs/payment-escrow.md)：付款人付款與放款，收款人可全額退回原付款人。沿用交易日誌與重試流程，提供合約、Go／HTTP 及 Chromium＋模擬 EVM 測試；已在公共 Sepolia 完成付款、放款與退款，見[交易收據與餘額驗收](docs/evidence/escrow-sepolia-2026-09-22/README.md)。

[合約與測試](docs/eth-vault.md)。

## 智慧合約錢包實驗

ERC-4337 v0.6 帳戶已透過命令列在 Ethereum Sepolia 完成部署與轉帳；[收據與限制](docs/erc4337-acceptance.md)也記錄了 v0.7 仍只有格式支援，尚無公鏈驗收。

## 公開 Demo

網址：https://wallet.tedlin.fyi/ 。訪客共用測試錢包；簽署新交易與匯出加密金鑰仍需錢包密碼。訪客可建立受密碼保護的 EVM 測試錢包（全站最多 20 個）。
