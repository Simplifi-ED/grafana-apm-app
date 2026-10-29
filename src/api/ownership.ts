import { getBackendSrv } from '@grafana/runtime';
import { lastValueFrom } from 'rxjs';
import pluginJson from '../plugin.json';

const BASE_URL = `/api/plugins/${pluginJson.id}/resources`;

function nsParam(namespace: string): string {
  return encodeURIComponent(namespace || '_');
}

async function fetchResource<T>(path: string, params?: Record<string, string>): Promise<T> {
  const url = params ? `${BASE_URL}${path}?${new URLSearchParams(params).toString()}` : `${BASE_URL}${path}`;
  const response = await lastValueFrom(getBackendSrv().fetch<T>({ url, method: 'GET' }));
  return response.data;
}

export type OwnershipSource = 'telemetry' | 'annotation' | 'claimed';

export interface OwnershipFields {
  team?: string;
  contact?: string;
  repo?: string;
  runbook?: string;
}

export interface OwnershipConflictLoser {
  source: OwnershipSource;
  fields: OwnershipFields;
}

export interface Ownership {
  team?: string;
  contact?: string;
  repo?: string;
  runbook?: string;
  source?: OwnershipSource;
  lastSeenMs?: number;
  stale: boolean;
  teamKnown: boolean;
  /** Lookup failed — not the same as unowned. */
  unavailable?: boolean;
  conflict?: {
    winner: OwnershipSource;
    losers: OwnershipConflictLoser[];
  };
  emptyHint?: {
    attrs: string[];
    snippet: string;
    docPath: string;
  };
}

export interface NamespaceOwnershipResponse {
  services: Record<string, Ownership>;
  owned: number;
  total: number;
  coverage: number;
  truncated?: boolean;
  scanned: number;
}

export interface FleetOwnershipResponse {
  owned: number;
  total: number;
  coverage: number;
  unowned: Array<{ namespace?: string; service: string }>;
  byTeam?: Record<string, number>;
  services?: Record<string, Ownership>;
}

export async function getOwnership(namespace: string, service: string): Promise<Ownership> {
  return fetchResource<Ownership>(`/services/${nsParam(namespace)}/${encodeURIComponent(service)}/ownership`);
}

export async function getNamespaceOwnership(
  namespace: string,
  from: number,
  to: number,
  environment?: string
): Promise<NamespaceOwnershipResponse> {
  const params: Record<string, string> = {
    from: String(Math.floor(from / 1000)),
    to: String(Math.floor(to / 1000)),
  };
  if (environment) {
    params.environment = environment;
  }
  return fetchResource<NamespaceOwnershipResponse>(`/namespaces/${nsParam(namespace)}/ownership`, params);
}

export async function getFleetOwnership(from: number, to: number): Promise<FleetOwnershipResponse> {
  return fetchResource<FleetOwnershipResponse>('/ownership/coverage', {
    from: String(Math.floor(from / 1000)),
    to: String(Math.floor(to / 1000)),
  });
}
