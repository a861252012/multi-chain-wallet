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
container=multi-chain-wallet-demo-app-1
running=$(docker inspect --format '{{.State.Running}}' "$container")
[[ "$running" == true ]] || { echo 'Application is not running; investigate before backup' >&2; exit 1; }
wallet_source=$(docker inspect --format '{{json .Mounts}}' "$container" | python3 -c '
import json
from pathlib import Path
import sys
try:
    all_mounts = json.load(sys.stdin)
    if any(m.get("Destination", "").startswith("/data/wallet/") for m in all_mounts):
        raise ValueError()
    mounts = [m for m in all_mounts if m.get("Destination") == "/data/wallet"]
    if len(mounts) != 1 or mounts[0].get("Type") != "bind" or mounts[0].get("RW") is not True:
        raise ValueError()
    source = mounts[0].get("Source")
    if source not in ("/opt/testnet-wallet-lab/wallet", "/opt/multi-chain-wallet/wallet"):
        raise ValueError()
    path = Path(source)
    if not path.is_dir() or path.resolve(strict=True) != path:
        raise ValueError()
except (ValueError, TypeError, AttributeError, OSError):
    raise SystemExit("Expected one writable wallet bind at a supported physical directory")
print(source)
')
for file in .env current-image compose.demo.yaml verify-demo.py; do
  [[ -f "$file" && ! -L "$file" ]] || { echo 'Backup configuration is incomplete or symlinked' >&2; exit 1; }
done
# Always bring the application back even if archive creation or transfer fails.
restart_application() {
  result=$?
  trap - EXIT
  if ! docker start "$container" >/dev/null || [[ $(docker inspect --format '{{.State.Running}}' "$container") != true ]]; then
    echo 'Application restart failed after backup; operator action required' >&2
    exit 1
  fi
  exit "$result"
}
trap restart_application EXIT
docker stop -t 45 "$container" >/dev/null
[[ $(docker inspect --format '{{.State.Running}}' "$container") == false ]] || { echo 'Application did not stop for backup' >&2; exit 1; }
tar --exclude='wallet/.lock' -czf - .env current-image compose.demo.yaml verify-demo.py -C "${wallet_source%/wallet}" wallet
REMOTE
age -d -i "$keys/backup-age.key" "$output.partial" | tar -tzf - >/dev/null
mv "$output.partial" "$output"
trap - EXIT
printf 'Encrypted backup verified: %s\n' "$output"
printf 'Keep the recovery key separately: %s\n' "$keys/backup-age.key"
