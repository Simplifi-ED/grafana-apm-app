import React, { useMemo } from 'react';
import { render, renderHook } from '@testing-library/react';
import { useFrozenWhileHidden } from './useFrozenWhileHidden';

describe('useFrozenWhileHidden', () => {
  it('tracks the latest value while active', () => {
    const { result, rerender } = renderHook(
      ({ value, active }: { value: string; active: boolean }) => useFrozenWhileHidden(value, active),
      { initialProps: { value: 'a', active: true } }
    );
    expect(result.current).toBe('a');
    rerender({ value: 'b', active: true });
    expect(result.current).toBe('b');
  });

  it('freezes the last active value while hidden', () => {
    const { result, rerender } = renderHook(
      ({ value, active }: { value: string; active: boolean }) => useFrozenWhileHidden(value, active),
      { initialProps: { value: 'a', active: true } }
    );
    rerender({ value: 'b', active: false });
    expect(result.current).toBe('a');
    rerender({ value: 'c', active: false });
    expect(result.current).toBe('a');
  });

  it('catches up when reactivated', () => {
    const { result, rerender } = renderHook(
      ({ value, active }: { value: string; active: boolean }) => useFrozenWhileHidden(value, active),
      { initialProps: { value: 'a', active: true } }
    );
    rerender({ value: 'b', active: false });
    rerender({ value: 'c', active: true });
    expect(result.current).toBe('c');
  });

  it('does not re-invoke a memoized scene builder while hidden', () => {
    const builds: number[] = [];
    function Probe({ range, active }: { range: number; active: boolean }) {
      const frozen = useFrozenWhileHidden(range, active);
      useMemo(() => {
        builds.push(frozen);
        return frozen;
      }, [frozen]);
      return null;
    }
    const { rerender } = render(<Probe range={1} active={true} />);
    expect(builds).toEqual([1]);
    rerender(<Probe range={2} active={false} />);
    rerender(<Probe range={3} active={false} />);
    expect(builds).toEqual([1]);
    rerender(<Probe range={4} active={true} />);
    expect(builds).toEqual([1, 4]);
  });
});
