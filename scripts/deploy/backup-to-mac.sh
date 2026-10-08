#!/bin/bash
# Run on the owner's Mac while temporary SSH access is enabled.
set -euo pipefail
[[ $# == 1 && $1 =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'Usage: backup-to-mac.sh VM_IPV4' >&2; exit 2; }
command -v age >/dev/null
command -v age-keygen >/dev/null
umask 077
keys="$HOME/.ssh/wallet-demo-secrets"
backups="$HOME/Documents/wallet-demo-backups"
mkdir -p "$keys" "$backups"
chmod 700 "$keys" "$backups"
if [[ ! -f "$keys/backup-age.key" ]]; then
  age-keygen -o "$keys/backup-age.key" 2>/dev/null
fi
recipient=$(age-keygen -y "$keys/backup-age.key")
output="$backups/wallet-demo-$(date -u +%Y%m%dT%H%M%SZ).tar.gz.age"
[[ ! -e "$output" && ! -e "$output.partial" ]]
trap 'rm -f "$output.partial"' EXIT
ssh -i "$HOME/.ssh/wallet-demo-admin" -o IdentitiesOnly=yes \
  -o StrictHostKeyChecking=yes -o UserKnownHostsFile="$HOME/.ssh/wallet-demo-known-hosts" \
  -o ConnectTimeout=10 "ubuntu@$1" 'sudo bash -s' <<'REMOTE' | age -r "$recipient" -o "$output.partial"
set -euo pipefail
cd /opt/multi-chain-wallet
exec 9>deploy.lock
flock -w 180 9
running=$(docker inspect --format '{{.State.Running}}' multi-chain-wallet-demo-app-1)
[[ "$running" == true ]] || { echo 'Application is not running; investigate before backup' >&2; exit 1; }
# Always bring the application back even if archive creation or transfer fails.
trap 'docker start multi-chain-wallet-demo-app-1 >/dev/null' EXIT
docker stop -t 45 multi-chain-wallet-demo-app-1 >/dev/null
tar --exclude='wallet/.lock' -czf - wallet .env current-image compose.demo.yaml verify-demo.py
REMOTE
age -d -i "$keys/backup-age.key" "$output.partial" | tar -tzf - >/dev/null
mv "$output.partial" "$output"
trap - EXIT
printf 'Encrypted backup verified: %s\n' "$output"
printf 'Keep the recovery key separately: %s\n' "$keys/backup-age.key"
