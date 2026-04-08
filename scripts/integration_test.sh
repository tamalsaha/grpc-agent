#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

PORT="${GRPC_AGENT_IT_PORT:-50053}"
SERVER_ADDR="127.0.0.1:${PORT}"
CLIENT_NAME="itest-client"
HOST_NAME="itest-host"

TMP_DIR="$(mktemp -d -t grpc-agent-itest-XXXXXX)"
SERVER_LOG="$TMP_DIR/server.log"
CLIENT_LOG="$TMP_DIR/client.log"

SERVER_PID=""
CLIENT_PID=""

cleanup() {
  if [[ -n "$CLIENT_PID" ]] && kill -0 "$CLIENT_PID" 2>/dev/null; then
    kill "$CLIENT_PID" 2>/dev/null || true
    wait "$CLIENT_PID" 2>/dev/null || true
  fi

  if [[ -n "$SERVER_PID" ]] && kill -0 "$SERVER_PID" 2>/dev/null; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi

  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

wait_for_log() {
  local file="$1"
  local pattern="$2"
  local timeout_seconds="$3"

  local start_ts now_ts
  start_ts="$(date +%s)"

  while true; do
    if [[ -f "$file" ]] && grep -q "$pattern" "$file"; then
      return 0
    fi

    now_ts="$(date +%s)"
    if (( now_ts - start_ts >= timeout_seconds )); then
      echo "Timed out waiting for pattern '$pattern' in $file" >&2
      if [[ -f "$file" ]]; then
        echo "---- $file ----" >&2
        cat "$file" >&2
      fi
      return 1
    fi

    sleep 1
  done
}

echo "Building grpc-agent and plugins..."
make all >/dev/null

echo "Starting server on ${SERVER_ADDR}..."
./grpc-agent init --port "$PORT" >"$SERVER_LOG" 2>&1 &
SERVER_PID="$!"
wait_for_log "$SERVER_LOG" "Server started on port" 20

echo "Starting join client ${CLIENT_NAME}..."
./grpc-agent join --server "$SERVER_ADDR" --name "$CLIENT_NAME" >"$CLIENT_LOG" 2>&1 &
CLIENT_PID="$!"
wait_for_log "$CLIENT_LOG" "Server response: Connected to server" 20

echo "Validating local mode via plugin..."
LOCAL_OUT="$(./grpc-agent exec --server "$SERVER_ADDR" --name "$HOST_NAME" local printf local-ok)"
if [[ "$LOCAL_OUT" != *"local-ok"* ]]; then
  echo "Local exec assertion failed. Output: $LOCAL_OUT" >&2
  exit 1
fi

echo "Validating remote mode via plugin and bidi server..."
REMOTE_OUT="$(./grpc-agent exec --server "$SERVER_ADDR" --name "$HOST_NAME" remote "$CLIENT_NAME" printf remote-ok)"
if [[ "$REMOTE_OUT" != *"remote-ok"* ]]; then
  echo "Remote exec assertion failed. Output: $REMOTE_OUT" >&2
  exit 1
fi

echo "Integration test passed"
