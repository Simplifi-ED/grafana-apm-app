import { useCallback, useState } from 'react';
import { config } from '@grafana/runtime';

const STORAGE_KEY = 'omnivya-apm:dismissed-banners';

/** Fleet ownership coverage alert on Service Inventory. */
export const BANNER_OWNERSHIP_COVERAGE = 'ownership-coverage';

interface StorageSchema {
  version: 1;
  ids: string[];
}

function activeOrgId(): string {
  const orgId = config?.bootData?.user?.orgId;
  return orgId === undefined || orgId === null ? '' : String(orgId);
}

function scopedId(id: string): string {
  return `${activeOrgId()}|${id}`;
}

function readIds(): Set<string> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return new Set();
    }
    const parsed = JSON.parse(raw) as StorageSchema;
    if (parsed?.version !== 1 || !Array.isArray(parsed.ids)) {
      return new Set();
    }
    return new Set(parsed.ids.filter((id): id is string => typeof id === 'string' && id.length > 0));
  } catch {
    return new Set();
  }
}

function writeIds(ids: Set<string>): void {
  try {
    const schema: StorageSchema = { version: 1, ids: Array.from(ids).sort() };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(schema));
  } catch {
    // private mode / quota
  }
}

export function isBannerDismissed(id: string): boolean {
  return readIds().has(scopedId(id));
}

export function dismissBanner(id: string): void {
  const ids = readIds();
  ids.add(scopedId(id));
  writeIds(ids);
}

/** Persist dismiss across refreshes (browser-local, org-scoped). */
export function useDismissedBanner(id: string): [boolean, () => void] {
  const [dismissed, setDismissed] = useState(() => isBannerDismissed(id));
  const dismiss = useCallback(() => {
    dismissBanner(id);
    setDismissed(true);
  }, [id]);
  return [dismissed, dismiss];
}
