# Multi-Chain Wallet

[English](README.md) · [简体中文](README.zh-CN.md) · [正體中文（台灣）](README.zh-TW.md) · [繁體中文（香港）](README.zh-HK.md)

**[Live demo (testnets only)](https://wallet.tedlin.fyi/)**

A multi-chain wallet built with Go for EVM, Solana and TRON. The demo runs on test networks. The main problem explored here is uncertain broadcasts: save signed bytes before sending, deduplicate retries by quote ID, and resume receipt checks after a restart.

![Wallet interface with local test data](docs/images/wallet-overview.png)

*Interface preview using local test data.*

## Run and verify

[Local setup](docs/wallet-reference.md#run-with-docker) · [Architecture](docs/architecture.md) · [Recovery checks](docs/demo-script.md) · [Go style and checks](docs/go-style.md) · [Verification evidence](docs/evidence/README.md)

## Features

- Create, restore and manage local wallets with encrypted backups.
- Send native coins and tokens, preview fees, and set or revoke ERC-20 allowances.
- Wrap and unwrap WETH, and swap WETH / test USDC through Uniswap V3 on Ethereum Sepolia.
- Build EVM activity records from transaction receipts and export them as CSV.

## Transaction handling

Signed transactions are saved before broadcast. Retries with the same account, network and quote ID reuse the saved transaction. When a broadcast result is uncertain, recovery uses the original signed bytes, subject to each network's expiry rules.

EVM receipt success, canonical block inclusion and finality are checked separately. After a restart, the wallet reloads its transaction journal to continue tracking saved transactions.

## Networks

| Network | Operations |
|---|---|
| Ethereum Sepolia | Native coin / ERC-20 transfers and approvals, WETH wrapping, Uniswap V3 swaps |
| Arbitrum, Base, OP Sepolia · Polygon Amoy | Native coin / ERC-20 transfers and approvals |
| Solana Devnet | SOL transfers, separate wallet |
| TRON Shasta | TRX / TRC-20 transfers, separate wallet |

## Solidity ETH vault

A Solidity deposit/withdraw contract with a Sepolia-only panel, per-address balances and reentrancy protection. The [2026-09-19 acceptance record](docs/evidence/vault-sepolia-2026-09-19/REPORT.md) documents public Sepolia deployment, Sourcify source verification and deposit/withdraw receipts. Etherscan-specific verification was incomplete at that time. Unconfigured environments remain disabled.

[Payment escrow](docs/payment-escrow.md) adds test USDC funding, payer-authorized release and recipient-authorized full refunds to the original payer. It reuses the durable transaction journal and retry flow, with contract, Go/HTTP and Chromium + simulated EVM tests. Public Sepolia payment, release and refund flows have been completed; see the [receipts and balance checks](docs/evidence/escrow-sepolia-2026-09-22/README.md).

[Contract and tests](docs/eth-vault.md).

## Smart contract wallet experiment

An ERC-4337 v0.6 account was deployed and used for a transfer on Ethereum Sepolia through the CLI. The [receipts and limits](docs/erc4337-acceptance.md) also distinguish this from v0.7 encoding support, which has no public-chain acceptance.

## Public demo

Live demo: https://wallet.tedlin.fyi/. Visitors share a test wallet; signing new transactions and exporting encrypted keys require its password. Visitors can create password-protected EVM test wallets (20 accounts total).
