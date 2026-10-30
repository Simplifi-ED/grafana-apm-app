package plugin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

func mockReplayCHServer(t *testing.T, handler func(rawSQL string) (columns []string, values [][]any, errMsg string)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Queries []struct {
				RefID  string `json:"refId"`
				RawSQL string `json:"rawSql"`
			} `json:"queries"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Queries) == 0 {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		q := req.Queries[0]
		cols, vals, errMsg := handler(q.RawSQL)
		type frame struct {
			Schema struct {
				Fields []map[string]any `json:"fields"`
			} `json:"schema"`
			Data struct {
				Values []any `json:"values"`
			} `json:"data"`
		}
		res := map[string]any{"error": errMsg, "frames": []frame{}}
		if errMsg == "" {
			var f frame
			for _, c := range cols {
				f.Schema.Fields = append(f.Schema.Fields, map[string]any{"name": c})
			}
			// Transpose rows → column vectors.
			colVals := make([]any, len(cols))
			for c := range cols {
				col := make([]any, len(vals))
				for r := range vals {
					if c < len(vals[r]) {
						col[r] = vals[r][c]
					}
				}
				colVals[c] = col
			}
			f.Data.Values = colVals
			res["frames"] = []frame{f}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"results": map[string]any{q.RefID: res}})
	}))
}

type replayHTTPApp struct {
	*App
	mux *http.ServeMux
}

func newReplayHTTPApp(t *testing.T, srv *httptest.Server) *replayHTTPApp {
	t.Helper()
	app := &App{
		grafanaURL: srv.URL,
		chClient:   queries.NewClickHouseClient(srv.URL, "tok", "clickhouse"),
		settings:   queries.PluginSettings{ClickHouseDataSource: queries.DataSourceRef{UID: "clickhouse"}},
	}
	mux := http.NewServeMux()
	app.registerRoutes(mux)
	return &replayHTTPApp{App: app, mux: mux}
}

func TestHandleSessionReplayProbe_UnavailableWithoutCH(t *testing.T) {
	app := &App{}
	mux := http.NewServeMux()
	app.registerRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/services/_/storefront/frontend/sessions/abc123/replay/probe?from=1&to=2", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp SessionReplayProbeResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Unavailable {
		t.Fatalf("expected unavailable, got %+v", resp)
	}
}

func TestHandleSessionReplayProbe_HasChunks(t *testing.T) {
	srv := mockReplayCHServer(t, func(rawSQL string) ([]string, [][]any, string) {
		if !strings.Contains(rawSQL, "storefront") || !strings.Contains(rawSQL, "sess-1") {
			t.Fatalf("unexpected SQL: %s", rawSQL)
		}
		return []string{"row_count", "event_count"}, [][]any{{float64(12), float64(10)}}, ""
	})
	defer srv.Close()
	app := newReplayHTTPApp(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/services/_/storefront/frontend/sessions/sess-1/replay/probe?from=1000&to=2000", nil)
	rr := httptest.NewRecorder()
	app.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp SessionReplayProbeResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.HasChunks || resp.Mode != "recording" || resp.ChunkCount != 12 {
		t.Fatalf("probe=%+v", resp)
	}
}

func TestHandleSessionReplay_ReassemblesChunks(t *testing.T) {
	full := `{"type":2,"data":{"node":{"type":0,"childNodes":[]}},"timestamp":2000}`
	mid := len(full) / 2
	srv := mockReplayCHServer(t, func(rawSQL string) ([]string, [][]any, string) {
		if !strings.Contains(rawSQL, "Body AS body") {
			t.Fatalf("expected event fetch SQL, got %s", rawSQL)
		}
		return []string{"ts_ms", "body", "event_id", "chunk", "total_chunks", "offset"}, [][]any{
			{float64(1990), `{"type":4,"data":{"href":"https://example.test/"},"timestamp":1990}`, "1", "1", "1", "1"},
			{float64(2000), full[:mid], "2", "1", "2", "2"},
			{float64(2000), full[mid:], "2", "2", "2", "3"},
		}, ""
	})
	defer srv.Close()
	app := newReplayHTTPApp(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/services/_/storefront/frontend/sessions/sess-1/replay?from=1000&to=3000", nil)
	rr := httptest.NewRecorder()
	app.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp SessionReplayResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.EventCount != 2 || resp.Mode != "recording" {
		t.Fatalf("resp=%+v", resp)
	}
	if len(resp.Events) != 2 {
		t.Fatalf("events=%d", len(resp.Events))
	}
}

func TestHandleSessionReplay_RejectsWideRange(t *testing.T) {
	app := &App{chClient: queries.NewClickHouseClient("http://example.invalid", "", "ch")}
	mux := http.NewServeMux()
	app.registerRoutes(mux)
	from := time.Now().Add(-48 * time.Hour).Unix()
	to := time.Now().Unix()
	req := httptest.NewRequest(http.MethodGet, "/services/_/svc/frontend/sessions/s1/replay?from="+itoa64(from)+"&to="+itoa64(to), nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSessionRecordings_ListsSessions(t *testing.T) {
	srv := mockReplayCHServer(t, func(rawSQL string) ([]string, [][]any, string) {
		if !strings.Contains(rawSQL, "GROUP BY session_id") {
			t.Fatalf("expected list SQL, got %s", rawSQL)
		}
		return []string{"session_id", "first_ms", "last_ms", "row_count", "event_count"}, [][]any{
			{"sess-a", float64(1000), float64(2000), float64(5), float64(4)},
			{"sess-b", float64(1500), float64(2500), float64(3), float64(3)},
		}, ""
	})
	defer srv.Close()
	app := newReplayHTTPApp(t, srv)
	req := httptest.NewRequest(http.MethodGet, "/services/_/storefront/frontend/sessions/recordings?from=1&to=2", nil)
	rr := httptest.NewRecorder()
	app.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp SessionRecordingsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Sessions) != 2 || resp.Sessions[0].SessionID != "sess-a" {
		t.Fatalf("sessions=%+v", resp.Sessions)
	}
}

func TestBuildReplayWhere_Escapes(t *testing.T) {
	got := buildReplayWhere("svc", "prod-k3s", "abc'def")
	if !strings.Contains(got, "ServiceName = 'svc'") {
		t.Fatalf("service: %s", got)
	}
	if !strings.Contains(got, "rum.sessionId'] = 'abc''def'") {
		t.Fatalf("session escape: %s", got)
	}
	if !strings.Contains(got, "cluster'] = 'prod-k3s'") {
		t.Fatalf("cluster: %s", got)
	}
}

func TestValidateReplayTimeRange(t *testing.T) {
	now := time.Now()
	if err := validateReplayTimeRange(now, now.Add(-time.Second)); err == nil {
		t.Fatal("expected inverted range error")
	}
	if err := validateReplayTimeRange(now.Add(-25*time.Hour), now); err == nil {
		t.Fatal("expected max range error")
	}
	if err := validateReplayTimeRange(now.Add(-time.Hour), now); err != nil {
		t.Fatal(err)
	}
}

func itoa64(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}
