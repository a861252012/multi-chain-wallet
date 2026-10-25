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

For an existing VM whose wallet is physically stored at `/opt/testnet-wallet-lab/wallet`, copy its private environment settings into the new installation and set `WALLET_DATA_DIR=/opt/testnet-wallet-lab/wallet` there. `compose.demo.yaml` uses this optional bind source; new installations default to `./wallet`. Confirm the directory already exists, retains its ownership and permissions, and is the exact `/data/wallet` mount of both versions. Do not create an empty replacement or copy the wallet while either application is running. Keeping one physical directory also preserves new journal entries if application code must be rolled back: stop the new application before restarting the old one against that same directory. Never restore a pre-upgrade data snapshot over transactions or accounts written during acceptance.

Read back the deployed `X-App-Version`, wallet state and browser behavior before resuming the timer and notifications, including `DEMO_NOTIFY_ENABLED`. A passing source/build test does not establish that this VM transition occurred.

## Routine image updates

`deploy.sh` checks current `main` before pulling and again immediately before stopping the app. If `main` moved during the pull, it exits without replacing the running container. This check is a point-in-time guard; it does not lock GitHub against a later push.

After stopping the old app and before replacing its container, deployment saves the last 1,000 application log lines, with timestamps, to a new private `container-<container-id>.*.log` file in `/opt/multi-chain-wallet`. Reading container logs has a 60-second limit; Compose container lookup and inspection are outside that timeout. A capture failure aborts the update and restores the previous image; partial output is retained. Rollback also tries to save the failed container's logs before replacing it, but a logging failure must not block recovery. These files are diagnostic snapshots, not full log archives; retain incident evidence and remove unneeded snapshots during operator maintenance.

## Encrypted VM backup and restore

After the new installation is running, `scripts/deploy/backup-to-mac.sh VM_IPV4` runs from the owner's Mac over the reviewed SSH route. It briefly stops the exact `multi-chain-wallet-demo-app-1` container under the deployment lock, archives its actual wallet bind with the new configuration, and restarts it even if archiving fails. The script accepts only one writable `/data/wallet` bind at `/opt/testnet-wallet-lab/wallet` or `/opt/multi-chain-wallet/wallet`, with no symlink path or nested mounts. It does not back up an old container before migration; use the separately reviewed old-installation backup procedure for that stage.

The archive stores that directory as `wallet/` alongside `.env`, `current-image`, `compose.demo.yaml` and `verify-demo.py`. It is encrypted on the Mac and decrypted only for archive verification before being reported successful. Keep the recovery key separate and preserve the old installation/scripts/image for migration rollback. A backup failure or restart failure is not success and requires investigation before retrying.

SSH and sudo are non-interactive; SSH sends keepalives every 15 seconds and tolerates four unanswered probes. Before stopping the application, the remote script requires its archive output to be a pipe or socket, not a terminal or regular file. Archiving has a 300-second limit followed by a 10-second termination grace period. On failure, hangup or broken pipe, cleanup detaches from SSH output and ignores further HUP/INT/TERM/PIPE signals while attempting restart. It records only the archive exit code and restart outcome in the root-private `/opt/multi-chain-wallet/backup-recovery.log`; inability to write that log must not prevent restart. Check the service and this log after an interrupted backup. A nonzero archive result remains a failure even when restart succeeds; SIGKILL or VM failure still requires operator recovery.

When restoring to another host or directory, inspect the archived `.env` privately: an absolute `WALLET_DATA_DIR` still names the original host path. Restore the complete wallet directory with its original ownership and permissions, then set the destination installation's `WALLET_DATA_DIR` to that verified physical directory, or remove the override only when `./wallet` is the intended restored location. Confirm the resolved Compose bind before starting. Do not blindly reuse the original override or start against an empty directory.

## Historical evidence

[The archive index](evidence/archive.json) records original SHA-256 values and immutable Git links for dated raw evidence. Historical logs and screenshots retain their original bytes at that revision. Current summaries and navigation use the canonical name. Do not rewrite past logs, receipts or screenshots to make them look like acceptance results for the current version, and do not rewrite Git history as part of a naming change.
