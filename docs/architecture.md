# Multi-Chain Wallet architecture

```mermaid
flowchart LR
 UI[Local browser: intent and confirmation] --> HTTP[Go HTTP / local-origin + CSRF guards]
 HTTP --> EVM[EVM wallet service per account and network]
 HTTP --> SOL[Independent Solana Devnet service]
 HTTP --> TRON[Independent TRON Shasta service]
 HTTP --> READ[Read-only public-address observation]
 EVM --> QUOTE[Integer amounts / simulation / bound quotes]
 QUOTE --> KEY[Password-decrypted signing key]
 KEY --> JOURNAL[Atomic signed transaction journal]
 JOURNAL --> RPC[EVM RPC with checked fallback]
 RPC --> RECEIPT[Canonical receipt / finality / activity]
 SOL --> ED[Ed25519 / blockhash-bound transfer]
 ED --> SJ[Solana signed-byte journal]
 SJ --> SRPC[Devnet RPC / genesis guard]
 TRON --> TJ[Local Protobuf / signed transaction journal]
 TJ --> TRPC[Shasta RPC / Solidity-node receipts]
 READ --> RPC
 RECEIPT --> UI
 SRPC --> UI
 TRPC --> UI
```

## Transaction behavior and evidence

| Question | Evidence to inspect |
|---|---|
| What happens if broadcast succeeds but the HTTP response is lost? | journal-before-broadcast, quote ID lookup, exact raw-byte retry, browser response-loss test |
| How can several replacement hashes refer to one payment? | shared EVM nonce, preservation of payload, group receipt winner and reorg behavior |
| How do you avoid floating point errors? | Go big.Int / integer lamports, decimal parsing, raw units in CSV and confirmations |
| How are forks handled? | receipt block-hash canonical comparison, fresh versioned state update, finalized RPC tag |
| Is adding an L2 just changing an RPC URL? | OP L1/operator fee reserve, receipt fee accounting, separate chain IDs and journals |
| What differs on Solana? | Ed25519, native message format, recent blockhash + last-valid height, signatures and finalized commitment |
| How much does the scanner cover? | saved start/cursor, errors preserve cursor, bounded scans, no complete-history claim |

Browser input is untrusted in every deployment mode: chain, contract, recipient, amount and operation are validated server-side. RPC endpoints are a trust boundary; fallback provides transport recovery, not independent consensus. Keys remain in the Go process during signing and cannot be guaranteed absent from every garbage-collected memory copy.

This is a testnet prototype with local and public deployment modes. Local mode is intended for a single operator and checks local Host/Origin and CSRF. In protected public mode, wallet data and writes require operator authentication. With `SHARED_DEMO=true`, visitors can view wallet data and create password-protected EVM test wallets without a website login; signing new transactions and exporting encrypted keys still require the wallet password. Existing-wallet administration remains restricted. These are server-enforced policies, not separate frontend trust levels.

EVM, Solana and TRON are separate implementations with different recovery/capacity limits. The shared demo is not a production custody service. There is no audited cryptography claim, fiat valuation, bridge or claim of production readiness.

## Implementation boundaries

- **Duplicate request versus duplicate business payment:** [`Service.Send`](../internal/wallet/service.go) holds `sendMu` and calls `FindByQuoteID` before signing. Reusing one quote returns its journal record; [`FindByQuoteID`](../internal/wallet/journal.go) includes archives. This does not identify the same business payment across different quotes, installations or lost journals. The single-in-flight rule does not replace durable upstream business deduplication. The separate [PaymentEscrow](payment-escrow.md) feature adds on-chain deduplication for `(buyer, order reference)` within one configured contract; it does not deduplicate arbitrary transfers or orders across contracts.
- **Externally consumed nonce:** quote and send can release the corresponding in-flight guard only when a fresh `eth_getTransactionCount(address, "finalized")` exceeds the signed transaction's nonce, with matching sender and chain. Network verification surrounds the query; unsupported tags, RPC errors and latest-only progress preserve the guard. Evidence is request-local and queried again after restart, rather than persisted as a payment outcome. History exposes `nonceConsumed` separately from `state` and `finalized`, and `canCreateTransaction` indicates that this snapshot has no remaining in-flight blocker after reconciliation. The original record and receipt lookup remain available; neither recovery nor the UI automatically retries a payment. A later receipt can still establish success or revert. Users must check the original payment before creating another one.
- **Crash versus returned broadcast error:** `Send` persists `pending` with signed bytes before calling RPC. Only after RPC returns does it attempt `submitted` or `broadcast_unknown`. A crash before that update may leave `pending`. `AppendAtomic` returns the initial version; `UpdateStateAtomicIfVersion` prevents an older broadcast result from overwriting a concurrent newer record. This is not a rule forbidding legitimate reorg updates. See the [recovery evidence matrix](demo-script.md#recovery-evidence-matrix).
- **OP fees:** [`RollupFee`](../internal/chain/opfees.go) estimates L1 data plus operator charges; `ReceiptFee` combines execution cost, receipt `l1Fee`, and an operator query bound to the receipt block hash. Operator fees were introduced by Isthmus. The estimate is not a type-2 on-chain cap on total charges. [Official fee specification](https://docs.optimism.io/op-stack/transactions/fees).
- **KMS:** Remote signing and KMS integration are not implemented; signing uses the local encrypted keystore.
- **TRON encoding:** constructing transaction bytes locally avoids relying on a remote transaction builder for recipient and contract parameters. TAPOS data, broadcast delivery and receipt observations still depend on RPC responses; local encoding does not eliminate all RPC or transport attacks.

## Type and storage boundaries

`internal/web` validates HTTP inputs and maps wallet results to response DTOs. `internal/wallet` owns transaction rules and converts durable records to domain types; `internal/chain` handles RPC wire values. For example, `journalRecordFromDisk` validates a saved EVM transaction before it is used by the wallet service. The signed raw bytes stay in the journal; a web response never serializes that disk record directly.

`TestLayerBoundaries` checks import direction and external field types in web DTOs. It does not prove every handler's data flow. See [Go conventions and verification](go-style.md) for the related tests and compatibility rules.

## Local wallet management

EVM wallets have local names (1–40 characters) and an archive flag stored in
`account.json`, separately from encrypted keys and transaction journals. The
wallet manager supports adding a named setup slot, completing creation/import,
renaming, archiving and restoring. Cancelling the name form creates no slot.
An unfinished slot remains available as “Continue setup”.

Archiving hides a wallet from the normal switcher; it does not erase keys,
remove on-chain assets, revoke signing access or stop background maintenance.
At least one wallet must remain unarchived. Archived wallets still count toward
the existing 20-wallet limit. Existing unnamed wallets receive a display name
without migrating their keys. Solana and TRON retain their separate wallet flows.
