# ClickStack migration — feature matrix

**Updated:** 2026-10-27  
**Mode:** When `jsonData.clickHouseDataSource.uid` is set, handlers use ClickHouse (`clickStackMode`). LGTM clients are not required.

**Status legend**

| Status | Meaning |
|--------|---------|
| ✅ Verified working | Unit tests + live ClickHouse SQL confirmed against Local-ClickHouse |
| 🟨 Implemented, integration-unverified | Code path exists; not exercised end-to-end through Grafana CH datasource in this session |
| 🟧 Partial | Works for a subset of the LGTM feature; gaps documented |
| ❌ Unsupported | Schema/data cannot support; returns unavailable / empty without fabricating |

---

## Core workflows

| Feature | Endpoint / UI | CH tables | Status | Notes |
|---------|---------------|-----------|--------|-------|
| Service inventory | `GET /services` → ServiceInventory | `otel_metrics_sum` (`traces.spanmetrics.calls`) + `otel_traces` p95 | 🟨 | Live inventory SQL verified; Grafana DS path unit-tested via mock |
| Capabilities | `GET /capabilities` | sum metrics probes | 🟨 | Detects spanmetrics + service graph + lists services/envs |
| Operations RED | `GET .../operations` | spanmetrics + traces quantiles | 🟨 | |
| Overview RED timeseries | Scenes panels | `otel_metrics_sum` + `otel_traces` | 🟨 | ClickHouse SQL scenes when no Prom UID |
| Endpoints | `GET .../endpoints` | spanmetrics → protocol groups | 🟨 | Maps CH operations into HTTP/DB/client/messaging groups |
| Service health | `GET .../health` | — | 🟧 | Still Prom when configured |
| Trace breakdown | `GET .../traces/breakdown` | `otel_traces` | 🟨 | Replaces Tempo TraceQL |
| Trace waterfall / Explore | Traces tab Tempo links | `otel_traces` available | 🟧 | Plugin breakdown CH; Explore Tempo URLs disabled without Tempo UID |
| Log patterns | `GET .../logs/patterns` | `otel_logs` Body clustering | 🟨 | Replaces Loki pattern API |
| Log Explore | Logs tab | — | 🟧 | Needs CH Explore or in-app search (patterns only today) |
| Issues / exceptions | `GET .../issues` | `otel_traces` StatusCode=Error | 🟧 | Server error spans only; **no Faro browser exceptions** in CH |
| Issue occurrences / drawer breadcrumbs | Loki Faro | — | ❌ | Faro streams not in this CH; replay still works via hyperdx_sessions |
| Frontend Web Vitals | `GET .../frontend` | `otel_traces` `webvitals` spans | 🟨 | HyperDX/@hyperdx/otel-web attributes (lcp/fcp/…); bullet charts in ClickStack UI |
| Frontend versions | `GET .../frontend/versions` | — | ❌ | Faro `app_version` not verified |
| Frontend sessions list | `GET .../frontend/sessions` | `hyperdx_sessions` | 🟨 | rum.sessionId list |
| Session replay | `.../replay`, probe, recordings | `hyperdx_sessions` | ✅ | Step 2; live chunk reassembly verified |
| Service map | `GET /service-map` | `traces_service_graph_*` | 🟨 | Live edge attrs verified (client/server) |
| Dependencies / connected | `GET .../dependencies`, `/connected` | — | 🟧 | Service map CH works; dedicated dep handlers still Prom-first |
| Database top queries / N+1 | Tempo | `otel_traces` db attrs | 🟧 | SQL builders not wired yet; Tempo path skipped in CH-only |
| Runtime / custom metrics / GraphQL | Prom | `otel_metrics_*` may hold `go_*` etc. | 🟧 | Not ported; empty/unavailable without Prom |
| Alerts / alert templates | Grafana + Prom ruler | N/A | 🟧 | Unrelated to telemetry storage |
| Scorecard | multi | N/A | 🟧 | Mixed; Console API ok |
| Jobs | KSM Prom | UNKNOWN | ❌ | Not verified in CH |
| Triage / watchlist / sourcemap | annotations / HTTP | N/A | ✅ | No LGTM dependency |
| Profiling | Pyroscope | N/A | ❌ | Out of scope |

---

## Live ClickHouse checks performed (MCP, not Grafana DS)

| Check | Result |
|-------|--------|
| `traces.spanmetrics.calls` Attributes | `service.name`, `span.kind`, `span.name`, `status.code`, http.* |
| Inventory SERVER aggregate | `envoy-gateway`, `nova-api`, `admin-api` with calls/errors |
| `otel_traces` p95 by service | Non-zero for nova-api / envoy-gateway |
| Service graph edges | client/server including nova-api→postgres, nova-frontend→envoy-gateway |
| `hyperdx_sessions` multi-chunk concat | Valid rrweb type=2 + timestamp |
| Log `exception.type` attrs | **Empty** in sampled window — Body text / error spans used instead |
| Faro / Alloy web vitals metrics | **Not found** as Faro names; HyperDX `webvitals` spans in `otel_traces` verified |

---

## Configuration (ClickStack-only)

1. Provision `grafana-clickhouse-datasource` → existing ClickHouse.
2. Plugin `jsonData.clickHouseDataSource.uid` (see `provisioning/plugins/apps-clickhouse.yaml.example`).
3. Leave Metrics/Traces/Logs empty.
4. `CheckHealth` must report ClickHouse OK without requiring Tempo/Mimir.
5. Restart Grafana after backend binary rebuild (`mise run deploy` / `backend:build`).

Docs: [schema.md](schema.md), [session-replay.md](session-replay.md), [configuration.md](../configuration.md).

---

## Remaining work (honest)

- Wire endpoints/health/dependencies/database handlers fully onto `chquery`.
- In-app log/trace search UI without Tempo/Loki Explore.
- Browser issues/vitals need HyperDX browser signal mapping (not Faro) — data shape TBD.
- End-to-end Grafana→`/api/ds/query` verification once CH DS is provisioned in the local Grafana stack.
- Player/recorder rrweb version skew for HyperDX 0.20 vs `@grafana/rrweb-player` 2.x.
