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

interface CardGridSkeletonProps {
  cards?: number;
  /** Min card width for auto-fill grid (px). */
  minCardWidth?: number;
  cardHeight?: number;
}

/** Optimistic card grid (Ops Status Board). */
export function CardGridSkeleton({ cards = 8, minCardWidth = 220, cardHeight = 140 }: CardGridSkeletonProps) {
  const styles = useStyles2(getStyles);
  return (
    <div
      className={styles.cardGrid}
      style={{ gridTemplateColumns: `repeat(auto-fill, minmax(${minCardWidth}px, 1fr))` }}
      role="status"
      aria-busy="true"
      aria-label="Loading"
    >
      {Array.from({ length: cards }, (_, i) => (
        <div key={i} className={styles.card} style={{ minHeight: cardHeight }}>
          <SkeletonBar width="55%" height={16} />
          <SkeletonBar width="35%" height={12} />
          <div className={styles.cardMetrics}>
            <SkeletonBar width="30%" height={22} />
            <SkeletonBar width="30%" height={22} />
            <SkeletonBar width="30%" height={22} />
          </div>
          <SkeletonBar width="100%" height={28} />
        </div>
      ))}
    </div>
  );
}

interface PanelSkeletonProps {
  height?: number | string;
  className?: string;
}

/** Large content panel placeholder (service map / graph canvas). */
export function PanelSkeleton({ height = 'min(60vh, 520px)', className }: PanelSkeletonProps) {
  const styles = useStyles2(getStyles);
  return (
    <div
      className={`${styles.panel}${className ? ` ${className}` : ''}`}
      style={{ height }}
      role="status"
      aria-busy="true"
      aria-label="Loading"
    >
      <SkeletonBar width={160} height={14} className={styles.panelHint} />
      <div className={styles.panelNodes}>
        <SkeletonBar width={88} height={48} />
        <SkeletonBar width={88} height={48} />
        <SkeletonBar width={88} height={48} />
        <SkeletonBar width={88} height={48} />
        <SkeletonBar width={88} height={48} />
        <SkeletonBar width={88} height={48} />
      </div>
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
  cardGrid: css`
    display: grid;
    gap: ${theme.spacing(2)};
    width: 100%;
    padding: ${theme.spacing(0.5)};
  `,
  card: css`
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(1.5)};
    padding: ${theme.spacing(2)};
    border: 1px solid ${theme.colors.border.weak};
    border-radius: ${theme.shape.radius.default};
    background: ${theme.colors.background.primary};
  `,
  cardMetrics: css`
    display: flex;
    gap: ${theme.spacing(1)};
    justify-content: space-between;
  `,
  panel: css`
    position: relative;
    width: 100%;
    border: 1px solid ${theme.colors.border.weak};
    border-radius: ${theme.shape.radius.default};
    background: ${theme.colors.background.primary};
    overflow: hidden;
  `,
  panelHint: css`
    position: absolute;
    top: ${theme.spacing(2)};
    left: ${theme.spacing(2)};
  `,
  panelNodes: css`
    position: absolute;
    inset: 0;
    display: flex;
    flex-wrap: wrap;
    align-content: center;
    justify-content: center;
    gap: ${theme.spacing(4)} ${theme.spacing(6)};
    padding: ${theme.spacing(6)};
  `,
});
