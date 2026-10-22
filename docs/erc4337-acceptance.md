# ERC-4337 鏈上驗收

重新執行本文件的本機指令前，先完成 [Go 1.26.10 工具鏈設定](go-style.md#本機工具鏈設定)。下列歷史鏈上結果仍保留原驗收範圍。

這個指令只驗證 Ethereum Sepolia、EntryPoint v0.6 與 SimpleAccount：第一次發送時部署帳戶，再轉 1 wei 給 owner。它沿用套件的 Builder、Signer 和 Bundler client，沒有接入錢包 UI。

## 套件範圍與本機測試

`internal/wallet/erc4337` 提供 v0.6／v0.7 UserOperation 的編碼、hash、Builder、Signer 與 Bundler JSON-RPC client。它與 EOA 發送流程及交易日誌分開；Paymaster、session key 與帳戶恢復尚未整合。格式支援不等於兩個版本都完成公鏈驗收，以下實際收據僅涵蓋 v0.6。

套件測試使用 `httptest` Mock Bundler，涵蓋編碼、簽章還原、RPC 錯誤、回應邊界與並行呼叫；不會向公鏈送出交易：

```sh
go test -race -count=1 ./internal/wallet/erc4337
go vet ./internal/wallet/erc4337
```

## 2026-09-16 驗收通過

- EntryPoint：`0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789`
- Factory：`0x9406Cc6185a346906296840746125a0E44976454`
- Smart account：`0xFeD97208981509cba39e536BF9745cB0862593b8`
- UserOperation hash：`0x3ee5271ee7223cd7fde5c92caa09126e942c140e4719389a63e5e87d263e1fee`
- [水龍頭入金 0.012 測試 ETH](https://sepolia.etherscan.io/tx/0x6a08b8b52c6d05a29eb0b0b74d881a7b1512e2bee573e7ef9155f22cfe00025e)。這筆是入金，不是 UserOperation 的執行交易。

[執行交易](https://sepolia.etherscan.io/tx/0x12610361a80bcc374feb68ca3c6b3b2fa53bf0b76b2faf0e610693240f9e6c9b) 位於區塊 `11717225`。Bundler 收據 `success: true`，節點交易收據 `status: 0x1`。驗收指令已核對：

- 帳戶在前一區塊尚未部署，該區塊已存在程式碼，owner 與 EntryPoint 正確。
- `AccountDeployed`、`UserOperationEvent` 的 hash、帳戶、Factory、nonce 與成功狀態符合預期。
- owner `0x88E151447b16749509e28edAdC6093B03d0f8177` 的餘額從 0 增加到 1 wei。
- 收據區塊與節點的 canonical block hash 一致。

公開資料保留 Bundler 原始回覆與節點收據。Bundler 的內層 receipt 省略 status；指令會另查節點確認成功，保存時使用核對後的節點收據。

這份證據只涵蓋 Sepolia、EntryPoint v0.6 和上述 SimpleAccount，沒有驗證 v0.7、UI、Paymaster 或 finalized 狀態。

### AA95 失敗與重試紀錄

原 hash `0x7c16a3f9662cfcf9381235410d18ebd675f56a87f5980f54e67f643bc8ab1266` 曾被 Bundler 接受，但[底層交易](https://sepolia.etherscan.io/tx/0xa3ec847a09acf70869643e166133abc587e1637f61d8c172e203278261702cb6) 的收據為 `status: 0x0`。以原交易參數在前一區塊重跑 `eth_call`，得到 `AA95 out of gas`：EntryPoint 的內層 gas 檢查未通過。先前只用較高 gas 的模擬，漏掉了這個條件。

後續改用 chain ID 形式的 Candide 公開端點，以零 gas 欄位要求估算，將估算值乘以 1.2 後取整數，再把調整後的 `verificationGasLimit` 加到 `preVerificationGas`。調整後的操作完成了上述部署與轉帳驗收。

這次同時改了多個條件，沒有逐項對照測試，無法確定更換端點或每一項 gas 調整是否必要，也不能由此認定 Bundler 估算錯誤。額外的 `preVerificationGas` 預留會增加付給 Bundler 的費用；這是本次驗收採用的保守處理，不是通用解法，也不保證其他操作成功。指令計算的 gas 費用上限仍不得超過 0.003 測試 ETH。

修正操作沿用相同帳戶、nonce 0、收款人與金額，費率提高 20%。原操作及新操作分開保存；相同 nonce 避免兩筆都執行。一般重跑指令仍只重送已保存的操作，不會自動替換或加價。

## 重跑公開證據

先將 `INDEPENDENT_SEPOLIA_RPC` 設為與 `--rpc` 不同供應商的 Sepolia HTTP(S) endpoint。以下不需要金鑰，也不會簽署、廣播或修改檔案：

```sh
go run ./cmd/accept-erc4337 --verify \
  --dir docs/evidence/erc4337-sepolia-v06 \
  --owner 0x88E151447b16749509e28edAdC6093B03d0f8177 \
  --confirm-rpc "$INDEPENDENT_SEPOLIA_RPC"
```

指令會重算 UserOperation hash、驗證簽章，再核對 Bundler 收據、節點交易收據、canonical block、AccountDeployed／UserOperationEvent、帳戶 owner／EntryPoint，以及收款地址在該區塊增加的 1 wei。缺少資料或任一項不符就失敗；不把收據成功直接當成 finalized。

## 建立自己的驗收

```sh
go run ./cmd/accept-erc4337 --confirm-rpc "$INDEPENDENT_SEPOLIA_RPC" --dir data/my-aa-test
# 只有兩個獨立供應商確認一致後，指令才會列出 sender；核對後再入金。
go run ./cmd/accept-erc4337 --confirm-rpc "$INDEPENDENT_SEPOLIA_RPC" --dir data/my-aa-test --send
go run ./cmd/accept-erc4337 --confirm-rpc "$INDEPENDENT_SEPOLIA_RPC" --dir data/my-aa-test --verify
```

`--confirm-rpc` 為必要參數，省略、相同 host、任一 RPC 非 Sepolia、回應地址為零／ABI padding 不正確或兩邊地址不同時，都不會列出可入金地址，也不會簽署或廣播。不同 URL 路徑／連接埠不算獨立來源；不同 host 也不保證不同經營者，選用獨立供應商是操作前提。這能阻擋單一 RPC 替換入金地址，不能防止兩家共同回報假資料，亦不等於本地 CREATE2 推導或共識證明。

指令使用 Candide 公開 Bundler `https://api.candide.dev/public/v3/11155111`，單筆 gas 上限為 0.003 測試 ETH。簽署結果會先寫入 `operation.json` 並同步磁碟，再送出；重跑只使用同一筆操作。若等待逾時，可用 `--verify` 查詢，或用 `--send` 重送原操作，不會自動加價。

`data/` 內保存專用測試 keystore 與自動產生的密碼，兩者在同一目錄，只適合這個測試用途。不要提交這個目錄。公開資料只包含已送出的操作，不含金鑰或密碼。

## 依據

- [SimpleAccount v0.6](https://github.com/eth-infinitism/account-abstraction/blob/v0.6.0/contracts/samples/SimpleAccount.sol)：部署與簽章規則。
- [Candide Bundler API](https://docs.candide.dev/wallet/abstractionkit/bundler/)：公開端點與 RPC 方法。
- [permissionless SimpleAccount](https://github.com/pimlicolabs/permissionless.js/blob/main/packages/permissionless/accounts/simple/toSimpleSmartAccount.ts)：Factory 地址。
