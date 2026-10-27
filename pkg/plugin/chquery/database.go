package chquery

import (
	"fmt"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// Database tab CH path (letter D): top queries + N+1 candidates from otel_traces.

const (
	// AttrDBStatement is the OTel db.statement span attribute.
	AttrDBStatement = "db.statement"
	// AttrDBSQLTable is the OTel db.sql.table span attribute.
	AttrDBSQLTable = "db.sql.table"
)

// TopDBQueriesSQL samples DB client spans with statements for aggregation in Go.
// Bounded by limit (spans). Duration is returned in milliseconds.
func TopDBQueriesSQL(service, environment string, from, to time.Time, limit int) string {
	if limit < 1 {
		limit = 500
	}
	where := fmt.Sprintf(`%s
  AND ServiceName = %s
  AND SpanAttributes['%s'] != ''
  AND SpanAttributes['%s'] != ''`,
		TimestampFilter(from, to), q(service), AttrDBSystem, AttrDBStatement)
	if environment != "" {
		where += fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	return fmt.Sprintf(`
SELECT
  TraceId AS trace_id,
  SpanAttributes['%s'] AS db_system,
  SpanAttributes['%s'] AS statement,
  SpanAttributes['%s'] AS db_table,
  Duration / 1e6 AS duration_ms
FROM %s
WHERE %s
ORDER BY Timestamp DESC
LIMIT %d`,
		AttrDBSystem, AttrDBStatement, AttrDBSQLTable,
		table(queries.CHTableTraces), where, limit,
	)
}

// NPlusOneCandidateSQL returns DB spans from traces that exceed the repeat floor
// (many DB client spans). Endpoint is approximated as the earliest root SpanName
// in the same TraceId (empty ParentSpanId); otherwise the earliest SpanName.
func NPlusOneCandidateSQL(service, environment string, from, to time.Time, minDBSpans, spanLimit int) string {
	if minDBSpans < 2 {
		minDBSpans = 2
	}
	if spanLimit < 1 {
		spanLimit = 2000
	}
	where := fmt.Sprintf(`%s
  AND ServiceName = %s
  AND SpanAttributes['%s'] != ''
  AND SpanAttributes['%s'] != ''`,
		TimestampFilter(from, to), q(service), AttrDBSystem, AttrDBStatement)
	if environment != "" {
		where += fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	rootWhere := fmt.Sprintf(`%s AND ServiceName = %s`, TimestampFilter(from, to), q(service))
	if environment != "" {
		rootWhere += fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	return fmt.Sprintf(`
WITH candidates AS (
  SELECT TraceId
  FROM %s
  WHERE %s
  GROUP BY TraceId
  HAVING count() >= %d
  ORDER BY count() DESC
  LIMIT 100
),
roots AS (
  SELECT TraceId, argMin(SpanName, Timestamp) AS endpoint
  FROM %s
  WHERE (%s)
    AND (ParentSpanId = '' OR ParentSpanId = '0000000000000000')
    AND TraceId IN (SELECT TraceId FROM candidates)
  GROUP BY TraceId
)
SELECT
  t.TraceId AS trace_id,
  coalesce(nullIf(r.endpoint, ''), '') AS endpoint,
  t.SpanAttributes['%s'] AS db_system,
  t.SpanAttributes['%s'] AS statement,
  t.SpanAttributes['%s'] AS db_table,
  t.Duration / 1e6 AS duration_ms
FROM %s AS t
INNER JOIN candidates AS c ON t.TraceId = c.TraceId
LEFT JOIN roots AS r ON t.TraceId = r.TraceId
WHERE %s
LIMIT %d`,
		table(queries.CHTableTraces), where, minDBSpans,
		table(queries.CHTableTraces), rootWhere,
		AttrDBSystem, AttrDBStatement, AttrDBSQLTable,
		table(queries.CHTableTraces), where, spanLimit,
	)
}

// ParseTopDBQueryRows maps CH rows into raw statement samples for aggregation.
func ParseTopDBQueryRows(rows []map[string]any) []DBSpanRow {
	out := make([]DBSpanRow, 0, len(rows))
	for _, r := range rows {
		stmt := ParseString(r["statement"])
		if stmt == "" {
			continue
		}
		out = append(out, DBSpanRow{
			TraceID:    ParseString(r["trace_id"]),
			System:     ParseString(r["db_system"]),
			Statement:  stmt,
			Table:      ParseString(r["db_table"]),
			DurationMs: ParseFloat(r["duration_ms"]),
			Endpoint:   ParseString(r["endpoint"]),
		})
	}
	return out
}

// DBSpanRow is one DB client span sampled from otel_traces.
type DBSpanRow struct {
	TraceID    string
	System     string
	Statement  string
	Table      string
	DurationMs float64
	Endpoint   string
}
