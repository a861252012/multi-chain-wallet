# Polygon、Solana 發送驗收 — 2026-09-16

兩條鏈各完成一筆原生幣轉帳。使用獨立測試錢包，直接呼叫 Multi-Chain Wallet 的 `Quote`、`Send`、`History`，再透過公開 RPC 核對交易。這次沒有走瀏覽器發送介面。

| 網路 | 發送金額 | 交易 | 區塊／slot | 結果 | 實際手續費 |
| --- | --- | --- | --- | --- | --- |
| Polygon Amoy | 0.001 POL | [0xa2effa44…](https://amoy.polygonscan.com/tx/0xa2effa4429a475a9ac3089f65a400ef19134348e840bcd457d31b4e2700c2f23) | 47743032 | `status=0x1`，finalized | 0.000630000001323 POL |
| Solana Devnet | 0.001 SOL | [3FwSEuZ9…](https://explorer.solana.com/tx/3FwSEuZ92sbM5jjy5qJRZ79u8m2JbEWCWwioabotfTpttPXTS1dBSCgYjcvZ48xSutZVY14Vu4NxNvRC8GM61Zij?cluster=devnet) | 499312462 | `meta.err=null`，finalized | 0.000005 SOL |

## 核對結果

核對時間：2026-09-16 13:29:05 UTC。

- Amoy：RPC chain ID 為 `80002`；交易金額為 `1000000000000000` wei，收款地址為 `0x219465ECA5EB591b7587E4B1Aa97F34971206Ab9`。成功收據的 block hash 與同高度區塊一致，`finalized` 區塊已超過交易高度。
- Solana：genesis hash 符合 Devnet；System Program transfer 的收款地址為 `8MdhSYukGpauTfZPnFC51mi6e9FLm1ndXGm5ztbeVjMF`，金額為 `1000000` lamports。收款餘額增加相同金額，signature status 與 finalized transaction 都沒有執行錯誤。
- 重新開啟兩個錢包後，Multi-Chain Wallet 從日誌讀回相同交易，Amoy 顯示 `succeeded`／`finalized=true`，Solana 顯示 `finalized`。

## 測試幣來源

透過 [OpenFaucet](https://openfaucet.org/) 的瀏覽器 proof-of-work 領取，沒有使用主網資產：

- Amoy：領取 [0.012 POL](https://amoy.polygonscan.com/tx/0x480d79c82aac0629785560ba61155d0186d14bac0cebfd8590f3d16562685d09)，測試發送地址為 `0x93e1BaB0a74a28e229c90606f1047a0F379D8757`。
- Solana：領取 [0.014 SOL](https://explorer.solana.com/tx/4pSBL3UU1ZZcu5PCSeF4EXVqVueDUMnBpCHJY2e9TwjhmCYnBkU8yBk6RqS3ccNDjbAN2VuwhrfpCWrV84dxhHLB?cluster=devnet)，測試發送地址為 `As4ez7f5peg3c5yAQxMAuiWQFBBTZ7ZMgioK6UrEq8K2`。

這次補齊 9 月 15 日留下的兩項原生幣發送驗收。加上前次紀錄，共有七個測試網、15 筆成功交易；不包含水龍頭入金，也不代表代幣、DEX、ERC-4337 或所有故障情境都已驗收。Finality 是當次 RPC 觀測結果。
