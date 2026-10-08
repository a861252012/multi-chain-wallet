# 付款託管：公共 Sepolia 驗收

2026-09-22（台灣時間），從 `https://wallet.tedlin.fyi` 的 Chromium 介面完成 **5 USDC 付款後退款**、**3.25 USDC 付款後放款**。兩條流程都核對成功收據、合約事件、訂單狀態與三方餘額。這裡記錄的是公共測試鏈交易。

## 合約與版本

- 網路：Ethereum Sepolia，chain ID `11155111`。
- 合約：[`0xE806A516cb5AA93Dde3124eab3ebFf49c0605078`](https://sepolia.etherscan.io/address/0xE806A516cb5AA93Dde3124eab3ebFf49c0605078)。
- 代幣：Circle 測試 USDC，`0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238`，6 位小數。
- 合約來源 commit：`456312f963462f2f438b288e5d3899c3d860ee65`；線上操作版本：`ede6d8f6ba59c968fdd847706a3c88edcb050616`。兩版合約來源相同。
- [CI 及公開站檢查](https://github.com/a861252012/multi-chain-wallet/actions/runs/35638920421) 通過。該操作版本映像 digest：`sha256:666367bb86b58d0907a8ec6536bd252496a30532d183c363078de5234ea1f382`。
- [Sourcify 原始碼驗證](https://repo.sourcify.dev/11155111/0xE806A516cb5AA93Dde3124eab3ebFf49c0605078)：creation、runtime 均為 `exact_match`。另以本機重新編譯結果比對鏈上 runtime，包含 immutable USDC 地址。

Sourcify 向 Etherscan 的同步驗證遇到每日配額上限；這份紀錄不宣稱 Etherscan 已驗證原始碼。

## 線上流程

付款人：`0x11A0130EecDF4648efed6a0E3A8901Bf9E44B293`。收款人：`0xe7A597C22c77E983B353CB39fb53a5479ec8c474`。兩者都是專用測試錢包；使用 Google 與 Circle 免費水龍頭取得測試幣。

| 訂單 | 授權 | 付款 | 結清 |
|---|---|---|---|
| `fl-20260922-refund-456312f`，5 USDC | [成功](https://sepolia.etherscan.io/tx/0xa17ef1edf6f15c899bb21bc78bb6039d43c9097e531f4815d2ebbc97dd35c15b) | [成功](https://sepolia.etherscan.io/tx/0xeb79075624896ba796cb162a2f97410adef7c1718aaa4b53324a6ba8c34b1677) | [原收款人全額退款](https://sepolia.etherscan.io/tx/0x5f20002b8a7710d7d97f28eeb04a672b16fe0ae5082e90ffe00cacd7a13c5569) |
| `fl-20260922-release-456312f`，3.25 USDC | [成功](https://sepolia.etherscan.io/tx/0xbcc09ff4901b598912e162e050fb7ee64d3e7d62e0886087df3c333b2cdae3da) | [成功](https://sepolia.etherscan.io/tx/0xa94ca2a86a5e0b17fbe0f9b45dd0c55dcadb061f6eff3a2bcff312d496a03b50) | [原付款人放款](https://sepolia.etherscan.io/tx/0x38e09fe3bca624fe0a53e056c0bdce2fa10b8495885fa817460dd2bcd7291405) |

每筆授權成功後先核對訂單仍未付款，再從 UI 送出付款。結清收據逐一檢查合約地址、事件名稱、訂單 ID、雙方地址和金額。退款回到原付款人，放款送到訂單指定的收款人。

| 觀察時點 | 付款人 USDC | 收款人 USDC | 合約 USDC | 未結清 USDC |
|---|---:|---:|---:|---:|
| 開始 | 20 | 0 | 0 | 0 |
| 5 USDC 付款後 | 15 | 0 | 5 | 5 |
| 5 USDC 退款後 | 20 | 0 | 0 | 0 |
| 3.25 USDC 付款後 | 16.75 | 0 | 3.25 | 3.25 |
| 3.25 USDC 放款後 | 16.75 | 3.25 | 0 | 0 |

餘額直接從 RPC 在各自固定區塊讀取，使用整數最小單位比對；不是只讀 UI。ETH 另用於 Gas，不列入 USDC 餘額。

UI 同時驗證付款人／收款人的操作按鈕、重整後還原訂單、退款前取消確認、已完成訂單拒絕重複付款，以及手機深色模式沒有水平溢位。收款錢包經 UI 建立並下載加密備份；備份與密碼不存入 repository。

## 複查與修正

線上截圖發現交易紀錄已成功，送出提示卻仍顯示等待收錄。修正讓提示沿用既有交易紀錄更新狀態，不增加 RPC 請求。隔離的 Chromium + Go + EVM 測試先重現失敗，再驗證成功；歷史查詢失敗時仍保留交易 hash，恢復後顯示成功。原 ETH 存提瀏覽器 E2E 一併回歸。

## 重查證據

- [交易、calldata 與收據](transactions.json)：部署、測試 ETH 補充及六筆託管流程交易。
- [USDC 餘額與未結清金額](balances.json)：五次固定區塊快照。
- [部署及原始碼驗證](contract.json)。
- [獨立收據複查](verification.json)：含檢查時間與當時的 finalized 狀態。

```sh
go run ./cmd/verify-onchain-evidence \
  --manifest docs/evidence/escrow-sepolia-2026-09-22/transactions.json \
  --network sepolia
```

此命令唯讀核對八筆交易的 chain ID、成功收據、發送者、目標與 calldata、canonical block 及 finalized 狀態；不簽名或廣播，也不代替上表的事件和餘額驗收。`finalized: false` 表示觀察時尚未最終確認，不代表交易失敗。

VM 已設定 `SEPOLIA_ESCROW_ADDRESS`，CI 另以 `EXPECTED_ESCROW_ADDRESS` 核對地址。設定更新前備份，保留原錢包目錄；完成後移除臨時 SSH 入站規則，重新連線確認已關閉。
