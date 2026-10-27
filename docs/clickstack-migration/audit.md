# ClickStack migration — architecture audit

**Date:** 2026-10-27  
**Repo:** `nais/grafana-apm-app` (local fork)  
**Scope:** Phase A inspection + Phase B query foundation (this step does **not** port APM pages)

## Goal

Run Grafana OSS + this app as the UI over an **already-deployed ClickStack** stack:

- Keep ClickHouse storage and existing OpenTelemetry collectors / `@hyperdx/browser`.
- Do **not** add Loki, Tempo, Mimir, or Prometheus as required telemetry backends for the ClickStack path.
- Use the existing Grafana `grafana-clickhouse-datasource` plugin (UID configured in app settings).

## Current architecture (LGTM path)

| Layer | Role |
|-------|------|
| Frontend (`src/`) | Scenes UI; calls Go resource API (`src/api/`) |
| Backend (`pkg/plugin/`) | Aggregates PromQL / LogQL / TraceQL; returns JSON DTOs |
| Metrics | `PrometheusClient` → Grafana datasource **proxy** `/api/datasources/proxy/uid/{mimir}` |
| Logs / Faro / issues | `DsQueryClient` → `/api/ds/query` against Loki (`expr`, instant/range) |
| Traces | Tempo UID for Explore links + some TraceQL analytics via Tempo API |
| Config | `jsonData.metricsDataSource` / `tracesDataSource` / `logsDataSource` (+ per-env overrides) |

Key files:

- `pkg/plugin/app.go` — settings, clients, `CheckHealth`, routes
- `pkg/plugin/queries/promql.go` — Prometheus/Mimir HTTP client
- `pkg/plugin/queries/dsquery.go` — Loki/Prom via `/api/ds/query` (uses `expr`, not `rawSql`)
- `pkg/plugin/queries/models.go` — `PluginSettings`, capabilities DTOs
- `pkg/plugin/otelconfig/` — PromQL/LogQL/TraceQL builders
- `src/components/AppConfig/AppConfig.tsx` — datasource UID UI

**Important:** `DsQueryClient` cannot issue ClickHouse SQL unchanged. The ClickHouse datasource expects `rawSql` + `format` (+ `queryType: "sql"`), not Loki/Prom `expr`.

## Grafana ClickHouse query contract (verified from plugin usage)

`POST /api/ds/query`:

```json
{
  "from": "<epoch_ms>",
  "to": "<epoch_ms>",
  "queries": [{
    "refId": "A",
    "datasource": { "type": "grafana-clickhouse-datasource", "uid": "<uid>" },
    "rawSql": "SELECT 1",
    "format": 1,
    "queryType": "sql"
  }]
}
```

- `format: 1` = table frames (foundation default); `2` = time series.
- Auth: Grafana service account token or forwarded user headers (same pattern as existing clients).
- Macros such as `$__timeFilter(Timestamp)` are expanded by the ClickHouse datasource using `from`/`to`.
- Response: standard Grafana data frames under `results.A.frames[]`.

Credentials must live on the Grafana datasource (`secureJsonData`); the app only stores the **UID**.

## ClickStack schema (verified)

Live inspection via ClickStack MCP against connection `Local-ClickHouse` (database `default`). See [schema.md](schema.md).

| Signal | Table | Notes |
|--------|-------|-------|
| Logs | `otel_logs` | OTel logs; `ServiceName`, `Body`, attribute maps |
| Traces | `otel_traces` | Spans; `Duration` ns; `StatusCode`; events/links arrays |
| Trace index | `otel_traces_trace_id_ts` | TraceId → time lookup (+ MV) |
| Metrics | `otel_metrics_{gauge,sum,histogram,summary,exponential_histogram}` | OTel metric tables; spanmetrics present as `traces.spanmetrics.*` / `traces_service_graph_*` |
| Session replay | `hyperdx_sessions` | HyperDX / browser session payloads |

Browser SDK and collectors are **out of scope** for this step (unchanged).

## Gaps vs APM product surface

The plugin today assumes **pre-aggregated PromQL span metrics** and **Loki Faro streams**. ClickStack stores:

- Raw spans/logs in CH tables
- OTel metrics (including spanmetrics/servicegraph names observed in CH)

Porting inventory/RED/issues/sessions will need SQL (or CH metric queries) that reproduce today’s DTO contracts — **not started in this step**.

## Foundation delivered (Phase B)

- `pkg/plugin/queries/clickhouse.go` — reusable `/api/ds/query` client, SQL safety, frame parsing, ping
- `jsonData.clickHouseDataSource` + App Config UI field
- `CheckHealth` probes ClickHouse when UID is set; LGTM checks remain when metrics DS is set
- Docs: this audit, schema inventory, configuration notes

## Non-goals (this step)

- Porting Service Inventory / Overview / Issues / Sessions pages
- Removing LGTM clients
- Deploying ClickHouse or changing OTel collectors / `@hyperdx/browser`
- Claiming end-to-end Grafana→ClickHouse verification without a provisioned Grafana CH datasource in this environment
