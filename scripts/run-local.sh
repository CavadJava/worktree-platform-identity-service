#!/usr/bin/env bash
# Starts platform-identity-service (backend, :8095) and platform-identity-admin
# (frontend dev server, :5173) in the background, logging to /tmp and
# tracking PIDs so stop-local.sh can cleanly tear them down.
#
# Usage:
#   ./scripts/run-local.sh
#   ./scripts/stop-local.sh   # when done
#
# Requires: local Postgres reachable at localhost:5433 (see
# platform-identity-service/.env for credentials), Go, Node/npm.

set -euo pipefail

cd "$(dirname "$0")/.."
ROOT_DIR="$(pwd)"
PID_DIR="/tmp/platform-identity-local"
mkdir -p "$PID_DIR"

BACKEND_PORT="${BACKEND_PORT:-8095}"
FRONTEND_PORT="${FRONTEND_PORT:-5173}"

echo "==> Checking Postgres on localhost:5433"
if ! pg_isready -h localhost -p 5433 >/dev/null 2>&1; then
  echo "Postgres is not reachable at localhost:5433 — start it before running this script." >&2
  exit 1
fi

echo "==> Freeing ports ${BACKEND_PORT} and ${FRONTEND_PORT} if already in use"
lsof -ti ":${BACKEND_PORT}" 2>/dev/null | xargs -r kill || true
lsof -ti ":${FRONTEND_PORT}" 2>/dev/null | xargs -r kill || true
sleep 1

echo "==> Starting platform-identity-service on :${BACKEND_PORT}"
(
  cd "$ROOT_DIR/platform-identity-service"
  PORT="$BACKEND_PORT" go run ./cmd/api > "$PID_DIR/backend.log" 2>&1 &
  echo $! > "$PID_DIR/backend.pid"
)
sleep 3
if ! kill -0 "$(cat "$PID_DIR/backend.pid")" 2>/dev/null; then
  echo "Backend failed to start — see $PID_DIR/backend.log" >&2
  cat "$PID_DIR/backend.log" >&2
  exit 1
fi

echo "==> Starting platform-identity-admin dev server on :${FRONTEND_PORT}"
(
  cd "$ROOT_DIR/platform-identity-admin"
  npm run dev > "$PID_DIR/frontend.log" 2>&1 &
  echo $! > "$PID_DIR/frontend.pid"
)
sleep 3
if ! kill -0 "$(cat "$PID_DIR/frontend.pid")" 2>/dev/null; then
  echo "Frontend failed to start — see $PID_DIR/frontend.log" >&2
  cat "$PID_DIR/frontend.log" >&2
  exit 1
fi

echo ""
echo "==> Both services are running:"
echo "    Backend:  http://localhost:${BACKEND_PORT}  (logs: $PID_DIR/backend.log)"
echo "    Frontend: http://localhost:${FRONTEND_PORT}  (logs: $PID_DIR/frontend.log)"
echo ""
echo "Stop with: ./scripts/stop-local.sh"
