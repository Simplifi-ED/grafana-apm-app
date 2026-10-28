/**
 * FrontendTab — the no-telemetry SetupPlaceholder must point at public Faro /
 * OpenTelemetry docs (not nais-specific SDK links).
 */
import React from 'react';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { FrontendTab } from './FrontendTab';
import * as client from '../../api/client';

jest.mock('../../api/client', () => ({
  ...jest.requireActual('../../api/client'),
  getFrontendMetrics: jest.fn(),
  getExceptionGroups: jest.fn().mockResolvedValue({ groups: [] }),
}));

jest.mock('../../utils/datasources', () => ({
  usePluginDatasources: () => ({ metricsUid: 'mimir', tracesUid: 'tempo', logsUid: 'loki', clickHouseUid: '' }),
  usePluginLabelOverrides: () => ({}),
  useServiceAlias: (name: string) => name,
}));

jest.mock('../../utils/timeRange', () => ({
  useTimeRange: () => ({ from: 'now-1h', to: 'now', fromMs: 1000, toMs: 2000 }),
}));

const getFrontendMetrics = client.getFrontendMetrics as jest.Mock;

function renderTab() {
  return render(
    <MemoryRouter>
      <FrontendTab service="my-app" namespace="team-a" environment="prod" />
    </MemoryRouter>
  );
}

describe('FrontendTab setup placeholder links', () => {
  beforeEach(() => getFrontendMetrics.mockReset());

  it('links to Faro and OpenTelemetry browser docs when no telemetry is found', async () => {
    getFrontendMetrics.mockResolvedValue({ available: false });
    renderTab();

    const faro = await screen.findByRole('link', { name: /Faro Quick Start Guide/ });
    expect(faro).toHaveAttribute('href', 'https://grafana.com/docs/faro-web-sdk/latest/tutorials/quick-start-browser/');

    const otelLinks = screen.getAllByRole('link', { name: /OpenTelemetry Browser SDK/ });
    expect(otelLinks.length).toBeGreaterThanOrEqual(1);
    expect(otelLinks[0]).toHaveAttribute('href', 'https://opentelemetry.io/docs/languages/js/getting-started/browser/');

    expect(screen.queryByRole('link', { name: /@nais\/apm/ })).not.toBeInTheDocument();
  });
});
