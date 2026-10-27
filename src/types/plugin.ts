/** Shared type definitions for plugin configuration (jsonData schema). */

/** Reference to a Grafana datasource by UID. */
export interface DsRef {
  uid?: string;
  type?: string;
}

/** Datasource reference with optional per-environment overrides. */
export interface EnvAwareDs {
  uid?: string;
  type?: string;
  byEnvironment?: Record<string, DsRef>;
}

/** Label name overrides for non-standard OTel pipelines (e.g. Tempo metrics generator). */
export interface LabelOverrides {
  /** Default: "service_name". Tempo metrics generator emits "service". */
  serviceNameLabel?: string;
  /** Default: "service_namespace". Use "k8s_namespace_name" for Tempo with k8s.namespace.name dimension. */
  serviceNamespaceLabel?: string;
  /** Default: "k8s_cluster_name". */
  deploymentEnvLabel?: string;
}

/** An entry in the ops watchlist: a specific service in a namespace to monitor. */
export interface OpsWatchlistEntry {
  namespace: string;
  service: string;
}

/** The plugin's jsonData schema — persisted in Grafana's plugin settings. */
export interface AppPluginSettings {
  metricsDataSource?: EnvAwareDs;
  tracesDataSource?: EnvAwareDs;
  logsDataSource?: EnvAwareDs;
  /**
   * Grafana ClickHouse datasource (grafana-clickhouse-datasource) for ClickStack.
   * Optional while the LGTM path remains; required for ClickHouse-backed queries.
   */
  clickHouseDataSource?: DsRef;
  metricNamespace?: string;
  durationUnit?: string;
  labelOverrides?: LabelOverrides;
  /** Ingress hostname → service name mapping for discovering on-prem callers via nais ingress. */
  ingressAliases?: Record<string, string>;
  /**
   * Inventory (spanmetrics) name → scrape/browser ServiceName for non-spanmetrics
   * signals (Runtime, Custom, HyperDX sessions/webvitals, browser Issues).
   * Canonical UI/URL stays the inventory name. Missing entry → empty panels + hint.
   */
  serviceAliases?: Record<string, string>;
  /** Global ops watchlist — services monitored on the Ops Status Board. Editable by any user via the backend API. */
  opsWatchlist?: OpsWatchlistEntry[];
}
