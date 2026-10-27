import React, { useCallback, useMemo } from 'react';
import { TimeRangePicker } from '@grafana/ui';
import { rangeUtil, isDateTime, DateTime, TimeRange } from '@grafana/data';
import { TimeZone } from '@grafana/schema';

/** Grafana raw range endpoints are DateTime (absolute) or string (relative). */
export function rawToUrlValue(v: DateTime | string): string {
  return isDateTime(v) ? v.toISOString() : v;
}

export interface AppTimeRangePickerProps {
  from: string;
  to: string;
  fromMs: number;
  toMs: number;
  onChange: (from: string, to: string) => void;
  /**
   * Optional max window length. Absolute picks and move/zoom that exceed the
   * cap are clamped to end at `to` with `from = to - maxRangeMs`.
   */
  maxRangeMs?: number;
}

/**
 * Native Grafana TimeRangePicker wired to string from/to (+ resolved ms for
 * move/zoom). Used by HeaderTimeControls (URL) and ServiceInventory (local).
 */
export function AppTimeRangePicker({ from, to, fromMs, toMs, onChange, maxRangeMs }: AppTimeRangePickerProps) {
  // Re-resolve on every fromMs/toMs change so a refresh tick keeps the picker's
  // resolved value current for relative ranges (the raw text is unchanged).
  const value: TimeRange = useMemo(
    () => rangeUtil.convertRawToRange({ from, to }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [from, to, fromMs, toMs]
  );

  const apply = useCallback(
    (nextFrom: string, nextTo: string) => {
      if (maxRangeMs == null) {
        onChange(nextFrom, nextTo);
        return;
      }
      const resolved = rangeUtil.convertRawToRange({ from: nextFrom, to: nextTo });
      const span = resolved.to.valueOf() - resolved.from.valueOf();
      if (span <= maxRangeMs) {
        onChange(nextFrom, nextTo);
        return;
      }
      // Clamp: keep the end of the window, freeze as absolute ISO.
      const clampedTo = resolved.to.toISOString();
      const clampedFrom = new Date(resolved.to.valueOf() - maxRangeMs).toISOString();
      onChange(clampedFrom, clampedTo);
    },
    [maxRangeMs, onChange]
  );

  const handleChange = useCallback(
    (next: TimeRange) => {
      apply(rawToUrlValue(next.raw.from), rawToUrlValue(next.raw.to));
    },
    [apply]
  );

  // Move/zoom operate on the resolved window and freeze it as absolute ISO.
  const shift = useCallback(
    (deltaSpans: number) => {
      const span = toMs - fromMs;
      apply(new Date(fromMs + deltaSpans * span).toISOString(), new Date(toMs + deltaSpans * span).toISOString());
    },
    [fromMs, toMs, apply]
  );

  const onMoveBackward = useCallback(() => shift(-1), [shift]);
  const onMoveForward = useCallback(() => shift(1), [shift]);

  const onZoom = useCallback(() => {
    const span = toMs - fromMs;
    const mid = (fromMs + toMs) / 2;
    apply(new Date(mid - span).toISOString(), new Date(mid + span).toISOString());
  }, [fromMs, toMs, apply]);

  // The app does not track a time zone in the URL; the picker renders in the
  // viewer's browser zone. Accept the callback to satisfy the API, no-op it.
  const onChangeTimeZone = useCallback((_tz: TimeZone) => {}, []);

  return (
    <TimeRangePicker
      value={value}
      onChange={handleChange}
      onChangeTimeZone={onChangeTimeZone}
      onMoveBackward={onMoveBackward}
      onMoveForward={onMoveForward}
      onZoom={onZoom}
    />
  );
}
