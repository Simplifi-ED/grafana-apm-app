import React from 'react';
import { render, screen, act, fireEvent } from '@testing-library/react';
import { PluginType } from '@grafana/data';
import AppConfig, { AppConfigProps } from './AppConfig';
import { testIds } from '../testIds';

const mockGet = jest.fn();
const mockFetch = jest.fn();

jest.mock('@grafana/runtime', () => ({
  ...jest.requireActual('@grafana/runtime'),
  getBackendSrv: () => ({
    get: mockGet,
    fetch: mockFetch,
  }),
}));

// Extract the settings payload the component POSTs on submit.
const submittedPayload = () => {
  const call = mockFetch.mock.calls.find((c) => String(c[0]?.url).endsWith('/settings'));
  if (!call) {
    throw new Error('settings POST was never issued');
  }
  return call[0].data as {
    jsonData: Record<string, unknown>;
    secureJsonData?: Record<string, unknown>;
  };
};

describe('Components/AppConfig', () => {
  let props: AppConfigProps;

  beforeEach(() => {
    jest.resetAllMocks();
    mockGet.mockImplementation(async (url: string) => {
      if (String(url).includes('/api/datasources/uid/')) {
        // Fully configured CH Explore defaults — no warning in most tests.
        return {
          uid: 'clickhouse',
          type: 'grafana-clickhouse-datasource',
          jsonData: {
            traces: { defaultTable: 'otel_traces' },
            logs: { defaultTable: 'otel_logs' },
          },
        };
      }
      return [
        { uid: 'mimir', name: 'Mimir', type: 'prometheus', isDefault: true },
        { uid: 'tempo', name: 'Tempo', type: 'tempo', isDefault: true },
        { uid: 'loki', name: 'Loki', type: 'loki', isDefault: true },
        { uid: 'clickhouse', name: 'ClickHouse', type: 'grafana-clickhouse-datasource', isDefault: true },
      ];
    });
    // fetch returns undefined; lastValueFrom rejects and is swallowed by the
    // component's catch, so window.location.reload is never reached in tests.
    mockFetch.mockReturnValue(undefined);
    jest.spyOn(console, 'error').mockImplementation(() => {});

    props = {
      plugin: {
        meta: {
          id: 'omnivya-apm-app',
          name: 'Application Observability',
          type: PluginType.app,
          enabled: true,
          jsonData: {
            metricsDataSource: { uid: 'mimir', type: 'prometheus' },
            tracesDataSource: { uid: 'tempo', type: 'tempo' },
            logsDataSource: { uid: 'loki', type: 'loki' },
          },
        },
      },
      query: {},
    } as unknown as AppConfigProps;
  });

  test('renders data source and detection fieldsets', async () => {
    const plugin = { meta: { ...props.plugin.meta, enabled: true } };

    await act(async () => {
      // @ts-ignore
      render(<AppConfig plugin={plugin} query={props.query} />);
    });

    expect(screen.queryByRole('group', { name: /data sources/i })).toBeInTheDocument();
    expect(screen.queryByRole('group', { name: /detection & overrides/i })).toBeInTheDocument();
    expect(screen.queryByTestId(testIds.appConfig.submit)).toBeInTheDocument();
  });

  test('updates deployment environment label without synthetic event errors', async () => {
    const plugin = { meta: { ...props.plugin.meta, enabled: true } };

    await act(async () => {
      // @ts-ignore
      render(<AppConfig plugin={plugin} query={props.query} />);
    });

    const input = screen.getByPlaceholderText('k8s_cluster_name') as HTMLInputElement;

    await act(async () => {
      fireEvent.change(input, { target: { value: 'cluster' } });
    });

    expect(input.value).toBe('cluster');
  });

  test('clears a configured service account token via reset + save (sends empty string)', async () => {
    const plugin = {
      meta: {
        ...props.plugin.meta,
        enabled: true,
        secureJsonFields: { serviceAccountToken: true },
      },
    };

    await act(async () => {
      // @ts-ignore
      render(<AppConfig plugin={plugin} query={props.query} />);
    });

    // A configured SecretInput renders a Reset button; clicking it signals intent
    // to clear the secret rather than leaving the stored value untouched.
    const resetButtons = screen.getAllByText('Reset');
    await act(async () => {
      fireEvent.click(resetButtons[0]);
    });

    await act(async () => {
      fireEvent.click(screen.getByTestId(testIds.appConfig.submit));
    });

    const payload = submittedPayload();
    // Explicit reset must send an empty string so Grafana overwrites/clears it.
    expect(payload.secureJsonData?.serviceAccountToken).toBe('');
  });

  test('does not send a configured secret that was left untouched', async () => {
    const plugin = {
      meta: {
        ...props.plugin.meta,
        enabled: true,
        secureJsonFields: { serviceAccountToken: true },
      },
    };

    await act(async () => {
      // @ts-ignore
      render(<AppConfig plugin={plugin} query={props.query} />);
    });

    await act(async () => {
      fireEvent.click(screen.getByTestId(testIds.appConfig.submit));
    });

    const payload = submittedPayload();
    // Nothing was touched, so no secure payload should be sent (secrets preserved).
    expect(payload.secureJsonData).toBeUndefined();
  });

  test('preserves and edits alert routing environment overrides', async () => {
    const plugin = {
      meta: {
        ...props.plugin.meta,
        jsonData: { ...props.plugin.meta.jsonData, alertRoutingEnvironments: { prod: 'prod-gcp' } },
      },
    };

    await act(async () => {
      // @ts-ignore
      render(<AppConfig plugin={plugin} query={props.query} />);
    });

    const routingInput = screen.getByPlaceholderText('e.g., prod-gcp') as HTMLInputElement;
    expect(routingInput.value).toBe('prod-gcp');

    await act(async () => {
      fireEvent.click(screen.getByTestId(testIds.appConfig.submit));
    });
    expect(submittedPayload().jsonData.alertRoutingEnvironments).toEqual({ prod: 'prod-gcp' });
  });

  test('auto-detects datasource overrides alongside routing-only environments', async () => {
    mockGet.mockResolvedValue([
      { uid: 'dev-tempo', name: 'dev-tempo', type: 'tempo', isDefault: false },
      { uid: 'prod-tempo', name: 'prod-tempo', type: 'tempo', isDefault: false },
    ]);
    const plugin = props.plugin;
    plugin.meta = {
      ...plugin.meta,
      jsonData: {
        ...plugin.meta.jsonData,
        alertRoutingEnvironments: { prod: 'prod-gcp', staging: 'staging-gcp' },
      },
    };

    await act(async () => {
      // @ts-ignore
      render(<AppConfig plugin={plugin} query={props.query} />);
    });
    await act(async () => {
      fireEvent.click(screen.getByTestId(testIds.appConfig.submit));
    });

    const { jsonData } = submittedPayload();
    expect(jsonData.tracesDataSource).toEqual({
      uid: 'tempo',
      type: 'tempo',
      byEnvironment: {
        dev: { uid: 'dev-tempo', type: 'tempo' },
        prod: { uid: 'prod-tempo', type: 'tempo' },
      },
    });
    expect(jsonData.alertRoutingEnvironments).toEqual({ prod: 'prod-gcp', staging: 'staging-gcp' });
  });

  test('does not auto-detect over existing datasource overrides', async () => {
    mockGet.mockResolvedValue([
      { uid: 'dev-tempo', name: 'dev-tempo', type: 'tempo', isDefault: false },
      { uid: 'prod-tempo', name: 'prod-tempo', type: 'tempo', isDefault: false },
    ]);
    const plugin = props.plugin;
    plugin.meta = {
      ...plugin.meta,
      jsonData: {
        ...plugin.meta.jsonData,
        tracesDataSource: {
          uid: 'tempo',
          type: 'tempo',
          byEnvironment: { prod: { uid: 'custom-prod-tempo', type: 'tempo' } },
        },
        alertRoutingEnvironments: { prod: 'prod-gcp' },
      },
    };

    await act(async () => {
      // @ts-ignore
      render(<AppConfig plugin={plugin} query={props.query} />);
    });
    await act(async () => {
      fireEvent.click(screen.getByTestId(testIds.appConfig.submit));
    });

    const { jsonData } = submittedPayload();
    expect(jsonData.tracesDataSource).toEqual({
      uid: 'tempo',
      type: 'tempo',
      byEnvironment: { prod: { uid: 'custom-prod-tempo', type: 'tempo' } },
    });
    expect(jsonData.alertRoutingEnvironments).toEqual({ prod: 'prod-gcp' });
  });

  test('persists serviceAliases inventory → scrape map on save', async () => {
    const plugin = {
      meta: {
        ...props.plugin.meta,
        enabled: true,
        jsonData: {
          ...props.plugin.meta.jsonData,
          clickHouseDataSource: { uid: 'clickhouse', type: 'grafana-clickhouse-datasource' },
          serviceAliases: { 'checkout-api': 'checkout' },
        },
      },
    };

    await act(async () => {
      // @ts-ignore
      render(<AppConfig plugin={plugin} query={props.query} />);
    });

    expect(screen.getByRole('group', { name: /service aliases \(clickstack\)/i })).toBeInTheDocument();
    expect(screen.getByDisplayValue('checkout-api')).toBeInTheDocument();
    expect(screen.getByDisplayValue('checkout')).toBeInTheDocument();

    await act(async () => {
      fireEvent.click(screen.getByTestId(testIds.appConfig.submit));
    });

    const payload = submittedPayload();
    expect(payload.jsonData.serviceAliases).toEqual({ 'checkout-api': 'checkout' });
  });

  test('warns when ClickHouse Explore traces/logs defaults are missing', async () => {
    mockGet.mockImplementation(async (url: string) => {
      if (String(url).includes('/api/datasources/uid/')) {
        return { uid: 'clickhouse', type: 'grafana-clickhouse-datasource', jsonData: {} };
      }
      return [{ uid: 'clickhouse', name: 'ClickHouse', type: 'grafana-clickhouse-datasource', isDefault: true }];
    });
    const plugin = {
      meta: {
        ...props.plugin.meta,
        enabled: true,
        jsonData: {
          clickHouseDataSource: { uid: 'clickhouse', type: 'grafana-clickhouse-datasource' },
        },
      },
    };

    await act(async () => {
      // @ts-ignore
      render(<AppConfig plugin={plugin} query={props.query} />);
    });

    expect(await screen.findByText(/ClickHouse Explore Traces\/Logs not configured/i)).toBeInTheDocument();
  });

  test('persists optional pyroscopeDataSource.uid on save', async () => {
    const plugin = {
      meta: {
        ...props.plugin.meta,
        enabled: true,
        jsonData: {
          ...props.plugin.meta.jsonData,
          pyroscopeDataSource: { uid: 'pyroscope', type: 'grafana-pyroscope-datasource' },
        },
      },
    };

    await act(async () => {
      // @ts-ignore
      render(<AppConfig plugin={plugin} query={props.query} />);
    });

    expect(screen.getByDisplayValue('pyroscope')).toBeInTheDocument();

    await act(async () => {
      fireEvent.click(screen.getByTestId(testIds.appConfig.submit));
    });

    const payload = submittedPayload();
    expect(payload.jsonData.pyroscopeDataSource).toEqual({
      uid: 'pyroscope',
      type: 'grafana-pyroscope-datasource',
    });
  });
});
