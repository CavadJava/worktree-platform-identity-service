#!/usr/bin/env bash
# Stops whatever run-local.sh started, using the PIDs it recorded.

set -euo pipefail

PID_DIR="/tmp/platform-identity-local"

stop_one() {
  local name="$1"
  local pid_file="$PID_DIR/$name.pid"
  if [[ -f "$pid_file" ]]; then
    local pid
    pid="$(cat "$pid_file")"
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid"
      echo "==> Stopped $name (pid $pid)"
    else
      echo "==> $name (pid $pid) was not running"
    fi
    rm -f "$pid_file"
  else
    echo "==> No recorded pid for $name (was run-local.sh ever started?)"
  fi
}

stop_one backend
stop_one frontend

# Also sweep the known ports in case something orphaned (e.g. `go run`'s
# child process outliving the pid we recorded for the parent).
lsof -ti :8095 2>/dev/null | xargs -r kill || true
lsof -ti :5173 2>/dev/null | xargs -r kill || true

echo "==> Done"
