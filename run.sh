#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

REDIS_ADDR="localhost:6379"
REDIS_HOST="${REDIS_ADDR%:*}"
REDIS_PORT="${REDIS_ADDR##*:}"

SERVER_BIN="./server"
PORTS=(8001 8002 8003)
PIDS=()

cleanup() {
  for pid in "${PIDS[@]:-}"; do
    if kill -0 "$pid" >/dev/null 2>&1; then
      kill "$pid" >/dev/null 2>&1 || true
    fi
  done
}

trap cleanup EXIT INT TERM

if ! command -v go >/dev/null 2>&1; then
  echo "go is not installed or not in PATH."
  exit 1
fi

if ! command -v redis-cli >/dev/null 2>&1; then
  echo "redis-cli is not installed or not in PATH."
  echo "Install Redis and make sure redis-cli is available."
  exit 1
fi

if ! redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping >/dev/null 2>&1; then
  echo "Redis is not reachable at $REDIS_ADDR."
  echo "Start Redis locally (expected address: $REDIS_ADDR) and run again."
  exit 1
fi

go build -o "$SERVER_BIN" ./cmd/server

for port in "${PORTS[@]}"; do
  "$SERVER_BIN" --port "$port" >"server-$port.log" 2>&1 &
  PIDS+=("$!")
done

cat <<'EOF'
Distributed Rate Limiter is running.
Rate limit: 3 requests per minute per IP (shared across all servers)

Servers:
  http://localhost:8001/api
  http://localhost:8002/api
  http://localhost:8003/api

Manual test (copy and paste these):
  curl http://localhost:8001/api   -> OK - served by :8001
  curl http://localhost:8002/api   -> OK - served by :8002
  curl http://localhost:8003/api   -> OK - served by :8003
  (4th request to any server)      -> 429 Too Many Requests
EOF

wait