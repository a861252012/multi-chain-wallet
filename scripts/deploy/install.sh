#!/bin/bash
# Run once on the dedicated Ubuntu VM, from a reviewed checkout, as root.
set -euo pipefail
[[ $EUID == 0 && $# == 0 ]] || { echo 'Usage: sudo bash scripts/deploy/install.sh' >&2; exit 2; }
command -v docker >/dev/null
docker compose version >/dev/null
command -v python3 >/dev/null
command -v curl >/dev/null
command -v flock >/dev/null
[[ -x /usr/local/bin/cosign ]] || { echo "Install verified Cosign v3.1.3 at /usr/local/bin/cosign first" >&2; exit 1; }
base=/opt/multi-chain-wallet
[[ ! -e "$base" && ! -e /usr/local/sbin/wallet-deploy ]] || { echo 'Already installed; review upgrades manually' >&2; exit 1; }
root=$(cd "$(dirname "$0")/../.." && pwd)
install -d -o root -g root -m 0750 "$base"
install -d -o 10001 -g 10001 -m 0700 "$base/wallet"
install -o root -g root -m 0644 "$root/compose.demo.yaml" "$base/compose.demo.yaml"
install -o root -g root -m 0644 "$root/scripts/deploy/verify-demo.py" "$base/verify-demo.py"
install -o root -g root -m 0755 "$root/scripts/deploy/deploy.sh" /usr/local/sbin/wallet-deploy
install -o root -g root -m 0755 "$root/scripts/deploy/poll.sh" /usr/local/sbin/wallet-demo-poll
install -o root -g root -m 0644 "$root/scripts/deploy/wallet-demo-update.service" /etc/systemd/system/wallet-demo-update.service
install -o root -g root -m 0644 "$root/scripts/deploy/wallet-demo-update.timer" /etc/systemd/system/wallet-demo-update.timer
systemctl daemon-reload
# Generate the application token on the VM; it never goes through CI or Git.
umask 077
python3 - <<'PY'
import secrets
from pathlib import Path
Path('/opt/multi-chain-wallet/.env').write_text('PUBLIC_ORIGIN=\nWALLET_ACCESS_TOKEN='+secrets.token_hex(32)+'\n')
PY
echo 'Installed. Set PUBLIC_ORIGIN in /opt/multi-chain-wallet/.env before enabling deployment.'
echo 'The application token is in that root-only file; keep it for the owner only.'
echo 'After first release and origin verification: systemctl enable --now wallet-demo-update.timer'
