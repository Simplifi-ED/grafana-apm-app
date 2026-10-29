import { getBackendSrv } from '@grafana/runtime';
import { lastValueFrom } from 'rxjs';
import pluginJson from '../plugin.json';

// Service scorecard: observability readiness checks from signals the plugin
// already probes. Fetch helpers mirror client.ts (module-private there).

const BASE_URL = `/api/plugins/${pluginJson.id}/resources`;

/** Encode namespace for URL path segments, using '_' as placeholder for empty. */
function nsParam(namespace: string): string {
  return encodeURIComponent(namespace || '_');
}

async function fetchResource<T>(path: string, params?: Record<string, string>): Promise<T> {
  const url = params ? `${BASE_URL}${path}?${new URLSearchParams(params).toString()}` : `${BASE_URL}${path}`;
  const response = await lastValueFrom(getBackendSrv().fetch<T>({ url, method: 'GET' }));
  return response.data;
}

export interface ScorecardCheck {
  key: string;
  label: string;
  ok: boolean;
  /** How to enable the capability — or, for a not-applicable check, why it doesn't apply. */
  hint: string;
  /** Check doesn't apply to this service (e.g. Faro for a pure JVM backend); excluded from score/total. */
  notApplicable?: boolean;
}

export interface ScorecardReadiness {
  /** Number of passing APPLICABLE checks; the fraction is score/total. N/A checks excluded from both. */
  score: number;
  total: number;
  checks: ScorecardCheck[];
}

export interface ScorecardResponse {
  readiness: ScorecardReadiness;
}

export async function getScorecard(
  namespace: string,
  service: string,
  environment?: string
): Promise<ScorecardResponse> {
  return fetchResource<ScorecardResponse>(
    `/services/${nsParam(namespace)}/${encodeURIComponent(service)}/scorecard`,
    environment ? { environment } : undefined
  );
}

export interface NamespaceServiceScorecard {
  service: string;
  environment?: string;
  score: number;
  total: number;
}

/** Namespace readiness rollup (team view without Console ownership). */
export interface NamespaceScorecardResponse {
  score: number;
  total: number;
  services: NamespaceServiceScorecard[];
  scanned: number;
  truncated?: boolean;
}

export async function getNamespaceScorecard(
  namespace: string,
  from: number,
  to: number,
  environment?: string
): Promise<NamespaceScorecardResponse> {
  const params: Record<string, string> = {
    from: String(Math.floor(from / 1000)),
    to: String(Math.floor(to / 1000)),
  };
  if (environment) {
    params.environment = environment;
  }
  return fetchResource<NamespaceScorecardResponse>(`/namespaces/${nsParam(namespace)}/scorecard`, params);
}
