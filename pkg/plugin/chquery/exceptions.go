package chquery

import (
	"fmt"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// ExceptionGroupsSQL groups error/exception log lines for a service.
// Uses OTel log exception attributes when present; otherwise groups by
// normalized Body. This is NOT the Faro browser exception pipeline.
func ExceptionGroupsSQL(service string, from, to time.Time, environment string, limit int) string {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	where := fmt.Sprintf(`%s
  AND ServiceName = %s
  AND (
    LogAttributes['exception.type'] != ''
    OR lower(SeverityText) IN ('error', 'fatal', 'critical')
  )`, TimestampFilter(from, to), q(service))
	if environment != "" {
		where += fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	return fmt.Sprintf(`
SELECT
  if(LogAttributes['exception.type'] != '',
     LogAttributes['exception.type'],
     replaceRegexpAll(substring(Body, 1, 120), '[0-9]+', '#')) AS fingerprint,
  any(if(LogAttributes['exception.message'] != '', LogAttributes['exception.message'], substring(Body, 1, 200))) AS message,
  any(LogAttributes['exception.type']) AS exception_type,
  count() AS occurrences,
  uniqExact(TraceId) AS traces,
  max(Timestamp) AS last_seen
FROM %s
WHERE %s
GROUP BY fingerprint
ORDER BY occurrences DESC
LIMIT %d`, table(queries.CHTableLogs), where, limit)
}

// ExceptionGroup is one grouped error.
type ExceptionGroup struct {
	Fingerprint   string
	Message       string
	ExceptionType string
	Occurrences   int64
	Traces        int64
}

// ParseExceptionGroupsMaps parses exception group rows.
func ParseExceptionGroupsMaps(rows []map[string]any) []ExceptionGroup {
	out := make([]ExceptionGroup, 0, len(rows))
	for _, r := range rows {
		out = append(out, ExceptionGroup{
			Fingerprint:   ParseString(r["fingerprint"]),
			Message:       ParseString(r["message"]),
			ExceptionType: ParseString(r["exception_type"]),
			Occurrences:   ParseInt64(r["occurrences"]),
			Traces:        ParseInt64(r["traces"]),
		})
	}
	return out
}

// ErrorSpansSQL counts error spans per service (supplement when logs lack exceptions).
func ErrorSpansSQL(service string, from, to time.Time, environment string) string {
	where := fmt.Sprintf(`%s
  AND ServiceName = %s
  AND %s`, TimestampFilter(from, to), q(service), TraceStatusErrorFilter())
	if environment != "" {
		where += fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	return fmt.Sprintf(`
SELECT
  SpanName AS span_name,
  count() AS errors,
  any(StatusMessage) AS status_message,
  max(Timestamp) AS last_seen
FROM %s
WHERE %s
GROUP BY span_name
ORDER BY errors DESC
LIMIT 50`, table(queries.CHTableTraces), where)
}
