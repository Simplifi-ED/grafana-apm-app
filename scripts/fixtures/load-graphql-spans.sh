#!/usr/bin/env bash
# Load ticket-16 GraphQL span fixture into a ClickHouse (CI / local / tunneled).
#
# Usage:
#   scripts/fixtures/load-graphql-spans.sh
#   CLICKHOUSE_URL=http://127.0.0.1:8123 CLICKHOUSE_USER=default \
#     CLICKHOUSE_PASSWORD=ci scripts/fixtures/load-graphql-spans.sh
#
# Env:
#   CLICKHOUSE_URL       default http://127.0.0.1:8123
#   CLICKHOUSE_USER      default default
#   CLICKHOUSE_PASSWORD  default empty
#   CLICKHOUSE_DATABASE  default default
#   SKIP_CREATE=1        skip otel_traces_minimal.sql (table already exists)
#   VERIFY_ONLY=1        only run the presence probe
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
URL="${CLICKHOUSE_URL:-http://127.0.0.1:8123}"
USER="${CLICKHOUSE_USER:-default}"
PASS="${CLICKHOUSE_PASSWORD:-}"
DB="${CLICKHOUSE_DATABASE:-default}"

auth=()
if [[ -n "$PASS" ]]; then
  auth=(-u "${USER}:${PASS}")
elif [[ -n "$USER" ]]; then
  auth=(-u "${USER}:")
fi

ch() {
  local q="$1"
  curl -sfS "${auth[@]}" \
    --get "${URL}/" \
    --data-urlencode "database=${DB}" \
    --data-urlencode "query=${q}"
}

ch_file() {
  local file="$1"
  curl -sfS "${auth[@]}" \
    "${URL}/?database=${DB}" \
    --data-binary @"${file}"
}

echo "==> ClickHouse ${URL} db=${DB}"

if [[ "${VERIFY_ONLY:-0}" != "1" ]]; then
  if [[ "${SKIP_CREATE:-0}" != "1" ]]; then
    echo "==> CREATE TABLE IF NOT EXISTS otel_traces (minimal)"
    ch_file "${ROOT}/otel_traces_minimal.sql"
  fi
  echo "==> INSERT graphql-otel-spans.sql"
  ch_file "${ROOT}/graphql-otel-spans.sql"
fi

echo "==> verify mapContains(SpanAttributes,'graphql.operation.name')"
VERIFY_SQL="SELECT count() AS n,
  SpanAttributes['graphql.operation.name'] AS op,
  SpanAttributes['graphql.operation.type'] AS typ
FROM otel_traces
WHERE mapContains(SpanAttributes, 'graphql.operation.name')
   OR mapContains(SpanAttributes, 'graphql.operation.type')
GROUP BY op, typ
ORDER BY n DESC
FORMAT PrettyCompact"
ch "${VERIFY_SQL}"
echo
N="$(ch "SELECT count() FROM otel_traces WHERE mapContains(SpanAttributes, 'graphql.operation.name') FORMAT TabSeparated")"
if [[ "${N}" -lt 1 ]]; then
  echo "FAIL: expected n>=1 GraphQL spans, got ${N}" >&2
  exit 1
fi
echo "OK: n=${N} GraphQL spans (ServiceName=graphql-demo)"
