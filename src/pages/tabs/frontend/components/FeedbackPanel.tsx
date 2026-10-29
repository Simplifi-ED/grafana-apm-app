import React from 'react';
import { Badge, useStyles2 } from '@grafana/ui';
import { GrafanaTheme2 } from '@grafana/data';
import { css } from '@emotion/css';
import { getFeedback, FeedbackEntry } from '../../../../api/client';
import { useFetch } from '../../../../utils/useFetch';
import { useTimeRange } from '../../../../utils/timeRange';
import { DataState } from '../../../../components/DataState';
import { typographyFontSize } from '../../../../utils/themeTypography';

interface FeedbackPanelProps {
  namespace: string;
  service: string;
  environment?: string;
  /** ClickStack / HyperDX path — empty-state copy differs from Faro/Loki. */
  clickStack?: boolean;
}

/**
 * Service-level user feedback list (letter G / ticket 21). Faro Loki reads
 * event_name=faro.feedback; ClickStack reads otel_traces SpanName=faro.feedback.
 */
export function FeedbackPanel({ namespace, service, environment, clickStack = false }: FeedbackPanelProps) {
  const styles = useStyles2(getStyles);
  const { fromMs, toMs } = useTimeRange();

  const { data, loading, error } = useFetch(
    () => getFeedback(namespace, service, fromMs, toMs, environment),
    [namespace, service, fromMs, toMs, environment]
  );

  const entries = data?.feedback ?? [];

  const emptyTitle = data?.hint?.includes('serviceAliases')
    ? 'Configure service alias'
    : data?.hint?.includes('faro.feedback') || data?.hint?.includes('ticket 21')
      ? 'Feedback events not detected'
      : 'No feedback';

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h6 className={styles.title}>Feedback</h6>
        <span className={styles.subtitle}>
          {clickStack
            ? 'User feedback from HyperDX faro.feedback actions'
            : 'User feedback captured via the browser SDK'}
        </span>
      </div>
      <DataState
        loading={loading}
        error={
          error
            ? 'Failed to load feedback'
            : data?.unavailable && !data?.hint
              ? clickStack
                ? 'Feedback telemetry is not available'
                : 'Loki is not available'
              : null
        }
        empty={entries.length === 0}
        emptyTitle={emptyTitle}
        emptyMessage={data?.hint ? data.hint : 'No feedback for this service in the selected time range.'}
      >
        <ul className={styles.list} aria-label="Feedback">
          {entries.map((entry, i) => (
            <FeedbackRow key={`${entry.timeMs}-${i}`} entry={entry} />
          ))}
        </ul>
      </DataState>
    </div>
  );
}

function FeedbackRow({ entry }: { entry: FeedbackEntry }) {
  const styles = useStyles2(getStyles);
  return (
    <li className={styles.item}>
      <div className={styles.meta}>
        <span className={styles.time}>{formatRelativeTime(entry.timeMs)}</span>
        {entry.category && <Badge text={entry.category} color="blue" />}
        {entry.appVersion && (
          <span className={styles.version} title={entry.appVersion}>
            {entry.appVersion.length > 10 ? entry.appVersion.slice(0, 10) : entry.appVersion}
          </span>
        )}
      </div>
      <div className={styles.message}>{entry.message || '(empty message)'}</div>
      {(entry.pageUrl || entry.email || entry.sessionId) && (
        <div className={styles.detail}>
          {entry.pageUrl && <span title={entry.pageUrl}>{truncate(entry.pageUrl, 48)}</span>}
          {entry.email && <span>{entry.email}</span>}
          {entry.sessionId && (
            <span className={styles.session} title={entry.sessionId}>
              {entry.sessionId.slice(0, 8)}…
            </span>
          )}
        </div>
      )}
    </li>
  );
}

function truncate(s: string, n: number): string {
  return s.length > n ? `${s.slice(0, n)}…` : s;
}

function formatRelativeTime(epochMs: number): string {
  const diffMs = Date.now() - epochMs;
  if (isNaN(diffMs) || diffMs < 0) {
    return new Date(epochMs).toLocaleString();
  }
  if (diffMs < 60_000) {
    return 'just now';
  }
  if (diffMs < 3_600_000) {
    return `${Math.floor(diffMs / 60_000)}m ago`;
  }
  if (diffMs < 86_400_000) {
    return `${Math.floor(diffMs / 3_600_000)}h ago`;
  }
  return `${Math.floor(diffMs / 86_400_000)}d ago`;
}

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    background: ${theme.colors.background.primary};
    border: 1px solid ${theme.colors.border.weak};
    border-radius: ${theme.shape.radius.default};
    padding: ${theme.spacing(1)};
    width: 100%;
    height: 100%;
    overflow: auto;
  `,
  header: css`
    display: flex;
    align-items: baseline;
    gap: ${theme.spacing(1)};
    padding: ${theme.spacing(0.5, 1)};
  `,
  title: css`
    margin: 0;
    font-size: ${typographyFontSize(theme, 'base')};
  `,
  subtitle: css`
    color: ${theme.colors.text.secondary};
    font-size: ${typographyFontSize(theme, 'sm')};
  `,
  list: css`
    list-style: none;
    margin: 0;
    padding: 0;
  `,
  item: css`
    padding: ${theme.spacing(1)};
    border-bottom: 1px solid ${theme.colors.border.weak};
    &:last-child {
      border-bottom: none;
    }
  `,
  meta: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1)};
    margin-bottom: ${theme.spacing(0.5)};
    font-size: ${typographyFontSize(theme, 'sm')};
  `,
  time: css`
    color: ${theme.colors.text.secondary};
  `,
  version: css`
    font-family: ${theme.typography.fontFamilyMonospace};
    color: ${theme.colors.text.secondary};
  `,
  message: css`
    white-space: pre-wrap;
    word-break: break-word;
  `,
  detail: css`
    display: flex;
    flex-wrap: wrap;
    gap: ${theme.spacing(1.5)};
    margin-top: ${theme.spacing(0.5)};
    color: ${theme.colors.text.secondary};
    font-size: ${typographyFontSize(theme, 'sm')};
  `,
  session: css`
    font-family: ${theme.typography.fontFamilyMonospace};
  `,
});
