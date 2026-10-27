import React from 'react';
import { useStyles2 } from '@grafana/ui';
import { GrafanaTheme2 } from '@grafana/data';
import { css } from '@emotion/css';
import { ClickStackLogEntry, getLogSearch } from '../../../api/analytics';
import { useFetch } from '../../../utils/useFetch';
import { DataState } from '../../../components/DataState';

interface ClickStackLogsPanelProps {
  namespace: string;
  service: string;
  fromMs: number;
  toMs: number;
  severity: string[];
  search: string;
}

export function ClickStackLogsPanel({ namespace, service, fromMs, toMs, severity, search }: ClickStackLogsPanelProps) {
  const styles = useStyles2(getStyles);
  const { data, loading, error } = useFetch(
    () => getLogSearch(namespace, service, fromMs, toMs, severity, search),
    [namespace, service, fromMs, toMs, severity, search]
  );
  const logs = data?.logs ?? [];

  return (
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
  tableScroll: css`
    overflow-x: auto;
    flex: 1;
    min-height: 0;
  `,
  table: css`
    width: 100%;
    border-collapse: collapse;
    font-size: ${theme.typography.sm.fontSize};
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
