package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// Service scorecard: an "observability readiness" score computed from signals
// the plugin already probes (span metrics, traces, logs, runtime metrics,
// browser telemetry, alert rules).

const (
	// scorecardCacheTTL is deliberately long: the checks are presence probes
	// whose answers change on deploy cadence, not query cadence.
	scorecardCacheTTL = 5 * time.Minute
	// scorecardProbeTimeout bounds each individual readiness probe.
	scorecardProbeTimeout = 10 * time.Second
	// scorecardTraceLookback is the window the Tempo search probe scans.
	scorecardTraceLookback = time.Hour
	// maxScorecardRuleScan caps how many alert rules the "has alert rules"
	// check inspects — rulers can hold tens of thousands of rules fleet-wide.
	maxScorecardRuleScan = 5000
)

// ScorecardCheck is one observability readiness check with an enablement hint.
type ScorecardCheck struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	OK    bool   `json:"ok"`
	// Hint says how to enable the capability — or, for a not-applicable check,
	// why it does not apply.
	Hint string `json:"hint"`
	// NotApplicable marks a check that does not apply to this service (e.g.
	// browser telemetry for a pure JVM backend). Not counted in Score/Total.
	NotApplicable bool `json:"notApplicable,omitempty"`
}

// ScorecardReadiness is the computed observability readiness score.
type ScorecardReadiness struct {
	// Score is the number of passing APPLICABLE checks; the fraction is
	// Score/Total. Not-applicable checks are excluded from both.
	Score  int              `json:"score"`
	Total  int              `json:"total"`
	Checks []ScorecardCheck `json:"checks"`
}

// ScorecardResponse is the /scorecard payload.
type ScorecardResponse struct {
	Readiness ScorecardReadiness `json:"readiness"`
}

// scorecardResponseCache lazily builds the dedicated long-TTL cache. Kept
// separate from respCache so the aggressive TTL doesn't leak into the
// query-shaped endpoints sharing that cache.
func (a *App) scorecardResponseCache() *responseCache {
	a.scorecardOnce.Do(func() {
		c := newResponseCache()
		c.ttl = scorecardCacheTTL
		a.scorecardCache = c
	})
	return a.scorecardCache
}

// handleScorecard returns the service scorecard.
// GET /services/{namespace}/{service}/scorecard?environment=
func (a *App) handleScorecard(w http.ResponseWriter, req *http.Request) {
	if !requireGET(w, req) {
		return
	}
	ctx := a.requestContext(req)
	namespace, service := parseServiceRef(req)
	if !requireServiceParam(w, service) {
		return
	}
	env := parseEnvironment(req)

	orgID := req.Header.Get("X-Grafana-Org-Id")
	ck := cacheKey("scorecard", orgID, namespace, service, env)
	headers := req.Header.Clone()

	cache := a.scorecardResponseCache()
	if cached, ok := cache.get(ck); ok {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		_, _ = w.Write(cached)
		return
	}
	data, err := cache.getOrCompute(ck, func() (any, error) {
		return a.computeScorecard(ctx, headers, namespace, service, env), nil
	})
	if err != nil {
		http.Error(w, "computing scorecard failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func (a *App) computeScorecard(ctx context.Context, headers http.Header, namespace, service, env string) ScorecardResponse {
	return ScorecardResponse{Readiness: a.computeReadiness(ctx, headers, namespace, service, env)}
}

// ---------------------------------------------------------------------------
// Observability readiness
// ---------------------------------------------------------------------------

func (a *App) computeReadiness(ctx context.Context, headers http.Header, namespace, service, env string) ScorecardReadiness {
	now := time.Now()
	token := a.resolveServiceToken(ctx)

	// Fixed order and fixed slots: each goroutine writes only its own index.
	checks := []ScorecardCheck{
		{Key: "spanMetrics", Label: "Span metrics (RED)",
			Hint: "Export span metrics (or enable a metrics-generator / spanmetrics connector) so RED panels have data."},
		{Key: "traces", Label: "Traces",
			Hint: "Export OTLP traces to your collector / ClickStack so distributed traces appear."},
		{Key: "logs", Label: "Logs",
			Hint: "Ship application logs via OTLP (or Loki in dual-mode) so log patterns and correlation work."},
		{Key: "runtimeMetrics", Label: "Runtime metrics",
			Hint: "Runtime metrics (JVM/Node.js/Go/process) arrive with the OTel agent — enable auto-instrumentation for your runtime."},
		{Key: "browserTelemetry", Label: "Browser telemetry",
			Hint: "Instrument your frontend with @hyperdx/browser, Grafana Faro, or the OpenTelemetry Browser SDK."},
		{Key: "alertRules", Label: "Alert rules",
			Hint: "No alert rule mentions this service — start with 'Alert on error rate' in the service actions menu."},
	}

	var wg sync.WaitGroup
	wg.Add(6)
	go func() { defer wg.Done(); checks[0].OK = a.checkSpanMetrics(ctx, namespace, service, env, now) }()
	go func() {
		defer wg.Done()
		checks[1].OK = a.checkTempoTraces(ctx, headers, token, namespace, service, env, now)
	}()
	go func() { defer wg.Done(); checks[2].OK = a.checkLokiLogs(ctx, headers, token, service, env, now) }()
	go func() { defer wg.Done(); checks[3].OK = a.checkRuntimeMetrics(ctx, namespace, service, env, now) }()
	go func() {
		defer wg.Done()
		checks[4].OK = a.checkBrowserTelemetry(ctx, service, env, now)
		// A pure backend has no frontend to instrument: when no Faro data
		// exists AND every SDK language the service reports is backend-only
		// (JVM, Go, .NET, …), "missing browser telemetry" is not an actionable
		// gap — mark the check not applicable instead of failing it. Node.js
		// deliberately stays applicable: Node services are frequently SSR
		// frontends/BFFs where Faro is expected.
		if !checks[4].OK {
			if langs := a.serviceSDKLanguages(ctx, namespace, service, env, now); len(langs) > 0 && allBackendOnlyLanguages(langs) {
				checks[4].NotApplicable = true
				checks[4].Hint = fmt.Sprintf(
					"Not applicable — %s backend with no browser frontend. If this service gets a frontend, instrument it with Faro or the OpenTelemetry Browser SDK.",
					strings.Join(langs, "/"))
			}
		}
	}()
	go func() { defer wg.Done(); checks[5].OK = a.checkAlertRules(ctx, headers, token, service) }()
	wg.Wait()

	score, total := 0, 0
	for _, c := range checks {
		if c.NotApplicable {
			continue
		}
		total++
		if c.OK {
			score++
		}
	}
	return ScorecardReadiness{Score: score, Total: total, Checks: checks}
}

// backendOnlySDKLanguages are telemetry.sdk.language values that cannot serve
// a browser frontend from the same service. nodejs is intentionally absent
// (SSR/BFF ambiguity); webjs is the Faro browser SDK itself.
var backendOnlySDKLanguages = map[string]bool{
	"java": true, "go": true, "dotnet": true, "python": true,
	"rust": true, "php": true, "ruby": true, "erlang": true,
	"elixir": true, "cpp": true, "swift": true,
}

func allBackendOnlyLanguages(langs []string) bool {
	for _, l := range langs {
		if !backendOnlySDKLanguages[l] {
			return false
		}
	}
	return true
}

// serviceSDKLanguages returns the distinct telemetry.sdk.language values the
// service reports on its span metrics (empty when unknown/unavailable) —
// the same label the service inventory uses for framework derivation.
func (a *App) serviceSDKLanguages(ctx context.Context, namespace, service, env string, at time.Time) []string {
	client := a.prom(ctx)
	if client == nil {
		return nil
	}
	filter := a.otelCfg.ServiceFilter(service, namespace)
	if m := envMatcher(a.otelCfg.Labels.DeploymentEnv, env); m != "" {
		filter += ", " + m
	}
	ctx, cancel := context.WithTimeout(ctx, scorecardProbeTimeout)
	defer cancel()
	results, err := client.InstantQuery(ctx, fmt.Sprintf(
		`group by (%s) (%s{%s})`, a.otelCfg.Labels.SDKLanguage, a.callsMetric(ctx), filter), at)
	if err != nil {
		log.DefaultLogger.Debug("Scorecard SDK-language probe failed", "error", err)
		return nil
	}
	langs := make([]string, 0, len(results))
	for _, r := range results {
		if lang := r.Metric[a.otelCfg.Labels.SDKLanguage]; lang != "" {
			langs = append(langs, lang)
		}
	}
	return langs
}

// promSelectorNonEmpty runs count(<selector>) as an instant query and reports
// whether any series matched. Errors degrade to false (check not passed).
func (a *App) promSelectorNonEmpty(ctx context.Context, selector string, at time.Time) bool {
	client := a.prom(ctx)
	if client == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, scorecardProbeTimeout)
	defer cancel()
	results, err := client.InstantQuery(ctx, fmt.Sprintf(`count(%s)`, selector), at)
	if err != nil {
		log.DefaultLogger.Debug("Scorecard prom probe failed", "selector", sanitizeLogValue(selector), "error", err)
		return false
	}
	return len(results) > 0
}

func (a *App) checkSpanMetrics(ctx context.Context, namespace, service, env string, at time.Time) bool {
	filter := a.otelCfg.ServiceFilter(service, namespace)
	if m := envMatcher(a.otelCfg.Labels.DeploymentEnv, env); m != "" {
		filter += ", " + m
	}
	return a.promSelectorNonEmpty(ctx, fmt.Sprintf(`%s{%s}`, a.callsMetric(ctx), filter), at)
}

func (a *App) checkRuntimeMetrics(ctx context.Context, namespace, service, env string, at time.Time) bool {
	// Same metric families as the Runtime tab's discovery query.
	filter := a.otelCfg.RuntimeFilter(service, namespace)
	if m := envMatcher(a.otelCfg.Labels.DeploymentEnv, env); m != "" {
		filter += ", " + m
	}
	return a.promSelectorNonEmpty(ctx, fmt.Sprintf(
		`{%s, __name__=~"jvm_.*|nodejs_.*|hikaricp_.*|db_client_connections_.*|kafka_consumer_.*|kafka_producer_.*|process_.*|system_.*|go_.*"}`,
		filter), at)
}

func (a *App) checkBrowserTelemetry(ctx context.Context, service, env string, at time.Time) bool {
	// Presence of any Alloy-generated Faro metric for this app — mirrors the
	// Frontend tab's histogram detection but as a single cheap probe.
	return a.promSelectorNonEmpty(ctx, fmt.Sprintf(
		`{%s, __name__=~".*faro_.*"}`, a.otelCfg.AlloyHistogramFilter(service, env)), at)
}

// checkTempoTraces asks Tempo's search API (through the datasource proxy) for
// one trace from the service in the recent window.
func (a *App) checkTempoTraces(ctx context.Context, headers http.Header, token, namespace, service, env string, at time.Time) bool {
	base := a.tempoURL(env)
	if base == "" {
		return false
	}
	traceql := fmt.Sprintf(`{%s="%s"}`, a.otelCfg.TraceQL.ServiceName, service)
	if namespace != "" {
		traceql = fmt.Sprintf(`{%s="%s" && %s="%s"}`,
			a.otelCfg.TraceQL.ServiceName, service, a.otelCfg.TraceQL.ServiceNamespace, namespace)
	}
	q := url.Values{
		"q":     {traceql},
		"limit": {"1"},
		"start": {fmt.Sprintf("%d", at.Add(-scorecardTraceLookback).Unix())},
		"end":   {fmt.Sprintf("%d", at.Unix())},
	}

	body, err := a.proxyGET(ctx, headers, token, base+"/api/search?"+q.Encode())
	if err != nil {
		log.DefaultLogger.Debug("Scorecard tempo probe failed", "error", err)
		return false
	}
	var parsed struct {
		Traces []json.RawMessage `json:"traces"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return false
	}
	return len(parsed.Traces) > 0
}

// checkLokiLogs asks Loki's index/stats API (cheap: index-only, no chunks)
// whether any streams exist for the service in the recent window.
func (a *App) checkLokiLogs(ctx context.Context, headers http.Header, token, service, env string, at time.Time) bool {
	base := a.lokiURL(env)
	if base == "" {
		return false
	}
	q := url.Values{
		"query": {fmt.Sprintf(`{%s="%s"}`, a.otelCfg.FaroLoki.ServiceName, service)},
		"start": {fmt.Sprintf("%d", at.Add(-scorecardTraceLookback).Unix())},
		"end":   {fmt.Sprintf("%d", at.Unix())},
	}

	body, err := a.proxyGET(ctx, headers, token, base+"/loki/api/v1/index/stats?"+q.Encode())
	if err != nil {
		log.DefaultLogger.Debug("Scorecard loki probe failed", "error", err)
		return false
	}
	var parsed struct {
		Streams int64 `json:"streams"`
		Entries int64 `json:"entries"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return false
	}
	return parsed.Streams > 0 || parsed.Entries > 0
}

// checkAlertRules scans Mimir ruler rules and Grafana-managed rules for any
// alerting rule that mentions the service, capped at maxScorecardRuleScan.
func (a *App) checkAlertRules(ctx context.Context, headers http.Header, token, service string) bool {
	ctx, cancel := context.WithTimeout(ctx, scorecardProbeTimeout)
	defer cancel()

	var (
		wg      sync.WaitGroup
		sources [2]*queries.RulesResponse
	)
	if prom := a.prom(ctx); prom != nil {
		wg.Go(func() {
			rules, err := prom.GetAlertRules(ctx)
			if err != nil {
				log.DefaultLogger.Debug("Scorecard mimir rules probe failed", "error", err)
				return
			}
			sources[0] = rules
		})
	}
	if a.grafanaURL != "" {
		wg.Go(func() {
			client := queries.NewPrometheusClient(a.grafanaURL+"/api/prometheus/grafana", token).WithAuthHeaders(headers)
			rules, err := client.GetAlertRules(ctx)
			if err != nil {
				log.DefaultLogger.Debug("Scorecard grafana rules probe failed", "error", err)
				return
			}
			sources[1] = rules
		})
	}
	wg.Wait()

	scanned := 0
	for _, src := range sources {
		if src == nil {
			continue
		}
		for _, group := range src.Groups {
			for _, rule := range group.Rules {
				if scanned >= maxScorecardRuleScan {
					return false
				}
				scanned++
				if rule.Type != "alerting" {
					continue
				}
				if ruleMentionsService(rule, service) {
					return true
				}
			}
		}
	}
	return false
}

// ruleMentionsService reports whether an alerting rule targets the service.
// Matching is deliberately conservative (quoted occurrences in the query
// expression, or exact label values) — short service names like "api" would
// otherwise substring-match half the fleet's rules.
func ruleMentionsService(rule queries.Rule, service string) bool {
	if service == "" {
		return false
	}
	// PromQL expressions: service_name="svc"; Grafana-managed rules expose the
	// JSON-encoded query data where quotes are escaped: \"svc\".
	if strings.Contains(rule.Query, `"`+service+`"`) || strings.Contains(rule.Query, `\"`+service+`\"`) {
		return true
	}
	for _, key := range []string{"service", "service_name", "app", "app_name", "deployment"} {
		if rule.Labels[key] == service {
			return true
		}
	}
	return false
}

// proxyGET performs an authenticated GET through the Grafana datasource proxy
// and returns the body on 2xx.
func (a *App) proxyGET(ctx context.Context, headers http.Header, token, reqURL string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, scorecardProbeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	applyProxyAuth(req, headers, token)

	resp, err := a.healthClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close() //nolint:errcheck
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 10<<20))
}
