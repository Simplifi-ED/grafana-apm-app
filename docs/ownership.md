# Service ownership

Directory entry on a service: **who runs it, how do I reach them**. Not a score.

## Resource attributes (source of truth)

Emit these on the workload (SDK `OTEL_RESOURCE_ATTRIBUTES`, collector `resource` /
k8s attributes processor). The plugin reads the latest non-empty values from
ClickHouse `otel_traces.ResourceAttributes`.

| Field   | Attribute              | Required for “owned” |
|---------|------------------------|----------------------|
| Team    | `owner.team`           | **Yes**              |
| Contact | `owner.contact`        | No (Slack URL, `#channel`, or email) |
| Repo    | `vcs.repository.url`   | No                   |
| Runbook | `owner.runbook.url`    | No                   |

Ownership is **service-level** (not per environment).

### Collector snippet (Kubernetes)

```yaml
processors:
  resource/owner:
    attributes:
      - key: owner.team
        value: platform
        action: upsert
      - key: owner.contact
        value: "https://slack.com/app_redirect?channel=platform"
        action: upsert
      - key: vcs.repository.url
        value: https://github.com/org/payments
        action: upsert
      - key: owner.runbook.url
        value: https://wiki.example/runbooks/payments
        action: upsert
```

### SDK env

```bash
export OTEL_RESOURCE_ATTRIBUTES="owner.team=platform,owner.contact=#platform,vcs.repository.url=https://github.com/org/payments"
```

## Annotation fallback

When telemetry is not tagged yet, publish a Grafana annotation:

| Tag | Value |
|-----|--------|
| `clickhouse-apm:owner` | (marker) |
| `service:<name>` | inventory service name |

Put the same keys in the annotation **text** as `key=value` lines (or tags
`team:…`, `contact:…`, `repo:…`, `runbook:…`).

## Precedence

`telemetry` → `claimed` (not in v1 UI) → `annotation`.

The UI always shows which source won. A source with `owner.team` takes precedence
over a source without it. If both telemetry and annotation have `owner.team` and
disagree, telemetry wins and the annotation values appear under conflict detail.
If neither source has `owner.team`, the merge selects non-empty telemetry first,
then non-empty annotation. If both sources are empty, no source wins. All fields
come from the selected source. A conflict is recorded only when both sources
have `owner.team` and differ.

## Freshness

`lastSeen` is the latest telemetry timestamp that carried ownership attrs.
**Stale** when that sighting is older than **7 days** (owner known, telemetry gone).

## Known teams (optional)

Plugin jsonData `knownTeams: string[]`. When non-empty, a team string not in the
list still displays but is flagged `teamKnown: false`. Empty list = free text only.

## States

| State | Meaning |
|-------|---------|
| Owned | `owner.team` present from a winning source |
| Unowned | no team — add `owner.team` (snippet above) |
| Stale | owned, but telemetry last-seen past threshold |
| Conflict | sources disagree; winner shown |
| Unverified | claimed in UI with no telemetry match (post-v1) |
