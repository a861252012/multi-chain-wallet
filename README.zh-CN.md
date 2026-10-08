# Multi-Chain Wallet

[English](README.md) · [简体中文](README.zh-CN.md) · [正體中文（台灣）](README.zh-TW.md) · [繁體中文（香港）](README.zh-HK.md)

**[在线 Demo（仅供测试链使用）](https://wallet.tedlin.fyi/)**

这是用 Go 开发的多链钱包，支持 EVM、Solana 和 TRON。Demo 使用测试网。重点是交易结果不明时如何处理：先保存签署内容再广播、用 quote ID 避免重复签名，重启后继续查收据。

![钱包界面，使用本地测试数据](docs/images/wallet-overview.png)

*界面预览使用本地测试数据。*

## 本地启动与验证

[本地启动](docs/wallet-reference.md#run-with-docker) · [架构](docs/architecture.md) · [交易恢复验证](docs/demo-script.md) · [Go 风格与检查](docs/go-style.md) · [验收证据](docs/evidence/README.md)

## 主要功能

- 创建、恢复及管理本地钱包，支持加密备份。
- 发送原生币与代币、预览手续费，设置或撤销 ERC-20 授权额度。
- 在 Ethereum Sepolia 包装／解包 WETH，通过 Uniswap V3 兑换 WETH 与测试 USDC。
- 根据交易收据整理 EVM 收支记录，支持 CSV 导出。

## 交易处理

交易签名后先保存，再发送到 RPC。同一账户、网络与 quote ID 的重试会复用已保存的交易。广播结果不明时，会按各链的有效期规则重发原始签名数据。

EVM 收据是否成功、所在区块是否仍在主链，以及是否 finalized，分别核对。进程重启后，钱包会读取交易日志，继续跟踪已保存的交易。

## 支持网络

| 网络 | 功能 |
|---|---|
| Ethereum Sepolia | 原生币／ERC-20 转账与授权、WETH 包装、Uniswap V3 兑换 |
| Arbitrum、Base、OP Sepolia · Polygon Amoy | 原生币／ERC-20 转账与授权 |
| Solana Devnet | SOL 转账，使用独立钱包 |
| TRON Shasta | TRX／TRC-20 转账，使用独立钱包 |

## Solidity 测试 ETH 存款箱

Solidity 存款／提款合约搭配 Sepolia 面板，包含每个地址独立余额与重入保护。[2026-09-19 验收记录](docs/evidence/vault-sepolia-2026-09-19/REPORT.md)记录了公共 Sepolia 部署、Sourcify 源码验证及存取结果；Etherscan 单独验证在当时尚未完成。未配置合约地址的环境仍禁用操作。

另有[测试 USDC 付款托管](docs/payment-escrow.md)：付款人付款与放款，收款人可全额退回原付款人。沿用交易日志与重试流程，提供合约、Go/HTTP 及 Chromium＋模拟 EVM 测试；已在公共 Sepolia 完成付款、放款与退款，见[交易收据与余额验收](docs/evidence/escrow-sepolia-2026-09-22/README.md)。

[合约与测试](docs/eth-vault.md)。

## 智能合约钱包实验

ERC-4337 v0.6 账户已通过命令行在 Ethereum Sepolia 完成部署与转账；[收据与限制](docs/erc4337-acceptance.md)也记录了 v0.7 仍只有格式支持，尚无公链验收。

## 公开 Demo

公开网址：https://wallet.tedlin.fyi/ 。访客共用测试钱包；签署新交易及导出加密密钥仍需钱包密码。访客可创建受密码保护的 EVM 测试钱包（全站最多 20 个）。
