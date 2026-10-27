import React, { useState } from 'react';
import { Button, Combobox, IconButton, Input, useStyles2 } from '@grafana/ui';
import { GrafanaTheme2 } from '@grafana/data';
import { css } from '@emotion/css';
import { CollapsibleSection } from '../../../components/CollapsibleSection';
import { ExploreFieldFilter } from '../../../api/analytics';

export const LOG_FIELD_KEYS = [
  'k8s.pod.name',
  'k8s.namespace.name',
  'host.name',
  'exception.type',
  'exception.message',
] as const;

export const TRACE_FIELD_KEYS = [
  'http.route',
  'http.method',
  'http.status_code',
  'db.system',
  'db.operation.name',
  'rpc.method',
  'k8s.pod.name',
  'k8s.namespace.name',
] as const;

interface ClickStackExploreFiltersProps {
  fieldKeys: readonly string[];
  fields: ExploreFieldFilter[];
  onFieldsChange: (fields: ExploreFieldFilter[]) => void;
  sqlWhere: string;
  onSqlWhereChange: (sql: string) => void;
  sqlHint: string;
}

/**
 * Nais-like field chips + collapsed Advanced SQL hatch for ClickStack Explore.
 * Free-text / severity|status stay on the parent tab filter bar.
 */
export function ClickStackExploreFilters({
  fieldKeys,
  fields,
  onFieldsChange,
  sqlWhere,
  onSqlWhereChange,
  sqlHint,
}: ClickStackExploreFiltersProps) {
  const styles = useStyles2(getStyles);
  const [draftKey, setDraftKey] = useState(fieldKeys[0] ?? '');
  const [draftValue, setDraftValue] = useState('');
  const [draftSql, setDraftSql] = useState(sqlWhere);

  const usedKeys = new Set(fields.map((f) => f.key));
  const keyOptions = fieldKeys.filter((k) => !usedKeys.has(k) || k === draftKey).map((k) => ({ label: k, value: k }));

  const addChip = () => {
    const key = draftKey.trim();
    const value = draftValue.trim();
    if (!key || !value || usedKeys.has(key)) {
      return;
    }
    onFieldsChange([...fields, { key, value }]);
    setDraftValue('');
    const next = fieldKeys.find((k) => k !== key && !usedKeys.has(k) && k !== key);
    if (next) {
      setDraftKey(next);
    }
  };

  const applySql = () => onSqlWhereChange(draftSql.trim());

  return (
    <div className={styles.wrap}>
      <div className={styles.chipRow}>
        <label className={styles.label}>Fields:</label>
        {fields.map((f) => (
          <span key={f.key} className={styles.chip}>
            {f.key}={f.value}
            <IconButton
              name="times"
              size="sm"
              tooltip="Remove filter"
              onClick={() => onFieldsChange(fields.filter((x) => x.key !== f.key))}
            />
          </span>
        ))}
        {keyOptions.length > 0 && (
          <>
            <Combobox options={keyOptions} value={draftKey} onChange={(v) => setDraftKey(v?.value ?? '')} width={22} />
            <Input
              placeholder="value"
              width={16}
              value={draftValue}
              onChange={(e) => setDraftValue(e.currentTarget.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault();
                  addChip();
                }
              }}
            />
            <Button size="sm" variant="secondary" onClick={addChip} disabled={!draftValue.trim()}>
              Add
            </Button>
          </>
        )}
      </div>

      <CollapsibleSection label="Advanced SQL">
        <div className={styles.sqlBody}>
          <p className={styles.hint}>{sqlHint}</p>
          <Input
            placeholder="e.g. SeverityText = 'error'"
            value={draftSql}
            onChange={(e) => setDraftSql(e.currentTarget.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault();
                applySql();
              }
            }}
          />
          <div className={styles.sqlActions}>
            <Button size="sm" variant="secondary" onClick={applySql}>
              Apply SQL
            </Button>
            {sqlWhere && (
              <Button
                size="sm"
                variant="secondary"
                fill="outline"
                onClick={() => {
                  setDraftSql('');
                  onSqlWhereChange('');
                }}
              >
                Clear
              </Button>
            )}
          </div>
        </div>
      </CollapsibleSection>
    </div>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  wrap: css`
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(1)};
    margin-bottom: ${theme.spacing(1.5)};
  `,
  chipRow: css`
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: ${theme.spacing(1)};
  `,
  label: css`
    color: ${theme.colors.text.secondary};
    font-size: ${theme.typography.sm.fontSize};
  `,
  chip: css`
    display: inline-flex;
    align-items: center;
    gap: ${theme.spacing(0.5)};
    padding: ${theme.spacing(0.25, 0.75)};
    border-radius: ${theme.shape.radius.default};
    background: ${theme.colors.background.secondary};
    border: 1px solid ${theme.colors.border.weak};
    font-family: ${theme.typography.fontFamilyMonospace};
    font-size: ${theme.typography.sm.fontSize};
  `,
  sqlBody: css`
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(1)};
    padding: ${theme.spacing(0, 1.5, 1.5)};
  `,
  hint: css`
    margin: 0;
    color: ${theme.colors.text.secondary};
    font-size: ${theme.typography.sm.fontSize};
  `,
  sqlActions: css`
    display: flex;
    gap: ${theme.spacing(1)};
  `,
});
