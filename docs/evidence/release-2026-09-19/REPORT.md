# 第一階段發布驗收 — 2026-09-19

第一階段完成。公共 Sepolia 合約部署、原始碼驗證及存提尚未執行。

- 基底／發布前 X-App-Version：`66ae38bb4721d15b1993cf6343670883061c305b`。
- 發布後 X-App-Version：`3a55ab854701074f6fb796ba23fe7953b67fbd8e`；HTTP 200，`{"mode":"wallet","status":"ok"}`。
- [Verify run 35370366472](https://github.com/a861252012/multi-chain-wallet/actions/runs/35370366472)；push event，全部必要 jobs 真正執行成功，沒有 skipped job。
- 歷史映像 tag：`sha-3a55ab854701074f6fb796ba23fe7953b67fbd8e`；原始 registry 路徑見封存紀錄。
- Repository digest：`sha256:37ca634d51b67c79445f73d7d5a8109bda569cddc1bbbb6a37ffd0a4b222ec18`。Actions docker push log 與獨立 registry inspect 相符。

## 必要 jobs

| Job | 結果 | 完成 UTC |
|---|---|---|
| [vulnerabilities](https://github.com/a861252012/multi-chain-wallet/actions/runs/35370366472/job/105682630700) | success | 2026-09-18T16:46:18Z |
| [deployment (ubuntu-latest)](https://github.com/a861252012/multi-chain-wallet/actions/runs/35370366472/job/105682630883) | success | 2026-09-18T16:47:17Z |
| [go](https://github.com/a861252012/multi-chain-wallet/actions/runs/35370366472/job/105682630944) | success | 2026-09-18T16:50:28Z |
| [deployment (ubuntu-24.04-arm)](https://github.com/a861252012/multi-chain-wallet/actions/runs/35370366472/job/105682630995) | success | 2026-09-18T16:47:08Z |
| [solidity](https://github.com/a861252012/multi-chain-wallet/actions/runs/35370366472/job/105682631846) | success | 2026-09-18T16:45:54Z |
| [browser](https://github.com/a861252012/multi-chain-wallet/actions/runs/35370366472/job/105682633357) | success | 2026-09-18T16:48:26Z |
| [publish](https://github.com/a861252012/multi-chain-wallet/actions/runs/35370366472/job/105684150805) | success | 2026-09-18T16:52:12Z |
| [live](https://github.com/a861252012/multi-chain-wallet/actions/runs/35370366472/job/105684678881) | success | 2026-09-18T16:54:31Z |

Go job 通過 race、vet、格式與 go fix 差異檢查；Browser job 在 Node 22 實跑 fixtures 與帶 race 的 simulated EVM E2E。兩個 deployment job 分別在 Linux AMD64 與 ARM64 執行隔離映像 smoke。publish 另建置及 smoke 確切發布映像，再推送 GHCR；live 於新版上線後實跑公開 UI。

## UI 與驗收邊界

- CI live：EVM 收款／發送／活動／智慧合約導覽成功；Solana/TRON 切換保留活動頁；375px 手機導覽成功，無水平溢出、無 page error，結束時版本未變。
- Vault API：`enabled=false`、contract/balance/balanceRaw 空字串。新版正確顯示「此環境尚未開放合約操作」，操作表單、餘額區及紀錄區隱藏。
- 補充桌面 1280×900 與手機 375×812 截圖已人工檢視；手機返回合約頁等待停用提示完成後截圖。
- 補充截圖程序攔截全部非 GET/HEAD 請求；POST `/api/wallet/token` 為代幣查詢，未放行其他非唯讀請求。
- 沒有領幣、簽署、廣播公共鏈交易、部署 Solidity、變更 VM 設定或讀取機密。

## 發布前核對

- `DEMO_DEPLOY_ENABLED=true`、`DEMO_RUNNER=ubuntu-latest`；`EXPECTED_VAULT_ADDRESS` 未設定，未更動任何變數。
- 176 個程式／建置／測試來源逐檔 manifest 相符，整體 SHA-256 為 `62073e35ea47dfd619eaec13779cb3bb1606e2a9c3ba2c319e649da67f75765f`。

## 證據與最後狀態

Actions logs、JSON、coverage、registry inspect、health 與截圖的原始位元組封存於固定 Git revision；參見[封存索引](../archive.json)。歷史映像路徑與當時畫面保留於原始證據，不把當日輸出改寫成目前版本。


後續第二階段的公共 Sepolia ETHVault 部署、原始碼驗證與存提結果，另見 [Vault 驗收報告](../vault-sepolia-2026-09-19/REPORT.md)。本頁第一階段仍僅記錄應用程式發布驗收。
