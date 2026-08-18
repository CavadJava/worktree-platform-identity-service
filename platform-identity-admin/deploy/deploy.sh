#!/usr/bin/env bash
# Generic build + deploy script for platform-identity-admin (static site).
# Not tied to any specific server — supply SSH_HOST/SSH_USER at call time.
#
# Usage:
#   SSH_HOST=your.server.ip SSH_USER=deploy VITE_PLATFORM_IDENTITY_API_URL=https://api.example.com/api/v1 \
#     ./deploy/deploy.sh
#
# Requires: an SSH key already authorized on the target host, Caddy
# installed on the target. This script builds the production bundle
# locally, ships dist/ to the target, and installs/reloads a Caddy site
# block that serves it as a static SPA (falling back to index.html for
# client-side routes).
#
# Not executed automatically as part of any plan — run it explicitly
# against a real host when you're ready to deploy.

set -euo pipefail

: "${SSH_HOST:?SSH_HOST is required, e.g. SSH_HOST=1.2.3.4}"
: "${SSH_USER:?SSH_USER is required, e.g. SSH_USER=deploy}"
SSH_KEY="${SSH_KEY:-}"
REMOTE_DIR="${REMOTE_DIR:-/opt/platform-identity-admin}"
DOMAIN="${DOMAIN:-}"
SITE_NAME="platform-identity-admin"

# The API base URL must be baked into the build (Vite env vars are
# compile-time), so it must be set correctly before building — point it at
# the backend's real public URL, not localhost, for anything but local
# testing of this script.
: "${VITE_PLATFORM_IDENTITY_API_URL:?VITE_PLATFORM_IDENTITY_API_URL is required, e.g. https://api.example.com/api/v1}"

SSH_OPTS=(-o StrictHostKeyChecking=accept-new)
if [[ -n "$SSH_KEY" ]]; then
  SSH_OPTS+=(-i "$SSH_KEY")
fi

cd "$(dirname "$0")/.."

echo "==> Installing dependencies and building production bundle"
npm ci
VITE_PLATFORM_IDENTITY_API_URL="$VITE_PLATFORM_IDENTITY_API_URL" npm run build

echo "==> Ensuring remote directory exists"
ssh "${SSH_OPTS[@]}" "${SSH_USER}@${SSH_HOST}" "sudo mkdir -p '$REMOTE_DIR' && sudo chown ${SSH_USER} '$REMOTE_DIR'"

echo "==> Syncing dist/ to remote"
rsync -az --delete -e "ssh ${SSH_OPTS[*]}" dist/ "${SSH_USER}@${SSH_HOST}:${REMOTE_DIR}/dist/"

if [[ -n "$DOMAIN" ]]; then
  echo "==> Installing Caddy site block for $DOMAIN"
  CADDY_BLOCK=$(cat <<EOF
${DOMAIN} {
	root * ${REMOTE_DIR}/dist
	encode gzip
	try_files {path} /index.html
	file_server
}
EOF
)
  ssh "${SSH_OPTS[@]}" "${SSH_USER}@${SSH_HOST}" "
    echo '$CADDY_BLOCK' | sudo tee /etc/caddy/sites/${SITE_NAME}.caddy > /dev/null &&
    sudo caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile || true &&
    sudo systemctl reload caddy
  "
else
  echo "==> DOMAIN not set — skipping Caddy site block install."
  echo "    dist/ was still synced to ${REMOTE_DIR}/dist on the remote."
  echo "    Set DOMAIN=admin.example.com to also install/reload a Caddy site block."
fi

echo "==> Deploy complete"
