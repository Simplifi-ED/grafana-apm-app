import React, { useMemo, useState } from 'react';
import { Badge, Icon, LinkButton, useStyles2 } from '@grafana/ui';
import { GrafanaTheme2 } from '@grafana/data';
import { css } from '@emotion/css';
import {
  EmbeddedScene,
  PanelBuilders,
  SceneFlexItem,
  SceneFlexLayout,
  SceneQueryRunner,
  SceneTimeRange,
} from '@grafana/scenes';
import { CustomMetric, getCustomMetrics } from '../api/client';
import { useFetch } from '../utils/useFetch';
import { useTimeRange } from '../utils/timeRange';
import { useSceneTimeSync } from '../utils/useSceneTimeSync';
import { usePluginDatasources, useServiceAlias } from '../utils/datasources';
import { buildClickHouseExploreUrl, buildMimirExploreUrl } from '../utils/explore';
import { escapeQueryString } from '../utils/sanitize';
import { otel } from '../otelconfig';
import { customMetricExploreQuery, customMetricPanels } from './customMetricQueries';
import { customMetricExploreSQL, customMetricPanelsCH } from './customMetricQueries.ch';
import { DataState } from './DataState';
import { typographyFontSize } from '../utils/themeTypography';

const CH_DS_TYPE = 'grafana-clickhouse-datasource';

interface CustomMetricsPanelProps {
  namespace: string;
  service: string;
  environment?: string;
}

type BadgeColor = React.ComponentProps<typeof Badge>['color'];

const TYPE_BADGE_COLOR: Record<string, BadgeColor> = {
  counter: 'blue',
  histogram: 'purple',
  gauge: 'green',
};

/**
 * Zero-config custom-metric discovery + auto-charting (#68 + letter E):
 * Mimir PromQL when `metricsUid` is set; ClickHouse SQL Scenes when only
 * ClickStack (`clickHouseUid`) is configured. High-cardinality families
 * (series > 100) stay Explore-link only. Collapsed by default on Overview.
 */
export function CustomMetricsPanel({ namespace, service, environment }: CustomMetricsPanelProps) {
  const styles = useStyles2(getStyles);
  const { from, to, fromMs, toMs } = useTimeRange();
  const { metricsUid, clickHouseUid } = usePluginDatasources(environment);
  const scrapeService = useServiceAlias(service);
  const clickStackOnly = Boolean(clickHouseUid) && !metricsUid;
  const [collapsed, setCollapsed] = useState(true);

  const { data, loading, error } = useFetch(
    () => getCustomMetrics(namespace, service, fromMs, toMs, environment),
    [namespace, service, fromMs, toMs, environment]
  );

  const metrics = data?.metrics ?? [];
  const aliasHint = data?.hint;

  // Collapsed-when-empty: no chrome unless alias-miss hint (operator action).
  if (!loading && !error && metrics.length === 0 && !aliasHint) {
    return null;
  }

  const promFilter = runtimeFilter(service, namespace);
  const chartable = metrics.filter((m) => !m.highCardinality);
  const showBody = !collapsed || !!error || (!!aliasHint && metrics.length === 0);

  const exploreUrlFor = (m: CustomMetric): string => {
    if (clickStackOnly && clickHouseUid) {
      return buildClickHouseExploreUrl(clickHouseUid, customMetricExploreSQL(m, scrapeService, namespace), {
        from,
        to,
      });
    }
    return buildMimirExploreUrl(metricsUid, customMetricExploreQuery(m, promFilter), { from, to });
  };

  return (
    <div className={styles.container}>
      <div className={styles.header} onClick={() => setCollapsed((c) => !c)}>
        <Icon name={collapsed ? 'angle-right' : 'angle-down'} />
        <h6 className={styles.title}>Custom metrics{metrics.length > 0 ? ` (${metrics.length})` : ''}</h6>
        <span className={styles.subtitle}>
          {clickStackOnly ? 'Auto-discovered ClickHouse application metrics' : 'Auto-discovered application metrics'}
        </span>
      </div>
      {showBody && (
        <DataState loading={loading} error={error ? 'Failed to load custom metrics' : null} empty={false}>
          {aliasHint && metrics.length === 0 && <div className={styles.hint}>{aliasHint}</div>}
          {!collapsed && chartable.length > 0 && clickStackOnly && clickHouseUid && (
            <AutoChartsCH
              metrics={chartable}
              scrapeService={scrapeService}
              namespace={namespace}
              clickHouseUid={clickHouseUid}
              from={from}
              to={to}
            />
          )}
          {!collapsed && chartable.length > 0 && !clickStackOnly && (
            <AutoCharts metrics={chartable} filter={promFilter} metricsUid={metricsUid} from={from} to={to} />
          )}
          {metrics.length > 0 && (
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Metric</th>
                  <th>Type</th>
                  <th className={styles.num}>Series</th>
                  <th className={styles.actions}></th>
                </tr>
              </thead>
              <tbody>
                {metrics.map((m) => (
                  <MetricRow key={m.name} metric={m} exploreUrl={exploreUrlFor(m)} />
                ))}
              </tbody>
            </table>
          )}
          {data?.truncated && (
            <div className={styles.hint}>Showing the first {metrics.length} metric families — more exist.</div>
          )}
        </DataState>
      )}
    </div>
  );
}

function MetricRow({ metric, exploreUrl }: { metric: CustomMetric; exploreUrl: string }) {
  const styles = useStyles2(getStyles);
  return (
    <tr>
      <td className={styles.nameCell}>
        <span className={styles.name} title={metric.help || metric.name}>
          {metric.name}
        </span>
        {metric.unit && <span className={styles.unit}>{metric.unit}</span>}
        {metric.highCardinality && (
          <span className={styles.cardinalityHint}>high cardinality — not auto-charted, open in Explore</span>
        )}
      </td>
      <td>
        <Badge text={metric.type} color={TYPE_BADGE_COLOR[metric.type] ?? 'orange'} />
      </td>
      <td className={styles.num}>{metric.series}</td>
      <td className={styles.actions}>
        <LinkButton size="sm" variant="secondary" icon="compass" href={exploreUrl} target="_blank">
          Open in Explore
        </LinkButton>
      </td>
    </tr>
  );
}

/** app/namespace matcher for app-emitted metrics (mirrors backend RuntimeFilter). */
function runtimeFilter(service: string, namespace: string): string {
  let f = `${otel.runtime.appLabel}="${escapeQueryString(service)}"`;
  if (namespace) {
    f += `, ${otel.runtime.namespaceLabel}="${escapeQueryString(namespace)}"`;
  }
  return f;
}

/**
 * Type-aware auto-charts for the low-cardinality families (#68 Phase 1). Each
 * family expands to one or more timeseries panels (summaries/timers → two:
 * throughput and avg) driven by the shared per-type PromQL. A wrapping flex
 * grid keeps the panels compact so the section stays scannable when expanded.
 */
function AutoCharts({
  metrics,
  filter,
  metricsUid,
  from,
  to,
}: {
  metrics: CustomMetric[];
  filter: string;
  metricsUid: string;
  from: string;
  to: string;
}) {
  const { fromMs, toMs } = useTimeRange();
  const scene = useMemo(() => {
    const items = metrics.flatMap((metric) =>
      customMetricPanels(metric, filter).map((spec) => {
        const runner = new SceneQueryRunner({
          datasource: { uid: metricsUid, type: 'prometheus' },
          queries: spec.queries.map((q) => ({ refId: q.refId, expr: q.expr, legendFormat: q.legendFormat })),
        });
        let builder = PanelBuilders.timeseries().setTitle(spec.title).setData(runner);
        if (metric.help) {
          builder = builder.setDescription(metric.help);
        }
        if (spec.unit) {
          builder = builder.setUnit(spec.unit);
        }
        return new SceneFlexItem({ minWidth: '320px', height: 200, body: builder.build() });
      })
    );

    return new EmbeddedScene({
      $timeRange: new SceneTimeRange({ from, to }),
      body: new SceneFlexLayout({ direction: 'row', wrap: 'wrap', children: items }),
    });
    // metricsUid/from/to/filter are the only inputs; metrics identity changes
    // with the fetch, which is the intended rebuild trigger.
  }, [metrics, filter, metricsUid, from, to]);

  // Header refresh re-resolves fromMs/toMs while the raw strings stay put —
  // re-query the live scene instead of rebuilding it.
  useSceneTimeSync(scene, fromMs, toMs);

  return <scene.Component model={scene} />;
}

/** ClickStack auto-charts: CH SQL SceneQueryRunners (letter E). */
function AutoChartsCH({
  metrics,
  scrapeService,
  namespace,
  clickHouseUid,
  from,
  to,
}: {
  metrics: CustomMetric[];
  scrapeService: string;
  namespace: string;
  clickHouseUid: string;
  from: string;
  to: string;
}) {
  const { fromMs, toMs } = useTimeRange();
  const scene = useMemo(() => {
    const items = metrics.flatMap((metric) =>
      customMetricPanelsCH(metric, scrapeService, namespace).map((spec) => {
        const runner = new SceneQueryRunner({
          datasource: { uid: clickHouseUid, type: CH_DS_TYPE },
          queries: [
            {
              refId: 'A',
              rawSql: spec.rawSql,
              format: 0,
              queryType: 'sql',
            } as never,
          ],
        });
        let builder = PanelBuilders.timeseries().setTitle(spec.title).setData(runner);
        if (metric.help) {
          builder = builder.setDescription(metric.help);
        }
        if (spec.unit) {
          builder = builder.setUnit(spec.unit);
        }
        return new SceneFlexItem({ minWidth: '320px', height: 200, body: builder.build() });
      })
    );

    return new EmbeddedScene({
      $timeRange: new SceneTimeRange({ from, to }),
      body: new SceneFlexLayout({ direction: 'row', wrap: 'wrap', children: items }),
    });
  }, [metrics, scrapeService, namespace, clickHouseUid, from, to]);

  useSceneTimeSync(scene, fromMs, toMs);

  return <scene.Component model={scene} />;
}

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    background: ${theme.colors.background.primary};
    border: 1px solid ${theme.colors.border.weak};
    border-radius: ${theme.shape.radius.default};
    padding: ${theme.spacing(1)};
    width: 100%;
  `,
  header: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1)};
    padding: ${theme.spacing(0.5, 1)};
    cursor: pointer;
  `,
  title: css`
    margin: 0;
    font-size: ${typographyFontSize(theme, 'base')};
  `,
  subtitle: css`
    color: ${theme.colors.text.secondary};
    font-size: ${typographyFontSize(theme, 'sm')};
  `,
  table: css`
    width: 100%;
    border-collapse: collapse;
    font-size: ${typographyFontSize(theme, 'sm')};
    th {
      text-align: left;
      color: ${theme.colors.text.secondary};
      font-weight: ${theme.typography.fontWeightMedium};
      padding: ${theme.spacing(0.5, 1)};
      border-bottom: 1px solid ${theme.colors.border.weak};
    }
    td {
      padding: ${theme.spacing(0.75, 1)};
      border-bottom: 1px solid ${theme.colors.border.weak};
      vertical-align: middle;
    }
  `,
  nameCell: css`
    max-width: 0;
    width: 60%;
  `,
  name: css`
    font-family: ${theme.typography.fontFamilyMonospace};
    overflow-wrap: anywhere;
  `,
  unit: css`
    margin-left: ${theme.spacing(1)};
    color: ${theme.colors.text.secondary};
    font-size: ${typographyFontSize(theme, 'sm')};
  `,
  cardinalityHint: css`
    display: block;
    color: ${theme.colors.warning.text};
    font-size: ${typographyFontSize(theme, 'sm')};
  `,
  num: css`
    text-align: right;
    white-space: nowrap;
    width: 1%;
  `,
  actions: css`
    text-align: right;
    white-space: nowrap;
    width: 1%;
  `,
  hint: css`
    padding: ${theme.spacing(0.5, 1)};
    color: ${theme.colors.text.secondary};
    font-size: ${typographyFontSize(theme, 'sm')};
  `,
});
