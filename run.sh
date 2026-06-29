#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

REDIS_ADDR="localhost:6379"
REDIS_HOST="${REDIS_ADDR%:*}"
REDIS_PORT="${REDIS_ADDR##*:}"

BUILD_DIR="$(mktemp -d "${TMPDIR:-/tmp}/distributed-rate-limiter.XXXXXX")"
SERVER_BIN="$BUILD_DIR/server"
PORTS=(8001 8002 8003)
PIDS=()

try_command() {
  "$@" >/dev/null 2>&1
}

wait_for_redis() {
  local retries="$1"
  local delay_seconds="$2"

  for ((i = 1; i <= retries; i++)); do
    if try_command redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping; then
      return 0
    fi
    sleep "$delay_seconds"
  done

  return 1
}

command_exists() {
  try_command command -v "$1"
}

port_is_in_use() {
  local port="$1"
  try_command lsof -nP -iTCP:"$port" -sTCP:LISTEN
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

  if [[ -d "$BUILD_DIR" ]]; then
    try_command rm -rf "$BUILD_DIR" || true
  fi
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

if ! wait_for_redis 5 1; then
  fail_with_help \
    "Redis is not reachable at $REDIS_ADDR." \
    "If Redis was just started, wait a few seconds and run this script again." \
    "Try starting Redis with:" \
    "  brew services start redis" \
    "Then run this script again (expected address: $REDIS_ADDR)."
fi

for port in "${PORTS[@]}"; do
  if port_is_in_use "$port"; then
    fail_with_help \
      "Port $port is already in use." \
      "Stop the existing process using that port and run this script again." \
      "To inspect it, run:" \
      "  lsof -nP -iTCP:$port -sTCP:LISTEN"
  fi
done

# Build the server binary once; it will be launched on multiple ports below.
go build -o "$SERVER_BIN" ./cmd/server

# Start one server per port in the background and track PIDs for cleanup.
for port in "${PORTS[@]}"; do
  "$SERVER_BIN" --port "$port" &
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
  Keep this terminal running. Open a second terminal for curl tests.
  curl http://localhost:8001/api   -> OK - served by :8001
  curl http://localhost:8002/api   -> OK - served by :8002
  curl http://localhost:8003/api   -> OK - served by :8003
  (4th request to any server)      -> 429 Too Many Requests

Manual helper script:
  bash client.sh
EOF

wait