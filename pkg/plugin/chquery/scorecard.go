package chquery

import (
	"fmt"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// Scorecard presence probes (letter D / ticket 22). Each returns count() AS c.

// SpanMetricsProbeSQL is true when traces.spanmetrics.calls exists for the inventory service.
func SpanMetricsProbeSQL(service, environment string, from, to time.Time) string {
	filt := FilterClause("", environment, service, "ResourceAttributes")
	return fmt.Sprintf(`
SELECT count() AS c
FROM %s
WHERE MetricName = %s
  AND %s
  AND (%s)
LIMIT 1`,
		table(queries.CHTableMetricsSum),
		q(MetricSpanCalls),
		MetricTimeFilter(from, to),
		filt,
	)
}

// TracesProbeSQL is true when otel_traces has any row for the service.
func TracesProbeSQL(service, environment string, from, to time.Time) string {
	where := fmt.Sprintf(`%s AND ServiceName = %s`, TimestampFilter(from, to), q(service))
	if environment != "" {
		where += fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	return fmt.Sprintf(`
SELECT count() AS c
FROM %s
WHERE %s
LIMIT 1`, table(queries.CHTableTraces), where)
}

// LogsProbeSQL is true when otel_logs has any row for the service (incl. k8s identity).
// scrapeOrBrowser may be empty; when set (serviceAliases) it broadens the match.
func LogsProbeSQL(service, scrapeOrBrowser, environment string, from, to time.Time) string {
	where := fmt.Sprintf(`%s AND %s`, TimestampFilter(from, to), LogServiceIdentityFilter(service, scrapeOrBrowser))
	if environment != "" {
		where += fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	return fmt.Sprintf(`
SELECT count() AS c
FROM %s
WHERE %s
LIMIT 1`, table(queries.CHTableLogs), where)
}

// RuntimeMetricsProbeSQL checks go_*/process_* for the aliased scrape ServiceName.
func RuntimeMetricsProbeSQL(scrapeService, namespace string, from, to time.Time) string {
	svc := ScrapeServiceFilter(scrapeService, namespace)
	tf := MetricTimeFilter(from, to)
	return fmt.Sprintf(`
SELECT count() AS c FROM (
  SELECT 1
  FROM %s
  WHERE match(MetricName, %s) AND %s AND (%s)
  LIMIT 1
  UNION ALL
  SELECT 1
  FROM %s
  WHERE match(MetricName, %s) AND %s AND (%s)
  LIMIT 1
)
LIMIT 1`,
		table(queries.CHTableMetricsGauge), q(RuntimeMetricNameMatch), tf, svc,
		table(queries.CHTableMetricsSum), q(RuntimeMetricNameMatch), tf, svc,
	)
}

// BrowserScorecardProbeSQL is true when HyperDX sessions ∨ webvitals ∨ browser Issues
// exist for the aliased browser ServiceName (ticket 22).
func BrowserScorecardProbeSQL(browserService, environment string, from, to time.Time) string {
	env := "1"
	if environment != "" {
		env = fmt.Sprintf("ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	tf := TimestampFilter(from, to)
	svc := q(browserService)
	return fmt.Sprintf(`
SELECT count() AS c FROM (
  SELECT 1
  FROM %s
  WHERE %s AND ServiceName = %s AND ResourceAttributes['rum.sessionId'] != '' AND (%s)
  LIMIT 1
  UNION ALL
  SELECT 1
  FROM %s
  WHERE %s AND ServiceName = %s AND SpanName = %s AND (%s)
  LIMIT 1
  UNION ALL
  SELECT 1
  FROM %s
  WHERE %s AND ServiceName = %s
    AND (SpanName = 'unhandledrejection' OR startsWith(SpanName, 'error.'))
    AND (%s)
  LIMIT 1
)
LIMIT 1`,
		table(queries.CHTableSessions), tf, svc, env,
		table(queries.CHTableTraces), tf, svc, q(SpanNameWebVitals), env,
		table(queries.CHTableTraces), tf, svc, env,
	)
}

// SDKLanguageProbeSQL returns distinct telemetry.sdk.language values for a service.
func SDKLanguageProbeSQL(service, environment string, from, to time.Time) string {
	filt := FilterClause("", environment, service, "ResourceAttributes")
	return fmt.Sprintf(`
SELECT ResourceAttributes['%s'] AS lang
FROM %s
WHERE MetricName = %s
  AND %s
  AND (%s)
  AND ResourceAttributes['%s'] != ''
GROUP BY lang
LIMIT 20`,
		ResSDKLanguage,
		table(queries.CHTableMetricsSum),
		q(MetricSpanCalls),
		MetricTimeFilter(from, to),
		filt,
		ResSDKLanguage,
	)
}
