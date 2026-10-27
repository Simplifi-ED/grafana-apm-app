import {
  SceneFlexLayout,
  SceneFlexItem,
  SceneQueryRunner,
  SceneTimeRange,
  PanelBuilders,
  EmbeddedScene,
  behaviors,
} from '@grafana/scenes';
import { DashboardCursorSync, TooltipDisplayMode } from '@grafana/schema';
import { HeatmapColorMode } from '@grafana/schema/dist/esm/raw/composable/heatmap/panelcfg/x/HeatmapPanelCfg_types.gen';
import { sanitizeLabelValue } from '../utils/sanitize';
import { buildDeployAnnotationsLayer } from './buildServiceScene';

export interface BuildClickStackServiceSceneParams {
  service: string;
  namespace: string;
  envFilter: string;
  percentile: string;
  percentileLabel: string;
  from: string;
  to: string;
  clickHouseUid: string;
  hasServerSpans: boolean;
  /** Heatmap y-axis unit; ClickHouse Duration buckets are ms. */
  durationUnit?: string;
}

const CH_DS_TYPE = 'grafana-clickhouse-datasource';

/** Escape a value for a ClickHouse string literal. */
function sqlStr(value: string): string {
  return `'${sanitizeLabelValue(value).replace(/'/g, "''")}'`;
}

function serviceFilterSql(service: string, environment: string): string {
  let where = `(Attributes['service.name'] = ${sqlStr(service)} OR ServiceName = ${sqlStr(service)})`;
  if (environment) {
    where += ` AND ResourceAttributes['cluster'] = ${sqlStr(environment)}`;
  }
  return where;
}

function tracesServiceFilterSql(service: string, environment: string): string {
  let where = `ServiceName = ${sqlStr(service)}`;
  if (environment) {
    where += ` AND ResourceAttributes['cluster'] = ${sqlStr(environment)}`;
  }
  return where;
}

function chQuery(uid: string, rawSql: string) {
  return new SceneQueryRunner({
    datasource: { uid, type: CH_DS_TYPE },
    queries: [
      {
        refId: 'A',
        rawSql,
        format: 0, // Time series
        queryType: 'sql',
      } as never,
    ],
  });
}

/**
 * Overview RED panels backed by ClickStack / ClickHouse spanmetrics + traces.
 * Used when no Prometheus metrics datasource is configured.
 */
export function buildClickStackServiceScene(params: BuildClickStackServiceSceneParams): EmbeddedScene | null {
  const {
    service,
    envFilter,
    percentile,
    percentileLabel,
    from,
    to,
    clickHouseUid,
    hasServerSpans,
    durationUnit = 'ms',
  } = params;

  if (!clickHouseUid || !service) {
    return null;
  }

  const metricFilt = serviceFilterSql(service, envFilter);
  const spanKind = hasServerSpans ? ` AND Attributes['span.kind'] = 'SPAN_KIND_SERVER'` : '';
  // otel_traces uses short labels (Server); some pipelines emit SPAN_KIND_SERVER.
  const traceKind = hasServerSpans ? ` AND SpanKind IN ('Server', 'SPAN_KIND_SERVER')` : '';
  const traceFilt = tracesServiceFilterSql(service, envFilter);

  const q = Number.parseFloat(percentile);
  const quantile = Number.isFinite(q) && q > 0 && q < 1 ? q : 0.95;

  const rateSql = `
SELECT
  $__timeInterval(TimeUnix) AS time,
  sum(Value) / $__interval_s AS rate
FROM \`default\`.\`otel_metrics_sum\`
WHERE MetricName = 'traces.spanmetrics.calls'
  AND ${metricFilt}${spanKind}
  AND $__timeFilter(TimeUnix)
GROUP BY time
ORDER BY time`.trim();

  const errorSql = `
SELECT
  $__timeInterval(TimeUnix) AS time,
  100 * sumIf(Value, Attributes['status.code'] = 'STATUS_CODE_ERROR') / nullIf(sum(Value), 0) AS error_pct
FROM \`default\`.\`otel_metrics_sum\`
WHERE MetricName = 'traces.spanmetrics.calls'
  AND ${metricFilt}${spanKind}
  AND $__timeFilter(TimeUnix)
GROUP BY time
ORDER BY time`.trim();

  const durationSql = `
SELECT
  $__timeInterval(Timestamp) AS time,
  quantile(${quantile})(Duration) / 1e6 AS duration_ms
FROM \`default\`.\`otel_traces\`
WHERE ${traceFilt}${traceKind}
  AND $__timeFilter(Timestamp)
GROUP BY time
ORDER BY time`.trim();

  const heatmapSql = `
SELECT
  $__timeInterval(Timestamp) AS time,
  toString(multiIf(
    Duration < 1000000, 1,
    Duration < 2000000, 2,
    Duration < 5000000, 5,
    Duration < 10000000, 10,
    Duration < 25000000, 25,
    Duration < 50000000, 50,
    Duration < 100000000, 100,
    Duration < 250000000, 250,
    Duration < 500000000, 500,
    Duration < 1000000000, 1000,
    Duration < 2500000000, 2500,
    Duration < 5000000000, 5000,
    Duration < 10000000000, 10000,
    Duration < 30000000000, 30000,
    Duration < 60000000000, 60000,
    120000
  )) AS le,
  count() AS value
FROM \`default\`.\`otel_traces\`
WHERE ${traceFilt}${traceKind}
  AND $__timeFilter(Timestamp)
GROUP BY time, le
ORDER BY time, toFloat64(le)`.trim();

  const panelScope = hasServerSpans ? 'inbound (SERVER)' : 'all';

  const timeRange = new SceneTimeRange({ from, to });

  return new EmbeddedScene({
    $timeRange: timeRange,
    $data: buildDeployAnnotationsLayer(service, envFilter || undefined),
    $behaviors: [new behaviors.CursorSync({ sync: DashboardCursorSync.Crosshair })],
    controls: [],
    body: new SceneFlexLayout({
      direction: 'column',
      children: [
        new SceneFlexItem({
          body: new SceneFlexLayout({
            direction: 'row',
            children: [
              new SceneFlexItem({
                height: 300,
                body: PanelBuilders.timeseries()
                  .setTitle('Duration')
                  .setDescription(`${percentileLabel} response time for ${panelScope} requests (ClickHouse traces)`)
                  .setData(chQuery(clickHouseUid, durationSql))
                  .setUnit('ms')
                  .build(),
              }),
              new SceneFlexItem({
                height: 300,
                body: PanelBuilders.timeseries()
                  .setTitle('Errors')
                  .setDescription(`Percentage of ${panelScope} requests with error status (ClickHouse spanmetrics)`)
                  .setData(chQuery(clickHouseUid, errorSql))
                  .setUnit('percent')
                  .build(),
              }),
              new SceneFlexItem({
                height: 300,
                body: PanelBuilders.timeseries()
                  .setTitle('Rate')
                  .setDescription(`Throughput of ${panelScope} requests per second (ClickHouse spanmetrics)`)
                  .setData(chQuery(clickHouseUid, rateSql))
                  .setUnit('reqps')
                  .build(),
              }),
            ],
          }),
        }),
        new SceneFlexItem({
          height: 220,
          body: PanelBuilders.heatmap()
            .setTitle('Duration Distribution')
            .setDescription(`Distribution of ${panelScope} request durations over time (ClickHouse traces)`)
            .setData(chQuery(clickHouseUid, heatmapSql))
            .setOption('calculate', false)
            .setOption('yAxis', { unit: durationUnit === 's' ? 's' : 'ms' })
            .setOption('color', {
              mode: HeatmapColorMode.Scheme,
              scheme: 'Oranges',
              steps: 128,
            })
            .setOption('cellGap', 1)
            .setOption('tooltip', { mode: TooltipDisplayMode.Single, yHistogram: true })
            .build(),
        }),
      ],
    }),
  });
}
