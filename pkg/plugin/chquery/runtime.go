package chquery

import (
	"fmt"
	"strings"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// RuntimeMetricNameMatch is the CH match() pattern for go_*/process_* families
// present in ClickHouse (research 06). JVM/Node/etc. stay omitted until emitted.
const RuntimeMetricNameMatch = "^(go_|process_)"

// ScrapeServiceFilter scopes otel_metrics_* rows to a scrape ServiceName
// (resolved via jsonData.serviceAliases — never the inventory name alone).
func ScrapeServiceFilter(scrapeService, namespace string) string {
	parts := []string{
		fmt.Sprintf("(%s = %s OR ServiceName = %s)",
			ServiceExpr("Attributes"), q(scrapeService), q(scrapeService)),
	}
	if namespace != "" {
		// Scrape metrics use Attributes['namespace'] and/or k8s resource attr.
		parts = append(parts, fmt.Sprintf(
			"(Attributes['namespace'] = %s OR ResourceAttributes['k8s.namespace.name'] = %s OR ResourceAttributes['%s'] = %s)",
			q(namespace), q(namespace), ResServiceNamespace, q(namespace),
		))
	}
	return strings.Join(parts, " AND ")
}

// RuntimeDiscoverSQL lists go_*/process_* MetricNames for a scrape service.
func RuntimeDiscoverSQL(scrapeService, namespace string, from, to time.Time) string {
	svc := ScrapeServiceFilter(scrapeService, namespace)
	tf := MetricTimeFilter(from, to)
	return fmt.Sprintf(`
SELECT MetricName, kind FROM (
  SELECT MetricName, 'gauge' AS kind
  FROM %s
  WHERE match(MetricName, %s) AND %s AND (%s)
  GROUP BY MetricName
  UNION ALL
  SELECT MetricName, 'sum' AS kind
  FROM %s
  WHERE match(MetricName, %s) AND %s AND (%s)
  GROUP BY MetricName
  UNION ALL
  SELECT MetricName, 'summary' AS kind
  FROM %s
  WHERE match(MetricName, %s) AND %s AND (%s)
  GROUP BY MetricName
)
ORDER BY MetricName
LIMIT 200`,
		table(queries.CHTableMetricsGauge), q(RuntimeMetricNameMatch), tf, svc,
		table(queries.CHTableMetricsSum), q(RuntimeMetricNameMatch), tf, svc,
		table(queries.CHTableMetricsSummary), q(RuntimeMetricNameMatch), tf, svc,
	)
}

// ParseRuntimeDiscovered returns the set of discovered MetricNames.
func ParseRuntimeDiscovered(rows []map[string]any) map[string]bool {
	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		if n := ParseString(r["MetricName"]); n != "" {
			out[n] = true
		}
		if n := ParseString(r["metricname"]); n != "" {
			out[n] = true
		}
	}
	return out
}

// RuntimeGaugeAvgsSQL averages selected gauges over the window for a scrape service.
func RuntimeGaugeAvgsSQL(scrapeService, namespace string, metricNames []string, from, to time.Time) string {
	svc := ScrapeServiceFilter(scrapeService, namespace)
	quoted := make([]string, 0, len(metricNames))
	for _, n := range metricNames {
		n = strings.TrimSpace(n)
		if n != "" {
			quoted = append(quoted, q(n))
		}
	}
	if len(quoted) == 0 {
		quoted = append(quoted, q("__none__"))
	}
	return fmt.Sprintf(`
SELECT
  MetricName,
  avg(Value) AS value,
  uniqExact(ResourceAttributes['k8s.pod.name']) AS pods
FROM %s
WHERE MetricName IN (%s)
  AND %s
  AND (%s)
GROUP BY MetricName
LIMIT 50`,
		table(queries.CHTableMetricsGauge),
		strings.Join(quoted, ", "),
		MetricTimeFilter(from, to),
		svc,
	)
}

// RuntimeGoInfoSQL returns go_info version labels for a scrape service.
func RuntimeGoInfoSQL(scrapeService, namespace string, from, to time.Time) string {
	svc := ScrapeServiceFilter(scrapeService, namespace)
	return fmt.Sprintf(`
SELECT
  Attributes['version'] AS version,
  count() AS c
FROM %s
WHERE MetricName = %s
  AND %s
  AND (%s)
GROUP BY version
ORDER BY c DESC
LIMIT 20`,
		table(queries.CHTableMetricsGauge),
		q("go_info"),
		MetricTimeFilter(from, to),
		svc,
	)
}

// RuntimeProcessCPURateSQL approximates avg(rate(process_cpu_seconds_total)) as
// increase over the window divided by window seconds.
func RuntimeProcessCPURateSQL(scrapeService, namespace string, from, to time.Time) string {
	svc := ScrapeServiceFilter(scrapeService, namespace)
	window := WindowSeconds(from, to)
	return fmt.Sprintf(`
SELECT
  (max(Value) - min(Value)) / %g AS cpu_rate
FROM %s
WHERE MetricName = %s
  AND %s
  AND (%s)
LIMIT 1`,
		window,
		table(queries.CHTableMetricsSum),
		q("process_cpu_seconds_total"),
		MetricTimeFilter(from, to),
		svc,
	)
}

// RuntimeGCSummarySQL returns avg pause (Sum/Count) and approximate GC rate
// from go_gc_duration_seconds summary snapshots.
func RuntimeGCSummarySQL(scrapeService, namespace string, from, to time.Time) string {
	svc := ScrapeServiceFilter(scrapeService, namespace)
	window := WindowSeconds(from, to)
	return fmt.Sprintf(`
SELECT
  avg(if(Count > 0, Sum / Count, NULL)) AS gc_pause_avg,
  (max(Count) - min(Count)) / %g AS gc_rate
FROM %s
WHERE MetricName = %s
  AND %s
  AND (%s)
LIMIT 1`,
		window,
		table(queries.CHTableMetricsSummary),
		q("go_gc_duration_seconds"),
		MetricTimeFilter(from, to),
		svc,
	)
}

// ParseGoRuntimeMaps builds a GoRuntime from CH gauge/info/cpu/gc result maps.
func ParseGoRuntimeMaps(
	gaugeRows, infoRows, cpuRows, gcRows []map[string]any,
) *queries.GoRuntime {
	if len(gaugeRows) == 0 && len(infoRows) == 0 && len(cpuRows) == 0 && len(gcRows) == 0 {
		return nil
	}
	goRt := &queries.GoRuntime{Status: queries.StatusDetected}
	byName := make(map[string]float64, len(gaugeRows))
	var maxPods float64
	for _, r := range gaugeRows {
		name := ParseString(r["MetricName"])
		if name == "" {
			name = ParseString(r["metricname"])
		}
		byName[name] = ParseFloat(r["value"])
		if p := ParseFloat(r["pods"]); p > maxPods {
			maxPods = p
		}
	}
	assign := func(metric string, dest *float64) {
		if v, ok := byName[metric]; ok {
			*dest = v
		}
	}
	assign("go_goroutines", &goRt.Goroutines)
	assign("go_threads", &goRt.Threads)
	assign("go_memstats_alloc_bytes", &goRt.MemAlloc)
	assign("go_memstats_sys_bytes", &goRt.MemSys)
	assign("process_open_fds", &goRt.OpenFDs)
	assign("process_max_fds", &goRt.MaxFDs)
	goRt.PodCount = int(maxPods)

	if len(cpuRows) > 0 {
		goRt.CPUUsage = ParseFloat(cpuRows[0]["cpu_rate"])
	}
	if len(gcRows) > 0 {
		goRt.GCPauseAvg = ParseFloat(gcRows[0]["gc_pause_avg"])
		goRt.GCRate = ParseFloat(gcRows[0]["gc_rate"])
	}
	for _, r := range infoRows {
		ver := ParseString(r["version"])
		c := int(ParseFloat(r["c"]))
		if ver == "" || c <= 0 {
			continue
		}
		goRt.Versions = append(goRt.Versions, queries.RuntimeVersion{Version: ver, Count: c})
	}
	// Detectable if any Go signal present.
	if goRt.Goroutines == 0 && goRt.MemAlloc == 0 && goRt.Threads == 0 &&
		len(goRt.Versions) == 0 && goRt.CPUUsage == 0 && goRt.OpenFDs == 0 {
		return nil
	}
	return goRt
}
