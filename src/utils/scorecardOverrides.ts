import { config } from '@grafana/runtime';
import { ScorecardCheck, ScorecardReadiness, ScorecardResponse } from '../api/scorecard';

const STORAGE_KEY = 'clickstack-apm:scorecard-na-overrides';

type OverrideMap = Record<string, string[]>;

function activeOrgId(): string {
  const orgId = config?.bootData?.user?.orgId;
  return orgId === undefined || orgId === null ? '' : String(orgId);
}

function serviceKey(namespace: string, service: string, environment?: string): string {
  return `${activeOrgId()}|${namespace || '_'}|${service}|${environment || ''}`;
}

function isStringArray(value: unknown): value is string[] {
  return Array.isArray(value) && value.every((item) => typeof item === 'string');
}

function readAll(): OverrideMap {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return {};
    }
    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return {};
    }
    const out: OverrideMap = {};
    for (const [key, value] of Object.entries(parsed as Record<string, unknown>)) {
      if (typeof key !== 'string' || !isStringArray(value)) {
        continue;
      }
      const cleaned = value.map((v) => v.trim()).filter((v) => v.length > 0);
      if (cleaned.length > 0) {
        out[key] = cleaned;
      }
    }
    return out;
  } catch {
    return {};
  }
}

function writeAll(map: OverrideMap): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(map));
}

/** Check keys the operator marked N/A for this service (browser-local). */
export function getScorecardNaOverrides(namespace: string, service: string, environment?: string): Set<string> {
  const list = readAll()[serviceKey(namespace, service, environment)] ?? [];
  return new Set(list);
}

/** Persist a manual N/A mark (or clear it). Throws if storage is unavailable. */
export function setScorecardNaOverride(
  namespace: string,
  service: string,
  checkKey: string,
  notApplicable: boolean,
  environment?: string
): void {
  const map = readAll();
  const key = serviceKey(namespace, service, environment);
  const next = new Set(map[key] ?? []);
  if (notApplicable) {
    next.add(checkKey);
  } else {
    next.delete(checkKey);
  }
  if (next.size === 0) {
    delete map[key];
  } else {
    map[key] = Array.from(next).sort();
  }
  writeAll(map);
}

function recomputeReadiness(checks: ScorecardCheck[]): ScorecardReadiness {
  const applicable = checks.filter((c) => !c.notApplicable);
  return {
    score: applicable.filter((c) => c.ok).length,
    total: applicable.length,
    checks,
  };
}

/**
 * Apply browser-local N/A overrides on top of the server scorecard.
 * Overridden checks keep their hint but are excluded from score/total.
 */
export function applyScorecardOverrides(response: ScorecardResponse, overrides: Set<string>): ScorecardResponse {
  if (!overrides.size || !response.readiness?.checks?.length) {
    return response;
  }
  const checks = response.readiness.checks.map((check) => {
    if (!overrides.has(check.key) || check.notApplicable) {
      return check;
    }
    return {
      ...check,
      notApplicable: true,
      ok: false,
      hint: check.hint || 'Marked N/A by operator (local override).',
    };
  });
  return { readiness: recomputeReadiness(checks) };
}
