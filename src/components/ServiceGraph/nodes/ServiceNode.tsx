import React, { memo, useMemo } from 'react';
import { Handle, Position, type NodeProps, type Node } from '@xyflow/react';
import { Icon, useStyles2, useTheme2 } from '@grafana/ui';
import { GrafanaTheme2, type IconName } from '@grafana/data';
import { css, cx } from '@emotion/css';

export interface ServiceNodeData {
  label: string;
  subtitle?: string;
  mainStat?: string;
  secondaryStat?: string;
  errorRate?: number;
  nodeType?: 'service' | 'database' | 'messaging' | 'external';
  isFocused?: boolean;
  dimmed?: boolean;
  isHub?: boolean;
  hubDegree?: number;
  callerCount?: number;
  [key: string]: unknown;
}

type ServiceNodeType = Node<ServiceNodeData>;

/** Outer diameter of the Tempo-style circle (matches Grafana Node Graph feel). */
export const NODE_CIRCLE_SIZE = 88;
/** Layout box includes title/subtitle under the circle. */
export const NODE_LAYOUT_WIDTH = 120;
export const NODE_LAYOUT_HEIGHT = 128;

const NODE_TYPE_CONFIG: Record<string, { icon: IconName; ring: string }> = {
  service: { icon: 'cube', ring: '#3871DC' },
  database: { icon: 'database', ring: '#336791' },
  messaging: { icon: 'envelope', ring: '#FF6600' },
  external: { icon: 'globe', ring: '#6E6E6E' },
};

export const ServiceNode = memo(({ data, sourcePosition, targetPosition }: NodeProps<ServiceNodeType>) => {
  const styles = useStyles2(getStyles);
  const theme = useTheme2();
  const nodeType = data.nodeType ?? 'service';
  const config = NODE_TYPE_CONFIG[nodeType] ?? NODE_TYPE_CONFIG.service;
  const errorRate = Math.min(1, Math.max(0, data.errorRate ?? 0));
  const isHub = data.isHub ?? false;
  const okPct = (1 - errorRate) * 100;
  const okColor = theme.isDark ? '#6CCF8E' : '#3AA76D';
  const errColor = theme.colors.error.main;

  const title = useMemo(() => {
    if (isHub && data.hubDegree) {
      return `Shared hub · ${data.hubDegree} connections`;
    }
    return data.subtitle || undefined;
  }, [isHub, data.hubDegree, data.subtitle]);

  return (
    <div className={cx(styles.wrap, data.isFocused && styles.focused, data.dimmed && styles.dimmed)}>
      <Handle type="target" position={targetPosition ?? Position.Top} className={styles.handle} />
      <div
        className={styles.circle}
        style={{
          background: `conic-gradient(${okColor} 0 ${okPct}%, ${errColor} ${okPct}% 100%)`,
        }}
        title={`${data.label}${errorRate > 0 ? ` · ${(errorRate * 100).toFixed(1)}% errors` : ''}`}
      >
        <div className={styles.inner}>
          {(data.mainStat || data.secondaryStat) && (
            <>
              {data.mainStat && <span className={styles.mainStat}>{data.mainStat}</span>}
              {data.secondaryStat && <span className={styles.secondaryStat}>{data.secondaryStat}</span>}
            </>
          )}
          {!data.mainStat && !data.secondaryStat && (
            <span className={styles.typeIcon} style={{ color: config.ring }}>
              <Icon name={isHub ? 'exchange-alt' : config.icon} size="lg" />
            </span>
          )}
        </div>
      </div>
      <div className={styles.caption}>
        <span className={styles.label} title={data.label}>
          {data.label}
        </span>
        {title && (
          <span className={styles.subtitle} title={title}>
            {title}
          </span>
        )}
      </div>
      <Handle type="source" position={sourcePosition ?? Position.Bottom} className={styles.handle} />
    </div>
  );
});

ServiceNode.displayName = 'ServiceNode';

const getStyles = (theme: GrafanaTheme2) => ({
  wrap: css`
    display: flex;
    flex-direction: column;
    align-items: center;
    width: ${NODE_LAYOUT_WIDTH}px;
    cursor: pointer;
    transition: opacity 0.15s ease;
  `,
  focused: css`
    filter: drop-shadow(0 0 6px ${theme.colors.primary.main});
  `,
  dimmed: css`
    opacity: 0.45;
  `,
  circle: css`
    position: relative;
    width: ${NODE_CIRCLE_SIZE}px;
    height: ${NODE_CIRCLE_SIZE}px;
    border-radius: 50%;
    padding: 5px;
    box-sizing: border-box;
    box-shadow: ${theme.shadows.z1};
    flex-shrink: 0;
  `,
  inner: css`
    width: 100%;
    height: 100%;
    border-radius: 50%;
    background: ${theme.colors.background.primary};
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 1px;
    padding: 4px;
    text-align: center;
  `,
  mainStat: css`
    font-size: 11px;
    font-weight: ${theme.typography.fontWeightMedium};
    color: ${theme.colors.text.primary};
    font-variant-numeric: tabular-nums;
    line-height: 1.15;
    max-width: 64px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  `,
  secondaryStat: css`
    font-size: 9px;
    color: ${theme.colors.text.secondary};
    font-variant-numeric: tabular-nums;
    line-height: 1.15;
    max-width: 64px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  `,
  typeIcon: css`
    display: inline-flex;
    opacity: 0.85;
  `,
  caption: css`
    margin-top: 6px;
    display: flex;
    flex-direction: column;
    align-items: center;
    max-width: ${NODE_LAYOUT_WIDTH}px;
    min-width: 0;
  `,
  label: css`
    font-size: 11px;
    font-weight: ${theme.typography.fontWeightMedium};
    color: ${theme.colors.text.primary};
    text-align: center;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 100%;
  `,
  subtitle: css`
    font-size: 10px;
    color: ${theme.colors.text.secondary};
    text-align: center;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 100%;
    line-height: 1.2;
  `,
  handle: css`
    width: 6px !important;
    height: 6px !important;
    opacity: 0 !important;
    pointer-events: none !important;
  `,
});
