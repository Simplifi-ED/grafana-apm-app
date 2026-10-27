import React, { useCallback, useEffect, useMemo, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';
import { RefreshPicker, defaultIntervals } from '@grafana/ui';
import { rangeUtil } from '@grafana/data';
import { useAutoRefresh } from '../utils/useInterval';
import { useUrlParams } from '../utils/useUrlState';

/** Off is an empty RefreshPicker value; stored as `refresh=off` when the default is not off. */
const OFF = '';
const OFF_PARAM = 'off';

interface RefreshControlProps {
  /** Called on every refresh tick and when the refresh button is clicked. */
  onRefresh: () => void;
  /**
   * Default interval written as a Grafana duration (`30s`, `1m`). Empty/off
   * by default — status boards pass `1m` so wall displays keep refreshing.
   */
  defaultInterval?: string;
  isLoading?: boolean;
}

/**
 * Native Grafana RefreshPicker for useFetch-based (non-Scene) pages.
 *
 * Persists the chosen interval in the URL (`refresh=30s`), consistent with
 * Grafana dashboards. Off by default (param removed). Includes the immediate
 * refresh button plus the interval dropdown — same control as Explore/dashboards.
 */
export function RefreshControl({ onRefresh, defaultInterval = OFF, isLoading }: RefreshControlProps) {
  const [searchParams] = useSearchParams();
  const setParams = useUrlParams();

  const refresh = useMemo(() => {
    const raw = searchParams.get('refresh');
    if (raw === null) {
      return defaultInterval;
    }
    if (raw === OFF_PARAM || raw === '') {
      return OFF;
    }
    return raw;
  }, [searchParams, defaultInterval]);

  const setRefresh = useCallback(
    (interval: string) => {
      if (!interval) {
        // Explicit off: omit when default is off; write refresh=off otherwise
        // so a non-off default (status boards) can actually be disabled.
        setParams({ refresh: defaultInterval === OFF ? null : OFF_PARAM });
        return;
      }
      if (interval === defaultInterval) {
        setParams({ refresh: null });
        return;
      }
      setParams({ refresh: interval });
    },
    [defaultInterval, setParams]
  );

  // Map URL value → ms. Accept Grafana durations (`30s`) and legacy numeric
  // ms strings from older status-board URLs (`60000`).
  const intervalMs = parseRefreshMs(refresh);

  // Pages recreate their refetch callbacks on every render; route through a
  // ref so an unstable onRefresh identity never restarts the interval timer.
  const onRefreshRef = useRef(onRefresh);
  useEffect(() => {
    onRefreshRef.current = onRefresh;
  });
  const stableOnRefresh = useCallback(() => onRefreshRef.current(), []);

  useAutoRefresh(stableOnRefresh, intervalMs);

  const value = intervalMs > 0 ? normalizeRefreshValue(refresh) : OFF;

  return (
    <div data-testid="refresh-control">
      <RefreshPicker
        value={value}
        intervals={defaultIntervals}
        onRefresh={stableOnRefresh}
        onIntervalChanged={(interval) => setRefresh(interval || OFF)}
        isLoading={isLoading}
        isOnCanvas
      />
    </div>
  );
}

/** Convert a URL refresh value to milliseconds; unknown/empty → 0 (off). */
function parseRefreshMs(raw: string): number {
  if (!raw || raw === OFF_PARAM) {
    return 0;
  }
  const asNum = Number(raw);
  if (Number.isFinite(asNum) && asNum > 0) {
    return asNum;
  }
  try {
    return rangeUtil.intervalToMs(raw);
  } catch {
    return 0;
  }
}

/**
 * Prefer a Grafana duration label in the picker. Legacy numeric ms URLs stay
 * functional via parseRefreshMs; display them as the matching default interval
 * when possible so the button label looks right.
 */
function normalizeRefreshValue(raw: string): string {
  if (!raw || raw === OFF_PARAM) {
    return OFF;
  }
  const asNum = Number(raw);
  if (Number.isFinite(asNum) && asNum > 0) {
    const match = defaultIntervals.find((label) => {
      try {
        return rangeUtil.intervalToMs(label) === asNum;
      } catch {
        return false;
      }
    });
    return match ?? raw;
  }
  return raw;
}
