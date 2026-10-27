package plugin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/otelconfig"
	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// scorecardFixture controls which observability signals the fake backends
// report for the probed service.
type scorecardFixture struct {
	spanMetrics bool
	traces      bool
	logs        bool
	runtime     bool
	browser     bool
	alertRule   bool
	sdkLanguage string // telemetry_sdk_language answered by the language probe ("" → no series)
}

// newScorecardApp wires fake Prometheus and Grafana-proxy (Tempo/Loki/rules)
// into an App.
func newScorecardApp(t *testing.T, fx scorecardFixture) *App {
	t.Helper()

	// Fake Mimir: instant queries answered by substring, /api/v1/rules empty.
	prom := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/api/v1/rules") {
			_, _ = w.Write([]byte(`{"status":"success","data":{"groups":[]}}`))
			return
		}
		query := r.URL.Query().Get("query")
		// The SDK-language probe also mentions the calls metric, so match it first.
		if strings.Contains(query, "telemetry_sdk_language") {
			result := `[]`
			if fx.sdkLanguage != "" {
				result = `[{"metric":{"telemetry_sdk_language":"` + fx.sdkLanguage + `"},"value":[1700000000,"1"]}]`
			}
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":` + result + `}}`))
			return
		}
		hit := false
		switch {
		case strings.Contains(query, "calls_total"):
			hit = fx.spanMetrics
		case strings.Contains(query, "jvm_"):
			hit = fx.runtime
		case strings.Contains(query, "faro_"):
			hit = fx.browser
		}
		result := `[]`
		if hit {
			result = `[{"metric":{},"value":[1700000000,"3"]}]`
		}
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":` + result + `}}`))
	}))
	t.Cleanup(prom.Close)

	// Fake Grafana: datasource proxy for Tempo search + Loki index/stats, and
	// the Grafana-managed rules API.
	grafana := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/proxy/uid/tempo/api/search"):
			if fx.traces {
				_, _ = w.Write([]byte(`{"traces":[{"traceID":"abc"}]}`))
			} else {
				_, _ = w.Write([]byte(`{"traces":[]}`))
			}
		case strings.Contains(r.URL.Path, "/proxy/uid/loki/loki/api/v1/index/stats"):
			if fx.logs {
				_, _ = w.Write([]byte(`{"streams":4,"chunks":10,"entries":100,"bytes":1000}`))
			} else {
				_, _ = w.Write([]byte(`{"streams":0,"chunks":0,"entries":0,"bytes":0}`))
			}
		case strings.Contains(r.URL.Path, "/api/prometheus/grafana/api/v1/rules"):
			groups := `[]`
			if fx.alertRule {
				groups = `[{"name":"g","file":"team-a","rules":[
					{"type":"alerting","name":"my-app error rate","query":"sum(rate(calls_total{service_name=\"my-app\"}[5m]))","labels":{}}
				]}]`
			}
			_, _ = w.Write([]byte(`{"status":"success","data":{"groups":` + groups + `}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(grafana.Close)

	app := &App{
		otelCfg:      otelconfig.Default(),
		respCache:    newResponseCache(),
		promClient:   queries.NewPrometheusClient(prom.URL, ""),
		healthClient: &http.Client{Timeout: 5 * time.Second},
		grafanaURL:   grafana.URL,
		settings: queries.PluginSettings{
			TracesDataSource: queries.EnvAwareDataSource{UID: "tempo"},
			LogsDataSource:   queries.EnvAwareDataSource{UID: "loki"},
		},
	}
	app.capCache = &cachedCapabilities{caps: defaultCaps(), fetchedAt: time.Now()}
	return app
}

func getScorecard(t *testing.T, app *App) ScorecardResponse {
	const namespace, service = "team-a", "my-app"
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/services/"+namespace+"/"+service+"/scorecard", nil)
	req.SetPathValue("namespace", namespace)
	req.SetPathValue("service", service)
	w := httptest.NewRecorder()
	app.handleScorecard(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp ScorecardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	return resp
}

func TestScorecardReadinessScore(t *testing.T) {
	cases := []struct {
		name      string
		fx        scorecardFixture
		wantScore int
		wantTotal int
		wantOK    map[string]bool
		wantNA    map[string]bool
	}{
		{
			name:      "nothing instrumented",
			fx:        scorecardFixture{},
			wantScore: 0,
			wantTotal: 6,
		},
		{
			name:      "fully instrumented",
			fx:        scorecardFixture{spanMetrics: true, traces: true, logs: true, runtime: true, browser: true, alertRule: true},
			wantScore: 6,
			wantTotal: 6,
		},
		{
			name:      "backend service, unknown language: Faro stays actionable",
			fx:        scorecardFixture{spanMetrics: true, traces: true, logs: true, runtime: true},
			wantScore: 4,
			wantTotal: 6,
			wantOK: map[string]bool{
				"spanMetrics": true, "traces": true, "logs": true,
				"runtimeMetrics": true, "browserTelemetry": false, "alertRules": false,
			},
		},
		{
			name:      "pure JVM backend: Faro not applicable, excluded from score",
			fx:        scorecardFixture{spanMetrics: true, traces: true, logs: true, runtime: true, sdkLanguage: "java"},
			wantScore: 4,
			wantTotal: 5,
			wantOK: map[string]bool{
				"spanMetrics": true, "traces": true, "logs": true,
				"runtimeMetrics": true, "browserTelemetry": false, "alertRules": false,
			},
			wantNA: map[string]bool{"browserTelemetry": true},
		},
		{
			name:      "node service: Faro stays applicable (SSR/BFF ambiguity)",
			fx:        scorecardFixture{spanMetrics: true, sdkLanguage: "nodejs"},
			wantScore: 1,
			wantTotal: 6,
			wantOK:    map[string]bool{"browserTelemetry": false},
			wantNA:    map[string]bool{"browserTelemetry": false},
		},
		{
			name:      "JVM service WITH faro data: check passes and stays applicable",
			fx:        scorecardFixture{browser: true, sdkLanguage: "java"},
			wantScore: 1,
			wantTotal: 6,
			wantOK:    map[string]bool{"browserTelemetry": true},
			wantNA:    map[string]bool{"browserTelemetry": false},
		},
		{
			name:      "alert rule via grafana-managed rules",
			fx:        scorecardFixture{alertRule: true},
			wantScore: 1,
			wantTotal: 6,
			wantOK:    map[string]bool{"alertRules": true},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newScorecardApp(t, tc.fx)
			resp := getScorecard(t, app)

			if resp.Readiness.Total != tc.wantTotal {
				t.Fatalf("total = %d, want %d", resp.Readiness.Total, tc.wantTotal)
			}
			if resp.Readiness.Score != tc.wantScore {
				t.Errorf("score = %d, want %d (checks: %+v)", resp.Readiness.Score, tc.wantScore, resp.Readiness.Checks)
			}
			byKey := map[string]ScorecardCheck{}
			for _, c := range resp.Readiness.Checks {
				byKey[c.Key] = c
				if c.Label == "" || c.Hint == "" {
					t.Errorf("check %q missing label/hint", c.Key)
				}
			}
			for _, gated := range []string{"graphql", "jobs", "profiling"} {
				if !byKey[gated].NotApplicable {
					t.Errorf("check %q should be NotApplicable until enablement", gated)
				}
			}
			for key, want := range tc.wantOK {
				if byKey[key].OK != want {
					t.Errorf("check %q ok = %v, want %v", key, byKey[key].OK, want)
				}
			}
			for key, want := range tc.wantNA {
				if byKey[key].NotApplicable != want {
					t.Errorf("check %q notApplicable = %v, want %v", key, byKey[key].NotApplicable, want)
				}
			}
		})
	}
}

func TestRuleMentionsService(t *testing.T) {
	cases := []struct {
		name    string
		rule    queries.Rule
		service string
		want    bool
	}{
		{"quoted in promql", queries.Rule{Query: `rate(calls_total{service_name="my-app"}[5m])`}, "my-app", true},
		{"escaped quotes (grafana json)", queries.Rule{Query: `{"expr":"up{app=\"my-app\"}"}`}, "my-app", true},
		{"label match", queries.Rule{Labels: map[string]string{"app": "my-app"}}, "my-app", true},
		{"substring of other service does not match", queries.Rule{Query: `rate(calls_total{service_name="my-app-worker"}[5m])`}, "my-app", false},
		{"no mention", queries.Rule{Query: `up == 0`, Labels: map[string]string{"team": "x"}}, "my-app", false},
		{"empty service", queries.Rule{Query: `""`}, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ruleMentionsService(tc.rule, tc.service); got != tc.want {
				t.Errorf("ruleMentionsService() = %v, want %v", got, tc.want)
			}
		})
	}
}
