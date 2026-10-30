package plugin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/config"
)

// settingsAPI is a fake Grafana /api/plugins/{id}/settings endpoint that only
// accepts the bearer tokens in allowed and records what it received.
type settingsAPI struct {
	t        *testing.T
	pluginID string
	allowed  map[string]bool
	settings map[string]any

	mu       sync.Mutex
	authSeen []string
	saved    map[string]any
}

func (s *settingsAPI) handler(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	auth := r.Header.Get("Authorization")
	if auth == "" && r.Header.Get("Cookie") != "" {
		auth = "cookie:" + r.Header.Get("Cookie")
	}
	s.authSeen = append(s.authSeen, r.Method+" "+auth)
	if r.URL.Path != "/api/plugins/"+s.pluginID+"/settings" {
		http.Error(w, `{"message":"Plugin not found"}`, http.StatusNotFound)
		return
	}
	if !s.allowed[auth] {
		http.Error(w, `{"message":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case http.MethodGet:
		_ = json.NewEncoder(w).Encode(s.settings)
	case http.MethodPost:
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &s.saved); err != nil {
			s.t.Errorf("invalid save payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}
}

func newWatchlistApp(srvURL, manualToken string) *App {
	return &App{
		grafanaURL:   srvURL,
		serviceToken: manualToken,
		healthClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// watchlistCtx mirrors what Grafana's resource adapter puts on every request.
func watchlistCtx(pluginID string, jsonData string, managedToken string) context.Context {
	ctx := backend.WithPluginContext(context.Background(), backend.PluginContext{
		PluginID:            pluginID,
		AppInstanceSettings: &backend.AppInstanceSettings{JSONData: json.RawMessage(jsonData)},
	})
	ctx = backend.WithUser(ctx, &backend.User{Role: "Admin"})
	if managedToken != "" {
		ctx = config.WithGrafanaConfig(ctx, config.NewGrafanaCfg(map[string]string{config.AppClientSecret: managedToken}))
	}
	return ctx
}

// Regression: on Grafana 13 with cookie_samesite=strict, the loopback
// settings call could not authenticate, so the Ops Status board failed with
// "failed to fetch plugin settings (500)". Reads must not depend on it.
func TestGetOpsWatchlist_ReadsPluginContextWithoutGrafanaAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("GET /ops-watchlist must not call the Grafana API, got %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	app := newWatchlistApp(srv.URL, "")
	req := httptest.NewRequest(http.MethodGet, "/ops-watchlist", nil).WithContext(
		watchlistCtx("clickhouse-apm-app", `{"opsWatchlist":[{"namespace":"ns","service":"api"}]}`, ""))
	rec := httptest.NewRecorder()
	app.handleOpsWatchlist(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got []OpsWatchlistEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0].Service != "api" || got[0].Namespace != "ns" {
		t.Errorf("watchlist = %+v", got)
	}
}

func TestGetOpsWatchlist_EmptyJSONDataReturnsEmptyList(t *testing.T) {
	app := newWatchlistApp("http://127.0.0.1:1", "")
	req := httptest.NewRequest(http.MethodGet, "/ops-watchlist", nil).WithContext(
		watchlistCtx("clickhouse-apm-app", ``, ""))
	rec := httptest.NewRecorder()
	app.handleOpsWatchlist(rec, req)

	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("status = %d, body = %q, want 200 []", rec.Code, rec.Body.String())
	}
}

func TestSetOpsWatchlist_FallsBackAcrossCredentials(t *testing.T) {
	api := &settingsAPI{
		t:        t,
		pluginID: "clickhouse-apm-app",
		// Managed SA lacks plugin permissions (older plugin.json iam); the
		// manually configured token works.
		allowed: map[string]bool{"Bearer manual-token": true},
		settings: map[string]any{
			"enabled":  true,
			"pinned":   true,
			"jsonData": map[string]any{"clickHouseDataSource": map[string]any{"uid": "clickhouse"}},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(api.handler))
	defer srv.Close()

	app := newWatchlistApp(srv.URL, "manual-token")
	body := `[{"namespace":"ns","service":"api","environment":"prod"},{"namespace":"","service":"dropped"}]`
	req := httptest.NewRequest(http.MethodPost, "/ops-watchlist", strings.NewReader(body)).WithContext(
		watchlistCtx("clickhouse-apm-app", `{}`, "managed-token"))
	rec := httptest.NewRecorder()
	app.handleOpsWatchlist(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s, auth seen = %v", rec.Code, rec.Body.String(), api.authSeen)
	}
	if len(api.authSeen) == 0 || api.authSeen[0] != "GET Bearer managed-token" {
		t.Errorf("managed service account must be tried first, auth seen = %v", api.authSeen)
	}
	if api.saved == nil {
		t.Fatalf("settings were not saved, auth seen = %v", api.authSeen)
	}
	if api.saved["enabled"] != true || api.saved["pinned"] != true {
		t.Errorf("enabled/pinned not preserved: %+v", api.saved)
	}
	jd, _ := api.saved["jsonData"].(map[string]any)
	if _, ok := jd["clickHouseDataSource"]; !ok {
		t.Errorf("existing jsonData keys dropped: %+v", jd)
	}
	wl, _ := jd["opsWatchlist"].([]any)
	if len(wl) != 1 {
		t.Errorf("opsWatchlist = %+v, want the single valid entry", jd["opsWatchlist"])
	}
}

func TestSetOpsWatchlist_UsesPluginIDFromContext(t *testing.T) {
	api := &settingsAPI{
		t:        t,
		pluginID: "renamed-apm-app",
		allowed:  map[string]bool{"Bearer managed-token": true},
		settings: map[string]any{"enabled": true, "jsonData": map[string]any{}},
	}
	srv := httptest.NewServer(http.HandlerFunc(api.handler))
	defer srv.Close()

	app := newWatchlistApp(srv.URL, "")
	req := httptest.NewRequest(http.MethodPost, "/ops-watchlist", strings.NewReader(`[]`)).WithContext(
		watchlistCtx("renamed-apm-app", `{}`, "managed-token"))
	rec := httptest.NewRecorder()
	app.handleOpsWatchlist(rec, req)

	if rec.Code != http.StatusOK || api.saved == nil {
		t.Fatalf("status = %d, body = %s, auth seen = %v", rec.Code, rec.Body.String(), api.authSeen)
	}
}

func TestSetOpsWatchlist_ReportsUpstreamStatusWhenAllCredentialsRejected(t *testing.T) {
	api := &settingsAPI{t: t, pluginID: "clickhouse-apm-app", allowed: map[string]bool{}}
	srv := httptest.NewServer(http.HandlerFunc(api.handler))
	defer srv.Close()

	app := newWatchlistApp(srv.URL, "manual-token")
	req := httptest.NewRequest(http.MethodPost, "/ops-watchlist", strings.NewReader(`[]`)).WithContext(
		watchlistCtx("clickhouse-apm-app", `{}`, "managed-token"))
	req.Header.Set("Cookie", "grafana_session=abc")
	rec := httptest.NewRecorder()
	app.handleOpsWatchlist(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "401") {
		t.Errorf("error should name the upstream status, got %q", rec.Body.String())
	}
	if len(api.authSeen) != 3 {
		t.Errorf("expected managed SA, manual token and session to be tried once each, got %v", api.authSeen)
	}
}
