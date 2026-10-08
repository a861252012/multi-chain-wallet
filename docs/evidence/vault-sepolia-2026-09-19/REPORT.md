# 公共 Sepolia ETHVault 第二階段 — 2026-09-19（部署、啟用與公共 UI 存提已完成）

## 目前結果

公共合約已部署，Sourcify 原始碼驗證 creation/runtime 均為 exact_match。2026-09-19 已啟用公開站，並透過已發布 UI 完成 0.0001 Sepolia ETH 存入及全額取回；成功收據、精確事件、API、UI、餘額與 Gas 均相符。Etherscan 個別站台驗證仍未完成，沒有把它列為成功。

- 線上 /healthz HTTP 200；release SHA：`3a55ab854701074f6fb796ba23fe7953b67fbd8e`。
- 沿用第一階段 `../release-2026-09-19/REPORT.md` 的 Actions、GHCR digest 與 UI 證據，沒有新增 commit 或 push。
- `/api/network` 與公共 RPC 皆確認 Ethereum Sepolia，chain ID `11155111`。
- 部署後、啟用前的歷史檢查：`/api/wallet/vault` 為 `enabled=false`、`contract=""`；當時尚未修改 VM 設定與 GitHub `EXPECTED_VAULT_ADDRESS`。其後啟用與存提結果見下方「線上啟用與 UI 存提驗收」，此項不是最終狀態。

## 查重與編譯

讀取指定四份文件、部署流程與現有 Solidity 編譯器設定；文件均記載公共部署待執行，未找到已發布 ETHVault 部署紀錄。另查當時三個公開站帳戶的公共 Sepolia nonce 與所有候選 CREATE 地址，沒有匹配合約。查核範圍見 `chain-preflight.json`；這不是所有外部部署者／CREATE2 的全鏈搜尋。VM 管理連線受阻，無法聲稱已核對 VM 所有歷史檔案。

`npm run check --prefix contracts` 成功，沒有覆寫既有 artifacts。使用同一組來源與設定另產生 deployment/runtime/metadata 證據；creation bytecode 與 repository artifact 完全一致。

- Compiler：`0.8.37+commit.f401782d.Emscripten.clang`
- Optimizer：enabled，200 runs；EVM：Cancun
- Creation bytecode：882 bytes；runtime：850 bytes
- 只部署 `ETHVault`，未部署攻擊者／拒收測試合約。

## 專用錢包與補款

- 新增帳戶：`Vault 驗收 2026-09-19`
- Account ID：`bed966601653c8aebaab79b4a1dfa089`
- 地址：`0x11A0130EecDF4648efed6a0E3A8901Bf9E44B293`
- 公開 UI 領取 `0.001 Sepolia ETH`，收據 status=1。
- 補款 hash：`0x351f2cb77db60bb6b7c07ca9fec6c037272603c7a4933faf4bb1c45337e54393`
- `funding-ui.png`、`funding-receipt.json` 保存實際結果。
- 密碼、加密 keystore、已簽部署交易只保存於 Mac `~/.ssh/wallet-demo-secrets/` 的專用 0600 檔案；沒有寫入 Git／本證據目錄或輸出秘密內容。

## 部署與原始碼驗證

- 合約：`0xDbB49ee9eC6ab924eA2D3e37Fd594b06648247bf`
- 部署 hash：`0x3642cc5228da53c16a288e5c9a1da1f8399bc65a9b8b333f165ec0f43bd3a51d`
- 區塊：`11732278`；receipt status=1，canonical block hash 相符。
- 更新檢查時間 `2026-09-18T18:17:17.385Z`：108 confirmations，已 finalized。
- Gas used：`259269`
- Effective gas price：`1061375204 wei`
- Gas cost：`275181687765876 wei` = `0.000275181687765876 ETH`
- 部署前錢包：`1000000000000000 wei`
- 部署後錢包：`724818312234124 wei`
- 部署 value=0；前餘額 − Gas = 後餘額，精確相符。
- Chain runtime 與編譯 runtime 完全相同，包括 metadata；部署 tx input 與 creation bytecode 完全相同。

部署前有兩次預檢因計算 `2 × baseFee + tip` 超過預設 2 gwei 而退出，皆未簽署。之後將交易 fee cap 固定上限 2 gwei，仍要求當時 baseFee+tip 可支付、總費用上限不超過 0.0007 ETH，保留 0.0002 ETH 餘額。實際 gas limit 314992、最大費用 0.000629984 ETH。沒有提高上限，也未降低任何驗收斷言。

Sourcify verification ID：`b289d5a6-9d8d-4126-b01f-bd191372c075`。
`2026-09-18T17:56:38Z` 驗證成功，match/creationMatch/runtimeMatch 均 `exact_match`。另以 contract GET API 獨立讀回完整結果，保存於 `sourcify-contract.json`。

- [Sourcify 驗證結果](https://sourcify.dev/server/v2/contract/11155111/0xDbB49ee9eC6ab924eA2D3e37Fd594b06648247bf?fields=all)
- [Sepolia 合約](https://sepolia.etherscan.io/address/0xDbB49ee9eC6ab924eA2D3e37Fd594b06648247bf#code)
- [部署交易](https://sepolia.etherscan.io/tx/0x3642cc5228da53c16a288e5c9a1da1f8399bc65a9b8b333f165ec0f43bd3a51d)

Sourcify 自動轉送至 Etherscan 回報當日 500 次額度已滿；Blockscout 回報 429。這些是外部 explorer 的轉送結果，不否定 Sourcify exact_match。Etherscan 手動表單已選 Standard JSON、正確 compiler 與 MIT，停在服務條款確認；尚不能宣稱 Etherscan 已驗證。

## 線上啟用與 UI 存提驗收

Oracle 登入恢復後，經使用者同意臨時新增本機 IP/32 TCP 22 規則。VM 仍使用第一階段同一 SHA/digest，但舊 Compose 缺少 vault 映射。僅補上該映射及 `.env` 的 `SEPOLIA_VAULT_ADDRESS`，取得既有 deploy.lock，stop-before-start 重建，verify-demo.py 通過。保留 既有 wallet 資料目錄掛載，沒有刪除 volume。完成後已移除臨時入站規則；見 `ssh-ingress-removed.png`。

GitHub `EXPECTED_VAULT_ADDRESS` 與容器設定各自核對為本報告地址；指定完整 release SHA 與地址的 `test:live` 成功，包含桌面合約啟用、Solana/TRON 導覽、手機選單、無水平溢位及頁面錯誤。此為本機對公開站執行既有唯讀驗收，沒有新增或冒稱遠端 Actions run；第一階段 Actions 證據保持原樣。

專用錢包透過公開 UI 完成兩次「預估費用 → 核對 → 密碼簽署 → 廣播 → 更新成功狀態」，不是 CLI 合約呼叫。review/success PNG 保存操作與結果。使用者錢包未參與。

| 項目 | 存入 | 取回 |
|---|---|---|
| Hash | `0x44528916b2b9d597dbef252b7020e9cabad0ffc8d424f00ab65cdb00493eb4b1` | `0x9f600e5772bd4570e0da9a4ea3950cac6c37cf8e1e28bdac30367c340ebe5b53` |
| 區塊 | 11732386 | 11732392 |
| Receipt | status=1 | status=1 |
| 事件 | Deposited | Withdrawn |
| 精確金額 | 100000000000000 wei | 100000000000000 wei |
| Gas used | 47270 | 32635 |
| Effective gas price | 1109462836 wei | 1053463660 wei |
| Gas cost | 52444308257720 wei | 34379786544100 wei |
| 錢包 before → after | 724818312234124 → 572374003976404 wei | 572374003976404 → 637994217432304 wei |
| balanceOf before → after | 0 → 100000000000000 wei | 100000000000000 → 0 wei |

收據事件的 emitter、帳戶與金額均精確相符，區塊 hash 與 canonical block 一致。最後收據快照：存入 4 confirmations、取回 2 confirmations，兩者當時尚未 finalized；不得把確認數等同 finality。API history 同樣為 succeeded，Gas 相符；合約存款回到基準 0，錢包總減少 86824094801820 wei，恰為兩筆 Gas 總和。

## 限制與 Git 狀態

- Etherscan 驗證尚未完成：Sourcify 轉送遭當日額度限制，手動條款尚未取得同意；Sourcify exact_match 已完成原始碼驗證。
- 這是測試網學習實作，未經主網安全審計；沒有使用主網資產或付費服務。
- 本次僅更新本機文件與證據，沒有 commit/push，線上 SHA 保持第一階段版本。


原始收據、執行紀錄、截圖與原始 SHA256SUMS manifest 的固定版本連結及逐檔雜湊，見[封存索引](../archive.json)。本頁僅同步導覽；當日驗收的版本與結果不變。
