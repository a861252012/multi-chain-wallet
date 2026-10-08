# Project naming and upgrades

The canonical repository is `a861252012/multi-chain-wallet`. Use **Multi-Chain Wallet** for the product title, `multi-chain-wallet` for commands and package names, and `MULTI_CHAIN_WALLET_` for project-specific test environment variables. Go belongs in the project description: a multi-chain wallet built with Go for EVM, Solana and TRON. The supported environments remain test networks.

When changing names, check the Go module/imports and command directory, all README editions, UI titles and translations, download names, Docker/Compose, npm manifests, deployment scripts, CI signing identities, and screenshots. Run the existing Go, browser, contract and deployment checks. A renamed screenshot must come from the current UI, not an edited historical capture.

## Existing local data

Before starting an existing local installation, set `WALLET_DATA_VOLUME` in its private `.env` to the existing Docker wallet volume. Find candidates with `docker volume ls --filter label=com.docker.compose.volume=wallet_data`, then inspect their Compose labels. Do not create an empty replacement volume or run `docker compose down -v` during an upgrade. Preserve keys, journals and archives together.

The browser migrates version 1 preference, contact, token, draft and pending-operation storage keys on page load. Existing values in the current namespace win. A failed write leaves the original value available for a later attempt. Authentication sessions expire across the rename; CLI Basic authentication now uses `multi-chain-wallet` as the username. The token itself does not change.

Solana backup version 1 keeps its immutable authenticated-data bytes; product naming does not change the encryption format. See [key format compatibility](key-format.md).

## Signed deployment transition

CI publishes only when `DEMO_DEPLOY_ENABLED=true` and `DEMO_DEPLOY_REPOSITORY` equals `a861252012/multi-chain-wallet`. Leave the repository pin unset until the VM transition is prepared and automatic updates are paused. Unit tests and build checks continue to run while publication and live deployment are skipped.

A repository rename does not update the installed VM scripts, existing container, data directory, registry package visibility or Cosign trust identity. The operator must first preserve a verified encrypted backup and the previous Compose/configuration/image for rollback. Pause the existing update timer and notification dispatcher during the transition, and disable CI release notifications with `DEMO_NOTIFY_ENABLED=false`.

Prepare the new installation and trust settings before setting `DEMO_DEPLOY_REPOSITORY` to the canonical repository. Then trigger a new push to `main` to build, test, publish and sign the first image under the new registry name. Keep VM automation paused. Confirm the new immutable image is anonymously pullable and signed by the exact new workflow identity and commit before using it. The live CI job waits for deployment; if that wait expires during migration, rerun its verification after the VM is ready. Publishing an image alone does not complete the transition.

Install the new scripts and service definitions with `/opt/multi-chain-wallet` as their base. Stop the previous Compose application before starting `multi-chain-wallet-demo`, preserve the full wallet data with its ownership and permissions, and never run both applications against the same wallet. The old `current-image` value is not accepted by the new repository pin: keep it with the previous installation for rollback rather than relabelling its digest as a newly published image. Bootstrap the new installation using a newly verified image. If verification fails, stop the new application and restore the preserved installation.

Read back the deployed `X-App-Version`, wallet state and browser behavior before resuming the timer and notifications, including `DEMO_NOTIFY_ENABLED`. A passing source/build test does not establish that this VM transition occurred.

## Historical evidence

[The archive index](evidence/archive.json) records original SHA-256 values and immutable Git links for dated raw evidence. Historical logs and screenshots retain their original bytes at that revision. Current summaries and navigation use the canonical name. Do not rewrite past logs, receipts or screenshots to make them look like acceptance results for the current version, and do not rewrite Git history as part of a naming change.
