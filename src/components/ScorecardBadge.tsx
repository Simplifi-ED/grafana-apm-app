import React, { useCallback, useMemo, useState } from 'react';
import { css, cx } from '@emotion/css';
import { AppEvents, GrafanaTheme2 } from '@grafana/data';
import { getAppEvents, locationService } from '@grafana/runtime';
import { Button, Icon, Toggletip, useStyles2 } from '@grafana/ui';
import { getScorecard, ScorecardCheck, ScorecardResponse } from '../api/scorecard';
import { buildAlertRuleUrl, getAlertTemplate } from '../api/client';
import { useFetch } from '../utils/useFetch';
import { typographyFontSize } from '../utils/themeTypography';
import { applyScorecardOverrides, getScorecardNaOverrides, setScorecardNaOverride } from '../utils/scorecardOverrides';

interface ScorecardBadgeProps {
  namespace: string;
  service: string;
  environment?: string;
}

/**
 * Compact observability-readiness badge for the service header: "4/6 observability
 * checks" colored by fraction. Clicking opens a toggletip listing each check with
 * an enablement hint. Renders nothing while loading or on error.
 */
export function ScorecardBadge({ namespace, service, environment }: ScorecardBadgeProps) {
  const styles = useStyles2(getStyles);
  const { data } = useFetch<ScorecardResponse>(
    () => getScorecard(namespace, service, environment),
    [namespace, service, environment]
  );
  const [overrideTick, setOverrideTick] = useState(0);
  const overrides = useMemo(() => {
    void overrideTick;
    return getScorecardNaOverrides(namespace, service, environment);
  }, [namespace, service, environment, overrideTick]);

  const display = useMemo(() => (data ? applyScorecardOverrides(data, overrides) : null), [data, overrides]);

  const onToggleNa = useCallback(
    (checkKey: string, notApplicable: boolean) => {
      setScorecardNaOverride(namespace, service, checkKey, notApplicable, environment);
      setOverrideTick((n) => n + 1);
    },
    [namespace, service, environment]
  );

  if (!display || !display.readiness || display.readiness.total === 0) {
    return null;
  }
  const { score, total } = display.readiness;
  const tone = score === total ? styles.good : score * 2 >= total ? styles.partial : styles.low;

  return (
    <Toggletip
      content={
        <ScorecardDetails
          data={display}
          namespace={namespace}
          service={service}
          environment={environment}
          overrides={overrides}
          onToggleNa={onToggleNa}
        />
      }
      placement="bottom"
    >
      <button type="button" className={cx(styles.badge, tone)} data-testid="scorecard-badge">
        <Icon name={score === total ? 'check-circle' : 'info-circle'} size="sm" />
        {score}/{total} observability checks
      </button>
    </Toggletip>
  );
}

function ScorecardDetails({
  data,
  namespace,
  service,
  environment,
  overrides,
  onToggleNa,
}: {
  data: ScorecardResponse;
  overrides: Set<string>;
  onToggleNa: (key: string, na: boolean) => void;
} & ScorecardBadgeProps) {
  const styles = useStyles2(getStyles);
  return (
    <div className={styles.details} data-testid="scorecard-details">
      <div className={styles.sectionTitle}>Observability readiness</div>
      <ul className={styles.checkList}>
        {data.readiness.checks.map((check) => {
          const manuallyNa = overrides.has(check.key);
          return (
            <li key={check.key} className={cx(styles.check, check.notApplicable && styles.checkNotApplicable)}>
              <span
                className={
                  check.notApplicable ? styles.checkIconNA : check.ok ? styles.checkIconOk : styles.checkIconMissing
                }
                role="img"
                aria-label={check.notApplicable ? 'Not applicable' : check.ok ? 'Enabled' : 'Not enabled'}
              >
                <Icon name={check.notApplicable ? 'minus-circle' : check.ok ? 'check-circle' : 'circle'} size="sm" />
              </span>
              <div>
                {check.label}
                {check.notApplicable && <span className={styles.naTag}>N/A</span>}
                {check.hint && (check.notApplicable || !check.ok) && (
                  <div className={styles.hint}>
                    {check.hint}
                    {!check.notApplicable && (
                      <CheckActions check={check} namespace={namespace} service={service} environment={environment} />
                    )}
                  </div>
                )}
                {manuallyNa && <div className={styles.hint}>Local override — excluded from score on this browser.</div>}
                {(!check.ok && !check.notApplicable) || manuallyNa ? (
                  <div className={styles.actions} data-testid={`check-na-${check.key}`}>
                    <Button
                      variant="secondary"
                      fill="text"
                      size="sm"
                      onClick={() => onToggleNa(check.key, !manuallyNa)}
                    >
                      {manuallyNa ? 'Clear N/A override' : 'Mark N/A'}
                    </Button>
                  </div>
                ) : null}
              </div>
            </li>
          );
        })}
      </ul>
    </div>
  );
}

/** Alert-rule shortcut for the alertRules check. */
function CheckActions({ check, namespace, service, environment }: { check: ScorecardCheck } & ScorecardBadgeProps) {
  const styles = useStyles2(getStyles);
  const [creatingAlert, setCreatingAlert] = useState(false);
  const canCreateAlert = !check.notApplicable && check.key === 'alertRules';

  const onCreateAlert = useCallback(async () => {
    setCreatingAlert(true);
    try {
      const template = await getAlertTemplate('error-rate', {
        namespace: namespace || undefined,
        service,
        environment: environment || undefined,
      });
      locationService.push(buildAlertRuleUrl(template.url));
    } catch (err) {
      setCreatingAlert(false);
      getAppEvents().publish({
        type: AppEvents.alertError.name,
        payload: ['Could not prepare alert rule', err instanceof Error ? err.message : String(err)],
      });
    }
  }, [namespace, service, environment]);

  if (!canCreateAlert) {
    return null;
  }
  return (
    <div className={styles.actions} data-testid={`check-actions-${check.key}`}>
      <Button variant="secondary" fill="text" size="sm" icon="bell" disabled={creatingAlert} onClick={onCreateAlert}>
        Alert on error rate
      </Button>
    </div>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  badge: css`
    display: inline-flex;
    align-items: center;
    gap: ${theme.spacing(0.5)};
    padding: 1px 8px;
    border-radius: ${theme.shape.radius.pill};
    font-size: ${typographyFontSize(theme, 'sm')};
    border: 1px solid transparent;
    background: none;
    cursor: pointer;
    vertical-align: middle;
  `,
  good: css`
    color: ${theme.colors.success.text};
    border-color: ${theme.colors.success.border};
    background-color: ${theme.colors.success.transparent};
  `,
  partial: css`
    color: ${theme.colors.warning.text};
    border-color: ${theme.colors.warning.border};
    background-color: ${theme.colors.warning.transparent};
  `,
  low: css`
    color: ${theme.colors.error.text};
    border-color: ${theme.colors.error.border};
    background-color: ${theme.colors.error.transparent};
  `,
  details: css`
    max-width: 360px;
    font-size: ${typographyFontSize(theme, 'sm')};
  `,
  sectionTitle: css`
    font-weight: ${theme.typography.fontWeightMedium};
    margin-bottom: ${theme.spacing(0.5)};
  `,
  checkList: css`
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(0.5)};
  `,
  check: css`
    display: flex;
    align-items: flex-start;
    gap: ${theme.spacing(1)};
  `,
  checkIconOk: css`
    color: ${theme.colors.success.text};
  `,
  checkIconMissing: css`
    color: ${theme.colors.text.disabled};
  `,
  checkIconNA: css`
    color: ${theme.colors.text.disabled};
  `,
  checkNotApplicable: css`
    color: ${theme.colors.text.secondary};
  `,
  naTag: css`
    margin-left: ${theme.spacing(0.5)};
    font-size: ${typographyFontSize(theme, 'sm')};
    color: ${theme.colors.text.disabled};
  `,
  hint: css`
    color: ${theme.colors.text.secondary};
  `,
  actions: css`
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: ${theme.spacing(1)};
    margin-top: ${theme.spacing(0.25)};
  `,
});
