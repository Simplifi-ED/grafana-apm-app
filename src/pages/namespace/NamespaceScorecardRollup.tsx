import React from 'react';
import { useStyles2, Icon } from '@grafana/ui';
import { GrafanaTheme2 } from '@grafana/data';
import { css } from '@emotion/css';
import { NamespaceScorecardResponse } from '../../api/scorecard';
import { typographyFontSize } from '../../utils/themeTypography';

interface NamespaceScorecardRollupProps {
  data: NamespaceScorecardResponse | null;
  loading?: boolean;
  onServiceClick: (service: string, environment?: string) => void;
}

/**
 * Namespace readiness rollup — ClickStack-native "team" view (namespace ≈ team)
 * without Console ownership metadata.
 */
export function NamespaceScorecardRollup({ data, loading, onServiceClick }: NamespaceScorecardRollupProps) {
  const styles = useStyles2(getStyles);
  if (loading || !data || data.total === 0) {
    return null;
  }
  const pct = data.total > 0 ? Math.round((100 * data.score) / data.total) : 0;
  const weak = data.services.filter((s) => s.total > 0 && s.score < s.total).slice(0, 5);

  return (
    <div className={styles.panel} data-testid="namespace-scorecard-rollup">
      <div className={styles.header}>
        <Icon name="check-circle" className={styles.icon} />
        <div>
          <div className={styles.title}>Observability readiness</div>
          <div className={styles.subtitle}>
            {data.score}/{data.total} checks across {data.services.length} service
            {data.services.length === 1 ? '' : 's'} ({pct}%)
            {data.truncated ? ` · scanned top ${data.scanned} by traffic` : ''}
          </div>
        </div>
      </div>
      {weak.length > 0 && (
        <ul className={styles.list}>
          {weak.map((s) => (
            <li key={`${s.service}/${s.environment ?? ''}`}>
              <button type="button" className={styles.link} onClick={() => onServiceClick(s.service, s.environment)}>
                {s.service}
                {s.environment ? ` (${s.environment})` : ''}
              </button>
              <span className={styles.score}>
                {s.score}/{s.total}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  panel: css`
    margin-bottom: ${theme.spacing(2)};
    padding: ${theme.spacing(1.5)};
    background: ${theme.colors.background.secondary};
    border: 1px solid ${theme.colors.border.weak};
    border-radius: ${theme.shape.radius.default};
  `,
  header: css`
    display: flex;
    gap: ${theme.spacing(1.5)};
    align-items: flex-start;
  `,
  icon: css`
    color: ${theme.colors.text.secondary};
    margin-top: ${theme.spacing(0.25)};
  `,
  title: css`
    font-weight: ${theme.typography.fontWeightMedium};
  `,
  subtitle: css`
    font-size: ${typographyFontSize(theme, 'sm')};
    color: ${theme.colors.text.secondary};
  `,
  list: css`
    list-style: none;
    margin: ${theme.spacing(1.5)} 0 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(0.5)};
  `,
  link: css`
    background: none;
    border: none;
    padding: 0;
    color: ${theme.colors.text.link};
    cursor: pointer;
    font: inherit;
    text-align: left;
    &:hover {
      text-decoration: underline;
    }
  `,
  score: css`
    margin-left: ${theme.spacing(1)};
    font-size: ${typographyFontSize(theme, 'sm')};
    color: ${theme.colors.text.secondary};
  `,
});
