import {
  EmbeddedScene,
  PanelBuilders,
  SceneFlexItem,
  SceneFlexLayout,
  SceneQueryRunner,
  SceneTimeRange,
  behaviors,
} from '@grafana/scenes';
import { DashboardCursorSync } from '@grafana/schema';
import {
  browserBreakdownSql,
  browserEventsTopSql,
  browserEventsVolumeSql,
  interactivitySql,
  jsExceptionsSql,
  layoutStabilitySql,
  longtaskP75Sql,
  measurementsVolumeSql,
  pageLoadVitalsSql,
  perPageSql,
  ratingDistributionSql,
} from './tabs/frontend/queries/clickstack-builders';

const CH_DS_TYPE = 'grafana-clickhouse-datasource';

/** format: 0/2 timeseries frames; 1 = table (docs/clickstack-migration/audit.md). */
function chQuery(uid: string, rawSql: string, format: 0 | 1 | 2 = 0) {
  return new SceneQueryRunner({
    datasource: { uid, type: CH_DS_TYPE },
    queries: [
      {
        refId: 'A',
        rawSql,
        format,
        queryType: 'sql',
      } as never,
    ],
  });
}

export interface BuildClickStackFrontendSceneParams {
  service: string;
  environment?: string;
  from: string;
  to: string;
  clickHouseUid: string;
}

/**
 * HyperDX / ClickStack Frontend tab — vitals trends, volume, exceptions,
 * CWV rating pie, browser + per-page tables (no Faro/Prom required).
 */
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
  const env = environment || '';

  const trendsRow = new SceneFlexLayout({
    direction: 'row',
    children: [
      new SceneFlexItem({
        minHeight: 260,
        body: PanelBuilders.timeseries()
          .setTitle('Page Load Vitals')
          .setDescription('P75 LCP, FCP, and TTFB from HyperDX webvitals spans')
          .setData(chQuery(clickHouseUid, pageLoadVitalsSql(service, env)))
          .setUnit('ms')
          .build(),
      }),
      new SceneFlexItem({
        minHeight: 260,
        body: PanelBuilders.timeseries()
          .setTitle('Interactivity (INP)')
          .setDescription('P75 Interaction to Next Paint')
          .setData(chQuery(clickHouseUid, interactivitySql(service, env)))
          .setUnit('ms')
          .build(),
      }),
      new SceneFlexItem({
        minHeight: 260,
        body: PanelBuilders.timeseries()
          .setTitle('Layout Stability (CLS)')
          .setDescription('P75 Cumulative Layout Shift')
          .setData(chQuery(clickHouseUid, layoutStabilitySql(service, env)))
          .build(),
      }),
    ],
  });

  const trafficRow = new SceneFlexLayout({
    direction: 'row',
    children: [
      new SceneFlexItem({
        minHeight: 220,
        body: PanelBuilders.timeseries()
          .setTitle('Web Vitals Measurements')
          .setDescription('Count of webvitals spans over time')
          .setData(chQuery(clickHouseUid, measurementsVolumeSql(service, env)))
          .setUnit('short')
          .setCustomFieldConfig('fillOpacity', 15)
          .build(),
      }),
      new SceneFlexItem({
        minHeight: 220,
        body: PanelBuilders.timeseries()
          .setTitle('JavaScript Exceptions')
          .setDescription('Browser error / unhandledrejection spans')
          .setData(chQuery(clickHouseUid, jsExceptionsSql(service, env)))
          .setUnit('short')
          .setCustomFieldConfig('fillOpacity', 15)
          .build(),
      }),
      new SceneFlexItem({
        minHeight: 220,
        width: '30%',
        body: PanelBuilders.piechart()
          .setTitle('Web Vitals Rating Distribution')
          .setDescription('good / needs-improvement / poor from CWV thresholds')
          .setData(chQuery(clickHouseUid, ratingDistributionSql(service, env), 1))
          .build(),
      }),
    ],
  });

  const tablesRow = new SceneFlexLayout({
    direction: 'row',
    children: [
      new SceneFlexItem({
        minHeight: 280,
        body: PanelBuilders.table()
          .setTitle('Browser Breakdown')
          .setDescription('Avg vitals by browser (UA joined via rum.sessionId)')
          .setData(chQuery(clickHouseUid, browserBreakdownSql(service, env), 1))
          .build(),
      }),
      new SceneFlexItem({
        minHeight: 280,
        body: PanelBuilders.table()
          .setTitle('Per-page Vitals')
          .setDescription('P75 vitals by URL path')
          .setData(chQuery(clickHouseUid, perPageSql(service, env), 1))
          .build(),
      }),
    ],
  });

  // Frontend measurements/events beyond Overview CWV charts (deferred follow-on).
  const eventsRow = new SceneFlexLayout({
    direction: 'row',
    children: [
      new SceneFlexItem({
        minHeight: 220,
        body: PanelBuilders.timeseries()
          .setTitle('Browser Events')
          .setDescription('UX event volume (click, navigation, routeChange, …) excluding webvitals/HTTP/errors')
          .setData(chQuery(clickHouseUid, browserEventsVolumeSql(service, env)))
          .setUnit('short')
          .setCustomFieldConfig('fillOpacity', 15)
          .build(),
      }),
      new SceneFlexItem({
        minHeight: 220,
        body: PanelBuilders.timeseries()
          .setTitle('Long Tasks (P75)')
          .setDescription('P75 duration of longtask spans (main-thread blocks)')
          .setData(chQuery(clickHouseUid, longtaskP75Sql(service, env)))
          .setUnit('ms')
          .build(),
      }),
      new SceneFlexItem({
        minHeight: 220,
        body: PanelBuilders.table()
          .setTitle('Top Browser Events')
          .setDescription('SpanName breakdown for HyperDX UX events')
          .setData(chQuery(clickHouseUid, browserEventsTopSql(service, env), 1))
          .build(),
      }),
    ],
  });

  return new EmbeddedScene({
    $timeRange: new SceneTimeRange({ from, to }),
    $behaviors: [new behaviors.CursorSync({ sync: DashboardCursorSync.Crosshair })],
    controls: [],
    body: new SceneFlexLayout({
      direction: 'column',
      children: [
        new SceneFlexItem({ body: trendsRow }),
        new SceneFlexItem({ body: trafficRow }),
        new SceneFlexItem({ body: tablesRow }),
        new SceneFlexItem({ body: eventsRow }),
      ],
    }),
  });
}
