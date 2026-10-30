package plugin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/chquery"
	"github.com/nais/grafana-otel-plugin/pkg/plugin/fingerprint"
	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

func TestFetchIssuesClickHouse_AliasMissHint(t *testing.T) {
	app := &App{
		settings: queries.PluginSettings{
			ClickHouseDataSource: queries.DataSourceRef{UID: "clickhouse"},
		},
		chClient: &queries.ClickHouseClient{},
	}
	req := httptest.NewRequest(http.MethodGet, "/services/acme/storefront/issues?from=1&to=2", nil)
	req.SetPathValue("namespace", "acme")
	req.SetPathValue("service", "storefront")
	w := httptest.NewRecorder()
	app.handleIssues(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp IssuesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Hint == "" || !strings.Contains(resp.Hint, "serviceAliases") {
		t.Fatalf("hint=%q", resp.Hint)
	}
	if resp.Sources.Browser {
		t.Fatal("browser source must be false on alias miss")
	}
}

func TestParseTraceExceptionGroups_FingerprintStable(t *testing.T) {
	rows := chquery.ParseTraceExceptionGroupsMaps([]map[string]any{
		{"exception_type": "Error", "message": "Element not found", "occurrences": int64(6), "sessions": int64(4)},
	})
	if len(rows) != 1 || rows[0].Sessions != 4 {
		t.Fatalf("%+v", rows)
	}
	fp := fingerprint.Compute(fingerprint.Event{Type: rows[0].ExceptionType, Value: rows[0].Message})
	if !strings.HasPrefix(fp.Value, "v1:") || fp.Title == "" {
		t.Fatalf("fingerprint=%+v", fp)
	}
}
