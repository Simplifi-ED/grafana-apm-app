package chquery

import (
	"fmt"
	"strings"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// LogServiceIdentityFilter matches otel_logs rows for an inventory service.
// Filelog pipelines often leave ServiceName empty and stamp k8s.* attrs instead,
// so we also match service.name / k8s.deployment.name / k8s.pod.name prefix.
// scrapeOrBrowser is optional (from jsonData.serviceAliases); when set it is
// tried alongside the inventory name.
func LogServiceIdentityFilter(inventory, scrapeOrBrowser string) string {
	names := make([]string, 0, 2)
	seen := map[string]bool{}
	for _, n := range []string{strings.TrimSpace(inventory), strings.TrimSpace(scrapeOrBrowser)} {
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		names = append(names, n)
	}
	if len(names) == 0 {
		return "1=0"
	}
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = q(n)
	}
	inList := strings.Join(quoted, ", ")
	parts := []string{
		fmt.Sprintf("ServiceName IN (%s)", inList),
		fmt.Sprintf("ResourceAttributes['service.name'] IN (%s)", inList),
		fmt.Sprintf("ResourceAttributes['k8s.deployment.name'] IN (%s)", inList),
	}
	for _, n := range names {
		parts = append(parts, fmt.Sprintf("startsWith(ResourceAttributes['k8s.pod.name'], %s)", q(n+"-")))
	}
	return "(" + strings.Join(parts, " OR ") + ")"
}

// LogSearchSQL returns recent log rows for a service (bounded).
func LogSearchSQL(service string, from, to time.Time, environment, severity string, limit int) string {
	return LogSearchSQLFiltered(service, "", from, to, environment, severity, "", nil, "", limit)
}

// LogSearchSQLFiltered returns recent log rows with body search, field chips, and optional SQL WHERE.
// scrapeOrBrowser is the optional serviceAliases target (e.g. nova-app for nova-api).
func LogSearchSQLFiltered(
	service, scrapeOrBrowser string,
	from, to time.Time,
	environment, severity, search string,
	fields []FieldFilter,
	sqlWhere string,
	limit int,
) string {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	where := fmt.Sprintf(`%s
  AND %s`, TimestampFilter(from, to), LogServiceIdentityFilter(service, scrapeOrBrowser))
	if environment != "" {
		where += fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	if severity != "" {
		orParts := make([]string, 0)
		for level := range strings.SplitSeq(severity, ",") {
			level = strings.ToLower(strings.TrimSpace(level))
			if level == "" {
				continue
			}
			orParts = append(orParts,
				fmt.Sprintf("lower(SeverityText) = %s", q(level)),
				fmt.Sprintf("positionCaseInsensitiveUTF8(Body, %s) > 0", q(`"level":"`+level+`"`)),
				fmt.Sprintf("positionCaseInsensitiveUTF8(Body, %s) > 0", q(`"level": "`+level+`"`)),
			)
		}
		if len(orParts) > 0 {
			where += " AND (" + strings.Join(orParts, " OR ") + ")"
		}
	}
	if search != "" {
		where += fmt.Sprintf(" AND positionCaseInsensitiveUTF8(Body, %s) > 0", q(search))
	}
	if chipSQL := FieldFiltersSQL(fields, "log"); chipSQL != "" {
		where += " AND " + chipSQL
	}
	if sqlWhere != "" {
		where += " AND (" + sqlWhere + ")"
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
	return LogPatternSQLFiltered(service, "", from, to, environment, "", limit)
}

// LogPatternSQLFiltered groups normalized log bodies, optionally restricting
// the result to one or more severity levels.
func LogPatternSQLFiltered(service, scrapeOrBrowser string, from, to time.Time, environment, severity string, limit int) string {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	where := fmt.Sprintf(`%s
  AND %s
  AND Body != ''`, TimestampFilter(from, to), LogServiceIdentityFilter(service, scrapeOrBrowser))
	if environment != "" {
		where += fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	if severity != "" {
		// Filelog rows often leave SeverityText empty and put level in JSON Body.
		orParts := make([]string, 0)
		for level := range strings.SplitSeq(severity, ",") {
			level = strings.ToLower(strings.TrimSpace(level))
			if level == "" {
				continue
			}
			orParts = append(orParts,
				fmt.Sprintf("lower(SeverityText) = %s", q(level)),
				fmt.Sprintf("positionCaseInsensitiveUTF8(Body, %s) > 0", q(`"level":"`+level+`"`)),
				fmt.Sprintf("positionCaseInsensitiveUTF8(Body, %s) > 0", q(`"level": "`+level+`"`)),
			)
		}
		if len(orParts) > 0 {
			where += " AND (" + strings.Join(orParts, " OR ") + ")"
		}
	}
	// Normalize digits to reduce cardinality of otherwise-identical messages.
	return fmt.Sprintf(`
SELECT
  replaceRegexpAll(substring(Body, 1, 200), '[0-9]+', '#') AS pattern,
  count() AS c,
  any(SeverityText) AS severity,
  min(toUnixTimestamp64Milli(Timestamp)) AS first_seen,
  max(toUnixTimestamp64Milli(Timestamp)) AS last_seen
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
	Pattern     string
	Count       int64
	Severity    string
	FirstSeenMs int64
	LastSeenMs  int64
}

// ParseLogPatternMaps parses pattern aggregations.
func ParseLogPatternMaps(rows []map[string]any) []LogPattern {
	out := make([]LogPattern, 0, len(rows))
	for _, r := range rows {
		out = append(out, LogPattern{
			Pattern:     ParseString(r["pattern"]),
			Count:       ParseInt64(r["c"]),
			Severity:    ParseString(r["severity"]),
			FirstSeenMs: ParseInt64(r["first_seen"]),
			LastSeenMs:  ParseInt64(r["last_seen"]),
		})
	}
	return out
}
