import React from 'react';
import { css, keyframes } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { useStyles2 } from '@grafana/ui';

const pulse = keyframes`
  0% { opacity: 0.55; }
  50% { opacity: 1; }
  100% { opacity: 0.55; }
`;

interface SkeletonBarProps {
  width?: number | string;
  height?: number | string;
  className?: string;
}

/** Single shimmer bar — Grafana-like skeleton without a new dependency. */
export function SkeletonBar({ width = '100%', height = 14, className }: SkeletonBarProps) {
  const styles = useStyles2(getStyles);
  return <span className={`${styles.bar}${className ? ` ${className}` : ''}`} style={{ width, height }} aria-hidden />;
}

interface TableSkeletonProps {
  /** Number of placeholder rows. */
  rows?: number;
  /** Column flex/widths for each cell (defaults to a services-table-like layout). */
  columns?: Array<{ width: number | string }>;
}

/** Optimistic table body: header-free rows of skeleton cells. */
export function TableSkeleton({
  rows = 8,
  columns = [{ width: 32 }, { width: 72 }, { width: '1.4fr' }, { width: '1fr' }, { width: '1fr' }, { width: '1fr' }],
}: TableSkeletonProps) {
  const styles = useStyles2(getStyles);
  const template = columns.map((c) => (typeof c.width === 'number' ? `${c.width}px` : c.width)).join(' ');
  return (
    <div className={styles.table} role="status" aria-busy="true" aria-label="Loading">
      {Array.from({ length: rows }, (_, row) => (
        <div key={row} className={styles.row} style={{ gridTemplateColumns: template }}>
          {columns.map((col, i) => (
            <div key={i} className={styles.cell}>
              <SkeletonBar
                width={typeof col.width === 'number' ? Math.max(col.width - 8, 16) : `${55 + ((row + i) % 4) * 10}%`}
                height={i === 1 ? 22 : 14}
              />
            </div>
          ))}
        </div>
      ))}
    </div>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  bar: css`
    display: inline-block;
    border-radius: ${theme.shape.radius.default};
    background: ${theme.colors.background.secondary};
    animation: ${pulse} 1.4s ease-in-out infinite;
    vertical-align: middle;
  `,
  table: css`
    width: 100%;
    display: flex;
    flex-direction: column;
    border-top: 1px solid ${theme.colors.border.medium};
  `,
  row: css`
    display: grid;
    align-items: center;
    column-gap: ${theme.spacing(1.5)};
    padding: ${theme.spacing(1)} ${theme.spacing(1.5)};
    border-bottom: 1px solid ${theme.colors.border.weak};
  `,
  cell: css`
    min-width: 0;
    display: flex;
    align-items: center;
  `,
});
