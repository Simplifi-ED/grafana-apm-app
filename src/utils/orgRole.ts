import { config } from '@grafana/runtime';

/** True when the signed-in user can perform Editor-gated plugin writes (triage, watchlist). */
export function canEditPlugin(): boolean {
  const role = String(config?.bootData?.user?.orgRole ?? '').toLowerCase();
  return role === 'admin' || role === 'editor';
}

export const EDITOR_ROLE_REQUIRED = 'Editor role required';
