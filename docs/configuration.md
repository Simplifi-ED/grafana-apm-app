# Configuration

How to set up datasources, authentication, and per-environment overrides for
ClickStack APM. For the full list of expected LGTM metrics and labels (dual-mode),
see [metrics-reference.md](metrics-reference.md).

---

## Datasources

### ClickStack / ClickHouse (recommended for this fork)

When `clickHouseDataSource` is set, the plugin runs in **ClickStack mode**:
inventory, RED, service map, log patterns, trace breakdown, issues (error spans),
sessions, and session replay query ClickHouse. **Mimir, Tempo, and Loki are not
required.** Existing OTel collectors and `@hyperdx/browser` keep writing to the
same ClickHouse.

| Signal     | Type                            | Example UID  | Purpose                           |
| ---------- | ------------------------------- | ------------ | --------------------------------- |
| ClickHouse | `grafana-clickhouse-datasource` | `clickhouse` | All ClickStack-backed APM queries |

Feature status: [clickstack-migration/feature-matrix.md](clickstack-migration/feature-matrix.md).

### Profiling (Pyroscope) — optional, ticket 18

The **Profiling** tab is always visible. When a Pyroscope DS is detected it
renders real flame graphs; otherwise it shows an enablement empty-state (no
fake profiles). Detection:

1. Optional preferred UID: plugin `jsonData.pyroscopeDataSource.uid`
2. Else auto-detect the first Grafana DS of type `grafana-pyroscope-datasource`
   (or legacy `phlare`) via `GET /api/datasources`
3. `GET .../capabilities` → `pyroscope: { available, uid? }`

| Item                   | Value                                                                                                   |
| ---------------------- | ------------------------------------------------------------------------------------------------------- |
| Suggested UID          | `pyroscope`                                                                                             |
| Optional plugin config | `jsonData.pyroscopeDataSource.uid` (AppConfig → Profiling)                                              |
| Example provisioning   | [`provisioning/datasources/pyroscope.yaml.example`](../provisioning/datasources/pyroscope.yaml.example) |
| Local overlay          | `docker compose -f docker-compose.yaml -f docker-compose.pyroscope.yaml up`                             |

Install the Grafana plugin alongside ClickHouse:

```sh
GF_INSTALL_PLUGINS=grafana-clickhouse-datasource,grafana-pyroscope-datasource
```

### LGTM path (optional dual-mode)

| Signal  | Type       | Default UID | Purpose                                                       |
| ------- | ---------- | ----------- | ------------------------------------------------------------- |
| Metrics | Prometheus | `mimir`     | Span metrics, service graph, browser metrics, runtime metrics |
| Traces  | Tempo      | `tempo`     | Trace detail, exemplar links                                  |
| Logs    | Loki       | `loki`      | Faro browser telemetry, application logs                      |

### ClickHouse setup

1. Install **grafana-clickhouse-datasource** in Grafana (if not already present).
2. Create a datasource that reaches your existing ClickHouse (host, port, database
   `default`, TLS/user as required). Store credentials only in the datasource's
   `secureJsonData` — never in this plugin's settings or git.
3. On the plugin **Configuration** page, set **ClickHouse (ClickStack)** to that
   datasource UID (or provision `jsonData.clickHouseDataSource`).
4. Save. Plugin health (`CheckHealth`) runs `SELECT 1` through `/api/ds/query`
   when the UID is set.

Provisioning sketch (adapt host/auth; do not commit secrets):

```yaml
apiVersion: 1
datasources:
  - name: ClickHouse
    type: grafana-clickhouse-datasource
    uid: clickhouse
    access: proxy
    jsonData:
      host: clickhouse.example.svc
      port: 9000
      protocol: native
      defaultDatabase: default
      username: readonly
    secureJsonData:
      password: ${CLICKHOUSE_PASSWORD}
```

Plugin app settings:

```yaml
jsonData:
  clickHouseDataSource:
    uid: clickhouse
    type: grafana-clickhouse-datasource
  # Inventory (spanmetrics) → scrape/browser ServiceName for Runtime, Custom,
  # sessions/webvitals, browser Issues. Missing → empty panels + configure hint.
  serviceAliases:
    checkout-api: checkout
    billing-api: billing
    storefront: storefront
```

Schema notes for this fork's ClickStack instance:
[docs/clickstack-migration/schema.md](clickstack-migration/schema.md).
Architecture audit:
[docs/clickstack-migration/audit.md](clickstack-migration/audit.md).

Query builders live in `pkg/plugin/chquery/`; the HTTP client is
`pkg/plugin/queries/clickhouse.go`.

### Setup (LGTM dual-mode)

1. Go to the plugin's **Configuration** page
2. Enter datasource UIDs for Mimir, Tempo, and Loki
3. Click **Auto-detect capabilities** to verify connectivity and detect metric names
4. Save

---

## Auto-Detection

On startup (and when you click **Auto-detect**), the plugin probes Mimir to
discover which metrics exist. This determines which features are enabled.

### What Gets Detected

| Setting              | Values                                                               | How                                                                                                       |
| -------------------- | -------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| Metric namespace     | `traces_span_metrics`, `traces_spanmetrics`, `spanmetrics`, _(bare)_ | Probes `{ns}_calls_total` in order                                                                        |
| Duration unit        | `ms` or `s`                                                          | Probes `{ns}_duration_milliseconds_bucket` then `{ns}_duration_seconds_bucket` then `{ns}_latency_bucket` |
| Service graph prefix | `traces_service_graph` or `service_graph`                            | Probes `{pfx}_request_total`                                                                              |

See [metrics-reference.md — Auto-Detection](metrics-reference.md#auto-detection--capabilities)
for the full probe order.

### Manual Override

If auto-detection fails (e.g., restricted Mimir access), you can set the
namespace and duration unit manually on the plugin configuration page.

---

## Per-Environment Datasource Overrides

If you run separate Tempo/Loki instances per environment (e.g., `dev-gcp`,
`prod-gcp`), you can configure per-environment overrides. When a user selects
an environment filter, trace and log links will route to the matching
datasource instead of the default.

Each override maps an environment name (matching `deployment.environment`) to
a Tempo UID and/or Loki UID:

```json
{
  "tracesDataSource": {
    "uid": "tempo",
    "type": "tempo",
    "byEnvironment": {
      "dev-gcp": { "uid": "dev-gcp-tempo", "type": "tempo" },
      "prod-gcp": { "uid": "prod-gcp-tempo", "type": "tempo" }
    }
  },
  "logsDataSource": {
    "uid": "loki",
    "type": "loki",
    "byEnvironment": {
      "dev-gcp": { "uid": "dev-gcp-loki", "type": "loki" },
      "prod-gcp": { "uid": "prod-gcp-loki", "type": "loki" }
    }
  }
}
```

When a user selects an environment filter, the app resolves the matching
datasource UID. If no override exists, the default datasource is used.

---

## Authentication

When Grafana runs behind an OAuth2 proxy (e.g., Wonderwall on Nais), the
plugin backend cannot use forwarded browser cookies for internal API calls
because the cookies belong to the proxy, not to Grafana itself.

### Option 1 — Auto-managed (recommended, Grafana 10.3+)

Enable Grafana's `externalServiceAccounts` feature toggle and set
`auth.managed_service_accounts_enabled = true` in `grafana.ini`. The plugin
will automatically receive a managed service account token — no manual
configuration needed.

### Option 2 — Manual service account token

1. Go to **Administration > Service accounts** and create a new account
2. Assign **Viewer** role (only read access to datasources is needed)
3. Generate a token (starts with `glsa_`)
4. Paste it into the **Grafana Service Account Token** field on the plugin
   configuration page

### When Auth Is Not Needed

Auth is not needed for local development with anonymous auth or when Grafana
handles authentication directly (no OAuth2 proxy in front).

The plugin uses a 3-tier fallback: auto-managed token → manual token →
forwarded user headers.

---

## Deploy Annotations

Time-series panels on the Backend (RED) and Frontend tabs render deploy markers from
Grafana **organization annotations** tagged `clickhouse-apm:deploy`,
`service:<app>`, `namespace:<ns>`, `env:<env>`, and `version:<sha>`. The
plugin queries the built-in `grafana` annotations datasource filtered by the
`clickhouse-apm:deploy` and `service:` tags (plus `env:` when an environment filter
is selected); the marker tooltip shows the annotation text, which carries the
deployed version and a link to the triggering workflow. To produce these
annotations from CI, use the reusable GitHub Action in
[`infra/actions/apm-deploy-annotation`](../infra/actions/apm-deploy-annotation/action.yml)
after a successful deploy — it POSTs one annotation to `/api/annotations`
using a service-account token with the `annotations:create` permission. No
plugin configuration is required; markers appear as soon as annotations exist.

---

## Framework Detection Troubleshooting

The service list shows a **type badge** (Ktor, Spring, Node.js, Go) next to
services where the framework can be detected from Prometheus metrics. If your
service doesn't show a badge:

1. **No Prometheus scraping** — detection metrics come from application-emitted
   Prometheus metrics (not OTel span metrics). Your pod must be scraped by
   Prometheus with `app` and `namespace` labels.
2. **Missing instrumentation library** — for JVM apps, you need Micrometer
   (typically via Spring Boot Actuator or Ktor metrics plugin). For Node.js,
   `prom-client` or the OTel Prometheus exporter. For Go, `prometheus/client_golang`.
3. **Label mismatch** — the `app` label on your Prometheus metrics must match
   the `service_name` on your OTel span metrics. In Nais, both default to the
   application name from `nais.yaml`.

See [metrics-reference.md — Framework Detection](metrics-reference.md#framework-detection)
for the full list of detection metrics.

---

## Metrics Pipeline

The plugin reads pre-aggregated **span metrics** and **service graph metrics**
from Mimir, not raw traces. These can be produced two ways:

### Option A: Tempo metrics-generator (recommended)

Tempo's built-in [metrics-generator](https://grafana.com/docs/tempo/latest/metrics-generator/)
derives metrics from ingested traces and remote-writes them to Mimir. This is
the approach used in production at Nav.

### Option B: OTel Collector connectors

The OTel Collector can generate equivalent metrics using the
[spanmetrics](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/connector/spanmetricsconnector)
and [servicegraph](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/connector/servicegraphconnector)
connectors. This is the approach used in the local dev setup.

See [metrics-reference.md](metrics-reference.md) for detailed collector
configuration and the full set of required dimensions.
