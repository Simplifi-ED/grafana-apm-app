package chquery

import (
	"fmt"
	"strings"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// InventorySQL returns rate/error aggregates from spanmetrics calls (SERVER spans).
// Latency is filled separately via LatencySQL (otel_traces quantiles) because
// histogram Sum/Count on this cluster is not a reliable p95 source.
func InventorySQL(from, to time.Time, namespace, environment string) string {
	filt := FilterClause(namespace, environment, "", "ResourceAttributes")
	return fmt.Sprintf(`
SELECT
  %s AS service,
  any(ResourceAttributes['%s']) AS namespace,
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
		ResServiceNamespace, ResCluster,
		MetricStatusErrorFilter(),
		table(queries.CHTableMetricsSum),
		q(MetricSpanCalls),
		MetricTimeFilter(from, to),
		MetricSpanKindServerFilter(),
		filt,
	)
}

// LatencySQL returns p95 duration in milliseconds from raw SERVER spans.
func LatencySQL(from, to time.Time, namespace, environment string) string {
	var parts []string
	parts = append(parts, TimestampFilter(from, to))
	parts = append(parts, TraceServerKindFilter())
	if namespace != "" {
		parts = append(parts, fmt.Sprintf("ResourceAttributes['%s'] = %s", ResServiceNamespace, q(namespace)))
	}
	if environment != "" {
		parts = append(parts, fmt.Sprintf("ResourceAttributes['%s'] = %s", ResCluster, q(environment)))
	}
	return fmt.Sprintf(`
SELECT
  ServiceName AS service,
  quantile(0.95)(Duration) / 1e6 AS p95_ms
FROM %s
WHERE %s
GROUP BY service
LIMIT 500`, table(queries.CHTableTraces), strings.Join(parts, " AND "))
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

// ParseInventoryMaps converts ClickHouse result maps into inventory rows.
func ParseInventoryMaps(callRows, latencyRows []map[string]any, windowSec float64) []queries.ServiceSummary {
	lat := make(map[string]float64, len(latencyRows))
	for _, r := range latencyRows {
		lat[ParseString(r["service"])] = ParseFloat(r["p95_ms"])
	}
	out := make([]queries.ServiceSummary, 0, len(callRows))
	for _, r := range callRows {
		svc := ParseString(r["service"])
		if svc == "" {
			continue
		}
		calls := ParseFloat(r["calls"])
		errors := ParseFloat(r["errors"])
		rate := calls / windowSec
		errRate := 0.0
		if calls > 0 {
			errRate = errors / calls
		}
		out = append(out, queries.ServiceSummary{
			Name:           svc,
			Namespace:      ParseString(r["namespace"]),
			Environment:    ParseString(r["environment"]),
			HasServerSpans: true,
			Rate:           rate,
			ErrorRate:      errRate,
			P95Duration:    lat[svc],
			DurationUnit:   "ms",
		})
	}
	return out
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

// InventoryCallsSeriesSQL returns per-service rate/error buckets for sparklines.
func InventoryCallsSeriesSQL(from, to time.Time, stepSec int64, namespace, environment, servicesCSV string) string {
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
		MetricSpanKindServerFilter(),
		filt,
		svcFilt,
	)
}

// InventoryDurationSeriesSQL returns per-service p95 duration buckets (ms).
func InventoryDurationSeriesSQL(from, to time.Time, stepSec int64, namespace, environment, servicesCSV string) string {
	if stepSec < 1 {
		stepSec = 60
	}
	var parts []string
	parts = append(parts, TimestampFilter(from, to))
	parts = append(parts, TraceServerKindFilter())
	if namespace != "" {
		parts = append(parts, fmt.Sprintf("ResourceAttributes['%s'] = %s", ResServiceNamespace, q(namespace)))
	}
	if environment != "" {
		parts = append(parts, fmt.Sprintf("ResourceAttributes['%s'] = %s", ResCluster, q(environment)))
	}
	names := ParseServiceNames(servicesCSV)
	if len(names) > 0 {
		quoted := make([]string, 0, len(names))
		for _, n := range names {
			quoted = append(quoted, q(n))
		}
		parts = append(parts, fmt.Sprintf("ServiceName IN (%s)", strings.Join(quoted, ", ")))
	}
	return fmt.Sprintf(`
SELECT
  ServiceName AS service,
  %s AS t,
  quantile(0.95)(Duration) / 1e6 AS p95_ms
FROM %s
WHERE %s
GROUP BY service, t
ORDER BY service, t
LIMIT 5000`,
		BucketMsExpr(ColTimestamp, int(stepSec)),
		table(queries.CHTableTraces),
		strings.Join(parts, " AND "),
	)
}

// AttachInventorySeries merges sparkline rows onto inventory summaries (matched by service name).
func AttachInventorySeries(summaries []queries.ServiceSummary, callRows, durationRows []map[string]any) {
	rates := make(map[string][]queries.DataPoint)
	errs := make(map[string][]queries.DataPoint)
	// Keep call rows ordered by t within each service (SQL ORDER BY service, t).
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
		if s := rates[name]; len(s) > 0 {
			summaries[i].RateSeries = s
		}
		if s := errs[name]; len(s) > 0 {
			summaries[i].ErrorSeries = s
		}
		if s := durs[name]; len(s) > 0 {
			summaries[i].DurationSeries = s
		}
	}
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

// OperationLatencySQL returns p50/p95/p99 per span name from raw spans.
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
func ParseOperations(callRows, latencyRows []map[string]any, windowSec float64) []queries.OperationSummary {
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
		kind:            ParseString(r["span_kind"]),
		httpMethod:      ParseString(r["http_method"]),
		httpRoute:       ParseString(r["http_route"]),
		rpcService:      ParseString(r["rpc_service"]),
		rpcMethod:       ParseString(r["rpc_method"]),
		dbSystem:        ParseString(r["db_system"]),
		messagingSystem: ParseString(r["messaging_system"]),
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
