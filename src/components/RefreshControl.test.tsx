import React from 'react';
import { render, screen, fireEvent, act } from '@testing-library/react';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { RefreshControl } from './RefreshControl';

// Grafana's RefreshPicker needs canvas/portal bits jsdom can't drive. Swap in a
// minimal stand-in that exposes the refresh button + interval <select> so
// tests exercise URL wiring and the auto-refresh timer.
jest.mock('@grafana/ui', () => {
  const actual = jest.requireActual('@grafana/ui');
  return {
    ...actual,
    RefreshPicker: (props: {
      value?: string;
      intervals?: string[];
      onRefresh?: () => void;
      onIntervalChanged: (interval: string) => void;
    }) => (
      <div>
        <button type="button" aria-label="Refresh" onClick={() => props.onRefresh?.()}>
          Refresh
        </button>
        <select
          aria-label="Refresh interval"
          value={props.value ?? ''}
          onChange={(e) => props.onIntervalChanged(e.currentTarget.value)}
        >
          <option value="">Off</option>
          {(props.intervals ?? ['30s', '1m', '5m']).map((o) => (
            <option key={o} value={o}>
              {o}
            </option>
          ))}
        </select>
      </div>
    ),
    defaultIntervals: ['5s', '10s', '30s', '1m', '5m', '15m', '30m', '1h', '2h', '1d'],
  };
});

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="search">{location.search}</div>;
}

function renderControl(onRefresh: jest.Mock, initialEntry = '/', defaultInterval?: string) {
  return render(
    <MemoryRouter initialEntries={[initialEntry]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <RefreshControl onRefresh={onRefresh} defaultInterval={defaultInterval} />
      <LocationProbe />
    </MemoryRouter>
  );
}

describe('RefreshControl', () => {
  afterEach(() => {
    jest.useRealTimers();
  });

  it('is off by default: no URL param, and never fires onRefresh', () => {
    jest.useFakeTimers();
    const onRefresh = jest.fn();
    renderControl(onRefresh);

    expect(screen.getByRole('combobox')).toHaveValue('');
    expect(screen.getByTestId('search')).toHaveTextContent('');

    act(() => {
      jest.advanceTimersByTime(600000);
    });
    expect(onRefresh).not.toHaveBeenCalled();
  });

  it('persists the selected interval to the URL as refresh=30s', () => {
    const onRefresh = jest.fn();
    renderControl(onRefresh);

    fireEvent.change(screen.getByRole('combobox'), { target: { value: '30s' } });
    expect(screen.getByTestId('search')).toHaveTextContent('refresh=30s');
  });

  it('removes the URL param when switched back to off', () => {
    const onRefresh = jest.fn();
    renderControl(onRefresh, '/?refresh=1m');

    fireEvent.change(screen.getByRole('combobox'), { target: { value: '' } });
    expect(screen.getByTestId('search')).not.toHaveTextContent('refresh');
  });

  it('writes refresh=off when disabling a non-off default interval', () => {
    const onRefresh = jest.fn();
    renderControl(onRefresh, '/', '1m');

    fireEvent.change(screen.getByRole('combobox'), { target: { value: '' } });
    expect(screen.getByTestId('search')).toHaveTextContent('refresh=off');
  });

  it('fires onRefresh on every tick when an interval is active', () => {
    jest.useFakeTimers();
    const onRefresh = jest.fn();
    renderControl(onRefresh, '/?refresh=30s');

    act(() => {
      jest.advanceTimersByTime(30000);
    });
    expect(onRefresh).toHaveBeenCalledTimes(1);

    act(() => {
      jest.advanceTimersByTime(30000);
    });
    expect(onRefresh).toHaveBeenCalledTimes(2);
  });

  it('fires onRefresh when the refresh button is clicked', () => {
    const onRefresh = jest.fn();
    renderControl(onRefresh);
    fireEvent.click(screen.getByRole('button', { name: 'Refresh' }));
    expect(onRefresh).toHaveBeenCalledTimes(1);
  });

  it('always invokes the latest onRefresh even when the prop identity changes', () => {
    jest.useFakeTimers();
    const first = jest.fn();
    const second = jest.fn();
    const { rerender } = render(
      <MemoryRouter
        initialEntries={['/?refresh=30s']}
        future={{ v7_startTransition: true, v7_relativeSplatPath: true }}
      >
        <RefreshControl onRefresh={first} />
      </MemoryRouter>
    );

    rerender(
      <MemoryRouter
        initialEntries={['/?refresh=30s']}
        future={{ v7_startTransition: true, v7_relativeSplatPath: true }}
      >
        <RefreshControl onRefresh={second} />
      </MemoryRouter>
    );

    act(() => {
      jest.advanceTimersByTime(30000);
    });
    expect(first).not.toHaveBeenCalled();
    expect(second).toHaveBeenCalledTimes(1);
  });

  it('treats an unknown refresh value in the URL as off', () => {
    jest.useFakeTimers();
    const onRefresh = jest.fn();
    renderControl(onRefresh, '/?refresh=bogus');

    expect(screen.getByRole('combobox')).toHaveValue('');
    act(() => {
      jest.advanceTimersByTime(600000);
    });
    expect(onRefresh).not.toHaveBeenCalled();
  });

  it('accepts legacy numeric ms refresh values from status-board URLs', () => {
    jest.useFakeTimers();
    const onRefresh = jest.fn();
    renderControl(onRefresh, '/?refresh=60000');

    expect(screen.getByRole('combobox')).toHaveValue('1m');
    act(() => {
      jest.advanceTimersByTime(60000);
    });
    expect(onRefresh).toHaveBeenCalledTimes(1);
  });
});
