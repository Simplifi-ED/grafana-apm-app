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
| Frontend versions | `GET .../frontend/versions` | `ResourceAttributes['service.version']` | ✅ | **G** + **21**: DEV `@hyperdx/otel-web` emits `service.version=APP_VERSION` (never `rum.version`) |
| Frontend Feedback | Feedback UI | `otel_traces` `SpanName=faro.feedback` | ✅ | **G** + **21**: contract live after `keep-browser-rum` sampling (#121); Support form optional extra proof |
| Frontend sessions list | `GET .../frontend/sessions` | `hyperdx_sessions` + error enrich from traces | 🟨 | **F** polish: errors/user enrich, alias hint, ClickStack empty copy |
| Session replay | replay / probe / recordings | `hyperdx_sessions` | ✅ | |
| Service map | `GET /service-map` | `traces_service_graph_*` | 🟨 | |
| Dependencies / connected | `GET .../dependencies`, `/connected` | service graph CH | 🟨 | **D** service/global/ns/detail + connected |
| Database top queries / N+1 | DB tab | `otel_traces` db attrs | 🟨 | **D** `chquery` top queries + N+1 |
| Runtime metrics | Runtime tab | `otel_metrics_*` `go_*`/`process_*` | 🟨 | **C** shipped: alias → scrape `ServiceName`; Go/process CH queries; miss → hint |
| Custom metrics | Custom tab | `otel_metrics_*` | 🟨 | **C** shipped: denylist discovery across gauge/sum/hist/summary |
| GraphQL | GraphQL / Server tab | `otel_metrics_*` probes ∨ `otel_traces` `graphql.operation.*` | 🟨 | **H** + **16**: plugin detect/query ready; no live Sanad GQL (demo removed; tab empty+hint until an app emits) |
| Alerts / templates | Alerts UI | Grafana UA + CH SQL | 🟨 | **D** list/firing (**10**); full catalog CH SQL eval (**27**) |
| Scorecard | Scorecard | CH probes | 🟨 | **D** Nais 6 + graphql/jobs N/A (**22**); **J** profiling scored via Pyroscope probe |
| Jobs | Jobs UI | KSM/Prom and/or `kube_*` in CH | ✅ | **I** + **17**: `kube_cronjob_*` in CH (`namespace=prod-nova`) via KSM + honor_labels |
| Triage / watchlist / sourcemap | annotations / HTTP | N/A | ✅ | |
| Profiling | Profiling tab | Pyroscope DS | ✅ | **J** + **18**: local compose Pyroscope DS; plugin detects (`available:true`) |
| Service identity aliases | plugin `jsonData.serviceAliases` | — | ✅ | **A** / **15** — Sanad seeds: `nova-api→nova-app`, `admin-api→admin-app`, `nova-frontend` (example + local provisioning) |

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
| Frontend `service.version` | ✅ DEV: `20260422164408-main` on `@hyperdx/otel-web` (**21**) |
| Feedback RUM events | ✅ `SpanName=faro.feedback` after keep-browser-rum (**21** / #121) |
| Runtime identity gap | spanmetrics `nova-api` vs scrape `nova-app` (**15**/ **06**) — aliases |
| GraphQL / Jobs / Pyroscope | Jobs ✅ KSM; Pyroscope ✅ local; GraphQL plugin-ready (no live emitter) |

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
| G | Versions + Feedback | ✅ unit + DEV CH smoke (**21**) |
| H | GraphQL | ✅ plugin path; live emitter optional (**16**) |
| I | Jobs | ✅ Path A/B + live `kube_cronjob_*` (**17**) |
| J | Profiling | ✅ plugin + local Pyroscope DS (**18**) |

Platform **17/18/21** live (DEV/local); **16** plugin-ready without demo. Remaining: merge Sanad #836 → prod; local Grafana UI smoke with aliases.
