package chquery

import (
	"fmt"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// TraceBreakdownSQL groups spans by name for a service (replaces Tempo TraceQL breakdown).
func TraceBreakdownSQL(service string, from, to time.Time, environment string, limit int) string {
	return TraceBreakdownSQLForDimension(service, from, to, environment, "name", limit)
}

// TraceBreakdownSQLForDimension groups spans by a curated span attribute.
func TraceBreakdownSQLForDimension(service string, from, to time.Time, environment, dimension string, limit int) string {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	dimensionExpr := traceDimensionExpr(dimension)
	where := fmt.Sprintf(`%s
  AND ServiceName = %s`, TimestampFilter(from, to), q(service))
	if environment != "" {
		where += fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	return fmt.Sprintf(`
SELECT
  %s AS dimension,
  SpanKind AS span_kind,
  count() AS spans,
  countIf(%s) AS errors,
  quantile(0.50)(Duration) / 1e6 AS p50_ms,
  quantile(0.95)(Duration) / 1e6 AS p95_ms,
  quantile(0.99)(Duration) / 1e6 AS p99_ms,
  avg(Duration) / 1e6 AS avg_ms
FROM %s
WHERE %s
GROUP BY dimension, span_kind
HAVING dimension != ''
ORDER BY spans DESC
LIMIT %d`, dimensionExpr, TraceStatusErrorFilter(), table(queries.CHTableTraces), where, limit)
}

// TraceSearchSQL finds recent traces for a service (bounded).
func TraceSearchSQL(service string, from, to time.Time, environment string, onlyErrors bool, limit int) string {
	status := ""
	if onlyErrors {
		status = "error"
	}
	return TraceSearchSQLFiltered(service, from, to, environment, status, "", 0, 0, limit)
}

// TraceSearchSQLFiltered finds recent traces with UI filter support.
func TraceSearchSQLFiltered(
	service string,
	from, to time.Time,
	environment, status, spanSearch string,
	minDurationMs, maxDurationMs float64,
	limit int,
) string {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	where := fmt.Sprintf(`%s
  AND ServiceName = %s
  AND TraceId != ''`, TimestampFilter(from, to), q(service))
	if environment != "" {
		where += fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	if status == "error" {
		where += " AND " + TraceStatusErrorFilter()
	}
	if status == "ok" {
		where += " AND " + TraceStatusNotErrorFilter()
	}
	if spanSearch != "" {
		where += fmt.Sprintf(" AND positionCaseInsensitiveUTF8(SpanName, %s) > 0", q(spanSearch))
	}
	if minDurationMs > 0 {
		where += fmt.Sprintf(" AND Duration >= %g * 1000000", minDurationMs)
	}
	if maxDurationMs > 0 {
		where += fmt.Sprintf(" AND Duration <= %g * 1000000", maxDurationMs)
	}
	return fmt.Sprintf(`
SELECT
  TraceId AS trace_id,
  toUnixTimestamp64Milli(min(Timestamp)) AS start_ts,
  toUnixTimestamp64Milli(max(Timestamp)) AS end_ts,
  count() AS spans,
  countIf(%s) AS errors,
  sum(Duration) / 1e6 AS total_ms
FROM %s
WHERE %s
GROUP BY trace_id
ORDER BY start_ts DESC
LIMIT %d`, TraceStatusErrorFilter(), table(queries.CHTableTraces), where, limit)
}

// TraceDetailSQL returns spans for one trace id (waterfall input).
func TraceDetailSQL(traceID string, from, to time.Time, limit int) string {
	if limit <= 0 || limit > 2000 {
		limit = 1000
	}
	return fmt.Sprintf(`
SELECT
  toUnixTimestamp64Milli(Timestamp) AS ts_ms,
  TraceId AS trace_id,
  SpanId AS span_id,
  ParentSpanId AS parent_span_id,
  SpanName AS span_name,
  SpanKind AS span_kind,
  ServiceName AS service,
  StatusCode AS status,
  Duration AS duration_ns,
  StatusMessage AS status_message
FROM %s
WHERE TraceId = %s
  AND %s
ORDER BY Timestamp ASC
LIMIT %d`, table(queries.CHTableTraces), q(traceID), TimestampFilter(from, to), limit)
}

func traceDimensionExpr(dimension string) string {
	switch dimension {
	case "http.route":
		return "SpanAttributes['http.route']"
	case "http.status_code":
		return "SpanAttributes['http.status_code']"
	case "db.system":
		return "SpanAttributes['db.system']"
	case "db.operation":
		return "SpanAttributes['db.operation.name']"
	case "messaging.system":
		return "SpanAttributes['messaging.system']"
	case "rpc.method":
		return "SpanAttributes['rpc.method']"
	default:
		return "SpanName"
	}
}

// TraceBreakdownRow is one span-name aggregate.
type TraceBreakdownRow struct {
	SpanName string
	Value    string
	SpanKind string
	Spans    int64
	Errors   int64
	P50Ms    float64
	P95Ms    float64
	P99Ms    float64
	AvgMs    float64
}

// ParseTraceBreakdownMaps parses breakdown rows.
func ParseTraceBreakdownMaps(rows []map[string]any) []TraceBreakdownRow {
	out := make([]TraceBreakdownRow, 0, len(rows))
	for _, r := range rows {
		value := ParseString(r["dimension"])
		if value == "" {
			value = ParseString(r["span_name"])
		}
		out = append(out, TraceBreakdownRow{
			SpanName: value,
			Value:    value,
			SpanKind: ParseString(r["span_kind"]),
			Spans:    ParseInt64(r["spans"]),
			Errors:   ParseInt64(r["errors"]),
			P50Ms:    ParseFloat(r["p50_ms"]),
			P95Ms:    ParseFloat(r["p95_ms"]),
			P99Ms:    ParseFloat(r["p99_ms"]),
			AvgMs:    ParseFloat(r["avg_ms"]),
		})
	}
	return out
}
