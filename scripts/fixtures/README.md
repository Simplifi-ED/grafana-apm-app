# ClickHouse fixtures (ticket 16)

Synthetic telemetry for CI / local smoke when live Sanad apps do not emit the signal.

## GraphQL spans

| File | Role |
|------|------|
| `otel_traces_minimal.sql` | Minimal `otel_traces` DDL matching ClickStack DESCRIBE |
| `graphql-otel-spans.sql` | Two `graphql-demo` spans (`GetInvoice` query, `CreateInvoice` mutation) |
| `load-graphql-spans.sh` | CREATE (optional) → INSERT → verify `mapContains(…,'graphql.operation.name')` |

### Load into ephemeral / CI ClickHouse

```bash
# Example: local CH with password auth on :18123
CLICKHOUSE_URL=http://127.0.0.1:18123 \
CLICKHOUSE_PASSWORD=ci \
  ./scripts/fixtures/load-graphql-spans.sh
```

Against an existing ClickStack `otel_traces` (skip DDL):

```bash
SKIP_CREATE=1 CLICKHOUSE_URL=http://127.0.0.1:8123 \
  ./scripts/fixtures/load-graphql-spans.sh
```

Verify only:

```bash
VERIFY_ONLY=1 CLICKHOUSE_URL=… ./scripts/fixtures/load-graphql-spans.sh
```

### Expected verify output

```
n≥1 rows for GetInvoice/query and CreateInvoice/mutation
ServiceName = graphql-demo
```

Plugin letter **H** then answers `GET …/services/demo/graphql-demo/graphql?environment=ci` with `detected:true`, `source:"spans"`.

### Live prod CH

MCP / HyperDX SQL users are **read-only**. Seed live cluster via OTLP:

```bash
# see scripts/graphql-demo/README.md
OTLP_ENDPOINT=http://monitoring-clickstack:4318/v1/traces \
OTLP_AUTH="$CLICKSTACK_INGESTION_KEY" \
ONCE=1 ./scripts/graphql-demo/emit_graphql_spans.py
```
