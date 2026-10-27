# ClickStack migration — feature matrix

**Updated:** 2026-10-27 (plan sync — wayfinder tickets 05/12/13/15/19/20)  
**Mode:** When `jsonData.clickHouseDataSource.uid` is set, handlers use ClickHouse (`clickStackMode`). LGTM clients are not required.  
**Phase 0 DoD:** full Nais UX parity with **real data** (ticket **05**). Soft-unavailable ≠ done for GraphQL / Jobs / Profiling / Feedback. Build order: ticket **12** (A→J).

**Status legend**

| Status | Meaning |
|--------|---------|
| ✅ Verified working | Unit tests + live ClickHouse SQL confirmed against Local-ClickHouse |
| 🟨 Implemented, integration-unverified | Code path exists; not exercised end-to-end through Grafana CH datasource in this session |
| 🟧 Partial | Works for a subset; gaps + plan tickets noted |
| 🚧 Planned / gated | Locked in plan; plugin and/or platform enablement not shipped yet |
| ❌ Unsupported (rejected for Phase 0) | Would be empty forever — not acceptable; use 🚧 + enablement ticket instead |

---

## Core workflows

| Feature | Endpoint / UI | CH tables | Status | Plan / notes |
|---------|---------------|-----------|--------|--------------|
| Service inventory | `GET /services` → ServiceInventory | `otel_metrics_sum` (`traces.spanmetrics.calls`) + hist duration | 🟨 | **B** shipped: all-kind discover, SERVER RED + CLIENT fallback, histogram p95/sparklines, `hasFrontend`/`sdkLanguage` |
| Capabilities | `GET /capabilities` | sum metrics probes | 🟨 | + browser Issues/Versions/Feedback probes (**13**) |
| Operations RED | `GET .../operations` | spanmetrics + traces quantiles | 🟨 | ADR 0003 SpanKind/StatusCode |
| Overview RED timeseries | Scenes panels | `otel_metrics_sum` + histogram | 🟨 | Histogram-primary duration (**04**) |
| Endpoints | `GET .../endpoints` | spanmetrics → protocol groups | 🟨 | |
| Service health | `GET .../health` | spanmetrics + service graph | 🟨 | **D** CH path: RED + prev-period + degraded ops/deps |
| Trace breakdown | `GET .../traces/breakdown` | `otel_traces` | 🟨 | |
| Trace waterfall / Explore | Traces tab | `otel_traces` | 🟨 | **E** CH-primary + chips/SQL; Tempo Explore secondary |
| Log patterns | `GET .../logs/patterns` | `otel_logs` Body clustering | 🟨 | |
| Log Explore | Logs tab | `otel_logs` | 🟨 | **E** CH-primary + chips/SQL; patterns kept; Loki Explore secondary |
| Issues / exceptions | `GET .../issues` | `otel_traces` | 🟨 | **F** shipped: HyperDX browser (`unhandledrejection`/`error.*`) + server `Events.exception` badged (**19**/ **20**); empty UX + alias/probe hints (**13**) |
| Issue occurrences / drawer | CH spans + `rum.sessionId` | `otel_traces`, `hyperdx_sessions` | 🟨 | **F** shipped; replay deep-link via `rum.sessionId`; Faro Loki not required |
| Frontend Web Vitals | `GET .../frontend` + tab | `otel_traces` `webvitals` | 🟨 | HyperDX live |
| Frontend versions | `GET .../frontend/versions` | `ResourceAttributes['service.version']` | 🚧 | **G** gated on **21** (`service.version` empty on browser today; do not use `rum.version`) |
| Frontend Feedback | Feedback UI | `otel_traces` `SpanName=faro.feedback` | 🚧 | Contract locked (**21**); Sanad emit pending deploy; letter **G** |
| Frontend sessions list | `GET .../frontend/sessions` | `hyperdx_sessions` + error enrich from traces | 🟨 | **F** polish: errors/user enrich, alias hint, ClickStack empty copy |
| Session replay | replay / probe / recordings | `hyperdx_sessions` | ✅ | |
| Service map | `GET /service-map` | `traces_service_graph_*` | 🟨 | |
| Dependencies / connected | `GET .../dependencies`, `/connected` | service graph CH | 🟨 | **D** service/global/ns/detail + connected |
| Database top queries / N+1 | DB tab | `otel_traces` db attrs | 🟨 | **D** `chquery` top queries + N+1 |
| Runtime metrics | Runtime tab | `otel_metrics_*` `go_*`/`process_*` | 🟨 | **C** shipped: alias → scrape `ServiceName`; Go/process CH queries; miss → hint |
| Custom metrics | Custom tab | `otel_metrics_*` | 🟨 | **C** shipped: denylist discovery across gauge/sum/hist/summary |
| GraphQL | GraphQL tab | TBD metrics/spans | 🚧 | **H** gated on **16** |
| Alerts / templates | Alerts UI | Grafana UA + CH SQL | 🟨 | **D** list/firing (**10**); full catalog CH SQL eval (**27**) |
| Scorecard | Scorecard | CH probes | 🟨 | **D** Nais 6 + graphql/jobs/profiling N/A (**22**) |
| Jobs | Jobs UI | KSM/Prom and/or `kube_*` in CH | 🚧 | **I** gated on **17** |
| Triage / watchlist / sourcemap | annotations / HTTP | N/A | ✅ | |
| Profiling | Profiling tab | Pyroscope DS | 🚧 | **J** gated on **18** |
| Service identity aliases | plugin `jsonData.serviceAliases` | — | 🟨 | **A** / **15** — resolve helper + AppConfig; Runtime/Custom CH queries in **C** |

---

## Live ClickHouse checks performed (MCP, not Grafana DS)

| Check | Result |
|-------|--------|
| `traces.spanmetrics.calls` Attributes | `service.name`, `span.kind`, `span.name`, `status.code`, http.* |
| Inventory SERVER aggregate | `envoy-gateway`, `nova-api`, `admin-api` with calls/errors |
| Duration | Prefer `traces.spanmetrics.duration` histogram; short windows often empty on traces-only p95 |
| Service graph edges | client/server including nova-api→postgres, nova-frontend→envoy-gateway |
| `hyperdx_sessions` multi-chunk concat | Valid rrweb type=2 + timestamp |
| Browser Issues | `nova-frontend` `SpanName=unhandledrejection` + `error.message`/`error.stack` + `Events.exception.*` + `rum.sessionId` (**20**) |
| Server exception events | `nova-api` `Events.Name=exception` (`exception.type`/`message`) |
| Frontend `service.version` | **Empty** on `@hyperdx/otel-web`; `rum.version=0.20.0` is SDK only |
| Feedback RUM events | **Absent** (only unrelated `GET /api/feedback` HTTP spans) |
| Runtime identity gap | spanmetrics `nova-api` vs scrape `nova-app` (**15**/ **06**) |
| GraphQL / Jobs / Pyroscope | No signal in this CH (**08**/ **09**/ **11**) → enablement **16–18** |

---

## Configuration (ClickStack-only)

1. Provision `grafana-clickhouse-datasource` → existing ClickHouse.
2. Plugin `jsonData.clickHouseDataSource.uid` (see `provisioning/plugins/apps-clickhouse.yaml.example`).
3. Add `jsonData.serviceAliases` (inventory → scrape/browser names) — ticket **15**.
4. Leave Metrics/Traces/Logs empty unless hybrid.
5. `CheckHealth` must report ClickHouse OK without requiring Tempo/Mimir.
6. Restart Grafana after backend binary rebuild (`mise run deploy` / `backend:build`).

Docs: [schema.md](schema.md), [session-replay.md](session-replay.md), [ADR 0003](../adr/0003-clickstack-query-contract.md), [configuration.md](../configuration.md).  
Plan tracker: `.scratch/clickstack-nais-parity/map.md`.

---

## Remaining work (plan spine A→J)

| Letter | Work | Enablement |
|--------|------|------------|
| A | ADR helpers + `serviceAliases` | 🟨 foundation shipped |
| B | Inventory polish | 🟨 unit-tested; live Grafana DS smoke pending |
| C | Runtime + Custom | 🟨 unit-tested; Sanad smoke via aliases pending |
| D | Alerts / Scorecard / DB / deps health | 🟨 shipped this session |
| E | Explore depth | 🟨 filter chips + Advanced SQL + CH-primary; Explore links UID-gated |
| F | Issues + Sessions UX | 🟨 unit-tested; Sanad smoke pending |
| G | Versions + Feedback | **21** |
| H | GraphQL | **16** |
| I | Jobs | **17** |
| J | Profiling | **18** |

Platform **16–18/21** start in parallel with A. Gated tabs: capability probe + empty-state (**13**), no fake zeros.
