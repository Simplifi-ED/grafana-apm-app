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
	fromMs, toMs := TimeBounds(from, to)
	filt := FilterClause(namespace, environment, "", "ResourceAttributes")
	return fmt.Sprintf(`
SELECT
  %s AS service,
  any(ResourceAttributes['%s']) AS namespace,
  any(ResourceAttributes['%s']) AS environment,
  sum(Value) AS calls,
  sumIf(Value, Attributes['%s'] = %s) AS errors
FROM %s
WHERE MetricName = %s
  AND TimeUnix >= fromUnixTimestamp64Milli(%d)
  AND TimeUnix <= fromUnixTimestamp64Milli(%d)
  AND Attributes['%s'] = %s
  AND (%s)
GROUP BY service
HAVING calls > 0
ORDER BY calls DESC
LIMIT 500`,
		ServiceExpr("Attributes"),
		ResServiceNamespace, ResCluster,
		AttrStatusCode, q(StatusError),
		table(queries.CHTableMetricsSum),
		q(MetricSpanCalls),
		fromMs, toMs,
		AttrSpanKind, q(SpanKindServer),
		filt,
	)
}

// LatencySQL returns p95 duration in milliseconds from raw SERVER spans.
func LatencySQL(from, to time.Time, namespace, environment string) string {
	fromMs, toMs := TimeBounds(from, to)
	var parts []string
	parts = append(parts, fmt.Sprintf("Timestamp >= fromUnixTimestamp64Milli(%d)", fromMs))
	parts = append(parts, fmt.Sprintf("Timestamp <= fromUnixTimestamp64Milli(%d)", toMs))
	parts = append(parts, fmt.Sprintf("SpanKind = %s", q(SpanKindServer)))
	if namespace != "" {
		parts = append(parts, fmt.Sprintf("ResourceAttributes['%s'] = %s", ResServiceNamespace, q(namespace)))
	}
	if environment != "" {
		parts = append(parts, fmt.Sprintf("ResourceAttributes['%s'] = %s", ResCluster, q(environment)))
	}
	var where strings.Builder
	where.WriteString(parts[0])
	for _, p := range parts[1:] {
		where.WriteString(" AND " + p)
	}
	return fmt.Sprintf(`
SELECT
  ServiceName AS service,
  quantile(0.95)(Duration) / 1e6 AS p95_ms
FROM %s
WHERE %s
GROUP BY service
LIMIT 500`, table(queries.CHTableTraces), where.String())
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

// OperationsSQL returns per-span-name rate/error for one service.
func OperationsSQL(service string, from, to time.Time, environment string) string {
	fromMs, toMs := TimeBounds(from, to)
	filt := FilterClause("", environment, service, "ResourceAttributes")
	return fmt.Sprintf(`
SELECT
  Attributes['%s'] AS span_name,
  Attributes['%s'] AS span_kind,
  sum(Value) AS calls,
  sumIf(Value, Attributes['%s'] = %s) AS errors
FROM %s
WHERE MetricName = %s
  AND TimeUnix >= fromUnixTimestamp64Milli(%d)
  AND TimeUnix <= fromUnixTimestamp64Milli(%d)
  AND (%s)
GROUP BY span_name, span_kind
HAVING calls > 0 AND span_name != ''
ORDER BY calls DESC
LIMIT 200`,
		AttrSpanName, AttrSpanKind,
		AttrStatusCode, q(StatusError),
		table(queries.CHTableMetricsSum),
		q(MetricSpanCalls),
		fromMs, toMs,
		filt,
	)
}

// OperationLatencySQL returns p50/p95/p99 per span name from raw spans.
func OperationLatencySQL(service string, from, to time.Time, environment string) string {
	fromMs, toMs := TimeBounds(from, to)
	where := fmt.Sprintf(`Timestamp >= fromUnixTimestamp64Milli(%d)
  AND Timestamp <= fromUnixTimestamp64Milli(%d)
  AND ServiceName = %s`, fromMs, toMs, q(service))
	if environment != "" {
		where += fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	return fmt.Sprintf(`
SELECT
  SpanName AS span_name,
  SpanKind AS span_kind,
  quantile(0.50)(Duration) / 1e6 AS p50_ms,
  quantile(0.95)(Duration) / 1e6 AS p95_ms,
  quantile(0.99)(Duration) / 1e6 AS p99_ms
FROM %s
WHERE %s
GROUP BY span_name, span_kind
LIMIT 200`, table(queries.CHTableTraces), where)
}

// ParseOperations merges call and latency maps into OperationSummary.
func ParseOperations(callRows, latencyRows []map[string]any, windowSec float64) []queries.OperationSummary {
	type key struct{ name, kind string }
	lat := make(map[key][3]float64, len(latencyRows))
	for _, r := range latencyRows {
		k := key{ParseString(r["span_name"]), ParseString(r["span_kind"])}
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
		l := lat[key{name, kind}]
		out = append(out, queries.OperationSummary{
			SpanName:     name,
			SpanKind:     friendlySpanKind(kind),
			SpanKindRaw:  kind,
			Rate:         calls / windowSec,
			ErrorRate:    errRate,
			P50Duration:  l[0],
			P95Duration:  l[1],
			P99Duration:  l[2],
			DurationUnit: "ms",
		})
	}
	return out
}

func friendlySpanKind(raw string) string {
	switch raw {
	case SpanKindServer:
		return "Server"
	case SpanKindClient:
		return "Client"
	case "SPAN_KIND_INTERNAL":
		return "Internal"
	case "SPAN_KIND_PRODUCER":
		return "Producer"
	case "SPAN_KIND_CONSUMER":
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
