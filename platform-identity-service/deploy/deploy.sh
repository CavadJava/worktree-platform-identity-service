#!/usr/bin/env bash
# Generic build + deploy script for platform-identity-service.
# Not tied to any specific server — supply SSH_HOST/SSH_USER at call time.
#
# Usage:
#   SSH_HOST=your.server.ip SSH_USER=deploy ./deploy/deploy.sh
#
# Requires: an SSH key already authorized on the target host, systemd on
# the target, and a real .env (this script does not create one — copy your
# production .env to the target manually or via a separate secrets step
# before the first run).

set -euo pipefail

: "${SSH_HOST:?SSH_HOST is required, e.g. SSH_HOST=1.2.3.4}"
: "${SSH_USER:?SSH_USER is required, e.g. SSH_USER=deploy}"
SSH_KEY="${SSH_KEY:-}"
REMOTE_DIR="${REMOTE_DIR:-/opt/platform-identity-service}"
SERVICE_NAME="platform-identity-service"

SSH_OPTS=(-o StrictHostKeyChecking=accept-new)
if [[ -n "$SSH_KEY" ]]; then
  SSH_OPTS+=(-i "$SSH_KEY")
fi

cd "$(dirname "$0")/.."

echo "==> Building linux/amd64 binary"
GOOS=linux GOARCH=amd64 go build -o "$SERVICE_NAME" ./cmd/api

echo "==> Ensuring remote directory exists"
ssh "${SSH_OPTS[@]}" "${SSH_USER}@${SSH_HOST}" "sudo mkdir -p '$REMOTE_DIR' && sudo chown ${SSH_USER} '$REMOTE_DIR'"

echo "==> Copying binary and systemd unit"
scp "${SSH_OPTS[@]}" "$SERVICE_NAME" "${SSH_USER}@${SSH_HOST}:${REMOTE_DIR}/${SERVICE_NAME}"
scp "${SSH_OPTS[@]}" deploy/platform-identity-service.service "${SSH_USER}@${SSH_HOST}:/tmp/${SERVICE_NAME}.service"

echo "==> Installing systemd unit and restarting service"
ssh "${SSH_OPTS[@]}" "${SSH_USER}@${SSH_HOST}" "
  sudo mv /tmp/${SERVICE_NAME}.service /etc/systemd/system/${SERVICE_NAME}.service &&
  sudo systemctl daemon-reload &&
  sudo systemctl enable ${SERVICE_NAME} &&
  sudo systemctl restart ${SERVICE_NAME} &&
  sudo systemctl status ${SERVICE_NAME} --no-pager
"

rm -f "$SERVICE_NAME"
echo "==> Deploy complete"
