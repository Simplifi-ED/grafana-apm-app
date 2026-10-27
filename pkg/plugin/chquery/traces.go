package chquery

import (
	"fmt"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// TraceBreakdownSQL groups spans by name for a service (replaces Tempo TraceQL breakdown).
func TraceBreakdownSQL(service string, from, to time.Time, environment string, limit int) string {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
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
  count() AS spans,
  countIf(StatusCode = 'Error') AS errors,
  quantile(0.50)(Duration) / 1e6 AS p50_ms,
  quantile(0.95)(Duration) / 1e6 AS p95_ms,
  avg(Duration) / 1e6 AS avg_ms
FROM %s
WHERE %s
GROUP BY span_name, span_kind
ORDER BY spans DESC
LIMIT %d`, table(queries.CHTableTraces), where, limit)
}

// TraceSearchSQL finds recent traces for a service (bounded).
func TraceSearchSQL(service string, from, to time.Time, environment string, onlyErrors bool, limit int) string {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	fromMs, toMs := TimeBounds(from, to)
	where := fmt.Sprintf(`Timestamp >= fromUnixTimestamp64Milli(%d)
  AND Timestamp <= fromUnixTimestamp64Milli(%d)
  AND ServiceName = %s
  AND TraceId != ''`, fromMs, toMs, q(service))
	if environment != "" {
		where += fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	if onlyErrors {
		where += " AND StatusCode = 'Error'"
	}
	return fmt.Sprintf(`
SELECT
  TraceId AS trace_id,
  min(Timestamp) AS start_ts,
  max(Timestamp) AS end_ts,
  count() AS spans,
  countIf(StatusCode = 'Error') AS errors,
  sum(Duration) / 1e6 AS total_ms
FROM %s
WHERE %s
GROUP BY trace_id
ORDER BY start_ts DESC
LIMIT %d`, table(queries.CHTableTraces), where, limit)
}

// TraceDetailSQL returns spans for one trace id (waterfall input).
func TraceDetailSQL(traceID string, from, to time.Time, limit int) string {
	if limit <= 0 || limit > 2000 {
		limit = 1000
	}
	fromMs, toMs := TimeBounds(from, to)
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
  AND Timestamp >= fromUnixTimestamp64Milli(%d)
  AND Timestamp <= fromUnixTimestamp64Milli(%d)
ORDER BY Timestamp ASC
LIMIT %d`, table(queries.CHTableTraces), q(traceID), fromMs, toMs, limit)
}

// TraceBreakdownRow is one span-name aggregate.
type TraceBreakdownRow struct {
	SpanName string
	SpanKind string
	Spans    int64
	Errors   int64
	P50Ms    float64
	P95Ms    float64
	AvgMs    float64
}

// ParseTraceBreakdownMaps parses breakdown rows.
func ParseTraceBreakdownMaps(rows []map[string]any) []TraceBreakdownRow {
	out := make([]TraceBreakdownRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, TraceBreakdownRow{
			SpanName: ParseString(r["span_name"]),
			SpanKind: ParseString(r["span_kind"]),
			Spans:    ParseInt64(r["spans"]),
			Errors:   ParseInt64(r["errors"]),
			P50Ms:    ParseFloat(r["p50_ms"]),
			P95Ms:    ParseFloat(r["p95_ms"]),
			AvgMs:    ParseFloat(r["avg_ms"]),
		})
	}
	return out
}
