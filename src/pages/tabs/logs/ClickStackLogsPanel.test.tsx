import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { ClickStackLogsPanel } from './ClickStackLogsPanel';
import * as analytics from '../../../api/analytics';
import { PLUGIN_BASE_URL } from '../../../constants';

jest.mock('../../../api/analytics', () => ({
  ...jest.requireActual('../../../api/analytics'),
  getLogSearch: jest.fn(),
}));

const getLogSearch = analytics.getLogSearch as jest.Mock;

describe('ClickStackLogsPanel', () => {
  beforeEach(() => {
    getLogSearch.mockReset();
    getLogSearch.mockResolvedValue({
      mode: 'clickhouse',
      logs: [
        {
          timestampMs: 1_700_000_000_000,
          severity: 'error',
          body: 'boom',
          traceId: 'aabbccddeeff00112233445566778899',
        },
      ],
    });
  });

  it('shows a ClickHouse Explore link when Loki UID is absent', async () => {
    render(
      <ClickStackLogsPanel
        namespace="ns"
        service="svc"
        fromMs={1}
        toMs={2}
        severity={[]}
        search=""
        clickHouseUid="clickhouse"
        from="now-1h"
        to="now"
      />
    );

    const link = await screen.findByRole('link', { name: /Open in Explore/i });
    const left = JSON.parse(new URL(link.getAttribute('href')!, 'http://grafana.local').searchParams.get('left')!);
    expect(left.datasource).toBe('clickhouse');
    expect(left.queries[0].queryType).toBe('logs');
    expect(left.queries[0].rawSql).toContain("ServiceName = 'svc'");
  });

  it('keeps from/to/env on the log→traces jump link', async () => {
    render(
      <ClickStackLogsPanel
        namespace="demo"
        service="checkout"
        fromMs={1}
        toMs={2}
        severity={[]}
        search=""
        clickHouseUid="clickhouse"
        from="now-6h"
        to="now"
        environment="prod"
      />
    );

    await waitFor(() => expect(screen.getByTitle('aabbccddeeff00112233445566778899')).toBeInTheDocument());
    const href = screen.getByTitle('aabbccddeeff00112233445566778899').getAttribute('href')!;
    expect(href.startsWith(`${PLUGIN_BASE_URL}/services/demo/checkout?`)).toBe(true);
    expect(href).toContain('tab=traces');
    expect(href).toContain('from=now-6h');
    expect(href).toContain('to=now');
    expect(href).toContain('env=prod');
    expect(href).toContain('traceId=aabbccddeeff00112233445566778899');
  });
});
