# graphql-demo — OTel GraphQL span emitter (ticket 16)

Sanad has **no GraphQL server**. This minimal emitter posts OTLP/HTTP traces with
`graphql.operation.name` / `.type` / `.document` so ClickStack `otel_traces` (and
plugin letter **H**) can detect GraphQL without faking UI.

Same ingest path as Nova/Admin:

| Mode | Endpoint |
|------|----------|
| In-cluster (prod-k3s) | `http://otel-collector-logs.observability.svc.cluster.local:4318` |
| ClickStack VM (Tailscale) | `http://monitoring-clickstack:4318` + `Authorization: <ingestion key>` |
| Local | any OTLP HTTP collector on `:4318` |

## Quick emit

```bash
# one-shot against local collector
ONCE=1 python3 scripts/graphql-demo/emit_graphql_spans.py

# ClickStack (needs ingestion key from Team Settings → API Keys)
OTLP_ENDPOINT=http://monitoring-clickstack:4318/v1/traces \
OTLP_AUTH="$CLICKSTACK_INGESTION_KEY" \
ENVIRONMENT=prod-k3s-omnivya \
ONCE=1 python3 scripts/graphql-demo/emit_graphql_spans.py
```

Loops every `INTERVAL_SEC` (default 15) unless `ONCE=1`.

## Docker

```bash
cd scripts/graphql-demo
OTLP_ENDPOINT=http://host.docker.internal:4318/v1/traces docker compose up --build
```

Set `OTLP_AUTH` in the environment when targeting ClickStack `:4318` (returns **401** without it).

## Kubernetes (deploy remaining)

Kustomize root: [`kustomization.yaml`](kustomization.yaml) (ConfigMap from `emit_graphql_spans.py`).

**Must deploy in `prod-nova`:** `otel-collector-logs` `filter/namespace` drops every other `k8s.namespace.name`.

```bash
kubectl apply -k scripts/graphql-demo
# after pods Running — wait ~30–60s then:
# SELECT … mapContains(SpanAttributes,'graphql.operation.name') … ServiceName='graphql-demo'
```

## Verify in CH

```sql
SELECT count() AS n,
       SpanAttributes['graphql.operation.name'] AS op,
       SpanAttributes['graphql.operation.type'] AS typ
FROM otel_traces
WHERE Timestamp >= now() - INTERVAL 1 DAY
  AND ServiceName = 'graphql-demo'
  AND (
    mapContains(SpanAttributes, 'graphql.operation.name')
    OR mapContains(SpanAttributes, 'graphql.operation.type')
  )
GROUP BY op, typ
ORDER BY n DESC;
```

CI alternative (no OTLP): [`../fixtures/README.md`](../fixtures/README.md).
