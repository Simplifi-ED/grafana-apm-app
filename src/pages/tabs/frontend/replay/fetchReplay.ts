/**
 * Session-replay data access.
 *
 * Primary path (ClickStack / HyperDX): plugin backend reads `hyperdx_sessions`
 * via the configured Grafana ClickHouse datasource, reassembles rr-web chunks
 * (`rr-web.event` / `rr-web.chunk` / `rr-web.total-chunks`), and returns
 * rrweb eventWithTime objects. Body is raw JSON — not the NAIS Faro
 * gzip+b64 Loki chunk format.
 *
 * Fallback (LGTM / Faro): when ClickHouse is not configured the backend
 * returns unavailable; if `logsUid` is set we still probe/fetch the legacy
 * Alloy faro.receiver chunk format from Loki so existing NAIS recordings keep
 * working.
 */
import { getBackendSrv } from '@grafana/runtime';
import { lastValueFrom } from 'rxjs';
import { gunzipSync, strFromU8 } from 'fflate';
import pluginJson from '../../../../plugin.json';
import { otel } from '../../../../otelconfig';
import { sanitizeLabelValue } from '../../../../utils/sanitize';
import { parseLogfmt } from '../exception-utils';

export type ReplayMode = 'snapshot' | 'recording';

/** rrweb eventWithTime — structurally typed so the heavy rrweb packages stay out of this module. */
export interface ReplayEventWithTime {
  type: number;
  data: unknown;
  timestamp: number;
}

export interface ReplayQueryOptions {
  /** Namespace path segment for the plugin resource API (use "" / "_" when none). */
  namespace: string;
  service: string;
  sessionId: string;
  fromMs: number;
  toMs: number;
  environment?: string;
  /**
   * Loki datasource UID — only used when the ClickHouse-backed backend reports
   * unavailable (LGTM Faro path).
   */
  logsUid?: string;
  /** Stream label used for the Loki environment matcher (defaults to otel.labels.deploymentEnv). */
  environmentLabel?: string;
}

export interface ReplayData {
  /** All rrweb events for the session, ordered by timestamp. */
  events: ReplayEventWithTime[];
  mode: ReplayMode;
  chunkCount: number;
  /** True when the backend hit a row/size cap. */
  truncated?: boolean;
}

export interface ReplayProbeResult {
  hasChunks: boolean;
  /** "recording" wins when a session has both snapshot and recording chunks. */
  mode: ReplayMode | null;
  chunkCount: number;
}

const RESOURCES_BASE = `/api/plugins/${pluginJson.id}/resources`;

/** The only Faro/Loki chunk encoding this plugin version understands. */
const SUPPORTED_ENC = 'gzip+b64';

function nsParam(namespace: string): string {
  return encodeURIComponent(namespace || '_');
}

function timeParams(fromMs: number, toMs: number): Record<string, string> {
  return {
    from: String(Math.floor(fromMs / 1000)),
    to: String(Math.floor(toMs / 1000)),
  };
}

function replayPath(opts: ReplayQueryOptions, suffix: '' | '/probe'): string {
  return (
    `${RESOURCES_BASE}/services/${nsParam(opts.namespace)}/${encodeURIComponent(opts.service)}` +
    `/frontend/sessions/${encodeURIComponent(opts.sessionId)}/replay${suffix}`
  );
}

/** Millisecond timestamp → Loki nanosecond string (string concat avoids float precision loss). */
function msToNs(ms: number): string {
  return `${Math.floor(ms)}000000`;
}

/**
 * Shared Loki log pipeline: stream selector + cheap line prefilters + exact session match.
 */
function buildLokiReplayPipeline(opts: ReplayQueryOptions): string {
  const fl = otel.faroLoki;
  const service = sanitizeLabelValue(opts.service);
  const sessionId = sanitizeLabelValue(opts.sessionId);
  const envLabel = opts.environmentLabel || otel.labels.deploymentEnv;
  const envStream = opts.environment ? `, ${envLabel}="${sanitizeLabelValue(opts.environment)}"` : '';
  const selector = `{${fl.serviceName}="${service}", ${fl.kind}=~"${fl.kindEvent}|${fl.kindReplay}"${envStream}}`;
  return `${selector} |= \`${fl.replayChunkEvent}\` |= \`${sessionId}\` | logfmt | ${fl.sessionId}="${sessionId}"`;
}

/** Decode one Faro/Loki chunk payload: base64 → gunzip → JSON rrweb eventWithTime[]. */
export function decodeReplayChunk(b64: string): ReplayEventWithTime[] {
  const bin = atob(b64);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) {
    bytes[i] = bin.charCodeAt(i);
  }
  const parsed = JSON.parse(strFromU8(gunzipSync(bytes)));
  return Array.isArray(parsed) ? parsed : [];
}

interface BackendProbeResponse {
  hasChunks?: boolean;
  mode?: ReplayMode | '';
  chunkCount?: number;
  unavailable?: boolean;
  error?: string;
}

interface BackendReplayResponse {
  events?: ReplayEventWithTime[];
  mode?: ReplayMode;
  chunkCount?: number;
  eventCount?: number;
  truncated?: boolean;
  unavailable?: boolean;
  error?: string;
}

async function fetchBackendProbe(opts: ReplayQueryOptions): Promise<BackendProbeResponse> {
  const params: Record<string, string> = { ...timeParams(opts.fromMs, opts.toMs) };
  if (opts.environment) {
    params.environment = opts.environment;
  }
  const res = await lastValueFrom(
    getBackendSrv().fetch<BackendProbeResponse>({
      url: `${replayPath(opts, '/probe')}?${new URLSearchParams(params).toString()}`,
      method: 'GET',
    })
  );
  return res.data ?? {};
}

async function fetchBackendReplay(opts: ReplayQueryOptions): Promise<BackendReplayResponse> {
  const params: Record<string, string> = { ...timeParams(opts.fromMs, opts.toMs) };
  if (opts.environment) {
    params.environment = opts.environment;
  }
  const res = await lastValueFrom(
    getBackendSrv().fetch<BackendReplayResponse>({
      url: `${replayPath(opts, '')}?${new URLSearchParams(params).toString()}`,
      method: 'GET',
    })
  );
  return res.data ?? {};
}

/**
 * Fetch and reassemble all replay events for a session.
 * Returns null when the session has no recording data in the range.
 */
export async function fetchReplay(opts: ReplayQueryOptions): Promise<ReplayData | null> {
  const backend = await fetchBackendReplay(opts);
  if (!backend.unavailable) {
    if (backend.error && !(backend.events && backend.events.length > 0)) {
      throw new Error(backend.error);
    }
    if (!backend.events || backend.events.length === 0) {
      return null;
    }
    return {
      events: backend.events,
      mode: backend.mode === 'snapshot' ? 'snapshot' : 'recording',
      chunkCount: backend.chunkCount ?? backend.eventCount ?? backend.events.length,
      truncated: backend.truncated,
    };
  }

  if (!opts.logsUid) {
    return null;
  }
  return fetchLokiReplay(opts);
}

/**
 * Cheap existence probe for the exception drawer.
 */
export async function probeReplay(opts: ReplayQueryOptions): Promise<ReplayProbeResult> {
  const backend = await fetchBackendProbe(opts);
  if (!backend.unavailable) {
    const mode = backend.mode === 'snapshot' || backend.mode === 'recording' ? backend.mode : null;
    return {
      hasChunks: Boolean(backend.hasChunks),
      mode: backend.hasChunks ? (mode ?? 'recording') : null,
      chunkCount: Number(backend.chunkCount ?? 0),
    };
  }

  if (!opts.logsUid) {
    return { hasChunks: false, mode: null, chunkCount: 0 };
  }
  return probeLokiReplay(opts);
}

async function fetchLokiReplay(opts: ReplayQueryOptions): Promise<ReplayData | null> {
  const res = await lastValueFrom(
    getBackendSrv().fetch<any>({
      url: `/api/datasources/proxy/uid/${encodeURIComponent(opts.logsUid!)}/loki/api/v1/query_range`,
      params: {
        query: buildLokiReplayPipeline(opts),
        limit: '1000',
        direction: 'forward',
        start: msToNs(opts.fromMs),
        end: msToNs(opts.toMs),
      },
      method: 'GET',
    })
  );

  const streams = res.data?.data?.result ?? [];
  const chunks: Array<{ seq: number; mode?: string; enc?: string; data: string }> = [];
  streams.forEach((stream: any) => {
    (stream.values ?? []).forEach((val: [string, string]) => {
      const p = parseLogfmt(val[1]);
      if (!p.event_data_data) {
        return;
      }
      chunks.push({
        seq: Number(p.event_data_chunk_seq),
        mode: p.event_data_mode,
        enc: p.event_data_enc,
        data: p.event_data_data,
      });
    });
  });

  if (chunks.length === 0) {
    return null;
  }

  chunks.sort((a, b) => a.seq - b.seq);

  const events: ReplayEventWithTime[] = [];
  const seen = new Set<number>();
  let decoded = 0;
  for (const chunk of chunks) {
    if (Number.isFinite(chunk.seq) && seen.has(chunk.seq)) {
      continue;
    }
    seen.add(chunk.seq);
    if (chunk.enc && chunk.enc !== SUPPORTED_ENC) {
      throw new Error(
        `Unsupported replay chunk encoding "${chunk.enc}" (this plugin understands "${SUPPORTED_ENC}") — update the plugin.`
      );
    }
    events.push(...decodeReplayChunk(chunk.data));
    decoded++;
  }

  return {
    events,
    mode: chunks.some((c) => c.mode === 'recording') ? 'recording' : 'snapshot',
    chunkCount: decoded,
  };
}

async function probeLokiReplay(opts: ReplayQueryOptions): Promise<ReplayProbeResult> {
  const rangeSec = Math.max(1, Math.ceil((opts.toMs - opts.fromMs) / 1000));
  const query = `sum by (event_data_mode) (count_over_time(${buildLokiReplayPipeline(opts)} [${rangeSec}s]))`;

  const res = await lastValueFrom(
    getBackendSrv().fetch<any>({
      url: `/api/datasources/proxy/uid/${encodeURIComponent(opts.logsUid!)}/loki/api/v1/query`,
      params: {
        query,
        time: msToNs(opts.toMs),
      },
      method: 'GET',
    })
  );

  const samples = res.data?.data?.result ?? [];
  let chunkCount = 0;
  let mode: ReplayMode | null = null;
  for (const sample of samples) {
    const value = Number(sample.value?.[1] ?? 0);
    if (!(value > 0)) {
      continue;
    }
    chunkCount += value;
    const sampleMode = sample.metric?.event_data_mode;
    if (sampleMode === 'recording') {
      mode = 'recording';
    } else if (sampleMode === 'snapshot' && mode !== 'recording') {
      mode = 'snapshot';
    }
  }

  return { hasChunks: chunkCount > 0, mode, chunkCount };
}
