#!/bin/bash
# Root-owned timer: fetch only this repository's current main release.
set -euo pipefail
export PATH=/usr/sbin:/usr/bin:/sbin:/bin
unset DOCKER_HOST DOCKER_CONTEXT DOCKER_CONFIG COMPOSE_FILE COMPOSE_PROJECT_NAME COMPOSE_PROFILES
readonly base=/opt/multi-chain-wallet
readonly repository=ghcr.io/a861252012/multi-chain-wallet
exec 8>"$base/poll.lock"
flock -w 600 8 || { echo 'Timed out waiting for another release check' >&2; exit 1; }
head=$(curl --fail --silent --show-error --max-time 20 -H 'Cache-Control: no-cache' https://api.github.com/repos/a861252012/multi-chain-wallet/commits/main | python3 -c 'import json,sys; print(json.load(sys.stdin)["sha"])')
[[ "$head" =~ ^[a-f0-9]{40}$ ]] || exit 1
if [[ -f "$base/current-image" ]]; then
  current=$(cat "$base/current-image")
  [[ "$current" =~ ^ghcr.io/a861252012/multi-chain-wallet@sha256:[a-f0-9]{64}$ ]] || exit 1
  revision=$(docker image inspect --format '{{index .Config.Labels "org.opencontainers.image.revision"}}' "$current")
  [[ "$revision" != "$head" ]] || exit 0
fi
# A tag only locates a candidate; wallet-deploy must authenticate its digest.
# Missing/invalid signatures and pull failures never stop the old app.
docker pull "$repository:sha-$head"
ref=$(docker image inspect --format '{{index .RepoDigests 0}}' "$repository:sha-$head")
[[ "$ref" =~ ^ghcr.io/a861252012/multi-chain-wallet@sha256:[a-f0-9]{64}$ ]] || exit 1
/usr/local/sbin/wallet-deploy "${ref#*@}" "$head"
