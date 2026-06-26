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

try_command() {
  "$@" >/dev/null 2>&1
}

command_exists() {
  try_command command -v "$1"
}

print_lines() {
  printf '%s\n' "$@"
}

fail_with_help() {
  print_lines "$@"
  exit 1
}

cleanup() {
  for pid in "${PIDS[@]:-}"; do
    if try_command kill -0 "$pid"; then
      try_command kill "$pid" || true
    fi
  done
}

trap cleanup EXIT INT TERM

if ! command_exists go; then
  fail_with_help \
    "go is not installed or not in PATH." \
    "This script supports macOS. Install Go with:" \
    "  brew install go"
fi

if ! command_exists redis-cli; then
  fail_with_help \
    "redis-cli is not installed or not in PATH." \
    "This script supports macOS. Install Redis with:" \
    "  brew install redis"
fi

if ! try_command redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping; then
  fail_with_help \
    "Redis is not reachable at $REDIS_ADDR." \
    "Try starting Redis with:" \
    "  brew services start redis" \
    "Then run this script again (expected address: $REDIS_ADDR)."
fi

# Build the server binary once; it will be launched on multiple ports below.
go build -o "$SERVER_BIN" ./cmd/server

# Start one server per port in the background, write per-port logs, and track PIDs for cleanup.
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

Manual helper script:
  bash client.sh
EOF

wait