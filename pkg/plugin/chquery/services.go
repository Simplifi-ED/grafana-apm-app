package chquery

import (
	"fmt"
	"strings"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// DiscoverInventorySQL lists services from spanmetrics without requiring SERVER
// (all SpanKinds). Optional sdk_language comes from ResourceAttributes when set.
func DiscoverInventorySQL(from, to time.Time, namespace, environment string) string {
	filt := FilterClause(namespace, environment, "", "ResourceAttributes")
	return fmt.Sprintf(`
SELECT
  %s AS service,
  any(%s) AS namespace,
  any(ResourceAttributes['%s']) AS environment,
  any(nullIf(ResourceAttributes['%s'], '')) AS sdk_language,
  sum(Value) AS calls
FROM %s
WHERE MetricName = %s
  AND %s
  AND (%s)
GROUP BY service
HAVING calls > 0
ORDER BY calls DESC
LIMIT 500`,
		ServiceExpr("Attributes"),
		NamespaceDisplayExpr("ResourceAttributes"),
		ResCluster, ResSDKLanguage,
		table(queries.CHTableMetricsSum),
		q(MetricSpanCalls),
		MetricTimeFilter(from, to),
		filt,
	)
}

// InventorySQL returns rate/error aggregates from spanmetrics calls (SERVER spans).
// Latency is filled separately via InventoryHistogramLatencySQL.
func InventorySQL(from, to time.Time, namespace, environment string) string {
	return inventoryCallsSQL(from, to, namespace, environment, SpanKindServer)
}

// InventoryClientSQL returns CLIENT-filtered rate/error aggregates for services
// that have no SERVER spans (ticket 03 CLIENT RED fallback).
func InventoryClientSQL(from, to time.Time, namespace, environment string) string {
	return inventoryCallsSQL(from, to, namespace, environment, SpanKindClient)
}

func inventoryCallsSQL(from, to time.Time, namespace, environment, spanKind string) string {
	filt := FilterClause(namespace, environment, "", "ResourceAttributes")
	return fmt.Sprintf(`
SELECT
  %s AS service,
  any(%s) AS namespace,
  any(ResourceAttributes['%s']) AS environment,
  sum(Value) AS calls,
  sumIf(Value, %s) AS errors
FROM %s
WHERE MetricName = %s
  AND %s
  AND %s
  AND (%s)
GROUP BY service
HAVING calls > 0
ORDER BY calls DESC
LIMIT 500`,
		ServiceExpr("Attributes"),
		NamespaceDisplayExpr("ResourceAttributes"),
		ResCluster,
		MetricStatusErrorFilter(),
		table(queries.CHTableMetricsSum),
		q(MetricSpanCalls),
		MetricTimeFilter(from, to),
		MetricSpanKindFilter(spanKind),
		filt,
	)
}

// LatencySQL is kept as a thin wrapper for callers that still expect the name;
// inventory uses histogram-primary InventoryHistogramLatencySQL instead.
func LatencySQL(from, to time.Time, namespace, environment string) string {
	return InventoryHistogramLatencySQL(from, to, namespace, environment, "", SpanKindServer)
}

// InventoryHistogramLatencySQL returns p95 duration (ms) from
// traces.spanmetrics.duration via quantilePrometheusHistogram (ADR 0003).
func InventoryHistogramLatencySQL(from, to time.Time, namespace, environment, servicesCSV, spanKind string) string {
	if spanKind == "" {
		spanKind = SpanKindServer
	}
	filt := FilterClause(namespace, environment, "", "ResourceAttributes")
	svcFilt := ServiceNamesFilter(servicesCSV)
	return fmt.Sprintf(`
WITH latest AS (
  SELECT
    %s AS service,
    Attributes['%s'] AS span_name,
    Attributes['%s'] AS status_code,
    Attributes['%s'] AS http_method,
    Attributes['collector.instance.id'] AS collector,
    ResourceAttributes['service.instance.id'] AS instance,
    argMax(BucketCounts, TimeUnix) AS bc,
    anyLast(ExplicitBounds) AS bounds
  FROM %s
  WHERE MetricName = %s
    AND %s
    AND %s
    AND (%s)
    AND (%s)
  GROUP BY service, span_name, status_code, http_method, collector, instance
),
merged AS (
  SELECT
    service,
    any(bounds) AS bounds,
    sumForEach(arrayMap(x -> toFloat64(x), bc)) AS bc_sum
  FROM latest
  GROUP BY service
),
exploded AS (
  SELECT
    service,
    arrayJoin(arrayZip(
      arrayConcat(bounds, [CAST('+Inf' AS Float64)]),
      arrayCumSum(bc_sum)
    )) AS pair
  FROM merged
)
SELECT
  service,
  quantilePrometheusHistogram(0.95)(pair.1, pair.2) AS p95_ms
FROM exploded
GROUP BY service
ORDER BY service
LIMIT 500`,
		ServiceExpr("Attributes"),
		AttrSpanName, AttrStatusCode, AttrHTTPMethod,
		table(queries.CHTableMetricsHistogram),
		q(MetricSpanDuration),
		MetricTimeFilter(from, to),
		MetricSpanKindFilter(spanKind),
		filt,
		svcFilt,
	)
}

// InventoryRow is one parsed inventory aggregate.
type InventoryRow struct {
	Service     string
	Namespace   string
	Environment string
	Calls       float64
	Errors      float64
	P95Ms       float64
}

// ParseInventoryMaps converts SERVER call + latency rows into summaries.
// Prefer MergeInventory for the full discover + CLIENT fallback path.
func ParseInventoryMaps(callRows, latencyRows []map[string]any, windowSec float64) []queries.ServiceSummary {
	return MergeInventory(nil, callRows, nil, latencyRows, nil, nil, windowSec)
}

// MergeInventory builds inventory summaries:
// discover (all kinds) → SERVER RED primary → CLIENT RED fallback → histogram p95.
// frontendNames marks hasFrontend (webvitals ∪ sessions ServiceNames).
func MergeInventory(
	discoverRows, serverCallRows, clientCallRows, serverLatRows, clientLatRows []map[string]any,
	frontendNames map[string]bool,
	windowSec float64,
) []queries.ServiceSummary {
	if windowSec < 1 {
		windowSec = 1
	}
	type agg struct {
		ns, env, sdk string
		serverCalls  float64
		serverErrors float64
		clientCalls  float64
		clientErrors float64
		hasServer    bool
		seen         bool
	}
	bySvc := make(map[string]*agg)

	ensure := func(svc string) *agg {
		if a, ok := bySvc[svc]; ok {
			return a
		}
		a := &agg{}
		bySvc[svc] = a
		return a
	}
	applyMeta := func(a *agg, r map[string]any) {
		if a.ns == "" {
			a.ns = ParseString(r["namespace"])
		}
		if a.env == "" {
			a.env = ParseString(r["environment"])
		}
		if a.sdk == "" {
			a.sdk = ParseString(r["sdk_language"])
		}
	}

	for _, r := range discoverRows {
		svc := ParseString(r["service"])
		if svc == "" {
			continue
		}
		a := ensure(svc)
		a.seen = true
		applyMeta(a, r)
	}
	for _, r := range serverCallRows {
		svc := ParseString(r["service"])
		if svc == "" {
			continue
		}
		a := ensure(svc)
		a.seen = true
		a.hasServer = true
		a.serverCalls = ParseFloat(r["calls"])
		a.serverErrors = ParseFloat(r["errors"])
		applyMeta(a, r)
	}
	for _, r := range clientCallRows {
		svc := ParseString(r["service"])
		if svc == "" {
			continue
		}
		a := ensure(svc)
		a.seen = true
		a.clientCalls = ParseFloat(r["calls"])
		a.clientErrors = ParseFloat(r["errors"])
		applyMeta(a, r)
	}

	serverLat := make(map[string]float64, len(serverLatRows))
	for _, r := range serverLatRows {
		serverLat[ParseString(r["service"])] = ParseFloat(r["p95_ms"])
	}
	clientLat := make(map[string]float64, len(clientLatRows))
	for _, r := range clientLatRows {
		clientLat[ParseString(r["service"])] = ParseFloat(r["p95_ms"])
	}

	out := make([]queries.ServiceSummary, 0, len(bySvc))
	for svc, a := range bySvc {
		if !a.seen || svc == "" {
			continue
		}
		calls, errors := a.serverCalls, a.serverErrors
		p95 := serverLat[svc]
		if !a.hasServer {
			calls, errors = a.clientCalls, a.clientErrors
			p95 = clientLat[svc]
		}
		errRate := 0.0
		if calls > 0 {
			errRate = errors / calls
		}
		s := queries.ServiceSummary{
			Name:           svc,
			Namespace:      a.ns,
			Environment:    a.env,
			SDKLanguage:    a.sdk,
			HasServerSpans: a.hasServer,
			Rate:           calls / windowSec,
			ErrorRate:      errRate,
			P95Duration:    p95,
			DurationUnit:   "ms",
			HasFrontend:    frontendNames != nil && frontendNames[svc],
		}
		out = append(out, s)
	}
	return out
}

// MarkHasFrontend sets HasFrontend when the inventory name or an alternate
// (resolved alias) appears in frontendNames.
func MarkHasFrontend(summaries []queries.ServiceSummary, frontendNames map[string]bool, aliasFor func(inventory string) string) {
	if len(frontendNames) == 0 {
		return
	}
	for i := range summaries {
		name := summaries[i].Name
		if frontendNames[name] {
			summaries[i].HasFrontend = true
			continue
		}
		if aliasFor != nil {
			if alt := aliasFor(name); alt != "" && alt != name && frontendNames[alt] {
				summaries[i].HasFrontend = true
			}
		}
	}
}

// sparklineWindow is the fixed lookback used for inventory mini-charts,
// matching the Prometheus path's [5m] series window.
const sparklineWindow = 5 * time.Minute

// SparklineBounds returns [from,to] for inventory series queries.
// Prefer the last 5 minutes ending at `to`, clamped to the selected range.
func SparklineBounds(from, to time.Time) (time.Time, time.Time) {
	start := to.Add(-sparklineWindow)
	if start.Before(from) {
		start = from
	}
	if !start.Before(to) {
		start = to.Add(-time.Minute)
	}
	return start, to
}

// ServiceNamesFilter builds an IN(...) predicate on ServiceExpr/ServiceName
// for the comma-separated `namespace/service` list the inventory UI passes.
func ServiceNamesFilter(servicesCSV string) string {
	names := ParseServiceNames(servicesCSV)
	if len(names) == 0 {
		return "1"
	}
	quoted := make([]string, 0, len(names))
	for _, n := range names {
		quoted = append(quoted, q(n))
	}
	in := strings.Join(quoted, ", ")
	return fmt.Sprintf("(%s IN (%s) OR ServiceName IN (%s))", ServiceExpr("Attributes"), in, in)
}

// ParseServiceNames extracts sanitized service names from "ns/name" CSV entries.
// Empty or "_" namespaces are allowed (name-only filter).
func ParseServiceNames(servicesCSV string) []string {
	if servicesCSV == "" {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	for part := range strings.SplitSeq(servicesCSV, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name := part
		if i := strings.IndexByte(part, '/'); i >= 0 {
			name = part[i+1:]
		}
		name = queries.MustSanitizeLabel(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

// InventoryCallsSeriesSQL returns per-service rate/error buckets for sparklines (SERVER).
func InventoryCallsSeriesSQL(from, to time.Time, stepSec int64, namespace, environment, servicesCSV string) string {
	return inventoryCallsSeriesSQL(from, to, stepSec, namespace, environment, servicesCSV, SpanKindServer)
}

// InventoryClientCallsSeriesSQL returns CLIENT-filtered rate/error sparkline buckets.
func InventoryClientCallsSeriesSQL(from, to time.Time, stepSec int64, namespace, environment, servicesCSV string) string {
	return inventoryCallsSeriesSQL(from, to, stepSec, namespace, environment, servicesCSV, SpanKindClient)
}

func inventoryCallsSeriesSQL(from, to time.Time, stepSec int64, namespace, environment, servicesCSV, spanKind string) string {
	if stepSec < 1 {
		stepSec = 60
	}
	filt := FilterClause(namespace, environment, "", "ResourceAttributes")
	svcFilt := ServiceNamesFilter(servicesCSV)
	return fmt.Sprintf(`
SELECT
  %s AS service,
  %s AS t,
  sum(Value) / %d AS rate,
  sumIf(Value, %s) / nullIf(sum(Value), 0) AS error_rate
FROM %s
WHERE MetricName = %s
  AND %s
  AND %s
  AND (%s)
  AND (%s)
GROUP BY service, t
ORDER BY service, t
LIMIT 5000`,
		ServiceExpr("Attributes"),
		BucketMsExpr(ColTimeUnix, int(stepSec)),
		stepSec,
		MetricStatusErrorFilter(),
		table(queries.CHTableMetricsSum),
		q(MetricSpanCalls),
		MetricTimeFilter(from, to),
		MetricSpanKindFilter(spanKind),
		filt,
		svcFilt,
	)
}

// InventoryDurationSeriesSQL returns per-service p95 duration buckets (ms) from
// the spanmetrics histogram (SERVER).
func InventoryDurationSeriesSQL(from, to time.Time, stepSec int64, namespace, environment, servicesCSV string) string {
	return inventoryHistogramSeriesSQL(from, to, stepSec, namespace, environment, servicesCSV, SpanKindServer)
}

// InventoryClientDurationSeriesSQL returns CLIENT-filtered histogram p95 sparklines.
func InventoryClientDurationSeriesSQL(from, to time.Time, stepSec int64, namespace, environment, servicesCSV string) string {
	return inventoryHistogramSeriesSQL(from, to, stepSec, namespace, environment, servicesCSV, SpanKindClient)
}

func inventoryHistogramSeriesSQL(from, to time.Time, stepSec int64, namespace, environment, servicesCSV, spanKind string) string {
	if stepSec < 1 {
		stepSec = 60
	}
	filt := FilterClause(namespace, environment, "", "ResourceAttributes")
	svcFilt := ServiceNamesFilter(servicesCSV)
	bucket := BucketMsExpr(ColTimeUnix, int(stepSec))
	return fmt.Sprintf(`
WITH latest AS (
  SELECT
    %s AS service,
    %s AS t,
    Attributes['%s'] AS span_name,
    Attributes['%s'] AS status_code,
    Attributes['%s'] AS http_method,
    Attributes['collector.instance.id'] AS collector,
    ResourceAttributes['service.instance.id'] AS instance,
    argMax(BucketCounts, TimeUnix) AS bc,
    anyLast(ExplicitBounds) AS bounds
  FROM %s
  WHERE MetricName = %s
    AND %s
    AND %s
    AND (%s)
    AND (%s)
  GROUP BY service, t, span_name, status_code, http_method, collector, instance
),
merged AS (
  SELECT
    service,
    t,
    any(bounds) AS bounds,
    sumForEach(arrayMap(x -> toFloat64(x), bc)) AS bc_sum
  FROM latest
  GROUP BY service, t
),
exploded AS (
  SELECT
    service,
    t,
    arrayJoin(arrayZip(
      arrayConcat(bounds, [CAST('+Inf' AS Float64)]),
      arrayCumSum(bc_sum)
    )) AS pair
  FROM merged
)
SELECT
  service,
  t,
  quantilePrometheusHistogram(0.95)(pair.1, pair.2) AS p95_ms
FROM exploded
GROUP BY service, t
ORDER BY service, t
LIMIT 5000`,
		ServiceExpr("Attributes"),
		bucket,
		AttrSpanName, AttrStatusCode, AttrHTTPMethod,
		table(queries.CHTableMetricsHistogram),
		q(MetricSpanDuration),
		MetricTimeFilter(from, to),
		MetricSpanKindFilter(spanKind),
		filt,
		svcFilt,
	)
}

// AttachInventorySeries merges sparkline rows onto inventory summaries (matched by service name).
func AttachInventorySeries(summaries []queries.ServiceSummary, callRows, durationRows []map[string]any) {
	AttachInventorySeriesSelective(summaries, callRows, durationRows, true)
}

// AttachInventorySeriesSelective merges series. When onlyMissing is true, only
// fills services that still lack series (CLIENT fallback path).
func AttachInventorySeriesSelective(summaries []queries.ServiceSummary, callRows, durationRows []map[string]any, overwrite bool) {
	rates := make(map[string][]queries.DataPoint)
	errs := make(map[string][]queries.DataPoint)
	lastT := make(map[string]int64)
	for _, r := range callRows {
		svc := ParseString(r["service"])
		if svc == "" {
			continue
		}
		t := ParseInt64(r["t"])
		if prev, ok := lastT[svc]; ok && t < prev {
			continue
		}
		lastT[svc] = t
		rates[svc] = append(rates[svc], queries.DataPoint{Timestamp: t, Value: ParseFloat(r["rate"])})
		errs[svc] = append(errs[svc], queries.DataPoint{Timestamp: t, Value: ParseFloat(r["error_rate"])})
	}
	durs := make(map[string][]queries.DataPoint)
	lastDT := make(map[string]int64)
	for _, r := range durationRows {
		svc := ParseString(r["service"])
		if svc == "" {
			continue
		}
		t := ParseInt64(r["t"])
		if prev, ok := lastDT[svc]; ok && t < prev {
			continue
		}
		lastDT[svc] = t
		durs[svc] = append(durs[svc], queries.DataPoint{Timestamp: t, Value: ParseFloat(r["p95_ms"])})
	}
	for i := range summaries {
		name := summaries[i].Name
		if s := rates[name]; len(s) > 0 && (overwrite || len(summaries[i].RateSeries) == 0) {
			summaries[i].RateSeries = s
		}
		if s := errs[name]; len(s) > 0 && (overwrite || len(summaries[i].ErrorSeries) == 0) {
			summaries[i].ErrorSeries = s
		}
		if s := durs[name]; len(s) > 0 && (overwrite || len(summaries[i].DurationSeries) == 0) {
			summaries[i].DurationSeries = s
		}
	}
}

// FrontendServiceNamesSQL returns ServiceNames with webvitals spans or
// hyperdx_sessions rows in the window (ticket 03 hasFrontend).
func FrontendServiceNamesSQL(from, to time.Time, environment string) string {
	envFilt := "1"
	if environment != "" {
		envFilt = fmt.Sprintf("ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	return fmt.Sprintf(`
SELECT DISTINCT service FROM (
  SELECT ServiceName AS service
  FROM %s
  WHERE SpanName = %s
    AND %s
    AND (%s)
  UNION ALL
  SELECT ServiceName AS service
  FROM %s
  WHERE ResourceAttributes['rum.sessionId'] != ''
    AND %s
    AND (%s)
)
WHERE service != ''
ORDER BY service
LIMIT 500`,
		table(queries.CHTableTraces),
		q(SpanNameWebVitals),
		TimestampFilter(from, to),
		envFilt,
		table(queries.CHTableSessions),
		TimestampFilter(from, to),
		envFilt,
	)
}

// ParseFrontendServiceNames builds a set from FrontendServiceNamesSQL rows.
func ParseFrontendServiceNames(rows []map[string]any) map[string]bool {
	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		if s := ParseString(r["service"]); s != "" {
			out[s] = true
		}
	}
	return out
}

// OperationsSQL returns per-span-name rate/error for one service, preserving
// the semantic attributes needed to classify HTTP, RPC, database, and
// messaging operations without relying on span-name heuristics.
func OperationsSQL(service string, from, to time.Time, environment string) string {
	filt := FilterClause("", environment, service, "ResourceAttributes")
	return fmt.Sprintf(`
SELECT
  Attributes['%s'] AS span_name,
  Attributes['%s'] AS span_kind,
  Attributes['%s'] AS http_method,
  Attributes['%s'] AS http_route,
  Attributes['%s'] AS rpc_service,
  Attributes['%s'] AS rpc_method,
  Attributes['%s'] AS db_system,
  Attributes['%s'] AS messaging_system,
  sum(Value) AS calls,
  sumIf(Value, %s) AS errors
FROM %s
WHERE MetricName = %s
  AND %s
  AND (%s)
GROUP BY span_name, span_kind, http_method, http_route, rpc_service, rpc_method, db_system, messaging_system
HAVING calls > 0 AND span_name != ''
ORDER BY calls DESC
LIMIT 200`,
		AttrSpanName, AttrSpanKind,
		AttrHTTPMethod, AttrHTTPRoute, AttrRPCService, AttrRPCMethod, AttrDBSystem, AttrMessagingSystem,
		MetricStatusErrorFilter(),
		table(queries.CHTableMetricsSum),
		q(MetricSpanCalls),
		MetricTimeFilter(from, to),
		filt,
	)
}

// OperationLatencyHistogramSQL returns p50/p95/p99 per operation from
// traces.spanmetrics.duration (ADR 0003 histogram-primary). Same GROUP BY dims
// as OperationsSQL so ParseOperations can join without attribute drift.
func OperationLatencyHistogramSQL(service string, from, to time.Time, environment string) string {
	filt := FilterClause("", environment, service, "ResourceAttributes")
	return fmt.Sprintf(`
WITH latest AS (
  SELECT
    Attributes['%s'] AS span_name,
    Attributes['%s'] AS span_kind,
    Attributes['%s'] AS http_method,
    Attributes['%s'] AS http_route,
    Attributes['%s'] AS rpc_service,
    Attributes['%s'] AS rpc_method,
    Attributes['%s'] AS db_system,
    Attributes['%s'] AS messaging_system,
    Attributes['%s'] AS status_code,
    Attributes['collector.instance.id'] AS collector,
    ResourceAttributes['service.instance.id'] AS instance,
    argMax(BucketCounts, TimeUnix) AS bc,
    anyLast(ExplicitBounds) AS bounds
  FROM %s
  WHERE MetricName = %s
    AND %s
    AND (%s)
  GROUP BY span_name, span_kind, http_method, http_route, rpc_service, rpc_method, db_system, messaging_system, status_code, collector, instance
),
merged AS (
  SELECT
    span_name, span_kind, http_method, http_route, rpc_service, rpc_method, db_system, messaging_system,
    any(bounds) AS bounds,
    sumForEach(arrayMap(x -> toFloat64(x), bc)) AS bc_sum
  FROM latest
  GROUP BY span_name, span_kind, http_method, http_route, rpc_service, rpc_method, db_system, messaging_system
),
exploded AS (
  SELECT
    span_name, span_kind, http_method, http_route, rpc_service, rpc_method, db_system, messaging_system,
    arrayJoin(arrayZip(
      arrayConcat(bounds, [CAST('+Inf' AS Float64)]),
      arrayCumSum(bc_sum)
    )) AS pair
  FROM merged
)
SELECT
  span_name, span_kind, http_method, http_route, rpc_service, rpc_method, db_system, messaging_system,
  quantilePrometheusHistogram(0.50)(pair.1, pair.2) AS p50_ms,
  quantilePrometheusHistogram(0.95)(pair.1, pair.2) AS p95_ms,
  quantilePrometheusHistogram(0.99)(pair.1, pair.2) AS p99_ms
FROM exploded
GROUP BY span_name, span_kind, http_method, http_route, rpc_service, rpc_method, db_system, messaging_system
ORDER BY p95_ms DESC
LIMIT 200`,
		AttrSpanName, AttrSpanKind,
		AttrHTTPMethod, AttrHTTPRoute, AttrRPCService, AttrRPCMethod, AttrDBSystem, AttrMessagingSystem,
		AttrStatusCode,
		table(queries.CHTableMetricsHistogram),
		q(MetricSpanDuration),
		MetricTimeFilter(from, to),
		filt,
	)
}

// OperationLatencySQL returns p50/p95/p99 per span name from raw traces.
// Fallback when spanmetrics duration histograms are absent.
func OperationLatencySQL(service string, from, to time.Time, environment string) string {
	where := fmt.Sprintf(`%s
  AND ServiceName = %s`, TimestampFilter(from, to), q(service))
	if environment != "" {
		where += fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	return fmt.Sprintf(`
SELECT
  SpanName AS span_name,
  SpanKind AS span_kind,
  SpanAttributes['%s'] AS http_method,
  SpanAttributes['%s'] AS http_route,
  SpanAttributes['%s'] AS rpc_service,
  SpanAttributes['%s'] AS rpc_method,
  SpanAttributes['%s'] AS db_system,
  SpanAttributes['%s'] AS messaging_system,
  quantile(0.50)(Duration) / 1e6 AS p50_ms,
  quantile(0.95)(Duration) / 1e6 AS p95_ms,
  quantile(0.99)(Duration) / 1e6 AS p99_ms
FROM %s
WHERE %s
GROUP BY span_name, span_kind, http_method, http_route, rpc_service, rpc_method, db_system, messaging_system
LIMIT 200`,
		AttrHTTPMethod, AttrHTTPRoute, AttrRPCService, AttrRPCMethod, AttrDBSystem, AttrMessagingSystem,
		table(queries.CHTableTraces), where)
}

// ParseOperations merges call and latency maps into OperationSummary.
// SpanKind is normalized so SPAN_KIND_CLIENT joins Client (traces fallback).
func ParseOperations(callRows, latencyRows []map[string]any, windowSec float64) []queries.OperationSummary {
	if windowSec < 1 {
		windowSec = 1
	}
	lat := make(map[operationKey][3]float64, len(latencyRows))
	for _, r := range latencyRows {
		k := operationKeyFromMap(r)
		lat[k] = [3]float64{ParseFloat(r["p50_ms"]), ParseFloat(r["p95_ms"]), ParseFloat(r["p99_ms"])}
	}
	out := make([]queries.OperationSummary, 0, len(callRows))
	for _, r := range callRows {
		name := ParseString(r["span_name"])
		kind := ParseString(r["span_kind"])
		calls := ParseFloat(r["calls"])
		errors := ParseFloat(r["errors"])
		errRate := 0.0
		if calls > 0 {
			errRate = errors / calls
		}
		k := operationKeyFromMap(r)
		l := lat[k]
		out = append(out, queries.OperationSummary{
			SpanName:        name,
			SpanKind:        friendlySpanKind(kind),
			SpanKindRaw:     kind,
			HTTPMethod:      k.httpMethod,
			HTTPRoute:       k.httpRoute,
			RPCService:      k.rpcService,
			RPCMethod:       k.rpcMethod,
			DBSystem:        k.dbSystem,
			MessagingSystem: k.messagingSystem,
			Rate:            calls / windowSec,
			ErrorRate:       errRate,
			P50Duration:     l[0],
			P95Duration:     l[1],
			P99Duration:     l[2],
			DurationUnit:    "ms",
		})
	}
	return out
}

type operationKey struct {
	name, kind, httpMethod, httpRoute, rpcService, rpcMethod, dbSystem, messagingSystem string
}

func operationKeyFromMap(r map[string]any) operationKey {
	return operationKey{
		name:            ParseString(r["span_name"]),
		kind:            normalizeSpanKindKey(ParseString(r["span_kind"])),
		httpMethod:      ParseString(r["http_method"]),
		httpRoute:       ParseString(r["http_route"]),
		rpcService:      ParseString(r["rpc_service"]),
		rpcMethod:       ParseString(r["rpc_method"]),
		dbSystem:        ParseString(r["db_system"]),
		messagingSystem: ParseString(r["messaging_system"]),
	}
}

// normalizeSpanKindKey maps SPAN_KIND_* and short enum labels to one join key.
func normalizeSpanKindKey(raw string) string {
	switch friendlySpanKind(raw) {
	case "Server":
		return SpanKindServer
	case "Client":
		return SpanKindClient
	case "Internal":
		return "SPAN_KIND_INTERNAL"
	case "Producer":
		return "SPAN_KIND_PRODUCER"
	case "Consumer":
		return "SPAN_KIND_CONSUMER"
	default:
		return raw
	}
}

func friendlySpanKind(raw string) string {
	switch raw {
	case SpanKindServer, TraceSpanKindServer:
		return "Server"
	case SpanKindClient, TraceSpanKindClient:
		return "Client"
	case "SPAN_KIND_INTERNAL", "Internal":
		return "Internal"
	case "SPAN_KIND_PRODUCER", "Producer":
		return "Producer"
	case "SPAN_KIND_CONSUMER", "Consumer":
		return "Consumer"
	default:
		return raw
	}
}

// DetectSpanMetricsSQL probes for the verified spanmetrics calls metric.
func DetectSpanMetricsSQL() string {
	return fmt.Sprintf(`
SELECT count() AS c
FROM %s
WHERE MetricName = %s
  AND TimeUnix > now() - INTERVAL 1 HOUR
LIMIT 1`, table(queries.CHTableMetricsSum), q(MetricSpanCalls))
}

// DetectServiceGraphSQL probes for service graph request totals.
func DetectServiceGraphSQL() string {
	return fmt.Sprintf(`
SELECT count() AS c
FROM %s
WHERE MetricName = %s
  AND TimeUnix > now() - INTERVAL 1 HOUR
LIMIT 1`, table(queries.CHTableMetricsSum), q(MetricSGRequest))
}

// ListEnvironmentsSQL returns distinct cluster values from recent spanmetrics.
func ListEnvironmentsSQL() string {
	return fmt.Sprintf(`
SELECT DISTINCT ResourceAttributes['%s'] AS env
FROM %s
WHERE MetricName = %s
  AND TimeUnix > now() - INTERVAL 24 HOUR
  AND ResourceAttributes['%s'] != ''
ORDER BY env
LIMIT 50`, ResCluster, table(queries.CHTableMetricsSum), q(MetricSpanCalls), ResCluster)
}

// ListServicesSQL returns distinct service names from recent spanmetrics.
func ListServicesSQL() string {
	return fmt.Sprintf(`
SELECT DISTINCT %s AS service
FROM %s
WHERE MetricName = %s
  AND TimeUnix > now() - INTERVAL 24 HOUR
ORDER BY service
LIMIT 500`, ServiceExpr("Attributes"), table(queries.CHTableMetricsSum), q(MetricSpanCalls))
}
