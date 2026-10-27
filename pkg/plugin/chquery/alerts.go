package chquery

import (
	"fmt"
	"strings"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// Alert template SQL for Grafana-managed rules evaluated against ClickHouse
// (letter D / tickets 26–27). Uses $__timeFilter so UA evaluation windows apply.

func alertServiceFilter(service, namespace, environment string) string {
	parts := []string{
		fmt.Sprintf("(%s = %s OR ServiceName = %s)", ServiceExpr("Attributes"), q(service), q(service)),
	}
	if namespace != "" {
		parts = append(parts, fmt.Sprintf("ResourceAttributes['%s'] = %s", ResServiceNamespace, q(namespace)))
	}
	if environment != "" {
		parts = append(parts, fmt.Sprintf("ResourceAttributes['%s'] = %s", ResCluster, q(environment)))
	}
	return strings.Join(parts, " AND ")
}

func alertTraceServiceFilter(service, environment string) string {
	parts := []string{fmt.Sprintf("ServiceName = %s", q(service))}
	if environment != "" {
		parts = append(parts, fmt.Sprintf("ResourceAttributes['%s'] = %s", ResCluster, q(environment)))
	}
	return strings.Join(parts, " AND ")
}

// ErrorRateAlertSQL returns the SERVER error ratio over $__timeFilter(TimeUnix).
func ErrorRateAlertSQL(service, namespace, environment string) string {
	filt := alertServiceFilter(service, namespace, environment)
	return fmt.Sprintf(`
SELECT
  if(sum(Value) = 0, 0,
     sumIf(Value, %s) / sum(Value)
  ) AS value
FROM %s
WHERE MetricName = %s
  AND $__timeFilter(%s)
  AND %s
  AND (%s)`,
		MetricStatusErrorFilter(),
		table(queries.CHTableMetricsSum),
		q(MetricSpanCalls),
		ColTimeUnix,
		MetricSpanKindServerFilter(),
		filt,
	)
}

// chInterval maps Prom-style window tokens ("1h", "5m") to ClickHouse INTERVAL clauses.
func chInterval(win string) string {
	switch strings.TrimSpace(win) {
	case "5m":
		return "INTERVAL 5 MINUTE"
	case "30m":
		return "INTERVAL 30 MINUTE"
	case "1h":
		return "INTERVAL 1 HOUR"
	case "6h":
		return "INTERVAL 6 HOUR"
	default:
		return "INTERVAL 1 HOUR"
	}
}

// SLOBurnRateAlertSQL approximates multi-window burn using fixed INTERVAL windows
// (Grafana $__timeFilter alone cannot express multi-window PromQL).
func SLOBurnRateAlertSQL(service, namespace, environment string, budget, factor float64, longWin, shortWin string) string {
	filt := alertServiceFilter(service, namespace, environment)
	ratio := func(win string) string {
		return fmt.Sprintf(`ifNull((
  SELECT if(sum(Value) = 0, 0, sumIf(Value, %s) / sum(Value))
  FROM %s
  WHERE MetricName = %s
    AND %s >= now() - %s
    AND %s
    AND (%s)
), 0)`,
			MetricStatusErrorFilter(),
			table(queries.CHTableMetricsSum),
			q(MetricSpanCalls),
			ColTimeUnix, chInterval(win),
			MetricSpanKindServerFilter(),
			filt,
		)
	}
	return fmt.Sprintf(`
SELECT
  (%s / %g) * if((%s / %g) >= %g, 1, 0) AS value`,
		ratio(longWin), budget, ratio(shortWin), budget, factor,
	)
}

// WebVitalsLCPAlertSQL returns LCP p75 (ms) from HyperDX webvitals spans.
func WebVitalsLCPAlertSQL(service, environment string) string {
	filt := alertTraceServiceFilter(service, environment)
	return fmt.Sprintf(`
SELECT
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['lcp']), SpanAttributes['lcp'] != '') AS value
FROM %s
WHERE $__timeFilter(%s)
  AND SpanName = %s
  AND (%s)`,
		table(queries.CHTableTraces),
		ColTimestamp,
		q(SpanNameWebVitals),
		filt,
	)
}

// ExceptionSpikeAlertSQL counts browser/server exception signals matching hashes
// (Faro hash attribute, or substring of error.message / StatusMessage).
func ExceptionSpikeAlertSQL(service, environment string, hashes []string) string {
	filt := alertTraceServiceFilter(service, environment)
	hashPreds := make([]string, 0, len(hashes)*3)
	for _, h := range hashes {
		hq := q(h)
		hashPreds = append(hashPreds,
			fmt.Sprintf("SpanAttributes['error.hash'] = %s", hq),
			fmt.Sprintf("SpanAttributes['faro.hash'] = %s", hq),
			fmt.Sprintf("positionCaseInsensitive(coalesce(nullIf(SpanAttributes['error.message'], ''), StatusMessage), %s) > 0", hq),
		)
	}
	hashClause := "1"
	if len(hashPreds) > 0 {
		hashClause = "(" + strings.Join(hashPreds, " OR ") + ")"
	}
	return fmt.Sprintf(`
SELECT count() AS value
FROM %s
WHERE $__timeFilter(%s)
  AND (%s)
  AND (
    SpanName = 'unhandledrejection'
    OR startsWith(SpanName, 'error.')
    OR has(Events.Name, 'exception')
  )
  AND %s`,
		table(queries.CHTableTraces),
		ColTimestamp,
		filt,
		hashClause,
	)
}

// NewExceptionsAlertSQL counts exception types seen in the last 30m that were
// absent from the preceding 7 days (stateless approximation, same as Loki path).
func NewExceptionsAlertSQL(service, environment string) string {
	filt := alertTraceServiceFilter(service, environment)
	typeExpr := `coalesce(
  nullIf(arrayElement(Events.Attributes, indexOf(Events.Name, 'exception'))['exception.type'], ''),
  nullIf(SpanAttributes['exception.type'], ''),
  nullIf(SpanAttributes['error.type'], ''),
  SpanName
)`
	return fmt.Sprintf(`
WITH recent AS (
  SELECT DISTINCT %s AS exception_type
  FROM %s
  WHERE %s >= now() - INTERVAL 30 MINUTE
    AND (%s)
    AND (
      has(Events.Name, 'exception')
      OR SpanName = 'unhandledrejection'
      OR startsWith(SpanName, 'error.')
    )
    AND exception_type != ''
),
baseline AS (
  SELECT DISTINCT %s AS exception_type
  FROM %s
  WHERE %s >= now() - INTERVAL 7 DAY - INTERVAL 30 MINUTE
    AND %s < now() - INTERVAL 30 MINUTE
    AND (%s)
    AND (
      has(Events.Name, 'exception')
      OR SpanName = 'unhandledrejection'
      OR startsWith(SpanName, 'error.')
    )
    AND exception_type != ''
)
SELECT count() AS value
FROM recent AS r
LEFT JOIN baseline AS b ON r.exception_type = b.exception_type
WHERE b.exception_type IS NULL`,
		typeExpr, table(queries.CHTableTraces), ColTimestamp, filt,
		typeExpr, table(queries.CHTableTraces), ColTimestamp, ColTimestamp, filt,
	)
}
