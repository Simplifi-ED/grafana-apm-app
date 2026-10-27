import React, { useMemo, useState } from 'react';
import { Alert, Button, useStyles2 } from '@grafana/ui';
import { GrafanaTheme2 } from '@grafana/data';
import { css } from '@emotion/css';
import { ClickStackTraceSpan, ClickStackTraceSummary, getTraceDetail, getTraceSearch } from '../../../api/analytics';
import { useFetch } from '../../../utils/useFetch';
import { DataState } from '../../../components/DataState';
import { formatDuration } from '../../../utils/format';

interface ClickStackTracesPanelProps {
  namespace: string;
  service: string;
  fromMs: number;
  toMs: number;
  status: string;
  spanSearch: string;
  durationMin: string;
  durationMax: string;
}

export function ClickStackTracesPanel({
  namespace,
  service,
  fromMs,
  toMs,
  status,
  spanSearch,
  durationMin,
  durationMax,
}: ClickStackTracesPanelProps) {
  const styles = useStyles2(getStyles);
  const [selectedTraceId, setSelectedTraceId] = useState('');
  const { data, loading, error } = useFetch(
    () => getTraceSearch(namespace, service, fromMs, toMs, status, spanSearch, durationMin, durationMax),
    [namespace, service, fromMs, toMs, status, spanSearch, durationMin, durationMax]
  );
  const detail = useFetch(
    () => getTraceDetail(namespace, service, selectedTraceId, fromMs, toMs),
    [namespace, service, selectedTraceId, fromMs, toMs],
    { skip: !selectedTraceId }
  );

  const traces = data?.traces ?? [];
  const selectedSpans = detail.data?.spans ?? [];

  return (
    <div className={styles.container}>
      <DataState
        loading={loading}
        error={error ? 'Failed to load ClickStack traces' : data?.mode === 'unavailable' ? data.note : null}
        empty={traces.length === 0}
        loadingText="Loading traces…"
        emptyTitle="No traces found"
        emptyMessage="No ClickHouse traces match the selected filters and time range."
      >
        <div className={styles.tableScroll}>
          <table className={styles.table} aria-label={`Traces for ${service}`}>
            <thead>
              <tr>
                <th scope="col">Trace</th>
                <th scope="col">Started</th>
                <th scope="col" className={styles.number}>
                  Spans
                </th>
                <th scope="col" className={styles.number}>
                  Errors
                </th>
                <th scope="col" className={styles.number}>
                  Duration
                </th>
              </tr>
            </thead>
            <tbody>
              {traces.map((trace) => (
                <TraceRow
                  key={trace.traceId}
                  trace={trace}
                  selected={trace.traceId === selectedTraceId}
                  onSelect={() => setSelectedTraceId(trace.traceId)}
                />
              ))}
            </tbody>
          </table>
        </div>
      </DataState>

      {selectedTraceId && (
        <section className={styles.detail} aria-label={`Trace ${selectedTraceId}`}>
          <div className={styles.detailHeader}>
            <div>
              <h4 className={styles.detailTitle}>Trace waterfall</h4>
              <code>{selectedTraceId}</code>
            </div>
            <Button variant="secondary" size="sm" onClick={() => setSelectedTraceId('')}>
              Close
            </Button>
          </div>
          <DataState
            loading={detail.loading}
            error={
              detail.error
                ? 'Failed to load trace details'
                : detail.data?.mode === 'unavailable'
                  ? detail.data.note
                  : null
            }
            empty={selectedSpans.length === 0}
            loadingText="Loading spans…"
            emptyTitle="No spans found"
            emptyMessage="The trace details are outside the selected time range."
          >
            <Waterfall spans={selectedSpans} />
          </DataState>
        </section>
      )}

      {data?.note && data.mode === 'clickhouse' && (
        <Alert severity="info" title="ClickHouse traces">
          {data.note}
        </Alert>
      )}
    </div>
  );
}

function TraceRow({
  trace,
  selected,
  onSelect,
}: {
  trace: ClickStackTraceSummary;
  selected: boolean;
  onSelect: () => void;
}) {
  const styles = useStyles2(getStyles);
  return (
    <tr className={selected ? styles.selectedRow : undefined} onClick={onSelect}>
      <td>
        <button className={styles.traceButton} type="button" onClick={onSelect}>
          {trace.traceId.slice(0, 16)}
        </button>
      </td>
      <td>{trace.startMs ? new Date(trace.startMs).toLocaleTimeString() : '—'}</td>
      <td className={styles.number}>{trace.spanCount}</td>
      <td className={styles.number}>{trace.errors || '—'}</td>
      <td className={styles.number}>{formatDuration(trace.totalMs, 'ms')}</td>
    </tr>
  );
}

function Waterfall({ spans }: { spans: ClickStackTraceSpan[] }) {
  const styles = useStyles2(getStyles);
  const { startMs, totalDuration } = useMemo(() => {
    const start = spans.reduce((min, span) => Math.min(min, span.timestampMs), Number.POSITIVE_INFINITY);
    const end = spans.reduce((max, span) => Math.max(max, span.timestampMs + span.durationMs), 0);
    return { startMs: Number.isFinite(start) ? start : 0, totalDuration: Math.max(end - start, 1) };
  }, [spans]);
  const byId = useMemo(() => new Map(spans.map((span) => [span.spanId, span])), [spans]);

  return (
    <div className={styles.waterfall}>
      {spans.map((span) => {
        let depth = 0;
        let parent = span.parentSpanId;
        const visited = new Set<string>();
        while (parent && byId.has(parent) && !visited.has(parent)) {
          visited.add(parent);
          depth += 1;
          parent = byId.get(parent)?.parentSpanId ?? '';
        }
        return (
          <div className={styles.spanRow} key={`${span.spanId}-${span.timestampMs}`}>
            <div className={styles.spanLabel} style={{ paddingLeft: `${depth * 16}px` }}>
              <strong>{span.spanName}</strong>
              <span className={styles.spanMeta}>
                {span.service} · {span.spanKind} · {formatDuration(span.durationMs, 'ms')}
              </span>
            </div>
            <div className={styles.barTrack}>
              <span
                className={span.status === 'Error' ? styles.errorBar : styles.bar}
                style={{
                  left: `${Math.max(0, ((span.timestampMs - startMs) / totalDuration) * 100)}%`,
                  width: `${Math.max(1, (span.durationMs / totalDuration) * 100)}%`,
                }}
              />
            </div>
          </div>
        );
      })}
    </div>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(2)};
  `,
  tableScroll: css`
    overflow-x: auto;
  `,
  table: css`
    width: 100%;
    border-collapse: collapse;
    font-size: ${theme.typography.sm.fontSize};
    th,
    td {
      padding: ${theme.spacing(0.75, 1)};
      border-bottom: 1px solid ${theme.colors.border.weak};
      text-align: left;
      white-space: nowrap;
    }
    th {
      color: ${theme.colors.text.secondary};
      font-weight: ${theme.typography.fontWeightMedium};
    }
    tbody tr {
      cursor: pointer;
      &:hover {
        background: ${theme.colors.action.hover};
      }
    }
  `,
  number: css`
    text-align: right !important;
    font-variant-numeric: tabular-nums;
  `,
  traceButton: css`
    border: 0;
    padding: 0;
    background: transparent;
    color: ${theme.colors.text.link};
    font-family: ${theme.typography.fontFamilyMonospace};
    cursor: pointer;
  `,
  selectedRow: css`
    background: ${theme.colors.action.selected};
  `,
  detail: css`
    border: 1px solid ${theme.colors.border.weak};
    border-radius: ${theme.shape.radius.default};
    padding: ${theme.spacing(1.5)};
  `,
  detailHeader: css`
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: ${theme.spacing(2)};
    margin-bottom: ${theme.spacing(1.5)};
  `,
  detailTitle: css`
    margin: 0 0 ${theme.spacing(0.5)};
  `,
  waterfall: css`
    display: flex;
    flex-direction: column;
  `,
  spanRow: css`
    display: grid;
    grid-template-columns: minmax(260px, 1fr) minmax(240px, 2fr);
    align-items: center;
    gap: ${theme.spacing(1)};
    min-height: 34px;
    border-bottom: 1px solid ${theme.colors.border.weak};
  `,
  spanLabel: css`
    display: flex;
    flex-direction: column;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  `,
  spanMeta: css`
    color: ${theme.colors.text.secondary};
    font-size: ${theme.typography.sm.fontSize};
  `,
  barTrack: css`
    position: relative;
    height: 12px;
    background: ${theme.colors.background.secondary};
    border-radius: 2px;
  `,
  bar: css`
    position: absolute;
    top: 2px;
    height: 8px;
    min-width: 3px;
    background: ${theme.colors.primary.main};
    border-radius: 2px;
  `,
  errorBar: css`
    position: absolute;
    top: 2px;
    height: 8px;
    min-width: 3px;
    background: ${theme.colors.error.main};
    border-radius: 2px;
  `,
});
