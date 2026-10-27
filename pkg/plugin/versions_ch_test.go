package plugin

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

func chHandlerClient(t *testing.T, handler http.HandlerFunc) *queries.ClickHouseClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := queries.NewClickHouseClient(srv.URL, "test-token", "clickhouse")
	return client
}

func TestHandleFrontendVersions_ClickStackAliasMissHint(t *testing.T) {
	app := &App{
		settings: queries.PluginSettings{
			ClickHouseDataSource: queries.DataSourceRef{UID: "clickhouse"},
		},
		chClient: &queries.ClickHouseClient{},
	}
	req := httptest.NewRequest(http.MethodGet, "/services/sanad/nova-frontend/frontend/versions?from=1&to=2", nil)
	req.SetPathValue("namespace", "sanad")
	req.SetPathValue("service", "nova-frontend")
	w := httptest.NewRecorder()
	app.handleFrontendVersions(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp FrontendVersionsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Hint == "" || !strings.Contains(resp.Hint, "serviceAliases") {
		t.Fatalf("hint=%q", resp.Hint)
	}
	if len(resp.Versions) != 0 {
		t.Fatalf("versions=%v", resp.Versions)
	}
}

func TestFetchFrontendVersionsClickHouse_EnablementHintWhenProbeEmpty(t *testing.T) {
	ch := chHandlerClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "service.version") {
			t.Fatalf("expected service.version probe, body=%s", body)
		}
		if strings.Contains(string(body), "rum.version") {
			t.Fatal("must not query rum.version")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":{"A":{"frames":[{
			"schema":{"fields":[{"name":"c","type":"number"}]},
			"data":{"values":[[0]]}
		}]}}}`))
	})
	app := &App{
		settings: queries.PluginSettings{
			ClickHouseDataSource: queries.DataSourceRef{UID: "clickhouse"},
			ServiceAliases:       map[string]string{"nova-frontend": "nova-frontend"},
		},
		chClient: ch,
	}
	from, to := time.Unix(1, 0), time.Unix(2, 0)
	resp := app.fetchFrontendVersionsClickHouse(t.Context(), ch, "nova-frontend", "prod", from, to)
	if !strings.Contains(resp.Hint, "ticket 21") || !strings.Contains(resp.Hint, "service.version") {
		t.Fatalf("hint=%q", resp.Hint)
	}
	if len(resp.Versions) != 0 {
		t.Fatalf("no fake zeros: %+v", resp.Versions)
	}
	if strings.Contains(resp.Hint, "rum.version") && !strings.Contains(resp.Hint, "Do not use rum.version") {
		t.Fatalf("hint should warn against rum.version: %q", resp.Hint)
	}
}

func TestFetchFrontendVersionsClickHouse_ParsesAdoption(t *testing.T) {
	n := 0
	ch := chHandlerClient(t, func(w http.ResponseWriter, r *http.Request) {
		n++
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(body), "count() AS c") {
			_, _ = w.Write([]byte(`{"results":{"A":{"frames":[{
				"schema":{"fields":[{"name":"c","type":"number"}]},
				"data":{"values":[[2]]}
			}]}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"results":{"A":{"frames":[{
			"schema":{"fields":[
				{"name":"version","type":"string"},
				{"name":"sessions","type":"number"},
				{"name":"err_sessions","type":"number"},
				{"name":"exceptions","type":"number"}
			]},
			"data":{"values":[["1.0.0","1.0.1"],[80,20],[4,0],[10,0]]}
		}]}}}`))
	})
	app := &App{
		settings: queries.PluginSettings{
			ServiceAliases: map[string]string{"nova-frontend": "nova-frontend"},
		},
		chClient: ch,
	}
	from, to := time.Unix(1, 0), time.Unix(3600, 0)
	resp := app.fetchFrontendVersionsClickHouse(t.Context(), ch, "nova-frontend", "", from, to)
	if resp.Hint != "" || resp.Unavailable {
		t.Fatalf("resp=%+v", resp)
	}
	if len(resp.Versions) != 2 {
		t.Fatalf("versions=%+v n=%d", resp.Versions, n)
	}
	if resp.Versions[0].Version != "1.0.0" || resp.Versions[0].Adoption != 0.8 {
		t.Fatalf("v0=%+v", resp.Versions[0])
	}
	if resp.Versions[0].ErrorFreeRate != 0.95 {
		t.Fatalf("errorFreeRate=%v", resp.Versions[0].ErrorFreeRate)
	}
}
