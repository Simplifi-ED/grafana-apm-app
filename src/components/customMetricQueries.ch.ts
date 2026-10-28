import { CustomMetric } from '../api/client';
import { sanitizeLabelValue } from '../utils/sanitize';

/**
 * ClickStack (ClickHouse) auto-chart SQL for Overview CustomMetricsPanel
 * (post-parity letter E / ticket 04). Mirrors PromQL shapes in
 * customMetricQueries.ts: rate / p95 / summary / gauge.
 *
 * Scenes run these as grafana-clickhouse-datasource SQL with
 * `$__timeInterval` / `$__timeFilter` / `$__interval_s` macros.
 */

function sqlStr(v: string): string {
  return `'${sanitizeLabelValue(v).replace(/'/g, "''")}'`;
}

/** Scrape ServiceName filter (alias-resolved), optional k8s/product namespace. */
export function customMetricScrapeFilter(scrapeService: string, namespace: string): string {
  let where = `(Attributes['service.name'] = ${sqlStr(scrapeService)} OR ServiceName = ${sqlStr(scrapeService)})`;
  if (namespace) {
    where += ` AND (Attributes['namespace'] = ${sqlStr(namespace)} OR ResourceAttributes['k8s.namespace.name'] = ${sqlStr(namespace)} OR ResourceAttributes['service.namespace'] = ${sqlStr(namespace)})`;
  }
  return where;
}

function grafanaUnit(unit: string): string | undefined {
  switch (unit) {
    case 'seconds':
      return 's';
    case 'milliseconds':
      return 'ms';
    case 'bytes':
      return 'bytes';
    default:
      return undefined;
  }
}

function table(name: string): string {
  return `\`default\`.\`${name}\``;
}

function rateSql(metricName: string, filter: string): string {
  return `
SELECT
  $__timeInterval(TimeUnix) AS time,
  sum(Value) / $__interval_s AS rate
FROM ${table('otel_metrics_sum')}
WHERE MetricName = ${sqlStr(metricName)}
  AND ${filter}
  AND $__timeFilter(TimeUnix)
GROUP BY time
ORDER BY time`.trim();
}

function gaugeSql(metricName: string, filter: string): string {
  return `
SELECT
  $__timeInterval(TimeUnix) AS time,
  avg(Value) AS avg
FROM ${table('otel_metrics_gauge')}
WHERE MetricName = ${sqlStr(metricName)}
  AND ${filter}
  AND $__timeFilter(TimeUnix)
GROUP BY time
ORDER BY time`.trim();
}

function summaryThroughputSql(metricName: string, filter: string): string {
  return `
SELECT
  $__timeInterval(TimeUnix) AS time,
  sum(Count) / $__interval_s AS throughput
FROM ${table('otel_metrics_summary')}
WHERE MetricName = ${sqlStr(metricName)}
  AND ${filter}
  AND $__timeFilter(TimeUnix)
GROUP BY time
ORDER BY time`.trim();
}

function summaryAvgSql(metricName: string, filter: string): string {
  return `
SELECT
  $__timeInterval(TimeUnix) AS time,
  sum(Sum) / nullIf(sum(Count), 0) AS avg
FROM ${table('otel_metrics_summary')}
WHERE MetricName = ${sqlStr(metricName)}
  AND ${filter}
  AND $__timeFilter(TimeUnix)
GROUP BY time
ORDER BY time`.trim();
}

/**
 * Histogram p95 via Prometheus-compatible cumulative buckets (same approach as
 * chquery spanmetrics duration), bucketed by $__timeInterval.
 */
function histogramP95Sql(metricName: string, filter: string): string {
  return `
WITH per_series AS (
  SELECT
    $__timeInterval(TimeUnix) AS time,
    Attributes,
    argMax(BucketCounts, TimeUnix) AS bc,
    anyLast(ExplicitBounds) AS bounds
  FROM ${table('otel_metrics_histogram')}
  WHERE MetricName = ${sqlStr(metricName)}
    AND ${filter}
    AND $__timeFilter(TimeUnix)
  GROUP BY time, Attributes
),
merged AS (
  SELECT
    time,
    any(bounds) AS bounds,
    sumForEach(arrayMap(x -> toFloat64(x), bc)) AS bc_sum
  FROM per_series
  GROUP BY time
),
exploded AS (
  SELECT
    time,
    arrayJoin(arrayZip(
      arrayConcat(bounds, [CAST('+Inf' AS Float64)]),
      arrayCumSum(bc_sum)
    )) AS pair
  FROM merged
)
SELECT
  time,
  quantilePrometheusHistogram(0.95)(pair.1, pair.2) AS p95
FROM exploded
GROUP BY time
ORDER BY time`.trim();
}

export interface CustomMetricCHPanelSpec {
  title: string;
  unit?: string;
  rawSql: string;
  legend: string;
}

/**
 * Build ClickHouse auto-chart panel spec(s) for one discovered family.
 * Summaries yield throughput + avg (two panels); others one panel.
 */
export function customMetricPanelsCH(
  metric: CustomMetric,
  scrapeService: string,
  namespace: string
): CustomMetricCHPanelSpec[] {
  const filter = customMetricScrapeFilter(scrapeService, namespace);
  const n = metric.name;
  const unit = grafanaUnit(metric.unit);

  switch (metric.chart) {
    case 'rate':
      return [{ title: n, rawSql: rateSql(n, filter), legend: 'rate' }];
    case 'p95':
      return [{ title: `${n} · p95`, unit, rawSql: histogramP95Sql(n, filter), legend: 'p95' }];
    case 'summary':
      return [
        { title: `${n} · throughput`, rawSql: summaryThroughputSql(n, filter), legend: 'throughput' },
        { title: `${n} · avg`, unit, rawSql: summaryAvgSql(n, filter), legend: 'avg' },
      ];
    case 'gauge':
    default:
      return [{ title: n, unit, rawSql: gaugeSql(n, filter), legend: 'avg' }];
  }
}

/** Primary CH SQL for Explore deep-link (first panel). */
export function customMetricExploreSQL(metric: CustomMetric, scrapeService: string, namespace: string): string {
  return customMetricPanelsCH(metric, scrapeService, namespace)[0].rawSql;
}
