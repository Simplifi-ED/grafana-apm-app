# ClickStack schema inventory

**Inspected:** 2026-10-27 via ClickStack MCP (`clickstack_list_sources`, `clickstack_describe_source`, `clickstack_sql`) against connection **Local-ClickHouse** (`connectionId` `6ad0e2cb978a70ad2dbf744c`).

**Normative query rules** (enums, time, env/ns, duration, inventory discover vs RED): see [ADR 0003 — ClickStack query contract](../adr/0003-clickstack-query-contract.md). This file remains the **live examples / verified shapes** reference.

**Legend**

- ✅ **Verified** — observed from live DESCRIBE / system tables / describe_source
- ⚠️ **Inferred** — reasonable from OTel/ClickStack conventions but not fully sampled
- ❌ **Unknown** — not confirmed; do not code against these without a follow-up query

Do not treat HyperDX UI defaults as authoritative without a verified row below.

---

## Databases & tables

| Database | Table | Engine | Approx rows (then) | Signal | Status |
|----------|-------|--------|--------------------|--------|--------|
| `default` | `otel_logs` | MergeTree | ~84k | Logs | ✅ |
| `default` | `otel_traces` | MergeTree | ~287k | Traces / spans | ✅ |
| `default` | `otel_traces_trace_id_ts` | MergeTree | ~56k | TraceId time index | ✅ |
| `default` | `otel_traces_trace_id_ts_mv` | MaterializedView | — | MV for trace index | ✅ |
| `default` | `otel_metrics_gauge` | MergeTree | ~67M | Gauge metrics | ✅ |
| `default` | `otel_metrics_sum` | MergeTree | ~85M | Sum / counters | ✅ |
| `default` | `otel_metrics_histogram` | MergeTree | ~45M | Histograms | ✅ |
| `default` | `otel_metrics_summary` | MergeTree | ~3M | Summaries | ✅ |
| `default` | `otel_metrics_exponential_histogram` | MergeTree | 0 | Exp. histograms | ✅ (empty) |
| `default` | `hyperdx_sessions` | MergeTree | ~2.2k | Session replay | ✅ |

ClickStack source mapping (HyperDX UI):

| Source name | kind | Table / metricTables | Status |
|-------------|------|----------------------|--------|
| Logs | log | `otel_logs` | ✅ |
| Traces | trace | `otel_traces` | ✅ (table via source config + DESCRIBE) |
| Metrics | metric | gauge/sum/histogram/summary tables above | ✅ |
| Sessions | session | `hyperdx_sessions` | ✅ |

---

## `otel_logs` (logs)

✅ Columns (from `clickstack_describe_source`):

| Column | Type |
|--------|------|
| Timestamp | DateTime64(9) |
| TraceId, SpanId | String |
| TraceFlags | UInt8 |
| SeverityText | LowCardinality(String) |
| SeverityNumber | UInt8 |
| ServiceName | LowCardinality(String) |
| Body | String |
| ResourceSchemaUrl, ScopeSchemaUrl | LowCardinality(String) |
| ResourceAttributes, ScopeAttributes, LogAttributes | Map(LowCardinality(String), String) |
| ScopeName | String |
| ScopeVersion | LowCardinality(String) |
| EventName | String |
| `__otel_materialized_*` | LowCardinality(String) materialized k8s / env fields |

✅ Sampled map keys include k8s resource attrs and various LogAttributes (`level`, `msg`, envoy access fields, postgres `record.*`, …).

⚠️ Exception / browser-error shape in logs: not fully catalogued here — sample for `exception.*` / Faro-like attrs before porting Issues.

---

## `otel_traces` (traces / spans)

✅ Columns (from `DESCRIBE TABLE default.otel_traces`):

| Column | Type |
|--------|------|
| Timestamp | DateTime64(9) |
| TraceId, SpanId, ParentSpanId, TraceState | String |
| SpanName, SpanKind, ServiceName, StatusCode | LowCardinality(String) |
| ResourceAttributes, SpanAttributes | Map(LowCardinality(String), String) |
| ScopeName, ScopeVersion, StatusMessage | String |
| Duration | UInt64 (**nanoseconds** — ClickStack `durationPrecision: 9`) |
| Events.Timestamp / Name / Attributes | Arrays |
| Links.TraceId / SpanId / TraceState / Attributes | Arrays |

✅ Sample: `count()` where `StatusCode = 'Error'` returned non-zero (~10k) — `StatusCode` values include at least `Error`.

⚠️ Full `SpanKind` / `StatusCode` enums and high-cardinality SpanAttributes keys: re-query with `clickstack_describe_source` when that call succeeds (it intermittently failed during this audit).

---

## Metrics tables

✅ Shared columns on gauge (and similarly sum/histogram families):  
`ServiceName`, `MetricName`, `MetricUnit`, `Attributes`, `ResourceAttributes`, `TimeUnix`, `StartTimeUnix`, `Value` (gauge), exemplars arrays, …

✅ Histogram-specific (from `system.columns` on `otel_metrics_histogram`):  
`Count`, `Sum`, `BucketCounts`, `ExplicitBounds`, `Min`, `Max`, `AggregationTemporality`.

✅ Metric names observed in this cluster include:

- Spanmetrics: `traces.spanmetrics.calls`, `traces.spanmetrics.duration` (unit ms)
- Service graph: `traces_service_graph_request_total`, `traces_service_graph_request_failed_total`, `traces_service_graph_request_server`, `traces_service_graph_request_client`
- App/runtime: `nova_http_request_duration_seconds`, `go_*`, envoy/controller metrics, …

⚠️ Attribute keys for spanmetrics series (service name label vs `ServiceName` column vs `Attributes['…']`) must be confirmed per metric with `clickstack_describe_metric` before rewriting RED queries.

---

## `hyperdx_sessions` (session replay)

✅ Table exists; column layout matches log-like OTel row shape (`Timestamp`, `TraceId`, `SpanId`, `ServiceName`, `Body`, attribute maps, materialized k8s fields) per `clickstack_describe_source`.

✅ **Verified recording format (2026-10-27, live sample against Local-ClickHouse):**

| Aspect | Value |
|--------|-------|
| Table | `default.hyperdx_sessions` |
| ScopeName | `rum.rr-web` |
| Service | `ServiceName` (observed: `nova-frontend`); also `ResourceAttributes['service.name']` |
| Session ID | `ResourceAttributes['rum.sessionId']` (32-char hex) |
| Environment | `ResourceAttributes['cluster']` (observed: `prod-k3s-omnivya`) — not `deployment.environment` |
| SDK | `ResourceAttributes['telemetry.sdk.name']` = `@hyperdx/otel-web`, `rum.version` ≈ `0.20.0` |
| Body | **Raw JSON string** — one rrweb `eventWithTime` per logical event (`type`, `data`, `timestamp`). **Not** NAIS Faro `gzip+b64` |
| Chunking | Large events split ~950KB: `LogAttributes['rr-web.event']`, `rr-web.chunk` (1-based), `rr-web.total-chunks`, `rr-web.offset`. Chunk >1 bodies are **string fragments** (do not parse alone) |
| Ordering | Prefer `Timestamp ASC`, then `rr-web.event`, then `rr-web.chunk`. Reassemble by sorting chunks within an event id before `JSON.parse` |
| Compression | None at rest in Body |
| Size | Observed p50 Body ~251 B, p95 ~890 KB, max ~973 KB per row; multi-chunk FullSnapshots concatenate larger |

⚠️ Avoid selecting full `Body` in unbounded queries. Cap rows and assembled bytes in the plugin.

⚠️ Player compatibility: recordings come from HyperDX's rrweb recorder; Grafana ships `@grafana/rrweb-player` 2.x. Major-version skew can degrade fidelity (see HyperDX issue #2569). Retrieval is correct; playback quality depends on player/recorder pairing.

---

## Browser telemetry (`@hyperdx/browser`)

✅ Session replay rows land in `hyperdx_sessions` (collector routes `rr-web.event` logs there). Other browser signals may also appear in `otel_traces` / `otel_logs` — out of scope for the replay vertical slice.

---

## Schema-discovery SQL (if MCP / agent connectivity is unavailable)

Run read-only against the same ClickHouse the Grafana datasource uses:

```sql
-- Tables
SELECT name, engine, total_rows, total_bytes
FROM system.tables
WHERE database = 'default' AND name NOT LIKE '.%'
ORDER BY name;

-- Columns
SELECT table, name, type
FROM system.columns
WHERE database = 'default'
  AND table IN (
    'otel_logs', 'otel_traces', 'hyperdx_sessions',
    'otel_metrics_gauge', 'otel_metrics_sum',
    'otel_metrics_histogram', 'otel_metrics_summary'
  )
ORDER BY table, position;

-- Smoke counts (bounded)
SELECT count() FROM default.otel_traces WHERE Timestamp > now() - INTERVAL 1 HOUR LIMIT 1;
SELECT ServiceName, count() AS c
FROM default.otel_logs
WHERE Timestamp > now() - INTERVAL 1 HOUR
GROUP BY ServiceName
ORDER BY c DESC
LIMIT 20;
```

Grafana datasource: point `grafana-clickhouse-datasource` at this host/db; put the datasource **UID** in plugin `jsonData.clickHouseDataSource.uid`. Never commit passwords.

---

## What this step does *not* claim

- End-to-end verification through **Grafana's** ClickHouse datasource from this repo's docker stack (not provisioned here yet).
- Completeness of browser/exception/session payload schemas beyond table/column inventory.
