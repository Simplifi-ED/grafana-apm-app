# graphql-demo — OTel GraphQL span emitter

Minimal emitter that posts OTLP/HTTP traces with `graphql.operation.name` /
`.type` / `.document` so ClickStack `otel_traces` (and the plugin GraphQL tab)
can detect GraphQL without a real GraphQL server.

| Mode | Endpoint |
|------|----------|
| In-cluster collector | `http://otel-collector.observability.svc.cluster.local:4318` |
| Remote ClickStack | `$OTLP_ENDPOINT` + `Authorization: <ingestion key>` |
| Local | any OTLP HTTP collector on `:4318` |

## Quick emit

```bash
# one-shot against local collector
ONCE=1 python3 scripts/graphql-demo/emit_graphql_spans.py

# remote ClickStack (set your host and ingestion key)
OTLP_ENDPOINT=http://clickhouse-host:4318/v1/traces \
OTLP_AUTH="$CLICKSTACK_INGESTION_KEY" \
ENVIRONMENT=prod \
ONCE=1 python3 scripts/graphql-demo/emit_graphql_spans.py
```

Loops every `INTERVAL_SEC` (default 15) unless `ONCE=1`.

## Docker

```bash
cd scripts/graphql-demo
OTLP_ENDPOINT=http://host.docker.internal:4318/v1/traces docker compose up --build
```

Set `OTLP_AUTH` in the environment when the collector requires a bearer token
(returns **401** without it).

## Kubernetes

Kustomize root: [`kustomization.yaml`](kustomization.yaml) (ConfigMap from `emit_graphql_spans.py`).

Default namespace is `demo` — change it if your collector filters namespaces.

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
