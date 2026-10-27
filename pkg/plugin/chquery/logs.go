package chquery

import (
	"fmt"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// LogSearchSQL returns recent log rows for a service (bounded).
func LogSearchSQL(service string, from, to time.Time, environment, severity string, limit int) string {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	fromMs, toMs := TimeBounds(from, to)
	where := fmt.Sprintf(`Timestamp >= fromUnixTimestamp64Milli(%d)
  AND Timestamp <= fromUnixTimestamp64Milli(%d)
  AND ServiceName = %s`, fromMs, toMs, q(service))
	if environment != "" {
		where += fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	if severity != "" {
		where += fmt.Sprintf(" AND lower(SeverityText) = lower(%s)", q(severity))
	}
	return fmt.Sprintf(`
SELECT
  toUnixTimestamp64Milli(Timestamp) AS ts_ms,
  SeverityText AS severity,
  Body AS body,
  TraceId AS trace_id,
  SpanId AS span_id
FROM %s
WHERE %s
ORDER BY Timestamp DESC
LIMIT %d`, table(queries.CHTableLogs), where, limit)
}

// LogPatternSQL groups recent log bodies by a truncated fingerprint for one service.
// This replaces Loki's pattern API with an in-ClickHouse GROUP BY.
func LogPatternSQL(service string, from, to time.Time, environment string, limit int) string {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	fromMs, toMs := TimeBounds(from, to)
	where := fmt.Sprintf(`Timestamp >= fromUnixTimestamp64Milli(%d)
  AND Timestamp <= fromUnixTimestamp64Milli(%d)
  AND ServiceName = %s
  AND Body != ''`, fromMs, toMs, q(service))
	if environment != "" {
		where += fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	// Normalize digits to reduce cardinality of otherwise-identical messages.
	return fmt.Sprintf(`
SELECT
  replaceRegexpAll(substring(Body, 1, 200), '[0-9]+', '#') AS pattern,
  count() AS c,
  any(SeverityText) AS severity,
  max(Timestamp) AS last_seen
FROM %s
WHERE %s
GROUP BY pattern
ORDER BY c DESC
LIMIT %d`, table(queries.CHTableLogs), where, limit)
}

// LogEntry is a single parsed log row.
type LogEntry struct {
	TimestampMs int64
	Severity    string
	Body        string
	TraceID     string
	SpanID      string
}

// ParseLogSearchMaps parses log search results.
func ParseLogSearchMaps(rows []map[string]any) []LogEntry {
	out := make([]LogEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, LogEntry{
			TimestampMs: ParseInt64(r["ts_ms"]),
			Severity:    ParseString(r["severity"]),
			Body:        ParseString(r["body"]),
			TraceID:     ParseString(r["trace_id"]),
			SpanID:      ParseString(r["span_id"]),
		})
	}
	return out
}

// LogPattern is one clustered pattern.
type LogPattern struct {
	Pattern  string
	Count    int64
	Severity string
}

// ParseLogPatternMaps parses pattern aggregations.
func ParseLogPatternMaps(rows []map[string]any) []LogPattern {
	out := make([]LogPattern, 0, len(rows))
	for _, r := range rows {
		out = append(out, LogPattern{
			Pattern:  ParseString(r["pattern"]),
			Count:    ParseInt64(r["c"]),
			Severity: ParseString(r["severity"]),
		})
	}
	return out
}
