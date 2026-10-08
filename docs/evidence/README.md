# Verification evidence

The records below distinguish local fault-recovery tests from transactions observed on public Sepolia. Each dated report identifies the application or contract revision it checked. A historical result does not certify a later commit or describe an unconfigured environment.

## Public testnet records

| Date (Taiwan time) | Scope | Record |
|---|---|---|
| 2026-09-19 | Application release: CI jobs, image digest, HTTP version and browser navigation. Contract operations were disabled at this stage. | [Release verification](release-2026-09-19/REPORT.md) |
| 2026-09-19 | ETHVault deployment, source verification, a 0.0001 test ETH deposit and full withdrawal, with receipts and balance checks. | [Vault verification](vault-sepolia-2026-09-19/REPORT.md) |
| 2026-09-22 | PaymentEscrow: 5 test USDC funded and refunded; 3.25 test USDC funded and released. Includes contract events and payer, recipient and escrow balances. | [Payment verification](escrow-sepolia-2026-09-22/README.md) |

The contract records document Sourcify verification. They do not claim that Etherscan-specific source verification was completed. The [archive index](archive.json) links the original release/vault reports, receipts, logs, screenshots and checksum manifests at an immutable Git revision, with SHA-256 for each file. Current summaries use the present project name; original evidence retains its recorded bytes and historical UI. A naming update is not a new chain acceptance run.

## Reproduce local behavior

The [recovery checks](../demo-script.md) explain journal-before-broadcast, same-quote retries, interrupted responses and restart recovery. [Payment escrow](../payment-escrow.md#測試入口) links the contract, Go/HTTP and browser tests. [Go verification](../go-style.md#驗證與限制) describes the checks run by CI.

These tests use temporary wallets, local RPC fixtures or a simulated EVM. Successful local tests establish the behavior of those cases; public-chain receipts are separate evidence. Access modes are described in the [architecture](../architecture.md#transaction-behavior-and-evidence).
