import {
  EmbeddedScene,
  PanelBuilders,
  SceneFlexItem,
  SceneFlexLayout,
  SceneQueryRunner,
  SceneTimeRange,
} from '@grafana/scenes';
import { sanitizeLabelValue } from '../utils/sanitize';

const CH_DS_TYPE = 'grafana-clickhouse-datasource';

function sqlStr(value: string): string {
  return `'${sanitizeLabelValue(value).replace(/'/g, "''")}'`;
}

function chQuery(uid: string, rawSql: string) {
  return new SceneQueryRunner({
    datasource: { uid, type: CH_DS_TYPE },
    queries: [
      {
        refId: 'A',
        rawSql,
        format: 0,
        queryType: 'sql',
      } as never,
    ],
  });
}

function webVitalsWhere(service: string, environment: string): string {
  let where = `ServiceName = ${sqlStr(service)}`;
  where += ` AND SpanName = 'webvitals'`;
  if (environment) {
    where += ` AND ResourceAttributes['cluster'] = ${sqlStr(environment)}`;
  }
  return where;
}

export interface BuildClickStackFrontendSceneParams {
  service: string;
  environment?: string;
  from: string;
  to: string;
  clickHouseUid: string;
}

/** Browser-vitals trends backed directly by HyperDX webvitals spans. */
export function buildClickStackFrontendScene({
  service,
  environment = '',
  from,
  to,
  clickHouseUid,
}: BuildClickStackFrontendSceneParams): EmbeddedScene | null {
  if (!service || !clickHouseUid) {
    return null;
  }
  const where = webVitalsWhere(service, environment);
  const pageLoadSql = `
SELECT
  $__timeInterval(Timestamp) AS time,
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['lcp']), SpanAttributes['lcp'] != '') AS lcp,
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['fcp']), SpanAttributes['fcp'] != '') AS fcp,
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['ttfb']), SpanAttributes['ttfb'] != '') AS ttfb
FROM \`default\`.\`otel_traces\`
WHERE ${where}
  AND $__timeFilter(Timestamp)
GROUP BY time
ORDER BY time`.trim();
  const interactionSql = `
SELECT
  $__timeInterval(Timestamp) AS time,
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['inp']), SpanAttributes['inp'] != '') AS inp,
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['cls']), SpanAttributes['cls'] != '') AS cls
FROM \`default\`.\`otel_traces\`
WHERE ${where}
  AND $__timeFilter(Timestamp)
GROUP BY time
ORDER BY time`.trim();

  return new EmbeddedScene({
    $timeRange: new SceneTimeRange({ from, to }),
    controls: [],
    body: new SceneFlexLayout({
      direction: 'column',
      children: [
        new SceneFlexItem({
          height: 260,
          body: PanelBuilders.timeseries()
            .setTitle('Page Load Vitals')
            .setDescription('P75 LCP, FCP, and TTFB from ClickStack webvitals spans')
            .setData(chQuery(clickHouseUid, pageLoadSql))
            .setUnit('ms')
            .build(),
        }),
        new SceneFlexItem({
          height: 260,
          body: PanelBuilders.timeseries()
            .setTitle('Interactivity and Layout')
            .setDescription('P75 INP and CLS from ClickStack webvitals spans')
            .setData(chQuery(clickHouseUid, interactionSql))
            .build(),
        }),
      ],
    }),
  });
}
