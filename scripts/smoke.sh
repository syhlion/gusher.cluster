#!/usr/bin/env bash
# smoke.sh — bring up the docker-compose stack (nats + master + slave, no
# Redis), run the assertive smoke client against it, and tear it down. Exits
# non-zero if the stack never becomes ready or the smoke check fails.
#
# Usage: make smoke   (or: scripts/smoke.sh)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

COMPOSE="docker compose -f docker-compose/docker-compose.yml"
MASTER_PING="http://127.0.0.1:7777/ping"
SLAVE_PING="http://127.0.0.1:8888/ping"

cleanup() {
  echo "--- tearing down stack ---"
  $COMPOSE down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

# the stack mounts docker-compose/public.pem (tracked in the repo, matching
# test/key/private.pem) to verify the smoke JWT signed below.
if [ ! -f docker-compose/public.pem ]; then
  echo "ERROR: docker-compose/public.pem missing (expected to be tracked in repo)"
  exit 1
fi

echo "--- building + starting stack ---"
$COMPOSE up -d --build

wait_ready() {
  local url="$1" name="$2" i
  for i in $(seq 1 60); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      echo "$name ready"
      return 0
    fi
    sleep 1
  done
  echo "ERROR: $name not ready after 60s"
  $COMPOSE logs --tail=50 || true
  return 1
}
wait_ready "$MASTER_PING" master
wait_ready "$SLAVE_PING" slave

echo "--- generating JWT ---"
go build -o "$ROOT/build/jwt-generate" test/jwtgenerate/jwtgenerate.go
TOKEN="$("$ROOT/build/jwt-generate" gen --private-key test/key/private.pem 2>/dev/null | grep -oE 'eyJ[A-Za-z0-9_.-]+' | head -1)"
if [ -z "$TOKEN" ]; then
  echo "ERROR: failed to generate JWT"
  exit 1
fi

echo "--- running smoke client ---"
SMOKE_JWT="$TOKEN" \
SMOKE_AUTH_URL="http://127.0.0.1:8888/auth" \
SMOKE_WS_URL="ws://127.0.0.1:8888/ws/TEST" \
SMOKE_PUSH_URL="http://127.0.0.1:7777/push/TEST/AA/EVENT" \
SMOKE_CONNECTIONS="${SMOKE_CONNECTIONS:-50}" \
  go run test/smoke/smoke.go

echo "--- smoke ok ---"
