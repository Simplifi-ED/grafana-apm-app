package chquery

import (
	"fmt"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// Faro-shaped feedback contract (ticket 21) — HyperDX addAction('faro.feedback', …).
const (
	SpanNameFaroFeedback     = "faro.feedback"
	AttrFeedbackMessage      = "feedback.message"
	AttrFeedbackCategory     = "feedback.category"
	AttrFeedbackEmail        = "feedback.email"
	AttrFeedbackFingerprint  = "feedback.fingerprint"
	AttrEventName            = "event_name"
	maxFeedbackRows          = 200
)

// BrowserFeedbackProbeSQL is true when SpanName=faro.feedback exists for the
// aliased browser ServiceName. Does not match product HTTP /api/feedback spans.
func BrowserFeedbackProbeSQL(service, environment string, from, to time.Time) string {
	where := fmt.Sprintf(`%s
  AND ServiceName = %s
  AND %s%s`,
		TimestampFilter(from, to), q(service), FaroFeedbackFilter(), tracesEnvFilter(environment))
	return fmt.Sprintf(`
SELECT count() AS c
FROM %s
WHERE %s
LIMIT 1`, table(queries.CHTableTraces), where)
}

// FaroFeedbackFilter matches Faro-shaped feedback action spans (ticket 21).
func FaroFeedbackFilter() string {
	return fmt.Sprintf(
		`(SpanName = %s OR SpanAttributes[%s] = %s)`,
		q(SpanNameFaroFeedback), q(AttrEventName), q(SpanNameFaroFeedback),
	)
}

// BrowserFeedbackSQL returns faro.feedback rows newest-first.
// service must already be the resolved browser ServiceName (via serviceAliases).
// Optional sessionID / fingerprint narrow the result (sanitized by the caller).
func BrowserFeedbackSQL(service string, from, to time.Time, environment, sessionID, fingerprint string, limit int) string {
	if limit <= 0 || limit > maxFeedbackRows {
		limit = maxFeedbackRows
	}
	where := fmt.Sprintf(`%s
  AND ServiceName = %s
  AND %s%s`,
		TimestampFilter(from, to), q(service), FaroFeedbackFilter(), tracesEnvFilter(environment))
	if sessionID != "" {
		where += fmt.Sprintf(" AND ResourceAttributes[%s] = %s", q(ResRumSessionID), q(sessionID))
	}
	if fingerprint != "" {
		where += fmt.Sprintf(" AND SpanAttributes[%s] = %s", q(AttrFeedbackFingerprint), q(fingerprint))
	}
	return fmt.Sprintf(`
SELECT
  toUnixTimestamp64Milli(Timestamp) AS time_ms,
  SpanAttributes[%s] AS message,
  SpanAttributes[%s] AS category,
  SpanAttributes[%s] AS email,
  ResourceAttributes[%s] AS session_id,
  SpanAttributes[%s] AS fingerprint,
  SpanAttributes[%s] AS page_url,
  ResourceAttributes[%s] AS app_version
FROM %s
WHERE %s
ORDER BY Timestamp DESC
LIMIT %d`,
		q(AttrFeedbackMessage), q(AttrFeedbackCategory), q(AttrFeedbackEmail),
		q(ResRumSessionID), q(AttrFeedbackFingerprint), q(AttrLocationHref),
		q(ResServiceVersion),
		table(queries.CHTableTraces), where, limit)
}

// FeedbackRow is one faro.feedback span from BrowserFeedbackSQL.
type FeedbackRow struct {
	TimeMs      int64
	Message     string
	Category    string
	Email       string
	SessionID   string
	Fingerprint string
	PageURL     string
	AppVersion  string
}

// ParseFeedbackRowsMaps parses BrowserFeedbackSQL rows.
func ParseFeedbackRowsMaps(rows []map[string]any) []FeedbackRow {
	out := make([]FeedbackRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, FeedbackRow{
			TimeMs:      ParseInt64(r["time_ms"]),
			Message:     ParseString(r["message"]),
			Category:    ParseString(r["category"]),
			Email:       ParseString(r["email"]),
			SessionID:   ParseString(r["session_id"]),
			Fingerprint: ParseString(r["fingerprint"]),
			PageURL:     ParseString(r["page_url"]),
			AppVersion:  ParseString(r["app_version"]),
		})
	}
	return out
}
