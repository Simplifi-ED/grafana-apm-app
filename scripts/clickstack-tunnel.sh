#!/usr/bin/env bash
# SSH LocalForward to prod ClickHouse on monitoring-clickstack (127.0.0.1 only).
# Requires: Tailscale up, Homebrew OpenSSH (/opt/homebrew/bin/ssh), YubiKey FIDO2 key
# enrolled as zine (see platforms-flux/.../docs/acces-ssh.md).
set -euo pipefail

SSH_BIN="${SSH_BIN:-/opt/homebrew/bin/ssh}"
HOST="${CLICKSTACK_SSH_HOST:-100.122.108.0}"
USER_NAME="${CLICKSTACK_SSH_USER:-zine}"
KEY="${CLICKSTACK_SSH_KEY:-$HOME/.ssh/zine@yubikey_sk}"
# fallback if user kept the default filename
if [[ ! -f "$KEY" && -f "$HOME/.ssh/id_ed25519_sk" ]]; then
  KEY="$HOME/.ssh/id_ed25519_sk"
fi
CONTROL_PATH="${CLICKSTACK_SSH_CONTROL:-$HOME/.ssh/cm-clickstack}"

if [[ ! -x "$SSH_BIN" ]]; then
  echo "missing FIDO2-capable ssh: brew install openssh (expected $SSH_BIN)" >&2
  exit 1
fi
if [[ ! -f "$KEY" ]]; then
  echo "missing FIDO2 key: $KEY" >&2
  echo "generate: $SSH_BIN-keygen -t ed25519-sk -O resident -O verify-required -C \"zine@yubikey\"" >&2
  echo "then send .pub to youba for enrollment (acces-ssh.md)" >&2
  exit 1
fi

# One UV (PIN+touch) for ControlMaster; reuse for port forwards.
exec env -u SSH_AUTH_SOCK "$SSH_BIN" \
  -i "$KEY" \
  -o ControlMaster=auto \
  -o "ControlPath=$CONTROL_PATH" \
  -o ControlPersist=30m \
  -o ExitOnForwardFailure=yes \
  -N \
  -L 8123:127.0.0.1:8123 \
  -L 9000:127.0.0.1:9000 \
  "${USER_NAME}@${HOST}"
