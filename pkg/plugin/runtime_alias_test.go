package plugin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

func TestHandleRuntime_ClickStackAliasMissHint(t *testing.T) {
	app := &App{
		settings: queries.PluginSettings{
			ClickHouseDataSource: queries.DataSourceRef{UID: "clickhouse"},
		},
		chClient: &queries.ClickHouseClient{},
	}
	req := httptest.NewRequest(http.MethodGet, "/services/sanad/nova-api/runtime?from=1&to=2", nil)
	req.SetPathValue("namespace", "sanad")
	req.SetPathValue("service", "nova-api")
	w := httptest.NewRecorder()
	app.handleRuntime(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp queries.RuntimeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Hint == "" || !strings.Contains(resp.Hint, "serviceAliases") || !strings.Contains(resp.Hint, "nova-api") {
		t.Fatalf("hint=%q", resp.Hint)
	}
	if resp.Go != nil || resp.Container != nil {
		t.Fatal("expected empty runtime payload on alias miss")
	}
}

func TestHandleRuntime_ClickStackAliasHitNoWrongNameQuery(t *testing.T) {
	app := &App{
		settings: queries.PluginSettings{
			ClickHouseDataSource: queries.DataSourceRef{UID: "clickhouse"},
			ServiceAliases:       map[string]string{"nova-api": "nova-app"},
		},
		chClient: &queries.ClickHouseClient{},
	}
	req := httptest.NewRequest(http.MethodGet, "/services/sanad/nova-api/runtime?from=1&to=2", nil)
	req.SetPathValue("namespace", "sanad")
	req.SetPathValue("service", "nova-api")
	w := httptest.NewRecorder()
	app.handleRuntime(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp queries.RuntimeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Hint != "" {
		t.Fatalf("unexpected hint on hit: %q", resp.Hint)
	}
}

func TestHandleCustomMetrics_ClickStackAliasMissHint(t *testing.T) {
	app := &App{
		settings: queries.PluginSettings{
			ClickHouseDataSource: queries.DataSourceRef{UID: "clickhouse"},
		},
		chClient: &queries.ClickHouseClient{},
	}
	req := httptest.NewRequest(http.MethodGet, "/services/sanad/admin-api/custom-metrics?from=1&to=2", nil)
	req.SetPathValue("namespace", "sanad")
	req.SetPathValue("service", "admin-api")
	w := httptest.NewRecorder()
	app.handleCustomMetrics(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var resp CustomMetricsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Hint == "" || !strings.Contains(resp.Hint, "admin-api") || !strings.Contains(resp.Hint, "serviceAliases") {
		t.Fatalf("hint=%q", resp.Hint)
	}
	if len(resp.Metrics) != 0 {
		t.Fatalf("metrics=%v", resp.Metrics)
	}
}
