package plugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/config"
)

// OpsWatchlistEntry is a single watched service in the ops watchlist. An entry
// is scoped to one environment; an empty Environment means the service is
// watched across all environments (legacy entries saved before env scoping).
type OpsWatchlistEntry struct {
	Namespace   string `json:"namespace"`
	Service     string `json:"service"`
	Environment string `json:"environment,omitempty"`
}

// watchlistMu serializes write operations to prevent concurrent read-modify-write races.
var watchlistMu sync.Mutex

// handleOpsWatchlist handles GET and POST requests for the shared ops watchlist.
// GET (any authenticated user) returns the current watchlist. POST replaces it
// and, because it writes plugin settings with the service-account token
// (Admin-scoped), requires Editor/Admin — otherwise a Viewer could drive an
// Admin action through the SA token.
func (a *App) handleOpsWatchlist(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		a.getOpsWatchlist(w, req)
	case http.MethodPost:
		if !a.requireEditor(w, req) {
			return
		}
		a.setOpsWatchlist(w, req)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// getOpsWatchlist reads jsonData from the plugin context Grafana attaches to
// every resource call, so reads never depend on authenticating a loopback
// call to the settings API (cookies are not forwarded reliably, e.g. with
// cookie_samesite=strict).
func (a *App) getOpsWatchlist(w http.ResponseWriter, req *http.Request) {
	jsonData, err := a.currentJSONData(req)
	if err != nil {
		log.DefaultLogger.Error("Failed to read plugin settings", "error", err)
		http.Error(w, "failed to fetch plugin settings: "+publicSettingsError(err), settingsErrorStatus(err))
		return
	}

	watchlist := jsonData["opsWatchlist"]
	if watchlist == nil {
		watchlist = []any{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(watchlist)
}

func (a *App) setOpsWatchlist(w http.ResponseWriter, req *http.Request) {
	var entries []OpsWatchlistEntry
	if err := json.NewDecoder(io.LimitReader(req.Body, 1<<20)).Decode(&entries); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Filter out incomplete entries
	valid := make([]OpsWatchlistEntry, 0, len(entries))
	for _, e := range entries {
		if e.Namespace != "" && e.Service != "" {
			valid = append(valid, e)
		}
	}

	// Serialize writes to prevent concurrent read-modify-write races
	watchlistMu.Lock()
	defer watchlistMu.Unlock()

	err := a.withSettingsAPI(req, func(cred settingsCredential) error {
		settings, err := a.fetchPluginSettings(req, cred)
		if err != nil {
			return err
		}
		if len(valid) > 0 {
			settings.JSONData["opsWatchlist"] = valid
		} else {
			delete(settings.JSONData, "opsWatchlist")
		}
		return a.savePluginSettings(req, cred, settings)
	})
	if err != nil {
		log.DefaultLogger.Error("Failed to save plugin settings", "error", err)
		http.Error(w, "failed to save plugin settings: "+publicSettingsError(err), settingsErrorStatus(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(valid)
}

// pluginSettings holds the subset of plugin settings needed for read-modify-write.
type pluginSettings struct {
	Enabled  bool           `json:"enabled"`
	Pinned   bool           `json:"pinned"`
	JSONData map[string]any `json:"jsonData"`
}

// defaultPluginID is used only when the request carries no plugin context.
const defaultPluginID = "clickhouse-apm-app"

func requestPluginID(req *http.Request) string {
	if id := backend.PluginConfigFromContext(req.Context()).PluginID; id != "" {
		return id
	}
	return defaultPluginID
}

// currentJSONData returns the plugin's jsonData, preferring the per-request
// plugin context and falling back to the settings API when it is absent.
func (a *App) currentJSONData(req *http.Request) (map[string]any, error) {
	if s := backend.PluginConfigFromContext(req.Context()).AppInstanceSettings; s != nil {
		jsonData := map[string]any{}
		if len(bytes.TrimSpace(s.JSONData)) > 0 {
			if err := json.Unmarshal(s.JSONData, &jsonData); err != nil {
				return nil, fmt.Errorf("failed to decode plugin jsonData: %w", err)
			}
		}
		return jsonData, nil
	}

	var jsonData map[string]any
	err := a.withSettingsAPI(req, func(cred settingsCredential) error {
		settings, err := a.fetchPluginSettings(req, cred)
		if err != nil {
			return err
		}
		jsonData = settings.JSONData
		return nil
	})
	return jsonData, err
}

// settingsCredential is one way of authenticating a settings API call.
type settingsCredential struct {
	name  string
	apply func(apiReq *http.Request)
}

// settingsCredentials lists the ways to authenticate against the settings API,
// in order: managed service account, configured service account token, then
// the caller's forwarded session. Which one works depends on the deployment
// (managed SA permissions, OAuth proxy, cookie SameSite policy), so callers
// try each in turn via withSettingsAPI.
func (a *App) settingsCredentials(req *http.Request) []settingsCredential {
	var creds []settingsCredential
	seen := map[string]bool{}
	addToken := func(name, token string) {
		if token == "" || seen[token] {
			return
		}
		seen[token] = true
		creds = append(creds, settingsCredential{name: name, apply: func(apiReq *http.Request) {
			apiReq.Header.Set("Authorization", "Bearer "+token)
			for _, v := range req.Header.Values("X-Grafana-Org-Id") {
				apiReq.Header.Add("X-Grafana-Org-Id", v)
			}
		}})
	}

	if token, err := config.GrafanaConfigFromContext(req.Context()).PluginAppClientSecret(); err == nil {
		addToken("managed service account", token)
	}
	addToken("configured service account token", a.serviceToken)

	creds = append(creds, settingsCredential{name: "forwarded user session", apply: func(apiReq *http.Request) {
		for _, key := range []string{"Cookie", "Authorization", "X-Grafana-Org-Id", "X-Grafana-Id"} {
			for _, v := range req.Header.Values(key) {
				apiReq.Header.Add(key, v)
			}
		}
	}})
	return creds
}

// withSettingsAPI runs fn with each credential until one is not rejected by
// Grafana (401/403/404). Any other error stops immediately.
func (a *App) withSettingsAPI(req *http.Request, fn func(settingsCredential) error) error {
	var lastErr error
	for _, cred := range a.settingsCredentials(req) {
		err := fn(cred)
		var apiErr *settingsAPIError
		if err == nil || !errors.As(err, &apiErr) || !apiErr.authRejected() {
			return err
		}
		log.DefaultLogger.Warn("Plugin settings API rejected credential, trying next",
			"credential", cred.name, "status", apiErr.status)
		lastErr = err
	}
	return lastErr
}

// settingsAPIError is a non-2xx response from Grafana's plugin settings API.
type settingsAPIError struct {
	status int
	body   string
}

func (e *settingsAPIError) Error() string {
	return fmt.Sprintf("grafana API returned %d: %s", e.status, e.body)
}

func (e *settingsAPIError) authRejected() bool {
	switch e.status {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		return true
	}
	return false
}

// publicSettingsError is the client-facing detail: the upstream status only,
// never the upstream body.
func publicSettingsError(err error) string {
	var apiErr *settingsAPIError
	if errors.As(err, &apiErr) {
		return fmt.Sprintf("grafana settings API returned %d (check the plugin service account has plugins:write on this app)", apiErr.status)
	}
	return "internal error"
}

func settingsErrorStatus(err error) int {
	var apiErr *settingsAPIError
	if errors.As(err, &apiErr) {
		return http.StatusBadGateway
	}
	return http.StatusInternalServerError
}

func (a *App) settingsURL(req *http.Request) string {
	return fmt.Sprintf("%s/api/plugins/%s/settings", a.grafanaURL, requestPluginID(req))
}

// fetchPluginSettings retrieves the plugin's current settings from the Grafana API.
func (a *App) fetchPluginSettings(req *http.Request, cred settingsCredential) (*pluginSettings, error) {
	apiReq, err := http.NewRequestWithContext(req.Context(), http.MethodGet, a.settingsURL(req), nil)
	if err != nil {
		return nil, err
	}
	cred.apply(apiReq)

	resp, err := a.healthClient.Do(apiReq)
	if err != nil {
		return nil, fmt.Errorf("grafana API request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, &settingsAPIError{status: resp.StatusCode, body: string(body)}
	}

	var settings pluginSettings
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return nil, fmt.Errorf("failed to decode plugin settings: %w", err)
	}
	if settings.JSONData == nil {
		settings.JSONData = make(map[string]any)
	}
	return &settings, nil
}

// savePluginSettings updates the plugin's settings via the Grafana API.
// Grafana merges secureJsonData on update, so omitting it keeps existing secrets.
func (a *App) savePluginSettings(req *http.Request, cred settingsCredential, settings *pluginSettings) error {
	body, err := json.Marshal(map[string]any{
		"enabled":  settings.Enabled,
		"pinned":   settings.Pinned,
		"jsonData": settings.JSONData,
	})
	if err != nil {
		return err
	}

	apiReq, err := http.NewRequestWithContext(req.Context(), http.MethodPost, a.settingsURL(req), bytes.NewReader(body))
	if err != nil {
		return err
	}
	apiReq.Header.Set("Content-Type", "application/json")
	cred.apply(apiReq)

	resp, err := a.healthClient.Do(apiReq)
	if err != nil {
		return fmt.Errorf("grafana API request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return &settingsAPIError{status: resp.StatusCode, body: string(respBody)}
	}
	return nil
}
