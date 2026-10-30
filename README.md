# ClickStack APM

Grafana app plugin for **[ClickStack](https://clickhouse.com/clickstack)** — OpenTelemetry
logs, traces, metrics, and HyperDX session replay already in ClickHouse, surfaced as
service inventory, RED views, dependency maps, issues, and one-click triage.

Forked from [nais/grafana-apm-app](https://github.com/nais/grafana-apm-app). This
repo targets ClickStack first; the classic LGTM (Mimir/Tempo/Loki) path remains as
optional dual-mode.

![Service Inventory](https://raw.githubusercontent.com/Simplifi-ED/grafana-apm-app/main/src/img/screenshot-service-inventory.png)

## Features

### ClickStack-backed (primary)

When `clickHouseDataSource` is set, queries go to ClickHouse — **no Mimir, Tempo, or
Loki required**. Keep your existing OTel collectors and `@hyperdx/browser`.

- **Service inventory** — discovery from spanmetrics + health sparklines
- **RED / operations / endpoints** — rate, errors, duration from ClickHouse span metrics
- **Service map** — topology from `traces_service_graph_*`
- **Log patterns** — clustering over `otel_logs`
- **Trace breakdown** — attribute analysis over `otel_traces` (replaces Tempo TraceQL metrics)
- **Issues** — error spans grouped for triage (server-side; Faro browser exceptions are not in this CH schema)
- **Frontend Web Vitals** — HyperDX / `@hyperdx/otel-web` spans
- **Session replay** — HyperDX `hyperdx_sessions` chunks, in-drawer player
- **Triage / watchlist** — Grafana annotations (no plugin database)

Status and gaps: [docs/clickstack-migration/feature-matrix.md](https://github.com/Simplifi-ED/grafana-apm-app/blob/main/docs/clickstack-migration/feature-matrix.md).

### Still useful (capability-gated)

- **Alert templates** — pre-filled Grafana alert rules
- **Profiling** — Pyroscope flame graphs when a Pyroscope datasource exists
- **LGTM dual-mode** — if you also configure Mimir/Tempo/Loki, remaining Prom/Tempo/Loki handlers keep working

![Service Overview](https://raw.githubusercontent.com/Simplifi-ED/grafana-apm-app/main/src/img/screenshot-service-overview.png)
![Server Endpoints](https://raw.githubusercontent.com/Simplifi-ED/grafana-apm-app/main/src/img/screenshot-service-server.png)
![Frontend Observability](https://raw.githubusercontent.com/Simplifi-ED/grafana-apm-app/main/src/img/screenshot-service-frontend.png)

## Prerequisites

- **Grafana** >= 13.3.0
- **grafana-clickhouse-datasource** pointing at your ClickStack ClickHouse
- OTel data in ClickHouse (`otel_traces`, `otel_logs`, `otel_metrics_*`, optional `hyperdx_sessions`)

Optional: Pyroscope; or Mimir/Tempo/Loki for dual-mode features not yet on ClickHouse.

## Installation

This fork is **not** on grafana.com yet. Install from a GitHub Release zip (see
[Releasing](#releasing)) or build locally:

```sh
# After downloading clickhouse-apm-app-<version>.zip from Releases
grafana-cli --pluginUrl file:///path/to/clickhouse-apm-app-<version>.zip plugins install clickhouse-apm-app
```

Or copy `dist/` (or the unzipped plugin folder) into Grafana's plugins directory
and allow unsigned loading while developing:

```sh
GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=clickhouse-apm-app
```

Enable the plugin under **Administration → Plugins**.

## Configuration (ClickStack)

1. Install/configure **grafana-clickhouse-datasource** (host, auth in the
   datasource's `secureJsonData` only).
2. Open this plugin's **Configuration** page → set **ClickHouse (ClickStack)** to
   that datasource UID (or provision `jsonData.clickHouseDataSource`).
3. Leave Metrics / Traces / Logs empty for ClickStack-only.
4. Save. Health check runs `SELECT 1` via `/api/ds/query`.

Example app provisioning: [`provisioning/plugins/apps-clickhouse.yaml.example`](https://github.com/Simplifi-ED/grafana-apm-app/blob/main/provisioning/plugins/apps-clickhouse.yaml.example).

Full setup, schema notes, and LGTM dual-mode:
[docs/configuration.md](https://github.com/Simplifi-ED/grafana-apm-app/blob/main/docs/configuration.md).

## Quick start (development)

```bash
git clone https://github.com/Simplifi-ED/grafana-apm-app.git
cd grafana-apm-app
pnpm install
```

### Against an existing ClickStack ClickHouse

```bash
# Optional: SSH LocalForward CH ports (set CLICKSTACK_SSH_HOST/USER/KEY; see scripts/clickstack-tunnel.sh)
mise run clickstack:tunnel

# Provisioning: copy *.local.yaml examples for ClickHouse DS + plugin
mise run deploy:up:clickstack
pnpm run dev
```

Open `http://localhost:3000/a/clickhouse-apm-app/services`.

### Local Grafana container

The default Docker Compose file starts Grafana only. It does not provision
Mimir, Tempo, Loki, or an OpenTelemetry Collector. Configure an external
ClickStack ClickHouse datasource, then run:

```bash
mise run deploy:up:clickstack
```

Use `mise run deploy` after Go backend changes. Debug with Delve:
`mise run dev:debug`.

### Commands

| Command                         | Description                                      |
| ------------------------------- | ------------------------------------------------ |
| `mise run all`                  | Full check, test, and build                      |
| `mise run check`                | Lint, typecheck, format                          |
| `mise run test`                 | Jest + Go tests                                  |
| `mise run build`                | Production frontend + backend                    |
| `mise run dev`                  | Local Grafana + frontend watch                  |
| `mise run deploy:up:clickstack` | Grafana only (expects CH on localhost)           |
| `mise run clickstack:tunnel`    | SSH LocalForward to ClickHouse (env-required)    |
| `mise run deploy`               | Rebuild and reload plugin in Grafana             |
| `mise run e2e`                  | Playwright against running Grafana               |
| `mise run release`              | Bump version, changelog, commit + tag (see below)|

## Architecture

```
src/   → React frontend (Scenes, @grafana/ui)
pkg/   → Go backend (gRPC); ClickHouse via grafana-clickhouse-datasource proxy
         chquery/ + queries/clickhouse.go when clickStackMode is on
```

Migration notes: [docs/clickstack-migration/](https://github.com/Simplifi-ED/grafana-apm-app/tree/main/docs/clickstack-migration).

## Documentation

| Document                                                                                                                         | Contents                                      |
| -------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------- |
| [docs/configuration.md](https://github.com/Simplifi-ED/grafana-apm-app/blob/main/docs/configuration.md)                           | Datasources, ClickHouse setup, dual-mode      |
| [docs/clickstack-migration/feature-matrix.md](https://github.com/Simplifi-ED/grafana-apm-app/blob/main/docs/clickstack-migration/feature-matrix.md) | What works on ClickHouse today                |
| [docs/clickstack-migration/schema.md](https://github.com/Simplifi-ED/grafana-apm-app/blob/main/docs/clickstack-migration/schema.md) | ClickStack table / attribute expectations     |
| [docs/metrics-reference.md](https://github.com/Simplifi-ED/grafana-apm-app/blob/main/docs/metrics-reference.md)                   | LGTM metrics reference (dual-mode)            |
| [CONTRIBUTING.md](https://github.com/Simplifi-ED/grafana-apm-app/blob/main/CONTRIBUTING.md)                                       | Dev setup and quality gates                   |
| [CHANGELOG.md](https://github.com/Simplifi-ED/grafana-apm-app/blob/main/CHANGELOG.md)                                             | Release history                               |

## Releasing

Two paths; pick one and stick to it.

### A — Manual (local script)

1. Put notes under `## Unreleased` in `CHANGELOG.md`.
2. Clean working tree, then:

```bash
mise run release          # patch
mise run release -- minor
mise run release -- major
# or: mise run release -- 1.2.3
```

That bumps `package.json`, stamps the changelog, runs `mise run all`, commits
`release: vX.Y.Z`, and creates annotated tag `vX.Y.Z`.

3. Publish:

```bash
git push origin main --tags
```

4. Create a GitHub Release from the tag (or let CI do it — see B) and attach a
   **signed** plugin zip if you need production Grafana.

### B — Automated (release-please + publish)

On push to `main`, [`.github/workflows/release-please.yml`](https://github.com/Simplifi-ED/grafana-apm-app/blob/main/.github/workflows/release-please.yml):

1. Opens/updates a **Release PR** (version + CHANGELOG from Conventional Commits).
2. Merging that PR cuts the tag and GitHub Release.
3. The publish job builds frontend/backend, **signs** the plugin, validates, and
   uploads `clickhouse-apm-app-<version>.zip` to that release.

**Repo secrets required for signed builds:**

| Secret                         | Purpose                                              |
| ------------------------------ | ---------------------------------------------------- |
| `GRAFANA_ACCESS_POLICY_TOKEN`  | Grafana Cloud access policy that can sign plugins    |
| `GRAFANA_PLUGIN_ROOT_URLS`     | Comma-separated Grafana root URLs for private signing |

Without those secrets you still get a GitHub Release/tag; the zip may be unsigned
(fine for local/`GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS`, not for locked-down prod).

Local sign smoke-test:

```bash
export GRAFANA_ACCESS_POLICY_TOKEN=...
pnpm run build && npx @grafana/sign-plugin@latest --rootUrls 'https://your-grafana.example/'
```

Grafana.com catalog publish is separate (plugin catalog submission) and not wired
yet for this fork.

## Contributing

See [CONTRIBUTING.md](https://github.com/Simplifi-ED/grafana-apm-app/blob/main/CONTRIBUTING.md).

## License

[Apache-2.0](https://github.com/Simplifi-ED/grafana-apm-app/blob/main/LICENSE) — same as upstream.
