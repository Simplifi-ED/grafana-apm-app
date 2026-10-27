import { useEffect, useRef, useState } from 'react';

interface FetchState<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
  /** True when we have stale data and a refetch is in progress */
  stale: boolean;
}

/** Turn getBackendSrv / fetch rejection values into a readable string. */
export function formatFetchError(error: unknown): string {
  if (error instanceof Error && error.message.trim()) {
    return error.message.trim();
  }
  if (typeof error === 'string' && error.trim()) {
    return error.trim();
  }
  if (typeof error === 'object' && error !== null) {
    const data = (error as { data?: unknown }).data;
    if (typeof data === 'string' && data.trim()) {
      return data.trim();
    }
    if (typeof data === 'object' && data !== null) {
      const msg = (data as { message?: unknown }).message;
      if (typeof msg === 'string' && msg.trim()) {
        return msg.trim();
      }
    }
    const message = (error as { message?: unknown }).message;
    if (typeof message === 'string' && message.trim()) {
      return message.trim();
    }
    const statusText = (error as { statusText?: unknown }).statusText;
    if (typeof statusText === 'string' && statusText.trim()) {
      return statusText.trim();
    }
  }
  return 'Request failed';
}

/**
 * Generic data-fetching hook with stale-while-revalidate semantics.
 *
 * When deps change, if we already have data, keeps showing the previous
 * result (with `stale: true`) while the new fetch runs in the background.
 * This prevents the UI from flashing a loading spinner on every refetch.
 *
 * @param fetcher  Async function that returns data. Called whenever deps change.
 * @param deps     Dependency array (like useEffect). Fetcher re-runs when deps change.
 * @param options  Optional config: `skip` to disable fetching, `initialData` for default.
 */
export function useFetch<T>(
  fetcher: () => Promise<T>,
  deps: React.DependencyList,
  options?: { skip?: boolean; initialData?: T | null }
): FetchState<T> & { refetch: () => void } {
  const skip = options?.skip ?? false;
  const initialData = options?.initialData ?? null;

  const [state, setState] = useState<FetchState<T>>({
    data: initialData,
    loading: !skip,
    error: null,
    stale: false,
  });

  // Keep fetcher in a ref so it's always current without being a dependency
  const fetcherRef = useRef(fetcher);

  useEffect(() => {
    fetcherRef.current = fetcher;
  });

  // Refetch trigger — increment to force a new fetch
  const [fetchTrigger, setFetchTrigger] = useState(0);

  useEffect(() => {
    if (skip) {
      setState((s) => (s.loading ? { ...s, loading: false } : s));
      return;
    }

    let cancelled = false;

    // Stale-while-revalidate: keep existing data visible, mark as stale
    setState((s) => ({
      ...s,
      loading: s.data === null,
      stale: s.data !== null,
      error: null,
    }));

    fetcherRef
      .current()
      .then((result) => {
        if (!cancelled) {
          setState({ data: result, loading: false, error: null, stale: false });
        }
      })
      .catch((e) => {
        if (!cancelled) {
          // On error, keep stale data if available.
          // getBackendSrv() rejects with plain objects ({ data, status, … }),
          // not Error — String(obj) → "[object Object]" in the UI Alert.
          setState((s) => ({
            data: s.data,
            loading: false,
            error: formatFetchError(e),
            stale: false,
          }));
        }
      });

    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, fetchTrigger, skip]);

  const refetch = () => setFetchTrigger((t) => t + 1);

  return { ...state, refetch };
}
