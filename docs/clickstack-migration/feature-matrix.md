# ClickStack migration — feature matrix

**Updated:** 2026-10-28 (closeout UI smoke vs prod CH: Versions/Feedback/Jobs/Runtime/aliases 10/10; wayfinder 16–18/21/27 closed)  
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
| Service inventory | `GET /services` → ServiceInventory | `otel_metrics_sum` (`traces.spanmetrics.calls`) + hist duration | ✅ | **B** live: p95 + sparklines; ns prefers `k8s.namespace.name` over product `service.namespace=sanad`; residual `graphql-demo` until CH TTL |
| Capabilities | `GET /capabilities` | sum metrics probes | ✅ | Live: spanmetrics+SG+CH+Pyroscope; Tempo/Loki N/A in CH mode |
| Operations RED | `GET .../operations` | spanmetrics calls + **duration hist** p50/p95/p99 | ✅ | Hist-primary latency (traces fallback); SpanKind join normalized |
| Overview RED timeseries | Scenes panels | `otel_metrics_sum` + histogram | ✅ | Covered via inventory sparklines + health/ops RED (same CH metrics) |
| Endpoints | `GET .../endpoints` | spanmetrics → protocol groups | ✅ | Derived from operations (same hist latency) |
| Service health | `GET .../health` | spanmetrics + service graph | ✅ | Live: prev-period + degradedOps/deps; p95 from ops hist |
| Trace breakdown | `GET .../traces/breakdown` | `otel_traces` | ✅ | Live: mode=spanmetrics, 18 rows (redis.dial / query …) |
| Trace waterfall / Explore | Traces tab | `otel_traces` | ✅ | **E** CH search live via Grafana DS (nova-api traces) |
| Log patterns | `GET .../logs/patterns` | `otel_logs` Body clustering | ✅ | **E** k8s identity + JSON body level match (filelog SeverityText empty) |
| Log Explore | Logs tab | `otel_logs` | ✅ | **E** match via alias→`k8s.deployment.name` / pod prefix when ServiceName empty |
| Issues / exceptions | `GET .../issues` | `otel_traces` | ✅ | **F** live: nova-api server exceptions (24h); nova-frontend browser `Element not found` (7d) |
| Issue occurrences / drawer | CH spans + `rum.sessionId` | `otel_traces`, `hyperdx_sessions` | ✅ | **F** shipped; replay deep-link via `rum.sessionId`; Faro Loki not required |
| Frontend Web Vitals | `GET .../frontend` + tab | `otel_traces` `webvitals` | ✅ | **G** CH scene: bullets + trends + volume/exceptions + CWV rating pie + browser/per-page tables (HyperDX; no Faro) |
| Frontend versions | `GET .../frontend/versions` | `ResourceAttributes['service.version']` | ✅ | **G** + **21**: DEV `@hyperdx/otel-web` emits `service.version=APP_VERSION` (never `rum.version`) |
| Frontend Feedback | Feedback UI | `otel_traces` `SpanName=faro.feedback` | ✅ | **G** + **21**: contract live after `keep-browser-rum` sampling (#121); Support form optional extra proof |
| Frontend sessions list | `GET .../frontend/sessions` | `hyperdx_sessions` + enrich from traces | ✅ | **F** enrich: errors + user/version/pages/UA from all browser spans (not errors-only) |
| Session replay | replay / probe / recordings | `hyperdx_sessions` | ✅ | |
| Service map | `GET /service-map` | `traces_service_graph_*` | ✅ | Live edges via Grafana CH DS (envoy/nova-api/postgres/valkey) |
| Dependencies / connected | `GET .../dependencies`, `/connected` | service graph CH | ✅ | **D** live: nova-api → valkey/postgres/nova; inbound envoy-gateway (p95 often 0 — graph totals only) |
| Database top queries / N+1 | DB tab | `otel_traces` db attrs | ✅ | Top queries live (redis EVALSHA…); N+1 API OK, 0 findings in 1h cap (needs `db.statement` + ≥10 repeats/trace) |
| Runtime metrics | Runtime tab | `otel_metrics_*` `go_*`/`process_*` | ✅ | **C** + aliases: `nova-api→nova-app` Go runtime live |
| Custom metrics | Custom tab | `otel_metrics_*` | ✅ | **C** live via aliases (`nova-api→nova-app`, 23 families); outer `GROUP BY MetricName, kind` (CH code 215) |
| GraphQL | GraphQL / Server tab | `otel_metrics_*` probes ∨ `otel_traces` `graphql.operation.*` | ✅ | Plugin path live (`detected:false` + hint); needs app emitter (**16**) |
| Alerts / templates | Alerts UI | Grafana UA + CH SQL | ✅ | **D**/**27** `GET /alert-templates/error-rate` returns CH SQL defaults |
| Scorecard | Scorecard | CH probes | ✅ | Logs+runtime via aliases; Jobs scored (kube_*); Profiling OK; GraphQL N/A |
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
| A | ADR helpers + `serviceAliases` | ✅ aliases live |
| B | Inventory polish | ✅ live Grafana DS (k8s ns prefer) |
| C | Runtime + Custom | ✅ both live via aliases |
| D | Alerts / Scorecard / DB / deps health | ✅ alerts + scorecard + deps/connected live |
| E | Explore depth | ✅ traces/logs/patterns live via Grafana CH DS |
| F | Issues + Sessions UX | ✅ sessions + Issues live |
| G | Versions + Feedback | ✅ unit + DEV CH smoke (**21**) |
| H | GraphQL | ✅ plugin path; live emitter optional (**16**) |
| I | Jobs | ✅ Path A/B + live `kube_cronjob_*` (**17**) |
| J | Profiling | ✅ plugin + local Pyroscope DS (**18**) |

Platform **17/18/21** live (DEV/local); **16** plugin-ready without demo. Core spine A→J verified via Grafana CH DS on Local-ClickHouse.

---

## Post-parity hardening (plan spine A→F)

Tracker: `.scratch/post-parity-hardening/map.md`.

| Letter | Work | Status |
|--------|------|--------|
| **A** | Triage pagination + release-please CI | 🚧 PR #5 |
| **B+C** | Alerts UX + `regressing-exceptions` template | 🚧 PR #6 |
| **D** | `/issues` isNew + returnedAfterResolve; badges; Overview strip; Alerts CTA | 🚧 this branch |
| **E** | Custom signals CH Overview auto-charts | 🚧 |
| **F** | Scorecard v2 depth | 🚧 |

Namespace-wide attention strip deferred (would N+1 `/issues` per service).
