// Package plugin implements the Grafana app plugin backend.
package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
	"github.com/grafana/grafana-plugin-sdk-go/config"
	"github.com/nais/grafana-otel-plugin/pkg/plugin/otelconfig"
	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
	"golang.org/x/sync/singleflight"
)

var (
	_ backend.CallResourceHandler   = (*App)(nil)
	_ instancemgmt.InstanceDisposer = (*App)(nil)
	_ backend.CheckHealthHandler    = (*App)(nil)
)

// App is the ClickStack APM plugin backend.
type App struct {
	backend.CallResourceHandler

	otelCfg      otelconfig.Config
	settings     queries.PluginSettings
	promClient   *queries.PrometheusClient
	chClient     *queries.ClickHouseClient // optional ClickStack / ClickHouse path
	healthClient *http.Client              // shared client for health checks
	grafanaURL   string                    // base URL for datasource proxy resolution
	serviceToken string                    // Grafana service account token for internal API calls

	capMu    sync.RWMutex
	capCache *cachedCapabilities
	capSF    singleflight.Group // coalesces concurrent capability probes on cache expiry

	respCache *responseCache // short-lived response cache for expensive queries

	scorecardOnce  sync.Once      // lazily builds the dedicated long-TTL scorecard cache
	scorecardCache *responseCache // see scorecardResponseCache()

	// ingressByService maps service_name → list of ingress hostnames that route to it.
	// Built from settings.IngressAliases (reversed). Used to expand caller queries.
	ingressByService map[string][]string
}

// NewApp creates a new App instance, parsing datasource configuration from jsonData.
func NewApp(_ context.Context, settings backend.AppInstanceSettings) (instancemgmt.Instance, error) {
	logger := log.DefaultLogger.With("component", "app")

	var app App
	app.otelCfg = otelconfig.Default()
	app.respCache = newResponseCache()
	app.healthClient = &http.Client{Timeout: 10 * time.Second}

	// Parse plugin settings from jsonData
	if len(settings.JSONData) > 0 {
		if err := json.Unmarshal(settings.JSONData, &app.settings); err != nil {
			logger.Warn("Failed to parse plugin jsonData", "error", err)
		}
	}

	// Apply label overrides — allows non-standard OTel pipelines (e.g. Tempo metrics generator,
	// which emits "service" instead of "service_name") to work without infra-side relabeling.
	if o := app.settings.LabelOverrides; (o != queries.LabelOverrides{}) {
		validLabel := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
		applied := 0
		if o.ServiceName != "" {
			if validLabel.MatchString(o.ServiceName) {
				app.otelCfg.Labels.ServiceName = o.ServiceName
				applied++
			} else {
				logger.Warn("Ignoring invalid serviceNameLabel override", "value", o.ServiceName)
			}
		}
		if o.ServiceNamespace != "" {
			if validLabel.MatchString(o.ServiceNamespace) {
				app.otelCfg.Labels.ServiceNamespace = o.ServiceNamespace
				applied++
			} else {
				logger.Warn("Ignoring invalid serviceNamespaceLabel override", "value", o.ServiceNamespace)
			}
		}
		if o.DeploymentEnv != "" {
			if validLabel.MatchString(o.DeploymentEnv) {
				app.otelCfg.Labels.DeploymentEnv = o.DeploymentEnv
				applied++
			} else {
				logger.Warn("Ignoring invalid deploymentEnvLabel override", "value", o.DeploymentEnv)
			}
		}
		if applied > 0 {
			logger.Info("Label overrides applied", "count", applied, "labels", app.otelCfg.Labels)
		}
	}

	// Build reverse ingress alias lookup: service_name → []hostnames.
	// Normalize hostnames (lowercase, strip trailing dots, strip standard ports)
	// so they match the normalized addresses produced by extractTopologyNodeName.
	if len(app.settings.IngressAliases) > 0 {
		normalized := make(map[string]string, len(app.settings.IngressAliases))
		app.ingressByService = make(map[string][]string, len(app.settings.IngressAliases))
		for hostname, svcName := range app.settings.IngressAliases {
			if hostname == "" || svcName == "" {
				continue
			}
			h := normalizeAddress(hostname)
			svc := strings.TrimSpace(svcName)
			normalized[h] = svc
			app.ingressByService[svc] = append(app.ingressByService[svc], h)
		}
		app.settings.IngressAliases = normalized
		logger.Info("Ingress aliases configured", "count", len(normalized), "services", len(app.ingressByService))
	}

	// Read Grafana service account token from secureJsonData.
	// When Grafana sits behind an OAuth2 proxy, the browser's session cookies are
	// for the proxy, not for Grafana. Since the plugin backend calls Grafana's
	// datasource proxy API on localhost (bypassing the proxy), we need a service
	// account token to authenticate those internal API calls.
	if token, ok := settings.DecryptedSecureJSONData["serviceAccountToken"]; ok && token != "" {
		app.serviceToken = token
		logger.Info("Service account token configured for internal API calls")
	}

	// Build internal Grafana URL for datasource proxy calls.
	// The plugin runs in the same process/pod as Grafana, so always use localhost
	// for API callbacks. GF_APP_URL is the external URL and may not be reachable
	// from inside the pod (ingress, network policies, OAuth2 proxy, etc.).
	app.grafanaURL = resolveInternalGrafanaURL(logger)

	if uid := app.settings.MetricsDataSource.UID; uid != "" {
		proxyURL := fmt.Sprintf("%s/api/datasources/proxy/uid/%s", app.grafanaURL, uid)
		app.promClient = queries.NewPrometheusClient(proxyURL, app.serviceToken)
		logger.Info("Metrics datasource configured via proxy", "uid", uid)
	} else {
		// Fallback for unconfigured plugin (e.g., first install before config is saved)
		logger.Warn("No metrics datasource configured — LGTM path inactive until configured")
	}

	if uid := app.settings.ClickHouseDataSource.UID; uid != "" {
		app.chClient = queries.NewClickHouseClient(app.grafanaURL, app.serviceToken, uid)
		logger.Info("ClickHouse datasource configured via /api/ds/query", "uid", uid)
	}

	logger.Info("Plugin initialized",
		"metricsDS", app.settings.MetricsDataSource.UID,
		"tracesDS", app.settings.TracesDataSource.UID,
		"logsDS", app.settings.LogsDataSource.UID,
		"clickHouseDS", app.settings.ClickHouseDataSource.UID,
		"envOverrides", len(app.settings.TracesDataSource.ByEnvironment)+len(app.settings.LogsDataSource.ByEnvironment)+len(app.settings.MetricsDataSource.ByEnvironment),
		"grafanaURL", app.grafanaURL,
	)

	mux := http.NewServeMux()
	app.registerRoutes(mux)
	app.CallResourceHandler = httpadapter.New(mux)

	return &app, nil
}

// Dispose is called when the plugin instance is shut down.
func (a *App) Dispose() {}

// ingressHostnames returns the configured ingress hostnames for a service, or nil.
func (a *App) ingressHostnames(service string) []string {
	if a.ingressByService == nil {
		return nil
	}
	return a.ingressByService[service]
}

// resolveInternalGrafanaURL builds the localhost URL for internal API calls.
// The plugin runs in the same process/pod as Grafana, so we always use localhost
// to avoid going through external ingress/load balancers/OAuth2 proxies.
// We preserve any sub-path from GF_APP_URL (for sub-path deployments).
func resolveInternalGrafanaURL(logger log.Logger) string {
	port := os.Getenv("GF_SERVER_HTTP_PORT")
	if port == "" {
		port = "3000"
	}
	base := "http://localhost:" + port

	// Preserve sub-path from GF_APP_URL if present (e.g., /grafana)
	if appURL := os.Getenv("GF_APP_URL"); appURL != "" {
		if u, err := url.Parse(appURL); err == nil && u.Path != "" && u.Path != "/" {
			base += strings.TrimRight(u.Path, "/")
		}
		logger.Info("Resolved internal Grafana URL", "internalURL", base, "GF_APP_URL", appURL)
	} else {
		logger.Info("Resolved internal Grafana URL", "internalURL", base, "GF_APP_URL", "(unset)")
	}

	return base
}

// proxyURL builds a Grafana datasource proxy URL for the given UID.
func (a *App) proxyURL(uid string) string {
	if uid == "" {
		return ""
	}
	return fmt.Sprintf("%s/api/datasources/proxy/uid/%s", a.grafanaURL, uid)
}

// tempoURL returns the Tempo proxy URL, optionally resolved for a specific environment.
func (a *App) tempoURL(env string) string {
	return a.proxyURL(a.settings.TracesDataSource.Resolve(env).UID)
}

// lokiURL returns the Loki proxy URL, optionally resolved for a specific environment.
func (a *App) lokiURL(env string) string {
	return a.proxyURL(a.settings.LogsDataSource.Resolve(env).UID)
}

// serviceGraphPrefix returns the detected service graph metric prefix.
// Falls back to "traces_service_graph" if not yet detected.
func (a *App) serviceGraphPrefix() string {
	a.capMu.RLock()
	defer a.capMu.RUnlock()
	if a.capCache != nil && a.capCache.caps.ServiceGraph.Prefix != "" {
		return a.capCache.caps.ServiceGraph.Prefix
	}
	return "traces_service_graph"
}

// callsMetric returns the detected span metrics calls metric name.
// Falls back to "traces_spanmetrics_calls_total" if not yet detected.
func (a *App) callsMetric(ctx context.Context) string {
	caps := a.cachedOrDetectCapabilities(ctx)
	if caps.SpanMetrics.CallsMetric != "" {
		return caps.SpanMetrics.CallsMetric
	}
	return "traces_spanmetrics_calls_total"
}

// promClientForRequest returns a PrometheusClient with auth resolved for the request.
// Auth priority: 1) auto-managed SA token (via IAM/externalServiceAccounts),
// 2) manual SA token (from secureJsonData), 3) forwarded user headers.
// Resolves the metrics datasource for the specified environment if configured.
func (a *App) promClientForRequest(r *http.Request) *queries.PrometheusClient {
	env := parseEnvironment(r)
	uid := a.settings.MetricsDataSource.Resolve(env).UID
	if uid == "" {
		if a.promClient == nil {
			return nil
		}
		if token := a.resolveServiceToken(r.Context()); token != "" {
			return a.promClient.WithServiceToken(token).WithAuthHeaders(r.Header)
		}
		return a.promClient.WithAuthHeaders(r.Header)
	}

	proxyURL := fmt.Sprintf("%s/api/datasources/proxy/uid/%s", a.grafanaURL, uid)
	client := queries.NewPrometheusClient(proxyURL, a.resolveServiceToken(r.Context()))
	return client.WithAuthHeaders(r.Header)
}

// resolveServiceToken returns the best available service account token.
// Prefers the auto-managed token from Grafana's externalServiceAccounts
// feature (PluginAppClientSecret), then falls back to the manual token
// from secureJsonData.
func (a *App) resolveServiceToken(ctx context.Context) string {
	cfg := config.GrafanaConfigFromContext(ctx)
	if cfg != nil {
		if token, err := cfg.PluginAppClientSecret(); err == nil && token != "" {
			return token
		}
	}
	return a.serviceToken
}

// requestContext builds a context with both auth-enhanced PrometheusClient and raw HTTP
// headers stored. This ensures that downstream code (capability detection, health checks)
// can access both the prom client and the user's auth headers.
func (a *App) requestContext(req *http.Request) context.Context {
	ctx := req.Context()
	ctx = withAuthContext(ctx, a.promClientForRequest(req))
	ctx = context.WithValue(ctx, httpHeadersCtxKey{}, req.Header.Clone())
	return ctx
}

// Context key for per-request PrometheusClient with forwarded auth.
type promClientCtxKey struct{}

// Context key for raw HTTP headers (used by health checks).
type httpHeadersCtxKey struct{}

// withAuthContext stores an auth-enhanced PrometheusClient in the context.
func withAuthContext(ctx context.Context, c *queries.PrometheusClient) context.Context {
	return context.WithValue(ctx, promClientCtxKey{}, c)
}

// httpHeaders returns the stored HTTP headers from context, or an empty header.
func httpHeaders(ctx context.Context) http.Header {
	if h, ok := ctx.Value(httpHeadersCtxKey{}).(http.Header); ok && h != nil {
		return h
	}
	return http.Header{}
}

// prom returns the per-request auth-enhanced PrometheusClient, or falls back to the base client.
func (a *App) prom(ctx context.Context) *queries.PrometheusClient {
	if c, ok := ctx.Value(promClientCtxKey{}).(*queries.PrometheusClient); ok && c != nil {
		return c
	}
	return a.promClient
}

// CheckHealth validates datasource connectivity.
//
// When a ClickHouse datasource UID is configured, connectivity is verified with
// SELECT 1 via /api/ds/query. The legacy LGTM (Mimir/Tempo) checks still run
// when a metrics datasource is configured so existing installs keep working
// during the ClickStack migration.
func (a *App) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	// Forward auth headers from the health check request for datasource proxy calls
	h := make(http.Header)
	for k, v := range req.Headers {
		h.Set(k, v)
	}

	// Resolve the best available service token
	token := a.resolveServiceToken(ctx)

	if a.promClient != nil {
		client := a.promClient.WithAuthHeaders(h)
		if token != "" {
			client = client.WithServiceToken(token)
		}
		ctx = withAuthContext(ctx, client)
	}

	type healthDetails struct {
		ClickHouse *clickHouseHealthStatus `json:"clickHouse,omitempty"`
		Legacy     *queries.Capabilities   `json:"legacy,omitempty"`
	}
	details := healthDetails{}
	var parts []string

	if a.chClient != nil {
		ch := a.chClient.WithAuthHeaders(h)
		if token != "" {
			ch = ch.WithServiceToken(token)
		}
		st := &clickHouseHealthStatus{UID: ch.DatasourceUID(), Available: true}
		if err := ch.Ping(ctx); err != nil {
			reason := summarizeClickHouseError(err)
			st.Available = false
			st.Error = reason
			details.ClickHouse = st
			raw, _ := json.Marshal(details)
			return &backend.CheckHealthResult{
				Status:      backend.HealthStatusError,
				Message:     "ClickHouse is not reachable (" + reason + ")",
				JSONDetails: raw,
			}, nil
		}
		details.ClickHouse = st
		parts = append(parts, "ClickHouse OK ("+ch.DatasourceUID()+")")
	}

	if a.promClient != nil {
		caps := a.detectCapabilities(ctx, h, token)
		details.Legacy = &caps

		if !caps.SpanMetrics.Detected {
			// ClickHouse / ClickStack mode is healthy without Mimir span metrics.
			if a.chClient == nil {
				raw, _ := json.Marshal(details)
				return &backend.CheckHealthResult{
					Status:      backend.HealthStatusError,
					Message:     "No span metrics detected. Ensure the OTel Collector spanmetrics connector writes to your metrics store (or ClickHouse in ClickStack mode).",
					JSONDetails: raw,
				}, nil
			}
			parts = append(parts, "LGTM span metrics not detected")
		} else {
			// Tempo is only required on the LGTM path. ClickStack serves traces from CH.
			if a.chClient == nil && !caps.Tempo.Available && len(caps.TempoByEnv) == 0 {
				raw, _ := json.Marshal(details)
				return &backend.CheckHealthResult{
					Status:      backend.HealthStatusError,
					Message:     "Tempo is not reachable: " + caps.Tempo.Error,
					JSONDetails: raw,
				}, nil
			}
			parts = append(parts, fmt.Sprintf("%s metrics, %d services", caps.SpanMetrics.Namespace, len(caps.Services)))
		}
	}

	if a.promClient == nil && a.chClient == nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "No datasources configured. Set metricsDataSource (LGTM) and/or clickHouseDataSource (ClickStack) in plugin settings.",
		}, nil
	}

	raw, err := json.Marshal(details)
	if err != nil {
		log.DefaultLogger.Warn("Failed to marshal health details", "error", err)
		raw = []byte("{}")
	}
	msg := "OK"
	if len(parts) > 0 {
		msg = "OK — " + strings.Join(parts, "; ")
	}
	return &backend.CheckHealthResult{
		Status:      backend.HealthStatusOk,
		Message:     msg,
		JSONDetails: raw,
	}, nil
}

// clickHouseHealthStatus is embedded in CheckHealth JSONDetails.
type clickHouseHealthStatus struct {
	UID       string `json:"uid"`
	Available bool   `json:"available"`
	Error     string `json:"error,omitempty"`
}

// appRoute pairs a net/http mux pattern with its handler. The pattern syntax is
// Go 1.22+ ServeMux ({name} wildcards), which is also OpenAPI path syntax.
type appRoute struct {
	pattern string
	handler http.HandlerFunc
}

// routes returns the full list of registered resource routes. It is the single
// source of truth for both route registration (registerRoutes) and the
// documentation-drift test (apidocs_test.go), which asserts every pattern here
// appears in docs/api/openapi.yaml. The receiver may be a zero-value *App; only
// the method values (not their execution) are read when enumerating patterns.
func (a *App) routes() []appRoute {
	return []appRoute{
		{"/capabilities", a.handleCapabilities},
		{"/services", a.handleServices},
		{"/services/{namespace}/{service}/health", a.handleHealth},
		{"/services/{namespace}/{service}/operations", a.handleOperations},
		{"/services/{namespace}/{service}/endpoints", a.handleEndpoints},
		{"/services/{namespace}/{service}/frontend", a.handleFrontendMetrics},
		{"/services/{namespace}/{service}/frontend/versions", a.handleFrontendVersions},
		{"/services/{namespace}/{service}/frontend/sessions", a.handleFrontendSessions},
		{"/services/{namespace}/{service}/frontend/sessions/recordings", a.handleSessionRecordings},
		{"/services/{namespace}/{service}/frontend/sessions/{sessionId}/replay", a.handleSessionReplay},
		{"/services/{namespace}/{service}/frontend/sessions/{sessionId}/replay/probe", a.handleSessionReplayProbe},
		{"/services/{namespace}/{service}/logs/patterns", a.handleLogPatterns},
		{"/services/{namespace}/{service}/logs/search", a.handleLogSearch},
		{"/services/{namespace}/{service}/sourcemap-doctor", a.handleSourcemapDoctor},
		{"/services/{namespace}/{service}/traces/breakdown", a.handleTraceBreakdown},
		{"/services/{namespace}/{service}/traces/search", a.handleTraceSearch},
		{"/services/{namespace}/{service}/traces/{traceId}", a.handleTraceDetail},
		{"/services/{namespace}/{service}/feedback", a.handleFeedback},
		{"/services/{namespace}/{service}/exceptions/groups", a.handleExceptionGroups},
		{"/services/{namespace}/{service}/issues", a.handleIssues},
		{"/services/{namespace}/{service}/issues/occurrences", a.handleIssueOccurrences},
		{"/services/{namespace}/{service}/triage", a.handleTriageStates},
		{"/services/{namespace}/{service}/triage/{fingerprint}", a.handleTriageAction},
		{"/services/{namespace}/{service}/triage/{fingerprint}/history", a.handleTriageHistory},
		{"/services/{namespace}/{service}/dependencies", a.handleServiceDependencies},
		{"/services/{namespace}/{service}/connected", a.handleConnectedServices},
		{"/services/{namespace}/{service}/graphql", a.handleGraphQLMetrics},
		{"/services/{namespace}/{service}/runtime", a.handleRuntime},
		{"/services/{namespace}/{service}/custom-metrics", a.handleCustomMetrics},
		{"/services/{namespace}/{service}/database/queries", a.handleDatabaseQueries},
		{"/services/{namespace}/{service}/database/nplusone", a.handleDatabaseNPlusOne},
		{"/services/{namespace}/{service}/scorecard", a.handleScorecard},
		{"/services/{namespace}/{service}/alerts", a.handleServiceAlerts},
		{"/jobs", a.handleJobs},
		{"/service-map", a.handleServiceMap},
		{"/service-map/clustered", a.handleClusteredServiceMap},
		{"/dependencies", a.handleGlobalDependencies},
		{"/dependencies/{name}", a.handleDependencyDetail},
		{"/namespaces/{namespace}/dependencies", a.handleNamespaceDependencies},
		{"/namespaces/{namespace}/alerts", a.handleNamespaceAlerts},
		{"/namespaces/{namespace}/issues-attention", a.handleNamespaceIssuesAttention},
		{"/namespaces/{namespace}/scorecard", a.handleNamespaceScorecard},
		{"/alert-templates/{kind}", a.handleAlertTemplate},
		{"/ops-watchlist", a.handleOpsWatchlist},
		{"/ping", a.handlePing},
	}
}

func (a *App) registerRoutes(mux *http.ServeMux) {
	for _, r := range a.routes() {
		mux.HandleFunc(r.pattern, r.handler)
	}
}
