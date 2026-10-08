# Wallet operating reference

[README / language editions](../README.md)

Use this guide to run the wallet, send test assets, manage backups and recover interrupted transactions. The Docker setup below is for localhost.

For an existing installation, follow [project naming and upgrades](project-naming.md) before starting Compose so it uses the existing wallet data volume.

[交易恢復驗證](demo-script.md) · [9 月 15 日歷史驗收](onchain-acceptance-2026-09-15.md) · [9 月 16 日後續驗收](onchain-acceptance-2026-09-16.md) · [程序中止恢復驗證](process-recovery.md)

## Why this project

Multi-Chain Wallet 處理以下交易生命週期問題：

- RPC 可能已收到交易，但 HTTP response 在回程中遺失。
- 同一個使用者操作可能因重試、並發請求或程序重啟而被處理多次。
- 「已廣播」、「已上鏈」、「執行成功」與「已 finalized」是不同狀態。
- EVM nonce、Solana recent blockhash 與 TRON TAPOS 的有效期限和復原方式不同。

交易簽署後先存檔，再送到 RPC。同一 quote ID 的重試沿用已保存的交易；廣播結果不明時，保留原始簽名資料供重送，並分開核對收據、所在區塊與最終確認狀態。

### Engineering highlights

| 問題 | 處理方式 | 可重跑證據 |
| --- | --- | --- |
| 廣播結果不確定 | 廣播前將 hash 與 signed raw transaction 原子寫入 journal；重試不重新簽署 | `TestUnknownBroadcastRestartReusesRaw` |
| 並發重送 | 同一 quote 只簽署／廣播一次，後續請求回傳已有記錄 | `TestConcurrentSendSignsExactQuoteOnce` |
| 程序中止 | 重啟後讀回 journal、保留 nonce 保護並重播同一筆 bytes | `TestProcessKillRestartReusesRaw` |
| 慢查詢覆蓋新狀態 | 以 version-checked journal update 保留較新的 receipt／reorg 觀測 | `TestSendConcurrentStateUpdatePreservesLatest*` |
| 鏈重組 | 重新核對 receipt block hash 與 canonical header，不只依賴 confirmation count | `TestHistoryUpdatesReorgDetectedFromRPC` |

### Scope and evidence

- **已實作：** 五個 EVM 測試網、Solana Devnet 與 TRON Shasta 的獨立錢包流程；Sepolia 上的 WETH／Uniswap V3 交換流程。
- **鏈上驗收：** [2026-09-15 歷史快照](onchain-acceptance-2026-09-15.md)記錄五個測試網共 13 筆成功交易；[2026-09-16 後續紀錄](onchain-acceptance-2026-09-16.md)補上 Polygon Amoy 與 Solana Devnet 各一筆原生幣轉帳。這些紀錄不代表目前版本已重新驗收。
- **測試證據：** Go unit/integration tests、race detector、Mock RPC 與 process-kill recovery test。Mock 通過不代表真實鏈或 production 環境已驗證。
- **使用限制：** 僅支援測試網，未經獨立安全稽核；公開共用 Demo 不提供正式託管服務、SLA 或 RPC quorum。

## One-click test tokens

The wallet has **領取測試幣** buttons. EVM and TRON requests use a locally configured test-only dispenser account; the recipient does not need to enter a wallet password. Transfers are real testnet transactions, not simulated balances. Solana calls Devnet `requestAirdrop` directly and reports unavailable/rate-limited results.

- Ethereum Sepolia: 0.001 ETH or 0.01 test USDC.
- Base / OP / Arbitrum Sepolia: 0.0001 ETH.
- Polygon Amoy: 0.1 POL, **requires POL stock in the dispenser**.
- TRON Shasta: 5 TRX from a separate dispenser wallet.
- Solana Devnet: request 0.01 SOL; the upstream faucet can refuse or time out.

The dispenser is disabled by default. Configure `TEST_FAUCET_CONFIG` with an owner-only (0600), ignored JSON file containing `accountId` and `password` for a dedicated EVM account registered in this installation. Optionally include `tronDir` and `tronPassword` for a separate Shasta wallet directory. Paths are configurable and interpreted inside the container. Never use a mainnet wallet or commit this file. A local installation needs funded EVM and TRON sources; cloning this repository does **not** include stock or credentials.

Amounts and networks are fixed by the server. EVM recipients must belong to the local account catalogue; TRON and Solana use the current local wallet address. EVM/TRON reuse matching transfers from the last hour, including unknown broadcasts, across restarts. The same-amount check includes manual transfers from the dispenser account. An hourly per-network limit also counts its other transactions. The dispenser shares the existing signer, network checks, journal and nonce protection; it cannot mint unlimited native coins. Refill its stock with a faucet when necessary. Automatic refill via third-party login/CAPTCHA is not implemented.

See [button acceptance evidence and limits](test-faucet-2026-09-15.md).

## Historical transaction evidence

The September 15 acceptance run completed **13 outgoing transactions across five testnets**: Ethereum Sepolia (native transfer, WETH wrap/unwrap, approvals, both WETH/USDC swap directions and USDC transfer), Base/OP/Arbitrum Sepolia (native transfers), and TRON Shasta (TRX and faucet test-USDT transfers). See [September 15 transaction links and limits](onchain-acceptance-2026-09-15.md). The [September 16 follow-up](onchain-acceptance-2026-09-16.md) records one native transfer each on Polygon Amoy and Solana Devnet, bringing those two historical records to 15 transactions across seven testnets. These dated results do not establish fresh acceptance of the current code or every supported operation.

Recheck the published receipts without a wallet or signing:

```sh
go run ./cmd/verify-onchain-evidence
```

This contacts public testnet RPCs, validates receipt inclusion and reports current finality observations. Snapshots and single-provider checks are not an independent consensus proof.

## Run with Docker

```sh
cp .env.example .env
# WALLET_ACCESS_TOKEN may remain empty for the localhost demo
docker compose build
docker compose run --rm --no-deps app go mod download
docker compose up -d
```

Open <http://localhost:8090> directly. The localhost demo does not require login. Keep the host port bound to `127.0.0.1`. Optionally set `WALLET_ACCESS_TOKEN` (at least 32 characters) to enable a browser session login; CLI clients can use Basic authentication with username `multi-chain-wallet`. Never commit credentials. `WALLET_DIR=/data/wallet` uses the dedicated `wallet_data` volume. Wallet persistence uses an encrypted keystore and an atomic transaction journal.

```sh
docker compose ps
docker compose logs -f app
docker compose restart app
```

Source and embedded HTML/CSS/JS changes require `restart app`. Changed Compose environment/volumes require `docker compose up -d --force-recreate app`.

## Use the wallet

1. Select an account and test network. Create a wallet with a password, or restore an English BIP-39 test mnemonic. New wallet passwords must contain 12–128 Unicode code points in both the UI and backend; spaces are preserved. Existing keystores remain decryptable and exportable with their original passwords, including those accepted under the previous byte-count rule.
2. Write down the generated 12 words **offline and in order**. They are displayed once and never saved in plaintext. Do not send them to an AI, logs, screenshots, or chat. Confirm backup to clear them from the screen.
3. Fund the displayed address using **領取測試幣** when a funded local dispenser is configured, or use an external faucet. The dashboard links to [Google's Sepolia faucet](https://cloud.google.com/application/web3/faucet/ethereum/sepolia), [Ethereum's faucet list](https://ethereum.org/developers/docs/networks/#sepolia), and [Circle's test USDC faucet](https://faucet.circle.com/). External services may require login, verification or impose limits. Native test coins are required for network fees even when transferring tokens; see the dispenser limits above.
4. Enter a recipient and amount, request a quote, and verify the network, recipient, asset contract, exact amount and maximum gas fee. Sepolia WETH and USDC use pinned local metadata that must match the RPC response. For any other ERC-20 or TRC-20, enter the exact smallest-unit integer shown in the signing review; this prevents an RPC-provided `decimals()` value from changing the signed amount. Enter the wallet password to sign and broadcast.
5. Refresh history to check the receipt. A broadcast result is not confirmation of execution. If the result is uncertain, use **重新廣播原交易**; this reuses the original signed bytes and hash.

The dashboard's first-transaction guide can prefill a **0.000001 ETH self-transfer** or **0.000001 ETH → WETH wrap**. These controls only fill the existing forms; quotes and password confirmation remain separate. A self-transfer returns the principal to the same address and consumes test ETH gas. A positive ETH balance does not prove it covers the selected transaction's fee. Funding checks distinguish zero balance from unavailable RPC data.

ERC-20: select a default or previously saved token for `transfer` or `approve`. The wallet reads its `symbol`, `decimals` and `balanceOf`; the UI no longer has a form for adding arbitrary token contracts. For approval, the recipient field is the **spender**, not a transfer recipient. Only an explicit finite allowance is permitted; `0` revokes. Existing nonzero allowance must first be set to zero before a new nonzero allowance is accepted. Token metadata is supplied by the contract and is not proof of legitimacy. Tokens with nonstandard metadata or special transfer semantics may be unsupported.

Refreshing the wallet queries the configured default tokens and browser-saved tokens for the selected account/network. Failed token queries retain the selected asset and last known balance, visibly marked as outdated; a successful retry clears that warning. Quotes independently recheck chain data. Token precision, raw amounts and event evidence are available in expandable details; recipients, asset contracts, readable amounts and fee limits remain visible for confirmation.

## Keys and backup

- BIP-39 English mnemonics; BIP-32/BIP-44 path `m/44'/60'/0'/0/0`, the first Ethereum address per mnemonic. Additional local accounts use independent backups.
- The optional BIP-39 passphrase is fixed to empty. A keystore encryption password is a separate concept; importing a mnemonic from a passphrase-protected wallet will not restore that wallet.
- Ethereum V3 keystore encrypted with geth StandardScrypt settings. Files use `0600`; wallet directory uses `0700`. A process lock prevents two app instances from operating the same directory.
- The backup control downloads a password-protected V3 keystore after verifying the password. The UI restores from mnemonic or a V3 scrypt keystore (up to 8 KB, bounded KDF parameters), re-encrypting imported keys with the chosen local password. Password changes are atomic and do not alter old backups; export a new backup afterward. Keep the backup password separately.
- Wallet passwords entered for signing are not written as plaintext files, and no unlocked session is retained. The optional test-faucet config is different: it stores a dedicated dispenser password in a `0600` JSON file and keeps it in process memory. Mutable key material is cleared where practical; Go garbage collection prevents a guarantee that every in-memory copy is erased.
- Preserve the **whole wallet volume**, including `journal.json`, when moving/restarting an active wallet. Restoring only the mnemonic does not restore this application's transaction history or pending-operation safeguards.

`docker compose down` keeps data. **`docker compose down -v` deletes the wallet, journal and other named volumes.** Do not run it as a routine stop command. If mnemonic display is lost during creation, the encrypted key may still exist: use the password-protected backup; the app never overwrites an existing wallet to retry creation.

## Transaction behavior and limits

- The selected EVM testnet chain ID (`11155111`, `421614`, `84532`, `11155420` or `80002`) is checked before RPC operations. Amounts use integer arithmetic; no floating-point currency math.
- ETH and token transactions use RPC gas estimation; failed estimation stops the operation. A quote binds action, recipient/spender, contract, calldata, nonce and fee caps for 120 seconds. The server rechecks nonce, funds and relevant token conditions before signing; it never silently raises an approved cap.
- Signed raw bytes and hash are synced to disk **before** broadcast. Repeated sends of one quote return the same journal entry. Timeouts and ambiguous responses remain uncertain; they are not treated as proof that nothing was sent.
- EVM deduplication is scoped to the same account/network and `quoteID`, including retained archives. Different quotes are not a business-level payment identity. An upstream payout service would need its own durable business key and request matching; an HTTP header alone would not provide that guarantee.
- The initial durable EVM state is `pending`. A returned broadcast error leads to a version-checked `broadcast_unknown` update; success leads to `submitted`. A process stopped before that update may leave `pending`. Concurrent newer journal updates are preserved. These states do not establish whether a node received the transaction; see the [recovery checks and evidence limits](demo-script.md).
- One outstanding nonce per account/network, with multiple replacement attempts allowed. Speed-up preserves the transaction payload; cancel signs a zero-value self-transfer at the same nonce. Both raise fee caps by at least 20% over known attempts and require a new preview/password confirmation. This local policy cannot guarantee node acceptance or inclusion; the original may win. A mined attempt resolves its nonce group, and a reorg can make it unresolved again.
- The active journal holds up to 1,000 records. At 900 records, history refresh archives finalized records outside the latest 100 before shortening the active file. Archives retain signed records and quote IDs for restart/idempotency. If there are not enough finalized records, capacity errors remain explicit. Preserve all `archive-*.json` files with the wallet volume.
- Background maintenance checks unfinalized records against canonical receipts. Finality is observed through the RPC `finalized` block tag and a canonical receipt-block hash recheck, never inferred from a confirmation threshold. RPC failures do not establish failure or finality. Finalized archives assume the chain does not violate finalized consensus; no local wallet can independently guarantee a remote node is truthful.
- Automatic activity synchronization is opt-in. It saves a per-account/network cursor and scans finalized block bodies and receipts, discovering ERC-20 Transfer log candidates without requiring a known contract filter (up to 200 token candidates per saved scan state). Unsupported block-receipt RPCs fall back to individual receipts. Errors retain the cursor; restarts resume. The displayed start block defines coverage. This is not a claim that the entire chain has already been indexed.
- A successful ERC-20 transaction receipt proves execution status; it does not by itself prove the recipient's economic balance change for fee-on-transfer, rebasing or malicious tokens.
- Wallet POST endpoints require a per-process CSRF token, exact-origin checks, allowed local Host, JSON media type and a bounded strict body. Restarting invalidates browser CSRF state; reload the page.
- Up to 20 independent local accounts can be created/restored and switched in the UI. Each account shares its encrypted key across supported test networks, while quotes, journals and activity indexes remain separate. Duplicate signing addresses are rejected to avoid independent nonce journals for the same key. Accounts and their background workers are loaded on first visit after startup. Mainnet and hardware signing are unsupported. Public access requires the protected or shared-demo configuration; shared visitors can create password-protected EVM test wallets, but cannot import or administer existing wallets.

## Network selection and RPC fallback

Use the network selector for Ethereum, Arbitrum, Base or OP Sepolia and Polygon Amoy, or open `/solana/` and `/tron/` for independent wallets. Native-currency/ERC-20 operations and replacement transactions are available on all five EVM testnets; the configured Uniswap exchange remains Ethereum Sepolia only. Assets received on one network are not balances on the other. Arbitrum gas estimation includes the parent-chain data component; see [Arbitrum gas estimation](https://docs.arbitrum.io/arbitrum-essentials/how-to-estimate-gas).

Configure `SEPOLIA_RPC_FALLBACK_URLS`, `ARBITRUM_SEPOLIA_RPC_FALLBACK_URLS`, `BASE_SEPOLIA_RPC_FALLBACK_URLS` or `OP_SEPOLIA_RPC_FALLBACK_URLS` with up to three comma-separated backups. Transport errors, HTTP 429 and server failures try the next endpoint using identical request bytes. Each endpoint is checked for the selected chain before forwarding operations. Deterministic JSON-RPC errors are not retried. There are per-attempt and overall request deadlines; fallback is not a promise of availability.

## Sepolia exchange

- **ETH → WETH / WETH → ETH:** deposit/withdraw against Uniswap's published Sepolia WETH9 at `0xfff9976782d46cc05630d1f6ebab18b2324d6b14`. The conversion is 1:1; ETH gas is additional.
- **WETH ↔ test USDC:** exact-input, single-pool Uniswap V3 swaps through the published Sepolia SwapRouter02 (`0x3bFA4769FB09eefC5a80d6E87c3B9C650f7Ae48E`), using QuoterV2 and the factory's `getPool`. Circle's test USDC is `0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238`.
- Choose a pool fee, input amount and slippage. Approve only the required input amount first, wait for its receipt, then request the swap quote. Revoke an existing nonzero approval before changing it. Each step has its own preview and password confirmation.
- The signed router call binds the wallet recipient, exact input, minimum output, pool fee and an on-chain deadline. No arbitrary router, token pair, recipient or calldata is accepted. Failed simulation, unavailable pool, insufficient allowance/balance or zero output stops the operation.
- Default slippage is 0.5%; supported range is 0.01–5%. Quotes expire after at most 120 seconds. Testnet pool prices do not represent USD valuations. No bridge, cross-protocol aggregator, LP creation or arbitrary token swapping is implemented. Pool comparison chooses the highest gross output among successful quotes from four fee tiers, not the best gas-adjusted return. Failed pools are displayed. Each executable quote independently rechecks the chosen pool and simulates the swap.

Read-only integration verification (no key or broadcast):

```sh
docker compose run --rm --no-deps \
  -e MULTI_CHAIN_WALLET_LIVE_RPC=https://ethereum-sepolia-rpc.publicnode.com \
  app go test ./internal/wallet -run TestSepoliaExchangeReadOnly -v -count=1
```

This checks the local implementation against Sepolia token metadata and live quotes. It does **not** claim a completed swap. References: [Uniswap Sepolia deployments](https://developers.uniswap.org/docs/protocols/v3/deployments/v3-ethereum-deployments), [Circle test USDC](https://developers.circle.com/stablecoins/usdc-contract-addresses).

## Guided exchange, observation and diagnostics

ETH → USDC guides wrap → finite approval (or reset when necessary) → swap. USDC → ETH guides approval → swap → unwrap the **actual WETH received in that swap's verified Transfer logs**, not the quoted output or the wallet's whole WETH balance. Each step requires its own quote and password. Session storage retains only public workflow data, quote IDs and hashes; lost broadcast responses can be reconciled with journal quote IDs. Ending the guide does not remove transactions or bypass in-flight checks. Browser storage is not a backup of transaction evidence.

Address-book labels (100 maximum) and watch addresses (20 maximum) are browser-local and separated by EVM network. Labels are rendered as text and never substitute for the full confirmation address. The receive panel provides the canonical checksummed address with one-click copy; the sender must still verify the network. History search filters the locally indexed send history by address, hash, asset or action. Details link to canonical receipt diagnostics.

The read-only observation area is available before wallet creation. Query native/token balances, a known transaction, or candidates in one specified finalized block. This never creates a signer or persistent index. It does not represent a full address transaction history. Diagnostics show endpoint numbers rather than provider URLs, HTTP transport counters and latency to response headers; counters reset with the process and are not a monitoring SLA. `/showcase` presents supported features and recorded transaction evidence, with mock checks labelled separately.

## Base and OP fees

Base Sepolia (`84532`) and OP Sepolia (`11155420`) use the OP Stack GasPriceOracle at `0x420000000000000000000000000000000000000F`. Quotes add a 2× reserve over `getL1FeeUpperBound(unsignedTxSize)` plus `getOperatorFee(gasLimit)`. Before signing, the current estimate must remain within that reserve. **The reserve is an estimate, not an on-chain cap:** type-2 transactions cannot cap L1 fees. Confirmed receipt fees include `l1Fee` from the same transaction/block and the operator fee queried at that block hash. Missing or inconsistent fee data fails the lookup rather than reporting an incomplete total.

Sources: [OP fee components](https://docs.optimism.io/op-stack/transactions/fees), [GasPriceOracle](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/L2/GasPriceOracle.sol), [Base RPC](https://docs.base.org/base-chain/api-reference/rpc-overview). Live node-reported finality can lag; the app does not replace it with a confirmation-count assumption.

## Solana Devnet

Open `/solana/`. This is one independent local SOL account, with BIP-39 + SLIP-0010 Ed25519 path `m/44'/501'/0'/0'` and an empty extra passphrase. It does not reuse the EVM private key or EVM account selector. Runtime seed encryption uses AES-256-GCM and scrypt N=262144/r=8/p=1, a random 32-byte salt and 12-byte nonce. The public address and format version are authenticated. The Multi-Chain Wallet Solana backup format supports local restore and password changes; it is not Ethereum V3 or a Solana CLI keypair file. Old backups retain their old password.

Native SOL transfer only: 9-decimal integer lamports, System Program transfer to an on-curve wallet address, confirmed blockhash, `getFeeForMessage`, pre-sign simulation, signed preflight, and genesis-hash guard. The pinned Devnet genesis is `EtWTRABZaYq6iMfeYKouRu166VU2xqa1wcaWoxPkrZBG`. `SOLANA_DEVNET_RPC_URL` may change the provider but not the allowed cluster. Testnet is intended primarily for validator testing; application work uses [Devnet](https://solana.com/docs/references/clusters).

Signed bytes/signature are persisted before broadcast. Retry preserves the exact bytes; quote IDs deduplicate sends across restart. Quotes expire after 60 seconds or the last valid block height, whichever comes first. Only one transfer remains outstanding at a time. A finalized receipt or an RPC-observed `expired_unconfirmed` state releases that gate; expiry does not prove the original transfer failed, so check it before sending another payment. Unknown/expired signatures stay in the journal for later investigation. Recent history shows 20 records, with a 1,000-record capacity limit. Operations are serialized within the Solana service and bounded by HTTP deadlines. No SPL-token transfers, Solana DEX, durable nonce, replacement/cancel, archival compaction, or independent RPC quorum is claimed for this first SOL implementation.

No existing wallet data is overwritten on create/restore. Preserve `solana-devnet/key.json` and `transactions.json`. Keep the local HTTP service on loopback.

## Receive and account for test assets

Copy the wallet's Sepolia address to receive ETH or ERC-20 tokens. In **活動 → 開發者：同步與匯入**, sync the latest 20 blocks, enter a starting block for the next batch, or import a known transaction hash. These controls are restricted to the operator and hidden in shared-demo mode. Event queries use the loaded default and browser-saved tokens, split into batches of at most 20 contracts. The optional scanner can discover additional token transfers in its configured range. Outgoing journal transactions are included automatically. Repeated imports and synchronization deduplicate by hash. `activity.json` stores transaction hashes; amounts are reconstructed from receipts.

Each page re-fetches up to 20 receipts with bounded concurrency. The view checks transaction identity, signature, chain ID, canonical block hash, receipt status and event provenance before calculating movements. Failed transactions count only sender gas. Pending, unavailable and observed orphaned transactions contribute no amounts; the page flags incomplete results. Self-transfers show both legs, leaving only the fee as net ETH change. Approval is not a token expenditure.

Supported evidence: direct transaction ETH value, ERC-20 `Transfer` logs, and the allowlisted WETH9 `Deposit`/`Withdrawal` logs. On Ethereum Sepolia, the server-configured `SEPOLIA_VAULT_ADDRESS` also enables ETHVault `Withdrawn` logs as ETH income for successful, canonical transactions sent by the wallet to that vault; matching `Deposited` logs annotate the outgoing transaction value without counting it twice. An unset vault address or another contract's events do not provide this evidence. General internal ETH calls, rebasing, fee-on-transfer semantics and historical ranges that have not been scanned may not be fully represented. Token events are evidence of emitted logs, not an independent balance reconciliation or financial audit.

Page totals cover **only that page's verified transactions**, and are grouped by contract address rather than token symbol. CSV exports that page's raw integer amounts, asset addresses, hashes and evidence, including unverified rows without amounts. Import raw amount columns as text in spreadsheets to preserve every digit. The activity view is a test-asset cash-flow record, not a full double-entry accounting system or the current wallet balance. Ordering follows insertion into the index; verified block timestamps are shown separately.

## Verification

```sh
docker compose run --rm --no-deps app go test -race ./...
docker compose run --rm --no-deps app go vet ./...
```

Tests use temporary wallets, published mnemonic fixtures and local mock RPCs. They cover derivation/restore, exact units, keystore encryption, input guards, ETH/token signed payloads, finite allowance/revocation, changed chain/nonce/funds, concurrent duplicate submissions, persistence failure, restart/rebroadcast identity and HTTP origin/CSRF restrictions. Test KDF parameters are deliberately reduced; runtime uses geth standard parameters.

**Mock tests are not Sepolia evidence.** A real end-to-end acceptance requires a user-created/funded test wallet and an actual transaction hash with a Sepolia receipt. No such outgoing transaction is claimed solely because tests pass. Existing read-only Sepolia lookups and UI fixture checks are separate evidence.

### On-chain automated broadcast & receipt verification script

The repository includes `go run ./cmd/send-and-verify` (also available through `scripts/send_and_verify.sh`) for balance checks, fee quoting, on-chain broadcasting, and receipt confirmation:

```sh
# Interactive self-transfer after funding:
./scripts/send_and_verify.sh --send

# Test fee quoting and zero-balance guard without broadcasting:
./scripts/send_and_verify.sh --test-quote

# Verify receipt and confirmation count for an existing transaction hash:
./scripts/send_and_verify.sh --hash 0xadfc05c5d4cf80c8c6b52f3a9eaa73e3c661b7a8d971d6cc57eabadb4677d22c
```

`--test-quote` never calls the send endpoint, regardless of balance. It distinguishes the expected zero-balance error from an RPC failure. With `--send`, it quotes, asks for explicit confirmation and a hidden password input, broadcasts, and polls `/api/transactions/{hash}` until the canonical receipt is mined.

## Code and references

- `cmd/multi-chain-wallet`: startup, configuration, shutdown.
- `internal/wallet`: mnemonic/keystore, exact amounts, ERC-20 ABI, quotes, signing, journal.
- `internal/chain`: Sepolia RPC and receipt checks.
- `internal/web`: HTTP guards and embedded vanilla HTML/CSS/JS.

Go module: `github.com/a861252012/multi-chain-wallet`. No frontend framework or Node build is required.

Standards: [BIP-39](https://github.com/bitcoin/bips/blob/master/bip-0039.mediawiki), [BIP-44](https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki), [EIP-1559](https://eips.ethereum.org/EIPS/eip-1559), [ERC-20](https://eips.ethereum.org/EIPS/eip-20).

## Browser and continuous verification

```sh
npm ci --prefix tests/browser
(cd tests/browser && npx playwright install chromium)
npm test --prefix tests/browser
npm run test:e2e --prefix tests/browser
```

`npm test` serves the repository HTML/JS with local mock API responses. It verifies network navigation, address-book text safety, read-only access before setup, RPC failure states, both guided exchange directions, pool selection, response-loss recovery, diagnostics and mobile layout. `npm run test:e2e` separately runs Chromium against a Go server and simulated EVM to check vault deposits/withdrawals, escrow payments/settlement and order recovery. It requires the Go toolchain and runs with the race detector. Both suites use temporary test data without accessing the runtime wallet or public RPC. GitHub Actions in `.github/workflows/verify.yml` runs both suites alongside isolated Go race/vet/coverage checks and Go CLI tests; consult [GitHub Actions](https://github.com/a861252012/multi-chain-wallet/actions) for actual remote run results.

Read-only new-network acceptance:

```sh
MULTI_CHAIN_WALLET_LIVE_NETWORKS=1 go test -run '^TestAdditionalNetworksReadOnly$' -v -count=1 ./internal/chain
```

`MULTI_CHAIN_WALLET_SOLANA_LIVE_SEND=1 go test -run '^TestSolanaDevnetSendAcceptance$' -v -count=1 ./internal/wallet` is an explicit **Devnet write opt-in**: it creates a disposable test wallet, requests faucet SOL and self-transfers. A faucet refusal is reported as SKIP, never outgoing acceptance. Do not include that opt-in in routine CI.

## Polygon Amoy and TRON Shasta

- `/net/polygon/`: chain ID **80002**, native **POL**, RPC `https://polygon-amoy.drpc.org`, explorer `https://amoy.polygonscan.com`. EVM account keys are shared, while quotes/journals stay network-specific. Native transfers, ERC-20 transfers/finite approvals, speed-up/cancel and receipt-derived activity use the existing EVM implementation. POL labels apply to amounts, fees, account activity and CSV assets. Legacy JSON keys such as `eth`, `feeEth`, `maxFeeEth`, `totalEth` and action `eth` remain wire-compatible and denote the selected network's 18-decimal native currency; check the network ID. Exchange contracts remain Ethereum Sepolia only.
- `/tron/`: **Shasta only**, pinned genesis `0000000000000000de1aa88295e1fcf982742f773e0419c5a9c134c994a9059e`. Uses an independent account and journal under `tron-shasta`. BIP-39/BIP-32 derivation is `m/44'/195'/0'/0/0`, empty extra passphrase, secp256k1 and Base58Check. Reuses the existing bounded Keystore V3 encryption, backup/restore and password-change implementation.
- TRX amounts use integer SUN (6 decimals). Custom TRC-20 contracts supply symbol, decimals and balance through constant calls. Transfer calldata is constructed locally from the confirmed recipient and amount, and simulated before quoting and signing. No preconfigured token is represented as official USDT. Only standard bool-returning `transfer` contracts are supported; a token symbol alone proves no issuer identity.
- Locally construct only the two supported protobuf contract types using the published TRON schema; never sign opaque node-generated transactions. Quote binds addresses, asset, amount, TAPOS reference, expiration and Energy `fee_limit`. SHA-256 of raw data is the transaction ID, and recoverable secp256k1 signatures are persisted before `broadcasthex`.
- Show available Bandwidth/Energy and reserve full bandwidth burn (serialized signed size plus 64 bytes of overhead), account-activation fees when required, and 2× simulated Energy cost. Chain parameters supply SUN prices. Quotes last 60 seconds and Energy reservation is limited to 100 test TRX per transaction. These are conservative estimates, not guarantees: `fee_limit` applies to Energy only, resource/fee conditions can change, and the actual fee comes from the receipt.
- Retry sends identical persisted bytes only while unexpired. Unknown outcomes are retained. One outstanding transaction is allowed; a solidified receipt or `expired_unconfirmed` releases the pending gate. The latter requires an absent receipt and a validated Solidity-node head timestamp beyond the transaction expiration; it does not prove the transfer failed. The signed record is retained and later receipts can still reconcile it. Local time or a failed RPC alone cannot release the gate. This relies on the configured node, not an independent RPC quorum. Native TRX self-transfers are rejected before signing. Latest history shows 20 local outgoing records (capacity 1,000). Successful VM execution returning `false` for TRC-20 is shown as failed transfer. No automated incoming TRON indexer, TRON DEX, staking/resource delegation, multisig, TRON archive compaction, or replacement mechanism is implemented.
- Optional `TRON_SHASTA_API_KEY` is a provider credential, never a wallet key. Redirects are rejected and genesis is rechecked immediately before broadcasts. Mainnets remain rejected. RPC observations rely on the configured provider; independent quorum is not implemented.

Official sources: [Polygon Amoy configuration](https://docs.polygon.technology/pos/reference/rpc-endpoints), [TRON networks](https://developers.tron.network/docs/networks), [resource model](https://developers.tron.network/docs/resource-model), [TRON protobuf schema](https://github.com/tronprotocol/protocol/blob/master/core/Tron.proto).

Read-only acceptance: `MULTI_CHAIN_WALLET_LIVE_NETWORKS=1 go test -run '^TestAdditionalNetworksReadOnly$/80002$' -v -count=1 ./internal/chain` and `MULTI_CHAIN_WALLET_LIVE_TRON=1 go test -run '^TestTronShastaReadOnly$' -v -count=1 ./internal/wallet`. These do not sign or broadcast. See [September 15 live acceptance](onchain-acceptance-2026-09-15.md) for outgoing Shasta transactions and the [September 16 follow-up](onchain-acceptance-2026-09-16.md) for the later Amoy native transfer; neither report is a new verification run.

## Interface preferences

The wallet uses a task-based workspace: an overview, send, receive, exchange, test funding, activity, and settings. Advanced diagnostics and setup details are available on demand. English, Simplified Chinese, and Traditional Chinese (Taiwan) can be switched without reloading or clearing form inputs. Light/dark appearance and language preferences persist in this browser across wallet pages.

UI translations are maintained in `internal/web/static/messages.js`, with Traditional Chinese as the source text and `i18n.js` handling display conversion. Add new UI messages to the table; keep user-defined names under `translate="no"`, and leave API values, amounts, addresses, signing data and unknown external errors unchanged. Set `MULTI_CHAIN_WALLET_UI_SCREENSHOTS` to an output directory when running the browser tests to save the optional English desktop/mobile, light/dark mock screenshots. These screenshots are UI fixtures, not chain acceptance evidence.
