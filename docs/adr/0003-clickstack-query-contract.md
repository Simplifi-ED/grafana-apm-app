# ClickStack query contract

ClickStack mode talks to OTel/HyperDX tables in ClickHouse with **different string enums and timestamp columns than Prometheus/Tempo**. We lock a query-layer contract (helpers in `pkg/plugin/chquery`) so every handler uses the same encodings, filters, and metric-vs-trace preferences — without changing collectors or adding cluster MVs.

## Rules

1. **SpanKind / StatusCode dual encoding**  
   - Metrics (`Attributes`): `SPAN_KIND_*`, `STATUS_CODE_*`.  
   - Traces (columns): short labels `Server`/`Client`/`…`, `Error`/`Ok`/`Unset`. Trace filters may defensively `IN (short, SPAN_KIND_*)`. Never cross-wire.

2. **Time**  
   - Metrics time column: `TimeUnix`. Traces/logs/sessions: `Timestamp`. All are `DateTime64(9)`.  
   - Bounds: `fromUnixTimestamp64Milli`. Buckets: `toUnixTimestamp(toStartOfInterval(...)) * 1000`. No raw epoch compares.

3. **Environment** = `ResourceAttributes['cluster']` only (v1). Do not use `deployment.environment.name` for inventory/env pickers.

4. **APM namespace** = `ResourceAttributes['service.namespace']`. Never `k8s.namespace.name`.

5. **Service identity**  
   - Metrics: `coalesce(nullIf(Attributes['service.name'], ''), ServiceName)`.  
   - Traces: `ServiceName`.  
   - Scrape↔inventory aliases (`nova-api` vs `nova-app`) are **out of this ADR** (wayfinder ticket 15).

6. **Duration**  
   - Inventory p95 + duration sparklines: **histogram-primary** on `traces.spanmetrics.duration` via `quantilePrometheusHistogram` (not `Sum/Count`, not short-window trace quantiles).  
   - Raw-trace quantiles: deep-dive / operations when spans exist.

7. **Inventory discovery vs RED**  
   - **Discover** services from spanmetrics **without** requiring SERVER (all kinds).  
   - **Default RED** series remain **SERVER-filtered**. How CLIENT-only rows present metrics/UX is decided in wayfinder ticket 03 (not fallback RED in this ADR).

## Why

Live Sanad CH showed metrics and traces disagreeing on enum strings; short-window Server spans are too sparse for inventory p95 while histograms are dense; env/namespace keys that look “OTel-standard” (`deployment.environment.name`, k8s ns) are wrong for this deployment’s inventory UX. Encoding these as Prom/Tempo idioms caused 404-empty charts and silent zeros.

## See also

- Facts: `.scratch/clickstack-nais-parity/research/01-ch-query-contract-facts.md`, `04-duration-from-spanmetrics.md`  
- Examples: `docs/clickstack-migration/schema.md`  
- Status: accepted (2026-10-27)
