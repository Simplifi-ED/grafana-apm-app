import { of, throwError } from 'rxjs';
import { gzipSync, strToU8 } from 'fflate';
import { fetchReplay, probeReplay, decodeReplayChunk, ReplayEventWithTime } from './fetchReplay';

const mockFetch = jest.fn();

jest.mock('@grafana/runtime', () => ({
  getBackendSrv: () => ({ fetch: (options: unknown) => mockFetch(options) }),
}));

jest.mock('../../../../plugin.json', () => ({ id: 'nais-apm-app' }), { virtual: true });

/** Round-trip half of the Faro SDK's transport: gzip the rrweb event array and base64 it. */
function encodeChunk(events: ReplayEventWithTime[]): string {
  return Buffer.from(gzipSync(strToU8(JSON.stringify(events)))).toString('base64');
}

/** One Loki logfmt line as Alloy's faro.receiver flattens a chunk event. */
function chunkLine(seq: number, mode: 'snapshot' | 'recording', events: ReplayEventWithTime[], enc = 'gzip+b64') {
  return (
    `kind=event event_name=faro.session_recording.chunk session_id=sess-1 ` +
    `event_data_chunk_seq=${seq} event_data_mode=${mode} event_data_enc=${enc} ` +
    `event_data_count=${events.length} event_data_data="${encodeChunk(events)}"`
  );
}

function lokiStreams(lines: string[]) {
  return of({
    data: { data: { result: [{ stream: {}, values: lines.map((l, i) => [`${1751500000000 + i}000000`, l]) }] } },
  });
}

const baseOpts = {
  namespace: 'team-a',
  service: 'my-app',
  sessionId: 'sess-1',
  fromMs: 1751500000000,
  toMs: 1751503600000,
  logsUid: 'loki-uid',
};

const ev = (timestamp: number, type = 3): ReplayEventWithTime => ({ type, data: { t: timestamp }, timestamp });

beforeEach(() => {
  mockFetch.mockReset();
});

describe('decodeReplayChunk', () => {
  it('round-trips gzip+base64 rrweb events', () => {
    const events = [ev(1000, 4), ev(1000, 2), ev(1500)];
    expect(decodeReplayChunk(encodeChunk(events))).toEqual(events);
  });
});

describe('fetchReplay (ClickHouse backend)', () => {
  it('calls the plugin resource API and returns reassembled events', async () => {
    const events = [ev(1000, 4), ev(1000, 2), ev(2000)];
    mockFetch.mockReturnValue(
      of({
        data: { events, mode: 'recording', chunkCount: 3, eventCount: 3 },
      })
    );

    const result = await fetchReplay(baseOpts);

    expect(mockFetch).toHaveBeenCalledTimes(1);
    const call = mockFetch.mock.calls[0][0];
    expect(call.url).toContain(
      '/api/plugins/nais-apm-app/resources/services/team-a/my-app/frontend/sessions/sess-1/replay?'
    );
    expect(call.url).toContain('from=1751500000');
    expect(call.url).toContain('to=1751503600');
    expect(result).toEqual({ events, mode: 'recording', chunkCount: 3, truncated: undefined });
  });

  it('returns null when the backend has no events', async () => {
    mockFetch.mockReturnValue(of({ data: { events: [], mode: 'recording', chunkCount: 0 } }));
    expect(await fetchReplay(baseOpts)).toBeNull();
  });

  it('throws on backend error payloads without events', async () => {
    mockFetch.mockReturnValue(of({ data: { events: [], error: 'recording data is malformed' } }));
    await expect(fetchReplay(baseOpts)).rejects.toThrow(/malformed/);
  });

  it('falls back to Loki when ClickHouse is unavailable', async () => {
    const chunk0 = [ev(1000, 4), ev(1000, 2)];
    mockFetch
      .mockReturnValueOnce(of({ data: { unavailable: true } }))
      .mockReturnValueOnce(lokiStreams([chunkLine(0, 'recording', chunk0)]));

    const result = await fetchReplay(baseOpts);
    expect(result!.events).toEqual(chunk0);
    expect(mockFetch.mock.calls[1][0].url).toContain('/api/datasources/proxy/uid/loki-uid/loki/api/v1/query_range');
  });

  it('returns null when ClickHouse is unavailable and no Loki UID is set', async () => {
    mockFetch.mockReturnValue(of({ data: { unavailable: true } }));
    expect(await fetchReplay({ ...baseOpts, logsUid: undefined })).toBeNull();
  });
});

describe('fetchReplay (Loki fallback)', () => {
  function mockUnavailableThenLoki(lines: string[]) {
    mockFetch.mockReturnValueOnce(of({ data: { unavailable: true } })).mockReturnValueOnce(lokiStreams(lines));
  }

  it('reassembles Faro chunks in seq order', async () => {
    const chunk0 = [ev(1000, 4), ev(1000, 2)];
    const chunk1 = [ev(2000), ev(3000)];
    mockUnavailableThenLoki([chunkLine(1, 'recording', chunk1), chunkLine(0, 'recording', chunk0)]);

    const result = await fetchReplay(baseOpts);
    expect(result!.mode).toBe('recording');
    expect(result!.chunkCount).toBe(2);
    expect(result!.events).toEqual([...chunk0, ...chunk1]);
  });

  it('rejects unknown chunk encodings', async () => {
    mockUnavailableThenLoki([chunkLine(0, 'recording', [ev(1, 4), ev(1, 2)], 'zstd+b64')]);
    await expect(fetchReplay(baseOpts)).rejects.toThrow(/Unsupported replay chunk encoding "zstd\+b64"/);
  });
});

describe('probeReplay (ClickHouse backend)', () => {
  it('calls the probe resource endpoint', async () => {
    mockFetch.mockReturnValue(of({ data: { hasChunks: true, mode: 'recording', chunkCount: 5 } }));

    const probe = await probeReplay(baseOpts);
    expect(mockFetch.mock.calls[0][0].url).toContain('/replay/probe?');
    expect(probe).toEqual({ hasChunks: true, mode: 'recording', chunkCount: 5 });
  });

  it('falls back to Loki when unavailable', async () => {
    mockFetch.mockReturnValueOnce(of({ data: { unavailable: true } })).mockReturnValueOnce(
      of({
        data: {
          data: {
            result: [
              { metric: { event_data_mode: 'snapshot' }, value: [1751503600, '2'] },
              { metric: { event_data_mode: 'recording' }, value: [1751503600, '3'] },
            ],
          },
        },
      })
    );

    expect(await probeReplay(baseOpts)).toEqual({ hasChunks: true, mode: 'recording', chunkCount: 5 });
  });

  it('reports absence when backend has no chunks', async () => {
    mockFetch.mockReturnValue(of({ data: { hasChunks: false, mode: '', chunkCount: 0 } }));
    expect(await probeReplay(baseOpts)).toEqual({ hasChunks: false, mode: null, chunkCount: 0 });
  });

  it('surfaces backend fetch failures', async () => {
    mockFetch.mockReturnValue(throwError(() => new Error('network down')));
    await expect(probeReplay(baseOpts)).rejects.toThrow(/network down/);
  });
});
