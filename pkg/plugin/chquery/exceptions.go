package chquery

import (
	"fmt"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// HyperDX / @hyperdx/otel-web browser error span selector (ticket 20).
const (
	SpanNameUnhandledRejection = "unhandledrejection"
	AttrComponent              = "component"
	AttrComponentError         = "error"
	AttrErrorObject            = "error.object"
	AttrErrorMessage           = "error.message"
	AttrErrorStack             = "error.stack"
	AttrLocationHref           = "location.href"
	AttrUserID                 = "userId"
	AttrUserEmail              = "userEmail"
	EventNameException         = "exception"
	EventAttrExceptionType     = "exception.type"
	EventAttrExceptionMessage  = "exception.message"
	ResRumSessionID            = "rum.sessionId"
)

// BrowserHyperDXErrorFilter matches HyperDX browser error spans in otel_traces.
func BrowserHyperDXErrorFilter() string {
	return fmt.Sprintf(
		`(SpanName = %s OR SpanAttributes[%s] = %s OR has(Events.Name, %s))`,
		q(SpanNameUnhandledRejection),
		q(AttrComponent),
		q(AttrComponentError),
		q(EventNameException),
	)
}

// ServerExceptionEventFilter matches server-side exception events, excluding
// HyperDX browser error shapes so the same span is never dual-badged.
func ServerExceptionEventFilter() string {
	return fmt.Sprintf(
		`has(Events.Name, %s) AND NOT (SpanName = %s OR SpanAttributes[%s] = %s)`,
		q(EventNameException),
		q(SpanNameUnhandledRejection),
		q(AttrComponent),
		q(AttrComponentError),
	)
}

// exceptionTypeExpr prefers SpanAttributes error.object, else Events exception.type.
func exceptionTypeExpr() string {
	return fmt.Sprintf(`if(SpanAttributes[%s] != '', SpanAttributes[%s],
    if(indexOf(Events.Name, %s) > 0,
       arrayElement(Events.Attributes, indexOf(Events.Name, %s))[%s],
       ''))`,
		q(AttrErrorObject), q(AttrErrorObject),
		q(EventNameException), q(EventNameException), q(EventAttrExceptionType))
}

// exceptionMessageExpr prefers SpanAttributes error.message, else Events exception.message.
func exceptionMessageExpr() string {
	return fmt.Sprintf(`if(SpanAttributes[%s] != '', SpanAttributes[%s],
    if(indexOf(Events.Name, %s) > 0,
       arrayElement(Events.Attributes, indexOf(Events.Name, %s))[%s],
       ''))`,
		q(AttrErrorMessage), q(AttrErrorMessage),
		q(EventNameException), q(EventNameException), q(EventAttrExceptionMessage))
}

func tracesEnvFilter(environment string) string {
	if environment == "" {
		return ""
	}
	return fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResCluster, q(environment))
}

func clampIssueLimit(limit int) int {
	if limit <= 0 || limit > 100 {
		return 50
	}
	return limit
}

// BrowserExceptionGroupsSQL groups HyperDX browser errors by type+message.
// service must already be the resolved browser ServiceName (via serviceAliases).
func BrowserExceptionGroupsSQL(service string, from, to time.Time, environment string, limit int) string {
	limit = clampIssueLimit(limit)
	where := fmt.Sprintf(`%s
  AND ServiceName = %s
  AND %s%s`, TimestampFilter(from, to), q(service), BrowserHyperDXErrorFilter(), tracesEnvFilter(environment))
	return fmt.Sprintf(`
SELECT
  %s AS exception_type,
  %s AS message,
  count() AS occurrences,
  uniqExactIf(ResourceAttributes[%s], ResourceAttributes[%s] != '') AS sessions,
  max(Timestamp) AS last_seen
FROM %s
WHERE %s
GROUP BY exception_type, message
HAVING exception_type != '' OR message != ''
ORDER BY occurrences DESC
LIMIT %d`,
		exceptionTypeExpr(), exceptionMessageExpr(),
		q(ResRumSessionID), q(ResRumSessionID),
		table(queries.CHTableTraces), where, limit)
}

// BrowserIssuesProbeSQL returns count()>0 when HyperDX browser exception spans exist.
func BrowserIssuesProbeSQL(service, environment string, from, to time.Time) string {
	where := fmt.Sprintf(`%s
  AND ServiceName = %s
  AND %s%s`, TimestampFilter(from, to), q(service), BrowserHyperDXErrorFilter(), tracesEnvFilter(environment))
	return fmt.Sprintf(`
SELECT count() AS c
FROM %s
WHERE %s
LIMIT 1`, table(queries.CHTableTraces), where)
}

// BrowserExceptionOccurrencesSQL returns raw HyperDX error spans for the drawer.
// Optional exType narrows the scan; fingerprint matching still happens Go-side.
func BrowserExceptionOccurrencesSQL(service string, from, to time.Time, environment, exType string, limit int) string {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	where := fmt.Sprintf(`%s
  AND ServiceName = %s
  AND %s%s`, TimestampFilter(from, to), q(service), BrowserHyperDXErrorFilter(), tracesEnvFilter(environment))
	if exType != "" {
		where += fmt.Sprintf(" AND (%s) = %s", exceptionTypeExpr(), q(exType))
	}
	return fmt.Sprintf(`
SELECT
  toUnixTimestamp64Milli(Timestamp) AS time_ms,
  %s AS exception_type,
  %s AS message,
  SpanAttributes[%s] AS stacktrace,
  ResourceAttributes[%s] AS session_id,
  SpanAttributes[%s] AS user_id,
  SpanAttributes[%s] AS user_email,
  SpanAttributes[%s] AS page_url,
  TraceId AS trace_id,
  SpanId AS span_id
FROM %s
WHERE %s
ORDER BY Timestamp DESC
LIMIT %d`,
		exceptionTypeExpr(), exceptionMessageExpr(),
		q(AttrErrorStack), q(ResRumSessionID),
		q(AttrUserID), q(AttrUserEmail), q(AttrLocationHref),
		table(queries.CHTableTraces), where, limit)
}

// ServerExceptionGroupsSQL groups server otel_traces exception events by type+message.
// service is the inventory / spanmetrics ServiceName (not the browser alias).
func ServerExceptionGroupsSQL(service string, from, to time.Time, environment string, limit int) string {
	limit = clampIssueLimit(limit)
	where := fmt.Sprintf(`%s
  AND ServiceName = %s
  AND %s%s`, TimestampFilter(from, to), q(service), ServerExceptionEventFilter(), tracesEnvFilter(environment))
	return fmt.Sprintf(`
SELECT
  %s AS exception_type,
  %s AS message,
  count() AS occurrences,
  max(Timestamp) AS last_seen
FROM %s
WHERE %s
GROUP BY exception_type, message
HAVING exception_type != '' OR message != ''
ORDER BY occurrences DESC
LIMIT %d`,
		exceptionTypeExpr(), exceptionMessageExpr(),
		table(queries.CHTableTraces), where, limit)
}

// ServerExceptionOccurrencesSQL returns raw server exception-event spans.
func ServerExceptionOccurrencesSQL(service string, from, to time.Time, environment, exType string, limit int) string {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	where := fmt.Sprintf(`%s
  AND ServiceName = %s
  AND %s%s`, TimestampFilter(from, to), q(service), ServerExceptionEventFilter(), tracesEnvFilter(environment))
	if exType != "" {
		where += fmt.Sprintf(" AND (%s) = %s", exceptionTypeExpr(), q(exType))
	}
	return fmt.Sprintf(`
SELECT
  toUnixTimestamp64Milli(Timestamp) AS time_ms,
  %s AS exception_type,
  %s AS message,
  '' AS stacktrace,
  ResourceAttributes['k8s.pod.name'] AS pod,
  TraceId AS trace_id,
  SpanId AS span_id
FROM %s
WHERE %s
ORDER BY Timestamp DESC
LIMIT %d`,
		exceptionTypeExpr(), exceptionMessageExpr(),
		table(queries.CHTableTraces), where, limit)
}

// ExceptionGroupsSQL groups error/exception log lines for a service.
// Uses OTel log exception attributes when present; otherwise groups by
// normalized Body. This is NOT the Faro browser exception pipeline.
func ExceptionGroupsSQL(service string, from, to time.Time, environment string, limit int) string {
	limit = clampIssueLimit(limit)
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

// ExceptionGroup is one grouped error (log-path or CH span path).
type ExceptionGroup struct {
	Fingerprint   string
	Message       string
	ExceptionType string
	Occurrences   int64
	Traces        int64
	Sessions      int64
}

// ParseExceptionGroupsMaps parses exception group rows (log path).
func ParseExceptionGroupsMaps(rows []map[string]any) []ExceptionGroup {
	out := make([]ExceptionGroup, 0, len(rows))
	for _, r := range rows {
		out = append(out, ExceptionGroup{
			Fingerprint:   ParseString(r["fingerprint"]),
			Message:       ParseString(r["message"]),
			ExceptionType: ParseString(r["exception_type"]),
			Occurrences:   ParseInt64(r["occurrences"]),
			Traces:        ParseInt64(r["traces"]),
			Sessions:      ParseInt64(r["sessions"]),
		})
	}
	return out
}

// TraceExceptionGroup is one HyperDX / Events.exception group row.
type TraceExceptionGroup struct {
	ExceptionType string
	Message       string
	Occurrences   int64
	Sessions      int64
}

// ParseTraceExceptionGroupsMaps parses browser/server exception group rows.
func ParseTraceExceptionGroupsMaps(rows []map[string]any) []TraceExceptionGroup {
	out := make([]TraceExceptionGroup, 0, len(rows))
	for _, r := range rows {
		out = append(out, TraceExceptionGroup{
			ExceptionType: ParseString(r["exception_type"]),
			Message:       ParseString(r["message"]),
			Occurrences:   ParseInt64(r["occurrences"]),
			Sessions:      ParseInt64(r["sessions"]),
		})
	}
	return out
}

// TraceExceptionOccurrence is one raw exception span for the drawer.
type TraceExceptionOccurrence struct {
	TimeMs        int64
	ExceptionType string
	Message       string
	Stacktrace    string
	SessionID     string
	UserID        string
	UserEmail     string
	PageURL       string
	Pod           string
	TraceID       string
	SpanID        string
}

// ParseTraceExceptionOccurrencesMaps parses occurrence rows from CH.
func ParseTraceExceptionOccurrencesMaps(rows []map[string]any) []TraceExceptionOccurrence {
	out := make([]TraceExceptionOccurrence, 0, len(rows))
	for _, r := range rows {
		out = append(out, TraceExceptionOccurrence{
			TimeMs:        ParseInt64(r["time_ms"]),
			ExceptionType: ParseString(r["exception_type"]),
			Message:       ParseString(r["message"]),
			Stacktrace:    ParseString(r["stacktrace"]),
			SessionID:     ParseString(r["session_id"]),
			UserID:        ParseString(r["user_id"]),
			UserEmail:     ParseString(r["user_email"]),
			PageURL:       ParseString(r["page_url"]),
			Pod:           ParseString(r["pod"]),
			TraceID:       ParseString(r["trace_id"]),
			SpanID:        ParseString(r["span_id"]),
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

// SessionErrorEnrichmentSQL counts HyperDX browser errors per rum.sessionId
// and harvests user identity for Sessions UX polish.
func SessionErrorEnrichmentSQL(service string, from, to time.Time, environment string) string {
	where := fmt.Sprintf(`%s
  AND ServiceName = %s
  AND ResourceAttributes[%s] != ''
  AND %s%s`,
		TimestampFilter(from, to), q(service), q(ResRumSessionID),
		BrowserHyperDXErrorFilter(), tracesEnvFilter(environment))
	return fmt.Sprintf(`
SELECT
  ResourceAttributes[%s] AS session_id,
  count() AS errors,
  any(SpanAttributes[%s]) AS user_id,
  any(SpanAttributes[%s]) AS user_email
FROM %s
WHERE %s
GROUP BY session_id`,
		q(ResRumSessionID), q(AttrUserID), q(AttrUserEmail),
		table(queries.CHTableTraces), where)
}
