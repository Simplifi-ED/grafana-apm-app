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

func TestHandleFeedback_ClickStackAliasMissHint(t *testing.T) {
	app := &App{
		settings: queries.PluginSettings{
			ClickHouseDataSource: queries.DataSourceRef{UID: "clickhouse"},
		},
		chClient: &queries.ClickHouseClient{},
	}
	req := httptest.NewRequest(http.MethodGet, "/services/acme/storefront/feedback?from=1&to=2", nil)
	req.SetPathValue("namespace", "acme")
	req.SetPathValue("service", "storefront")
	w := httptest.NewRecorder()
	app.handleFeedback(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp FeedbackResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Hint == "" || !strings.Contains(resp.Hint, "serviceAliases") {
		t.Fatalf("hint=%q", resp.Hint)
	}
	if len(resp.Feedback) != 0 {
		t.Fatalf("feedback=%v", resp.Feedback)
	}
}

func TestFetchFeedbackClickHouse_EnablementHintWhenProbeEmpty(t *testing.T) {
	ch := chHandlerClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "faro.feedback") {
			t.Fatalf("expected faro.feedback probe, body=%s", body)
		}
		if strings.Contains(string(body), "/api/feedback") {
			t.Fatal("must not query product HTTP feedback")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":{"A":{"frames":[{
			"schema":{"fields":[{"name":"c","type":"number"}]},
			"data":{"values":[[0]]}
		}]}}}`))
	})
	app := &App{
		settings: queries.PluginSettings{
			ServiceAliases: map[string]string{"storefront": "storefront"},
		},
		chClient: ch,
	}
	from, to := time.Unix(1, 0), time.Unix(2, 0)
	resp := app.fetchFeedbackClickHouse(t.Context(), ch, "storefront", "", from, to, "", "")
	if !strings.Contains(resp.Hint, "ticket 21") || !strings.Contains(resp.Hint, "faro.feedback") {
		t.Fatalf("hint=%q", resp.Hint)
	}
	if len(resp.Feedback) != 0 {
		t.Fatalf("no fake zeros: %+v", resp.Feedback)
	}
}

func TestFetchFeedbackClickHouse_ParsesRows(t *testing.T) {
	ch := chHandlerClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(body), "count() AS c") {
			_, _ = w.Write([]byte(`{"results":{"A":{"frames":[{
				"schema":{"fields":[{"name":"c","type":"number"}]},
				"data":{"values":[[1]]}
			}]}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"results":{"A":{"frames":[{
			"schema":{"fields":[
				{"name":"time_ms","type":"number"},
				{"name":"message","type":"string"},
				{"name":"category","type":"string"},
				{"name":"email","type":"string"},
				{"name":"session_id","type":"string"},
				{"name":"fingerprint","type":"string"},
				{"name":"page_url","type":"string"},
				{"name":"app_version","type":"string"}
			]},
			"data":{"values":[
				[9000],
				["broken button"],
				["bug"],
				["a@b.c"],
				["sess-9"],
				["v1:abc"],
				["https://x/y"],
				["1.72.1"]
			]}
		}]}}}`))
	})
	app := &App{
		settings: queries.PluginSettings{
			ServiceAliases: map[string]string{"storefront": "storefront"},
		},
		chClient: ch,
	}
	from, to := time.Unix(1, 0), time.Unix(3600, 0)
	resp := app.fetchFeedbackClickHouse(t.Context(), ch, "storefront", "prod", from, to, "", "")
	if resp.Hint != "" || resp.Unavailable || len(resp.Feedback) != 1 {
		t.Fatalf("resp=%+v", resp)
	}
	f := resp.Feedback[0]
	if f.Message != "broken button" || f.Category != "bug" || f.AppVersion != "1.72.1" || f.SessionID != "sess-9" {
		t.Fatalf("%+v", f)
	}
}
