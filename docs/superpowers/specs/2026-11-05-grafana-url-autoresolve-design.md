# Automatic Grafana base URL for backend → Grafana API calls

**Status:** approved design, 2026-11-05
**Branch:** `fix/grafana-url-autoresolve`

## Problem

Since 0.34.1 (PR #31, catalog verification fixes) the backend resolves the
Grafana base URL from `jsonData.internalGrafanaUrl`, else the SDK `AppURL()`
(`[server] root_url`). Before, it used `http://localhost:${GF_SERVER_HTTP_PORT:-3000}`,
which the catalog rejects (no `GF_*` env reads).

On the ClickStack VM, `root_url` is a Tailscale name. The plugin is a static Go
binary using the pure-Go resolver against `127.0.0.53`, which does not resolve
`*.ts.net`:

```
dial tcp: lookup monitoring-clickstack.tailec7cf.ts.net on 127.0.0.53:53: no such host
```

Every ClickHouse call fails. The UI shows "Configured data source is
unavailable" because `summarizeClickHouseError` collapses DNS, dial, timeout and
HTTP 400 into one message, which hid the cause.

## Goals

- Backend reaches its own Grafana with zero configuration on co-located installs
  (VM, Docker, Kubernetes pod).
- No `GF_*` environment reads (catalog rule).
- Never send the service-account token to a non-Grafana listener.
- Failures name the category and host.

## Non-goals

- Nested datasource or frontend-only architecture.
- Changing VM DNS / resolver configuration.
- Re-probing after startup (see Decisions).

## Design

### 1. Candidate resolution — `pkg/plugin/grafanaurl.go` (new)

`resolveGrafanaURL(ctx, override, logger) string` builds candidates in order:

| # | Candidate | Source |
| - | --------- | ------ |
| 1 | `jsonData.internalGrafanaUrl` | Explicit override, if non-empty |
| 2a | `http://127.0.0.1:3000` + path of `AppURL()` | Loopback, only when `root_url` has a sub-path (`serve_from_sub_path = true`) |
| 2b | `http://127.0.0.1:3000` | Loopback at root (sub-path stripped by a proxy, or no sub-path) |
| 3 | `AppURL()` | SDK, from `root_url` |

Each candidate is normalised with `queries.SafeBaseURL`; invalid ones are
skipped with a warning. Duplicates are removed (e.g. `root_url` already is
`http://127.0.0.1:3000`).

Each remaining candidate is probed: `GET {base}/api/health`, 2 s timeout, no
credentials sent. A candidate is accepted when:

- status is 200, and
- body is JSON with non-empty `version` and `database` fields, and
- if `useragent.FromContext(ctx).GrafanaVersion()` is non-empty, `version`
  equals it (prefix match tolerated for build suffixes, e.g. `13.3.0` vs
  `13.3.0+security-01`).

The first accepted candidate wins. If none is accepted, return the first valid
override or `AppURL` candidate (pre-change behaviour; a guessed loopback is
never used unverified), else "", and log at error level that probing failed.
Health probes reject any redirect.

Every probe result is logged at debug with candidate, outcome and reason; the
winner at info: `Resolved Grafana URL grafanaURL=… source=… probed=true|false`.

The probe uses `queries.NewSafeRequest` + `queries.DoValidated` with a
dedicated short-timeout client and no redirect following to a different host.

### 2. Lifecycle

- Called once from `NewApp` (replaces the current `resolveGrafanaURL`).
  Grafana is serving when it instantiates the app, so probing is valid.
- Grafana recreates the instance on settings save → a new override applies
  without restart.
- `a.grafanaURL` stays a string; the ~30 consumers are unchanged.

### 3. Error reporting — `pkg/plugin/cherror.go`

`summarizeClickHouseError` gains categories. `<host>` is the URL host only
(no path, query or credentials).

| Detected in error text | Message |
| ---------------------- | ------- |
| `no such host`, `server misbehaving` | `Plugin cannot resolve Grafana host <host> (DNS)` |
| `connection refused`, `i/o timeout`, `deadline exceeded` | `Plugin cannot reach Grafana at <host>` |
| `returned 401`, `returned 403` | `Grafana rejected plugin credentials (<code>). Check managed service accounts.` |
| `x509`, `tls:` | `TLS error calling Grafana at <host>` |
| `returned 400` | `Grafana rejected the ClickHouse query (400)` |
| authentication / access denied (ClickHouse) | unchanged |
| datasource not configured / uid empty | unchanged |
| other | unchanged (truncated, JSON stripped) |

Host is extracted from the `Post "https://host/..."` / `lookup host on` fragment
when present; otherwise the message omits it.

### 4. Visibility

- Capabilities response gains `grafanaUrl` (string; base URL only).
- AppConfig "Authentication (fallback)" section shows
  `Plugin reaches Grafana at: <grafanaUrl>` as read-only text when present.

### 5. Temporary VM unblock — `platforms-flux`

In `terraform/hetzner-vm/nix/modules/grafana.nix` `appProvisioning`:

```yaml
internalGrafanaUrl: http://127.0.0.1:3000  # TEMP until omnivya-apm-app > 0.35.0
```

Removed in the same change that bumps `apmVersion` to the release containing
this fix. Also verify the `root_url` hostname spelling (`tailec7cf`).

## Decisions

- **Loopback before `root_url`:** a co-located backend should not depend on DNS,
  ingress, TLS or OAuth proxies. Restores pre-0.34.1 behaviour without env
  reads.
- **Port 3000 guess:** Grafana's default. Non-default ports use the override.
  The version check makes a wrong guess harmless (rejected, falls through).
- **No re-probe after startup:** the URL is verified before use, Grafana's
  address does not change while the plugin runs, and instance recreation on
  settings save covers reconfiguration. Add later only if a real case appears.

## Testing (TDD)

`pkg/plugin/grafanaurl_test.go` with `httptest` servers:

- override accepted when healthy, even if loopback is also healthy
- loopback chosen when `AppURL` host does not resolve
- `AppURL` chosen when loopback is down
- version mismatch rejected
- 200 with non-Grafana body rejected
- no candidate healthy → first valid candidate returned
- sub-path preserved on loopback candidate; IPv6 base preserved
- duplicate candidates probed once

Loopback candidate construction is injectable (`loopbackBase` var / param) so
tests point it at an `httptest` server.

`pkg/plugin/cherror_test.go`: one case per table row, including host extraction.

`pkg/plugin/app_url_test.go`: updated for the new order.

Frontend: AppConfig test renders `grafanaUrl` when capabilities include it.

Gate: `mise run all`.
