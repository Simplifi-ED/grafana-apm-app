#!/usr/bin/env bash
# SSH LocalForward ClickHouse HTTP (8123) and native (9000) to localhost.
#
# Required env (no personal/infra defaults):
#   CLICKSTACK_SSH_HOST  — SSH host or Tailscale IP of the ClickHouse host
#   CLICKSTACK_SSH_USER  — SSH username
#   CLICKSTACK_SSH_KEY   — path to private key
#
# Optional:
#   SSH_BIN                 — ssh binary (default: ssh on PATH)
#   CLICKSTACK_SSH_CONTROL  — ControlMaster socket path
set -euo pipefail

SSH_BIN="${SSH_BIN:-ssh}"
HOST="${CLICKSTACK_SSH_HOST:?set CLICKSTACK_SSH_HOST}"
USER_NAME="${CLICKSTACK_SSH_USER:?set CLICKSTACK_SSH_USER}"
KEY="${CLICKSTACK_SSH_KEY:?set CLICKSTACK_SSH_KEY}"
CONTROL_PATH="${CLICKSTACK_SSH_CONTROL:-$HOME/.ssh/cm-clickstack}"

if ! command -v "$SSH_BIN" >/dev/null 2>&1 && [[ ! -x "$SSH_BIN" ]]; then
  echo "ssh not found: set SSH_BIN or install OpenSSH" >&2
  exit 1
fi
if [[ ! -f "$KEY" ]]; then
  echo "missing SSH key: $KEY" >&2
  exit 1
fi

exec "$SSH_BIN" \
  -i "$KEY" \
  -o ControlMaster=auto \
  -o "ControlPath=$CONTROL_PATH" \
  -o ControlPersist=30m \
  -o ExitOnForwardFailure=yes \
  -N \
  -L 8123:127.0.0.1:8123 \
  -L 9000:127.0.0.1:9000 \
  "${USER_NAME}@${HOST}"
