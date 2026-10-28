import React, { useState } from 'react';
import { LinkButton, useStyles2 } from '@grafana/ui';
import { GrafanaTheme2 } from '@grafana/data';
import { css } from '@emotion/css';
import { ClickStackLogEntry, ExploreFieldFilter, getLogSearch } from '../../../api/analytics';
import { useFetch } from '../../../utils/useFetch';
import { DataState } from '../../../components/DataState';
import { buildLokiExploreUrl } from '../../../utils/explore';
import { ClickStackExploreFilters, LOG_FIELD_KEYS } from '../explore/ClickStackExploreFilters';
import { typographyFontSize } from '../../../utils/themeTypography';

interface ClickStackLogsPanelProps {
  namespace: string;
  service: string;
  fromMs: number;
  toMs: number;
  severity: string[];
  search: string;
  /** Relative range strings for Explore deep links. */
  from?: string;
  to?: string;
  /** When set, show secondary “Open in Explore” (Loki). Hidden in CH-only mode. */
  exploreLogsUid?: string;
  serviceNameLabel?: string;
}

export function ClickStackLogsPanel({
  namespace,
  service,
  fromMs,
  toMs,
  severity,
  search,
  from = 'now-1h',
  to = 'now',
  exploreLogsUid = '',
  serviceNameLabel,
}: ClickStackLogsPanelProps) {
  const styles = useStyles2(getStyles);
  const [fields, setFields] = useState<ExploreFieldFilter[]>([]);
  const [sqlWhere, setSqlWhere] = useState('');
  const { data, loading, error } = useFetch(
    () => getLogSearch(namespace, service, fromMs, toMs, severity, search, fields, sqlWhere),
    [namespace, service, fromMs, toMs, severity, search, fields, sqlWhere]
  );
  const logs = data?.logs ?? [];

  return (
    <div className={styles.panel}>
      <div className={styles.toolbar}>
        <ClickStackExploreFilters
          fieldKeys={LOG_FIELD_KEYS}
          fields={fields}
          onFieldsChange={setFields}
          sqlWhere={sqlWhere}
          onSqlWhereChange={setSqlWhere}
          sqlHint="Optional AND predicate on otel_logs (service + time already scoped). Not LogQL."
        />
        {exploreLogsUid ? (
          <LinkButton
            variant="secondary"
            size="sm"
            icon="compass"
            className={styles.exploreLink}
            href={buildLokiExploreUrl(exploreLogsUid, service, {
              namespace,
              from,
              to,
              serviceNameLabel,
            })}
            tooltip="Open this service in Grafana Explore (Loki)"
          >
            Open in Explore
          </LinkButton>
        ) : null}
      </div>
      <DataState
        loading={loading}
        error={error ? 'Failed to load ClickStack logs' : data?.mode === 'unavailable' ? data.note : null}
        empty={logs.length === 0}
        loadingText="Loading logs…"
        emptyTitle="No logs found"
        emptyMessage="No ClickHouse log entries match the selected filters and time range."
      >
        <div className={styles.tableScroll}>
          <table className={styles.table} aria-label={`Logs for ${service}`}>
            <thead>
              <tr>
                <th scope="col">Time</th>
                <th scope="col">Severity</th>
                <th scope="col">Message</th>
                <th scope="col">Trace</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((entry, index) => (
                <LogRow key={`${entry.timestampMs}-${entry.traceId ?? ''}-${index}`} entry={entry} styles={styles} />
              ))}
            </tbody>
          </table>
        </div>
      </DataState>
    </div>
  );
}

function LogRow({ entry, styles }: { entry: ClickStackLogEntry; styles: ReturnType<typeof getStyles> }) {
  return (
    <tr>
      <td className={styles.timestamp}>{entry.timestampMs ? new Date(entry.timestampMs).toLocaleTimeString() : '—'}</td>
      <td>
        <span className={styles.severity}>{entry.severity || 'unknown'}</span>
      </td>
      <td className={styles.body} title={entry.body}>
        {entry.body || '—'}
      </td>
      <td className={styles.trace}>
        {entry.traceId ? (
          <a href={`?tab=traces&traceId=${encodeURIComponent(entry.traceId)}`} title={entry.traceId}>
            {entry.traceId.slice(0, 12)}
          </a>
        ) : (
          '—'
        )}
      </td>
    </tr>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  panel: css`
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
  `,
  toolbar: css`
    display: flex;
    flex-wrap: wrap;
    align-items: flex-start;
    gap: ${theme.spacing(1)};
  `,
  exploreLink: css`
    margin-left: auto;
  `,
  tableScroll: css`
    overflow-x: auto;
    flex: 1;
    min-height: 0;
  `,
  table: css`
    width: 100%;
    border-collapse: collapse;
    font-size: ${typographyFontSize(theme, 'sm')};
    table-layout: fixed;
    th,
    td {
      padding: ${theme.spacing(0.75, 1)};
      border-bottom: 1px solid ${theme.colors.border.weak};
      text-align: left;
      vertical-align: top;
    }
    th {
      color: ${theme.colors.text.secondary};
      font-weight: ${theme.typography.fontWeightMedium};
    }
    th:nth-child(1),
    td:nth-child(1) {
      width: 90px;
    }
    th:nth-child(2),
    td:nth-child(2) {
      width: 90px;
    }
    th:nth-child(4),
    td:nth-child(4) {
      width: 110px;
    }
  `,
  timestamp: css`
    color: ${theme.colors.text.secondary};
    white-space: nowrap;
  `,
  severity: css`
    color: ${theme.colors.warning.text};
    text-transform: lowercase;
    white-space: nowrap;
  `,
  body: css`
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-family: ${theme.typography.fontFamilyMonospace};
  `,
  trace: css`
    font-family: ${theme.typography.fontFamilyMonospace};
    white-space: nowrap;
  `,
});
