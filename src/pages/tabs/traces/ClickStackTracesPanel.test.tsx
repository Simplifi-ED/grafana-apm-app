import React from 'react';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { ClickStackTracesPanel } from './ClickStackTracesPanel';
import * as analytics from '../../../api/analytics';

jest.mock('../../../api/analytics', () => ({
  ...jest.requireActual('../../../api/analytics'),
  getTraceSearch: jest.fn(),
  getTraceDetail: jest.fn(),
}));

const getTraceSearch = analytics.getTraceSearch as jest.Mock;
const getTraceDetail = analytics.getTraceDetail as jest.Mock;

const TRACE_A = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa';
const TRACE_B = 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb';

const SEARCH = {
  mode: 'clickhouse' as const,
  traces: [
    {
      traceId: TRACE_A,
      startMs: 1_700_000_000_000,
      endMs: 1_700_000_000_100,
      spanCount: 2,
      errors: 0,
      totalMs: 12,
    },
    {
      traceId: TRACE_B,
      startMs: 1_700_000_000_200,
      endMs: 1_700_000_000_300,
      spanCount: 1,
      errors: 1,
      totalMs: 40,
    },
  ],
};

const DETAIL_A = {
  mode: 'clickhouse' as const,
  traceId: TRACE_A,
  spans: [
    {
      timestampMs: 1_700_000_000_000,
      traceId: TRACE_A,
      spanId: 'root',
      parentSpanId: '',
      spanName: 'GET /orders',
      spanKind: 'Server',
      service: 'svc',
      status: 'Ok',
      durationMs: 12,
    },
    {
      timestampMs: 1_700_000_000_001,
      traceId: TRACE_A,
      spanId: 'child',
      parentSpanId: 'root',
      spanName: 'db.query',
      spanKind: 'Client',
      service: 'svc',
      status: 'Ok',
      durationMs: 4,
    },
  ],
};

function renderPanel(props: Partial<React.ComponentProps<typeof ClickStackTracesPanel>> = {}) {
  return render(
    <ClickStackTracesPanel
      namespace="ns"
      service="svc"
      fromMs={1000}
      toMs={2000}
      status=""
      spanSearch=""
      durationMin=""
      durationMax=""
      {...props}
    />
  );
}

describe('ClickStackTracesPanel', () => {
  beforeEach(() => {
    getTraceSearch.mockReset();
    getTraceDetail.mockReset();
    Element.prototype.scrollIntoView = jest.fn();
  });

  it('expands the waterfall under the clicked row', async () => {
    getTraceSearch.mockResolvedValue(SEARCH);
    getTraceDetail.mockResolvedValue(DETAIL_A);
    renderPanel();

    await waitFor(() => expect(screen.getByRole('table', { name: 'Traces for svc' })).toBeInTheDocument());

    fireEvent.click(screen.getByTestId(`trace-row-${TRACE_A}`));

    await waitFor(() => expect(screen.getByTestId('trace-waterfall')).toBeInTheDocument());
    expect(screen.getByText('GET /orders')).toBeInTheDocument();
    expect(screen.getByText('db.query')).toBeInTheDocument();
    expect(getTraceDetail).toHaveBeenCalledWith('ns', 'svc', TRACE_A, 1000, 2000);

    // Waterfall is the next row after the selected trace — not a detached footer.
    const row = screen.getByTestId(`trace-row-${TRACE_A}`);
    const detailSection = screen.getByLabelText(`Trace ${TRACE_A}`);
    expect(row.nextElementSibling).toContainElement(detailSection);
  });

  it('toggles closed on a second click of the same row', async () => {
    getTraceSearch.mockResolvedValue(SEARCH);
    getTraceDetail.mockResolvedValue(DETAIL_A);
    renderPanel();

    await waitFor(() => expect(screen.getByTestId(`trace-row-${TRACE_A}`)).toBeInTheDocument());

    fireEvent.click(screen.getByTestId(`trace-row-${TRACE_A}`));
    await waitFor(() => expect(screen.getByTestId('trace-waterfall')).toBeInTheDocument());

    fireEvent.click(screen.getByTestId(`trace-row-${TRACE_A}`));
    await waitFor(() => expect(screen.queryByTestId('trace-waterfall')).not.toBeInTheDocument());
  });

  it('does not double-toggle when the Trace ID button is clicked', async () => {
    getTraceSearch.mockResolvedValue(SEARCH);
    getTraceDetail.mockResolvedValue(DETAIL_A);
    renderPanel();

    await waitFor(() => expect(screen.getByTestId(`trace-row-${TRACE_A}`)).toBeInTheDocument());

    const row = screen.getByTestId(`trace-row-${TRACE_A}`);
    fireEvent.click(within(row).getByRole('button', { name: TRACE_A.slice(0, 16) }));

    await waitFor(() => expect(screen.getByTestId('trace-waterfall')).toBeInTheDocument());
    expect(screen.getByLabelText(`Trace ${TRACE_A}`)).toBeInTheDocument();
  });

  it('shows ClickHouse Explore links when Tempo UID is absent', async () => {
    getTraceSearch.mockResolvedValue(SEARCH);
    getTraceDetail.mockResolvedValue(DETAIL_A);
    renderPanel({ clickHouseUid: 'clickhouse' });

    await waitFor(() => expect(screen.getByRole('link', { name: /Open in Explore/i })).toBeInTheDocument());
    const serviceLink = screen.getByRole('link', { name: /Open in Explore/i });
    const left = JSON.parse(
      new URL(serviceLink.getAttribute('href')!, 'http://grafana.local').searchParams.get('left')!
    );
    expect(left.datasource).toBe('clickhouse');
    expect(left.queries[0].queryType).toBe('table');

    fireEvent.click(screen.getByTestId(`trace-row-${TRACE_A}`));
    await waitFor(() => expect(screen.getByTestId('trace-waterfall')).toBeInTheDocument());
    const exploreLinks = screen.getAllByRole('link', { name: /Open in Explore/i });
    expect(exploreLinks.length).toBeGreaterThanOrEqual(2);
    const parsed = exploreLinks.map((a) =>
      JSON.parse(new URL(a.getAttribute('href')!, 'http://grafana.local').searchParams.get('left')!)
    );
    expect(parsed.some((p) => p.queries[0].queryType === 'table')).toBe(true);
    const traceLeft = parsed.find((p) => p.queries[0].queryType === 'traces');
    expect(traceLeft?.queries[0].rawSql).toContain(TRACE_A);
  });

  it('preselects initialTraceId even when missing from search results', async () => {
    getTraceSearch.mockResolvedValue({ mode: 'clickhouse', traces: [] });
    getTraceDetail.mockResolvedValue(DETAIL_A);
    renderPanel({ clickHouseUid: 'clickhouse', initialTraceId: TRACE_A });

    await waitFor(() => expect(screen.getByTestId('trace-waterfall')).toBeInTheDocument());
    expect(getTraceDetail).toHaveBeenCalledWith('ns', 'svc', TRACE_A, 1000, 2000);
  });
});
