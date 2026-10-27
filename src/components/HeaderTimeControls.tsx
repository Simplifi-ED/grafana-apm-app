import React, { useCallback } from 'react';
import { useTimeRange } from '../utils/timeRange';
import { useUrlParams } from '../utils/useUrlState';
import { AppTimeRangePicker } from './AppTimeRangePicker';
import { RefreshControl } from './RefreshControl';

export interface HeaderTimeControlsProps {
  /**
   * Called on refresh button / auto-refresh. Defaults to re-resolving the
   * shared relative time range. Pass a page handler when absolute ranges also
   * need an explicit refetch (useFetch pages).
   */
  onRefresh?: () => void;
  /** Default auto-refresh interval (`1m` for status boards). Off when omitted. */
  defaultRefresh?: string;
  isLoading?: boolean;
}

/**
 * The one universal time control: Grafana TimeRangePicker + RefreshPicker,
 * both wired to the shared from/to URL params (utils/timeRange). Replaces the
 * per-tab SceneTimePicker/SceneRefreshPicker and the Combobox quick-range
 * pickers on list pages so every view shares one window.
 *
 * Relative expressions (now-1h) are stored verbatim in the URL — exactly as
 * useTimeRange already persists them — so a refresh re-resolves them to fresh
 * timestamps. Absolute picks and move/zoom freeze the window as ISO strings.
 * Every write goes through useUrlParams so from+to land in ONE atomic URL
 * transaction (one history entry).
 */
export function HeaderTimeControls({ onRefresh, defaultRefresh, isLoading }: HeaderTimeControlsProps) {
  const { from, to, fromMs, toMs, refresh } = useTimeRange();
  const setParams = useUrlParams();

  const onChange = useCallback(
    (nextFrom: string, nextTo: string) => {
      setParams({ from: nextFrom, to: nextTo });
    },
    [setParams]
  );

  return (
    <>
      <AppTimeRangePicker from={from} to={to} fromMs={fromMs} toMs={toMs} onChange={onChange} />
      <RefreshControl onRefresh={onRefresh ?? refresh} defaultInterval={defaultRefresh} isLoading={isLoading} />
    </>
  );
}
