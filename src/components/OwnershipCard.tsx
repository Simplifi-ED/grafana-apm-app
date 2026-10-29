import React, { useCallback, useState } from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { Badge, Button, Icon, Tooltip, useStyles2 } from '@grafana/ui';
import { Ownership, OwnershipSource } from '../api/ownership';
import { typographyFontSize } from '../utils/themeTypography';

export interface OwnershipCardProps {
  data: Ownership | null;
  loading?: boolean;
  /** Compact strip for headers: Team · Slack · Repo */
  compact?: boolean;
}

function contactHref(contact: string): string | undefined {
  if (contact.startsWith('http://') || contact.startsWith('https://') || contact.startsWith('mailto:')) {
    return contact;
  }
  if (contact.includes('@') && !contact.startsWith('#')) {
    return `mailto:${contact}`;
  }
  if (contact.startsWith('#')) {
    return undefined; // Slack channel name — display only without workspace URL
  }
  return undefined;
}

function sourceLabel(source?: OwnershipSource): string {
  switch (source) {
    case 'telemetry':
      return 'telemetry';
    case 'annotation':
      return 'annotation';
    case 'claimed':
      return 'claimed';
    default:
      return '';
  }
}

function formatLastSeen(ms?: number): string | null {
  if (!ms) {
    return null;
  }
  const days = Math.floor((Date.now() - ms) / (24 * 60 * 60 * 1000));
  if (days <= 0) {
    return 'last seen today';
  }
  if (days === 1) {
    return 'last seen 1d ago';
  }
  return `last seen ${days}d ago`;
}

/**
 * Single ownership primitive — directory strip/card for service header,
 * scorecard, and namespace/inventory hygiene.
 */
export function OwnershipCard({ data, loading, compact }: OwnershipCardProps) {
  const styles = useStyles2(getStyles);
  const [copied, setCopied] = useState(false);
  const onCopy = useCallback(() => {
    const snippet = data?.emptyHint?.snippet;
    if (!snippet) {
      return;
    }
    void navigator.clipboard.writeText(snippet).then(
      () => {
        setCopied(true);
        window.setTimeout(() => setCopied(false), 1500);
      },
      () => {
        setCopied(false);
      }
    );
  }, [data?.emptyHint?.snippet]);

  if (loading || !data) {
    return null;
  }

  if (data.unavailable) {
    return (
      <div className={styles.empty} data-testid="ownership-unavailable">
        <Icon name="exclamation-triangle" className={styles.emptyIcon} />
        <div>
          <div className={styles.emptyTitle}>Ownership unavailable</div>
          <div className={styles.emptyHint}>Could not query telemetry or annotations.</div>
        </div>
      </div>
    );
  }

  const owned = Boolean(data.team);
  if (!owned) {
    return (
      <div className={styles.empty} data-testid="ownership-unowned">
        <Icon name="user" className={styles.emptyIcon} />
        <div>
          <div className={styles.emptyTitle}>No owner</div>
          <div className={styles.emptyHint}>
            Add <code>owner.team</code> to the workload
            {data.emptyHint?.snippet ? (
              <>
                {' '}
                <Button size="sm" variant="secondary" fill="outline" onClick={onCopy}>
                  {copied ? 'Copied' : 'Copy snippet'}
                </Button>
              </>
            ) : null}
          </div>
        </div>
      </div>
    );
  }

  const parts: React.ReactNode[] = [];
  parts.push(
    <span key="team" className={styles.field}>
      <Icon name="users-alt" size="sm" />
      <span>{data.team}</span>
      {!data.teamKnown ? (
        <Tooltip content="Team not in plugin knownTeams list">
          <Badge text="unknown team" color="orange" />
        </Tooltip>
      ) : null}
    </span>
  );
  if (data.contact) {
    const href = contactHref(data.contact);
    parts.push(
      <span key="contact" className={styles.field}>
        {href ? (
          <a href={href} target="_blank" rel="noopener noreferrer">
            {data.contact}
          </a>
        ) : (
          data.contact
        )}
      </span>
    );
  }
  if (data.repo) {
    parts.push(
      <span key="repo" className={styles.field}>
        <a href={data.repo} target="_blank" rel="noopener noreferrer">
          Repo
        </a>
      </span>
    );
  }
  if (data.runbook && !compact) {
    parts.push(
      <span key="runbook" className={styles.field}>
        <a href={data.runbook} target="_blank" rel="noopener noreferrer">
          Runbook
        </a>
      </span>
    );
  }

  const lastSeen = formatLastSeen(data.lastSeenMs);

  return (
    <div className={compact ? styles.strip : styles.card} data-testid="ownership-owned" data-source={data.source}>
      <div className={styles.row}>
        {parts.map((p, i) => (
          <React.Fragment key={i}>
            {i > 0 ? <span className={styles.dot}>·</span> : null}
            {p}
          </React.Fragment>
        ))}
        {data.source ? <Badge text={sourceLabel(data.source)} color="blue" /> : null}
        {data.stale ? (
          <Tooltip content="Ownership attrs not seen in telemetry for 7+ days">
            <Badge text="stale" color="orange" />
          </Tooltip>
        ) : null}
      </div>
      {!compact && lastSeen ? <div className={styles.meta}>{lastSeen}</div> : null}
      {data.conflict?.losers?.length ? (
        <details className={styles.conflict}>
          <summary>Other source disagreed</summary>
          {data.conflict.losers.map((l) => (
            <div key={l.source} className={styles.meta}>
              {l.source}: {l.fields.team || '—'}
              {l.fields.contact ? ` · ${l.fields.contact}` : ''}
            </div>
          ))}
        </details>
      ) : null}
    </div>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  strip: css({
    display: 'inline-flex',
    flexDirection: 'column',
    gap: 2,
    maxWidth: 420,
  }),
  card: css({
    display: 'flex',
    flexDirection: 'column',
    gap: theme.spacing(0.5),
    padding: theme.spacing(1, 1.5),
    borderRadius: theme.shape.radius.default,
    background: theme.colors.background.secondary,
  }),
  row: css({
    display: 'flex',
    flexWrap: 'wrap',
    alignItems: 'center',
    gap: theme.spacing(0.5),
    fontSize: typographyFontSize(theme, 'sm'),
  }),
  field: css({
    display: 'inline-flex',
    alignItems: 'center',
    gap: 4,
    a: { color: theme.colors.text.link },
  }),
  dot: css({ color: theme.colors.text.secondary }),
  meta: css({
    fontSize: typographyFontSize(theme, 'sm'),
    color: theme.colors.text.secondary,
  }),
  conflict: css({
    fontSize: typographyFontSize(theme, 'sm'),
    color: theme.colors.text.secondary,
  }),
  empty: css({
    display: 'inline-flex',
    alignItems: 'flex-start',
    gap: theme.spacing(1),
    padding: theme.spacing(0.5, 1),
    borderRadius: theme.shape.radius.default,
    background: theme.colors.background.secondary,
    fontSize: typographyFontSize(theme, 'sm'),
  }),
  emptyIcon: css({ color: theme.colors.text.secondary, marginTop: 2 }),
  emptyTitle: css({ fontWeight: theme.typography.fontWeightMedium }),
  emptyHint: css({
    color: theme.colors.text.secondary,
    code: { fontSize: '0.9em' },
  }),
});
