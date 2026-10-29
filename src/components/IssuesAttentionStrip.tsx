import React from 'react';
import { Alert, Button, Icon, useStyles2 } from '@grafana/ui';
import { GrafanaTheme2 } from '@grafana/data';
import { css } from '@emotion/css';
import { IssuesAttention, NamespaceServiceAttention } from '../api/client';
import { typographyFontSize } from '../utils/themeTypography';

interface IssuesAttentionStripProps {
  attention?: IssuesAttention | null;
  loading?: boolean;
  /** Fetch failure — shown instead of silently hiding the strip. */
  error?: string | null;
  onViewIssues: () => void;
  onViewAlerts?: () => void;
  /** Optional per-service breakdown (namespace rollup). */
  byService?: NamespaceServiceAttention[];
  /** When true, mention that the namespace probe was capped. */
  truncated?: boolean;
  scanned?: number;
  viewLabel?: string;
}

/**
 * Overview / namespace attention strip for new / returned-after-resolve issue
 * counts (post-parity letter D). Hidden when both counts are zero.
 */
export function IssuesAttentionStrip({
  attention,
  loading,
  error,
  onViewIssues,
  onViewAlerts,
  byService,
  truncated,
  scanned,
  viewLabel = 'View Issues',
}: IssuesAttentionStripProps) {
  const styles = useStyles2(getStyles);
  if (loading) {
    return null;
  }
  if (error && !attention) {
    return (
      <Alert severity="warning" title="Could not load issues attention">
        {error}
      </Alert>
    );
  }
  if (!attention) {
    return null;
  }
  const { new: newCount, returnedAfterResolve } = attention;
  if (newCount <= 0 && returnedAfterResolve <= 0) {
    return null;
  }

  const parts: string[] = [];
  if (newCount > 0) {
    parts.push(`${newCount} new`);
  }
  if (returnedAfterResolve > 0) {
    parts.push(`${returnedAfterResolve} returned after resolve`);
  }

  const topServices = (byService ?? []).slice(0, 3);
  let scopeDetail = 'in the current time range.';
  if (topServices.length > 0) {
    const names = topServices.map((s) => s.service).join(', ');
    const more =
      (byService?.length ?? 0) > topServices.length ? ` +${(byService?.length ?? 0) - topServices.length} more` : '';
    scopeDetail = `across ${byService?.length ?? 0} service${(byService?.length ?? 0) === 1 ? '' : 's'} (${names}${more}).`;
    if (truncated && scanned != null) {
      scopeDetail += ` Scanned top ${scanned} by traffic.`;
    }
  }

  return (
    <div className={styles.strip} role="status" data-testid="issues-attention-strip">
      <Icon name="exclamation-circle" className={styles.icon} />
      <div className={styles.body}>
        <div className={styles.title}>Issues needing attention</div>
        <div className={styles.detail}>
          {parts.join(' · ')} {scopeDetail}
        </div>
      </div>
      <div className={styles.actions}>
        <Button size="sm" variant="secondary" onClick={onViewIssues}>
          {viewLabel}
        </Button>
        {onViewAlerts && (
          <Button size="sm" variant="secondary" fill="outline" onClick={onViewAlerts}>
            Alert templates
          </Button>
        )}
      </div>
    </div>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  strip: css`
    display: flex;
    align-items: flex-start;
    gap: ${theme.spacing(1.5)};
    margin-bottom: ${theme.spacing(2)};
    padding: ${theme.spacing(1.5)};
    background: ${theme.colors.background.secondary};
    border: 1px solid ${theme.colors.border.weak};
    border-radius: ${theme.shape.radius.default};
  `,
  icon: css`
    margin-top: ${theme.spacing(0.25)};
    color: ${theme.colors.warning.text};
    flex-shrink: 0;
  `,
  body: css`
    flex: 1;
    min-width: 0;
  `,
  title: css`
    font-weight: ${theme.typography.fontWeightMedium};
    margin-bottom: ${theme.spacing(0.25)};
  `,
  detail: css`
    font-size: ${typographyFontSize(theme, 'sm')};
    color: ${theme.colors.text.secondary};
  `,
  actions: css`
    display: flex;
    flex-wrap: wrap;
    gap: ${theme.spacing(1)};
    flex-shrink: 0;
  `,
});
