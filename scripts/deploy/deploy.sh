#!/bin/bash
# Root-owned, fixed deployment configuration. Arguments never become shell code.
set -euo pipefail
export PATH=/usr/sbin:/usr/bin:/sbin:/bin
unset DOCKER_HOST DOCKER_CONTEXT DOCKER_CONFIG COMPOSE_FILE COMPOSE_PROJECT_NAME COMPOSE_PROFILES
readonly base=/opt/multi-chain-wallet
readonly repository=ghcr.io/a861252012/multi-chain-wallet
[[ $# == 2 && $1 =~ ^sha256:[a-f0-9]{64}$ && $2 =~ ^[a-f0-9]{40}$ ]] || { echo 'Invalid deployment arguments' >&2; exit 2; }
readonly image="$repository@$1"
readonly revision="$2"
cd "$base"
exec 9>deploy.lock
flock -w 180 9
# Query inside the deployment lock; a slower old CI run must not overwrite main.
head=$(curl --fail --silent --show-error --max-time 20 -H 'Cache-Control: no-cache' https://api.github.com/repos/a861252012/multi-chain-wallet/commits/main | python3 -c 'import json,sys; print(json.load(sys.stdin)["sha"])')
[[ "$head" == "$revision" ]] || { echo 'Refusing stale commit; main has moved' >&2; exit 1; }
# A tag and an OCI revision label are attacker-controlled registry metadata.
# Verify the exact digest, workflow identity and source commit before any pull/stop.
TUF_ROOT="$base/.sigstore" /usr/local/bin/cosign verify \
  --certificate-identity 'https://github.com/a861252012/multi-chain-wallet/.github/workflows/verify.yml@refs/heads/main' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  --certificate-github-workflow-repository 'a861252012/multi-chain-wallet' \
  --certificate-github-workflow-ref 'refs/heads/main' \
  --certificate-github-workflow-trigger 'push' \
  --certificate-github-workflow-sha "$revision" \
  "$image" >/dev/null
docker pull "$image"
actual=$(docker image inspect --format '{{index .Config.Labels "org.opencontainers.image.revision"}}' "$image")
[[ "$actual" == "$revision" ]] || { echo 'Image revision mismatch' >&2; exit 1; }
previous=''
if [[ -f current-image ]]; then
  previous=$(cat current-image)
  [[ "$previous" =~ ^ghcr.io/a861252012/multi-chain-wallet@sha256:[a-f0-9]{64}$ ]] || exit 2
fi
compose() { APP_IMAGE="$1" docker compose --env-file "$base/.env" -f "$base/compose.demo.yaml" "${@:2}"; }
verify() {
  local container
  container=$(compose "$1" ps -q app)
  [[ -n "$container" ]] || return 1
  python3 "$base/verify-demo.py" "$container" "$2"
}
# Preflight must succeed before stopping the running application.
compose "$image" config --quiet
rollback() {
  local result=$?
  trap - EXIT
  if [[ $result != 0 ]]; then
    echo 'Deployment failed; retaining wallet/journal data' >&2
    if [[ -n "$previous" ]]; then
      local old_revision
      old_revision=$(docker image inspect --format '{{index .Config.Labels "org.opencontainers.image.revision"}}' "$previous")
      if compose "$previous" up -d --no-build --pull never --wait --wait-timeout 90 && verify "$previous" "$old_revision"; then
        echo 'Previous image restored; deployment still reported failed' >&2
      else
        echo 'ROLLBACK FAILED: operator action required' >&2
      fi
    else
      compose "$image" stop app || true
      echo 'First deployment failed; no previous image available' >&2
    fi
  fi
  exit "$result"
}
trap rollback EXIT
# flock in the wallet requires stop-before-start. Never run compose down -v.
compose "$image" stop app
compose "$image" up -d --no-build --pull never --wait --wait-timeout 90
verify "$image" "$revision"
printf '%s\n' "$image" > current-image.new
mv current-image.new current-image
trap - EXIT
printf 'Deployed %s\n' "$revision"
