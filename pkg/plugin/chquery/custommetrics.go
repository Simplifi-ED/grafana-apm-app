package chquery

import (
	"fmt"
	"strings"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// DenylistMatchPattern anchors the Prom-style CustomMetrics.Denylist so
// MetricName is matched as a full string (research 07).
func DenylistMatchPattern(denylist string) string {
	d := strings.TrimSpace(denylist)
	if d == "" {
		return "^$" // match nothing useful → NOT match keeps all
	}
	// Strip outer anchors if callers already provided them.
	d = strings.TrimPrefix(d, "^")
	d = strings.TrimSuffix(d, "$")
	return "^(" + d + ")$"
}

// CustomMetricsDiscoverSQL lists non-platform MetricNames across gauge/sum/
// histogram/summary for a scrape ServiceName (table kind is authoritative).
func CustomMetricsDiscoverSQL(scrapeService, namespace, denylist string, from, to time.Time, limit int) string {
	if limit < 1 {
		limit = 50
	}
	svc := ScrapeServiceFilter(scrapeService, namespace)
	tf := MetricTimeFilter(from, to)
	deny := q(DenylistMatchPattern(denylist))
	return fmt.Sprintf(`
SELECT MetricName, kind, any(description) AS description, any(unit) AS unit FROM (
  SELECT MetricName, 'gauge' AS kind, any(MetricDescription) AS description, any(MetricUnit) AS unit
  FROM %s
  WHERE NOT match(MetricName, %s) AND %s AND (%s)
  GROUP BY MetricName
  UNION ALL
  SELECT MetricName, 'sum' AS kind, any(MetricDescription) AS description, any(MetricUnit) AS unit
  FROM %s
  WHERE NOT match(MetricName, %s) AND %s AND (%s)
  GROUP BY MetricName
  UNION ALL
  SELECT MetricName, 'histogram' AS kind, any(MetricDescription) AS description, any(MetricUnit) AS unit
  FROM %s
  WHERE NOT match(MetricName, %s) AND %s AND (%s)
  GROUP BY MetricName
  UNION ALL
  SELECT MetricName, 'summary' AS kind, any(MetricDescription) AS description, any(MetricUnit) AS unit
  FROM %s
  WHERE NOT match(MetricName, %s) AND %s AND (%s)
  GROUP BY MetricName
)
ORDER BY MetricName
LIMIT %d`,
		table(queries.CHTableMetricsGauge), deny, tf, svc,
		table(queries.CHTableMetricsSum), deny, tf, svc,
		table(queries.CHTableMetricsHistogram), deny, tf, svc,
		table(queries.CHTableMetricsSummary), deny, tf, svc,
		limit,
	)
}

// CustomMetricSeriesCountSQL returns uniqExact(Attributes) cardinality proxy.
func CustomMetricSeriesCountSQL(scrapeService, namespace, metricName, kind string, from, to time.Time) string {
	svc := ScrapeServiceFilter(scrapeService, namespace)
	var tbl string
	switch kind {
	case "sum", "counter":
		tbl = table(queries.CHTableMetricsSum)
	case "histogram":
		tbl = table(queries.CHTableMetricsHistogram)
	case "summary":
		tbl = table(queries.CHTableMetricsSummary)
	case "gauge":
		tbl = table(queries.CHTableMetricsGauge)
	default:
		tbl = table(queries.CHTableMetricsGauge)
	}
	return fmt.Sprintf(`
SELECT uniqExact(Attributes) AS series
FROM %s
WHERE MetricName = %s
  AND %s
  AND (%s)
LIMIT 1`,
		tbl,
		q(metricName),
		MetricTimeFilter(from, to),
		svc,
	)
}

// CustomMetricFamily is one discovered non-platform metric from CH.
type CustomMetricFamily struct {
	Name        string
	Kind        string // gauge | sum | histogram | summary
	Description string
	Unit        string
	Series      int
}

// ParseCustomMetricFamilies converts discovery rows (no series counts yet).
func ParseCustomMetricFamilies(rows []map[string]any) []CustomMetricFamily {
	out := make([]CustomMetricFamily, 0, len(rows))
	seen := make(map[string]bool, len(rows))
	for _, r := range rows {
		name := ParseString(r["MetricName"])
		if name == "" {
			name = ParseString(r["metricname"])
		}
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		kind := strings.ToLower(ParseString(r["kind"]))
		out = append(out, CustomMetricFamily{
			Name:        name,
			Kind:        kind,
			Description: ParseString(r["description"]),
			Unit:        ParseString(r["unit"]),
		})
	}
	return out
}

// ChartHintForKind maps OTel table kind → UI chart hint (prefer table over name).
func ChartHintForKind(kind string) string {
	switch strings.ToLower(kind) {
	case "sum", "counter":
		return "rate"
	case "histogram":
		return "p95"
	case "summary":
		return "summary"
	default:
		return "gauge"
	}
}

// TypeForKind maps table kind to the Type string Custom Metrics UI expects.
func TypeForKind(kind string) string {
	switch strings.ToLower(kind) {
	case "sum":
		return "counter"
	case "histogram":
		return "histogram"
	case "summary":
		return "summary"
	default:
		return "gauge"
	}
}
