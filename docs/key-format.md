# Multi-Chain Wallet Solana Key Format Compatibility

This document records the wire format and compatibility invariants for Solana v1 keystore backups.

## Solana v1 Backup Format

Solana seed backups (`key.json`) store an AES-256-GCM ciphertext derived via Scrypt:

- `version`: integer, fixed to `1`.
- `address`: Solana Base58 public key.
- `scryptN`: CPU/memory cost parameter (power of 2, `2 <= N <= 262144`).
- `salt`: hex-encoded 32-byte salt.
- `nonce`: hex-encoded 12-byte AES-GCM nonce.
- `ciphertext`: hex-encoded 48-byte payload (32-byte encrypted Ed25519 seed + 16-byte GCM authentication tag).

## Authenticated Data and Compatibility Invariant

AES-256-GCM authenticated data (AAD) binds the public address to the ciphertext:

- AAD = `solanaV1AuthDomain + address`
- Protocol domain constant (21 bytes): `\x66\x6c\x6f\x77\x6c\x65\x64\x67\x65\x72\x2d\x73\x6f\x6c\x61\x6e\x61\x2d\x76\x31\x3a`
- Wire hex representation: `666c6f776c65646765722d736f6c616e612d76313a`

These exact 21 bytes are frozen at the protocol level to guarantee that existing backups decrypt identically without alteration. This is an immutable wire constant, independent of product branding, and does not constitute a cryptographic format migration.
