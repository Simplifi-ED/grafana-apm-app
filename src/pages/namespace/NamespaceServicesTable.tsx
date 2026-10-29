import React, { useMemo } from 'react';
import { Icon, Input, MultiCombobox, Pagination, RadioButtonGroup, useStyles2, useTheme2 } from '@grafana/ui';
import { GrafanaTheme2 } from '@grafana/data';
import { css } from '@emotion/css';
import { ServiceSummary } from '../../api/client';
import { Ownership } from '../../api/ownership';
import { formatDuration } from '../../utils/format';
import { useTableSort, SortHeader, getTableStyles } from '../../components/SortableTable';
import { FrameworkBadge } from '../../components/FrameworkBadge';
import { Sparkline } from '../../components/Sparkline';
import { sparklineColors } from '../../utils/colors';
import { getServiceHealth, deltaArrow } from '../../utils/health';
import { HealthIndicator } from '../../components/HealthIndicator';
import { typographyFontSize } from '../../utils/themeTypography';

const PAGE_SIZE = 10;

type SortField = 'name' | 'rate' | 'errorRate' | 'p95Duration' | 'team';

const HEALTH_FILTER_OPTIONS = [
  { label: 'All', value: '' },
  { label: 'Unhealthy only', value: 'unhealthy' },
];

interface NamespaceServicesTableProps {
  services: ServiceSummary[];
  sparklineMap?: Map<string, ServiceSummary>;
  previousMap?: Map<string, ServiceSummary>;
  ownershipByService?: Record<string, Ownership>;
  teamFilters?: string[];
  /** When true, team filter is disabled (ownership scan capped). */
  ownershipTruncated?: boolean;
  onTeamFiltersChange?: (teams: string[]) => void;
  showEnvironment?: boolean;
  search: string;
  healthFilter?: string;
  page: number;
  onSearchChange: (q: string) => void;
  onHealthFilterChange: (filter: string) => void;
  onPageChange: (p: number) => void;
  onServiceClick: (namespace: string, service: string, environment?: string) => void;
}

export function NamespaceServicesTable({
  services,
  sparklineMap,
  previousMap,
  ownershipByService,
  teamFilters = [],
  ownershipTruncated,
  onTeamFiltersChange,
  showEnvironment,
  search,
  healthFilter,
  page,
  onSearchChange,
  onHealthFilterChange,
  onPageChange,
  onServiceClick,
}: NamespaceServicesTableProps) {
  const tableStyles = useStyles2(getTableStyles);
  const styles = useStyles2(getLocalStyles);
  const theme = useTheme2();
  const sc = sparklineColors(theme);
  const { sortField, sortDir, toggleSort, comparator } = useTableSort<SortField>('rate');

  const teamOptions = useMemo(() => {
    const teams = new Set<string>();
    for (const own of Object.values(ownershipByService ?? {})) {
      if (own.team) {
        teams.add(own.team);
      }
    }
    return [...teams].sort().map((t) => ({ label: t, value: t }));
  }, [ownershipByService]);

  const filtered = useMemo(() => {
    let result = services;
    if (search) {
      const q = search.toLowerCase();
      result = result.filter((s) => s.name.toLowerCase().includes(q));
    }
    if (healthFilter === 'unhealthy') {
      result = result.filter((s) => getServiceHealth(s.errorRate, s.p95Duration, s.durationUnit) !== 'healthy');
    }
    if (teamFilters.length > 0) {
      result = result.filter((s) => {
        const team = ownershipByService?.[s.name]?.team;
        return team != null && teamFilters.includes(team);
      });
    }
    const sorted = [...result];
    if (sortField === 'team') {
      sorted.sort((a, b) => {
        const ta = ownershipByService?.[a.name]?.team ?? '';
        const tb = ownershipByService?.[b.name]?.team ?? '';
        const cmp = ta.localeCompare(tb);
        return sortDir === 'asc' ? cmp : -cmp;
      });
      return sorted;
    }
    return sorted.sort(comparator);
  }, [services, search, healthFilter, teamFilters, ownershipByService, comparator, sortField, sortDir]);

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const safePage = Math.min(page, totalPages);
  const paginated = filtered.slice((safePage - 1) * PAGE_SIZE, safePage * PAGE_SIZE);

  return (
    <div>
      <div className={styles.toolbar}>
        <Input
          prefix={<Icon name="search" />}
          placeholder="Filter services…"
          value={search}
          onChange={(e) => {
            onSearchChange(e.currentTarget.value);
          }}
          width={28}
        />
        <RadioButtonGroup
          options={HEALTH_FILTER_OPTIONS}
          value={healthFilter ?? ''}
          onChange={onHealthFilterChange}
          size="sm"
        />
        {ownershipTruncated && <span className={styles.count}>Team filter unavailable (ownership scan truncated)</span>}
        {onTeamFiltersChange && !ownershipTruncated && (teamOptions.length > 0 || teamFilters.length > 0) && (
          <MultiCombobox
            options={teamOptions.length > 0 ? teamOptions : teamFilters.map((t) => ({ label: t, value: t }))}
            value={teamFilters}
            onChange={(selected) => onTeamFiltersChange(selected.map((o) => o.value).filter(Boolean) as string[])}
            placeholder="All teams"
            width={24}
          />
        )}
        <span className={styles.count}>
          {filtered.length} {filtered.length === 1 ? 'service' : 'services'}
        </span>
      </div>
      <table className={tableStyles.table}>
        <colgroup>
          <col style={{ width: showEnvironment ? '28%' : '32%' }} />
          {showEnvironment && <col style={{ width: '8%' }} />}
          <col style={{ width: '12%' }} />
          <col style={{ width: '10%' }} />
          <col style={{ width: '10%' }} />
          <col style={{ width: '10%' }} />
          <col style={{ width: showEnvironment ? '22%' : '26%' }} />
        </colgroup>
        <thead>
          <tr>
            <SortHeader field="name" label="Service" sortField={sortField} sortDir={sortDir} onSort={toggleSort} />
            {showEnvironment && <th>Environment</th>}
            <SortHeader field="team" label="Team" sortField={sortField} sortDir={sortDir} onSort={toggleSort} />
            <SortHeader field="rate" label="Rate" sortField={sortField} sortDir={sortDir} onSort={toggleSort} />
            <SortHeader field="errorRate" label="Error %" sortField={sortField} sortDir={sortDir} onSort={toggleSort} />
            <SortHeader field="p95Duration" label="P95" sortField={sortField} sortDir={sortDir} onSort={toggleSort} />
            <th style={{ textAlign: 'right' }}>Trend</th>
          </tr>
        </thead>
        <tbody>
          {paginated.map((svc) => {
            const sparkKey = `${svc.namespace}/${svc.name}/${svc.environment ?? ''}`;
            const spark = sparklineMap?.get(sparkKey);
            const prev = previousMap?.get(sparkKey);
            const health = getServiceHealth(svc.errorRate, svc.p95Duration, svc.durationUnit);
            const errArrow = deltaArrow(svc.errorRate, prev?.errorRate);
            const p95Arrow = deltaArrow(svc.p95Duration, prev?.p95Duration);
            const team = ownershipByService?.[svc.name]?.team;

            return (
              <tr
                key={`${svc.name}/${svc.environment ?? ''}`}
                className={tableStyles.clickableRow}
                onClick={() => onServiceClick(svc.namespace, svc.name, svc.environment)}
              >
                <td className={tableStyles.nameCell}>
                  <div className={tableStyles.nameCellInner}>
                    <HealthIndicator status={health} />
                    <FrameworkBadge framework={svc.framework} className={styles.badgeBefore} />
                    <span>{svc.name}</span>
                  </div>
                </td>
                {showEnvironment && <td>{svc.environment ?? '—'}</td>}
                <td className={styles.teamCell}>{team || '—'}</td>
                <td className={tableStyles.numCell}>{svc.rate.toFixed(2)} req/s</td>
                <td className={svc.errorRate > 0 ? tableStyles.errorCell : tableStyles.numCell}>
                  {svc.errorRate.toFixed(2)}%{errArrow && <span className={styles.arrow}> {errArrow}</span>}
                </td>
                <td className={tableStyles.numCell}>
                  {formatDuration(svc.p95Duration, svc.durationUnit)}
                  {p95Arrow && <span className={styles.arrow}> {p95Arrow}</span>}
                </td>
                <td style={{ textAlign: 'right' }}>
                  <div className={styles.sparkContainer}>
                    <Sparkline
                      data={spark?.durationSeries?.map((p) => p.v)}
                      color={sc.duration}
                      width={60}
                      height={20}
                    />
                    <Sparkline data={spark?.rateSeries?.map((p) => p.v)} color={sc.rate} width={60} height={20} />
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
      {totalPages > 1 && (
        <div className={styles.pagination}>
          <Pagination currentPage={safePage} numberOfPages={totalPages} onNavigate={onPageChange} />
        </div>
      )}
    </div>
  );
}

const getLocalStyles = (theme: GrafanaTheme2) => ({
  badgeBefore: css`
    vertical-align: middle;
  `,
  arrow: css`
    color: ${theme.colors.text.secondary};
    font-size: ${typographyFontSize(theme, 'sm')};
  `,
  sparkContainer: css`
    display: flex;
    gap: ${theme.spacing(0.5)};
    justify-content: flex-end;
  `,
  toolbar: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1.5)};
    margin-bottom: ${theme.spacing(1.5)};
  `,
  count: css`
    color: ${theme.colors.text.secondary};
    font-size: ${typographyFontSize(theme, 'sm')};
  `,
  teamCell: css`
    color: ${theme.colors.text.secondary};
    font-size: ${typographyFontSize(theme, 'sm')};
  `,
  pagination: css`
    margin-top: ${theme.spacing(1.5)};
    display: flex;
    justify-content: flex-end;
  `,
});
