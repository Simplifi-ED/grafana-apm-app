package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/useragent"
	"github.com/grafana/grafana-plugin-sdk-go/config"
	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// loopbackGrafanaBase is Grafana's default listen address. A wrong guess is
// harmless: the health probe (with version check) rejects it.
const loopbackGrafanaBase = "http://127.0.0.1:3000"

const grafanaProbeTimeout = 2 * time.Second

var errProbeRedirect = errors.New("health probe redirected")

var grafanaProbeClient = &http.Client{
	Timeout: grafanaProbeTimeout,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return errProbeRedirect
	},
}

type grafanaURLCandidate struct {
	source string
	base   string
	// fallback marks candidates eligible when no probe succeeds. Guessed
	// loopback addresses are never used unverified.
	fallback bool
}

// resolveGrafanaURL picks the base URL for backend → Grafana API calls:
// override → loopback (with and without the root_url sub-path) → SDK AppURL,
// taking the first that passes a Grafana /api/health probe.
func resolveGrafanaURL(ctx context.Context, override string, logger log.Logger) string {
	return resolveGrafanaURLWith(ctx, override, loopbackGrafanaBase, logger)
}

func resolveGrafanaURLWith(ctx context.Context, override, loopback string, logger log.Logger) string {
	candidates := grafanaURLCandidates(ctx, override, loopback, logger)

	wantVersion := ""
	if ua := useragent.FromContext(ctx); !ua.IsUnknown() {
		wantVersion = ua.GrafanaVersion()
	}

	for _, c := range candidates {
		if err := probeGrafana(ctx, c.base, wantVersion); err != nil {
			logger.Debug("Grafana URL candidate rejected", "source", c.source, "grafanaURL", c.base, "reason", err.Error())
			continue
		}
		logger.Info("Resolved Grafana URL", "grafanaURL", c.base, "source", c.source, "probed", true)
		return c.base
	}

	for _, c := range candidates {
		if c.fallback {
			logger.Error("No Grafana URL candidate passed the health probe; using unverified fallback", "grafanaURL", c.base, "source", c.source)
			return c.base
		}
	}
	logger.Error("Grafana URL unavailable: no candidate passed the health probe and no AppURL/override is set")
	return ""
}

func grafanaURLCandidates(ctx context.Context, override, loopback string, logger log.Logger) []grafanaURLCandidate {
	appURL, err := config.GrafanaConfigFromContext(ctx).AppURL()
	if err != nil {
		logger.Debug("SDK AppURL unavailable", "error", err)
		appURL = ""
	}

	var raw []grafanaURLCandidate
	if override != "" {
		raw = append(raw, grafanaURLCandidate{"plugin setting internalGrafanaUrl", override, true})
	}
	if loopback != "" {
		if sub := subPath(appURL); sub != "" {
			raw = append(raw, grafanaURLCandidate{"loopback + root_url sub-path", strings.TrimRight(loopback, "/") + sub, false})
		}
		raw = append(raw, grafanaURLCandidate{"loopback", loopback, false})
	}
	if appURL != "" {
		raw = append(raw, grafanaURLCandidate{"SDK AppURL", appURL, true})
	}

	seen := make(map[string]int, len(raw))
	out := make([]grafanaURLCandidate, 0, len(raw))
	for _, c := range raw {
		base := queries.SafeBaseURL(c.base)
		if base == "" {
			logger.Warn("Ignoring invalid Grafana URL candidate", "source", c.source, "value", c.base)
			continue
		}
		if i, dup := seen[base]; dup {
			out[i].fallback = out[i].fallback || c.fallback
			continue
		}
		seen[base] = len(out)
		c.base = base
		out = append(out, c)
	}
	return out
}

func subPath(appURL string) string {
	if appURL == "" {
		return ""
	}
	u, err := url.Parse(appURL)
	if err != nil {
		return ""
	}
	return strings.TrimRight(u.EscapedPath(), "/")
}

type grafanaHealth struct {
	Version  string `json:"version"`
	Database string `json:"database"`
}

// probeGrafana checks base is a Grafana instance (matching wantVersion when
// known). No credentials are sent.
func probeGrafana(ctx context.Context, base, wantVersion string) error {
	endpoint, err := url.JoinPath(base, "api", "health")
	if err != nil {
		return err
	}
	pctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), grafanaProbeTimeout)
	defer cancel()
	req, err := queries.NewSafeRequestFromString(pctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := queries.DoValidated(grafanaProbeClient, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	var h grafanaHealth
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&h); err != nil {
		return fmt.Errorf("not a Grafana health response: %w", err)
	}
	if h.Version == "" || h.Database == "" {
		return errors.New("not a Grafana health response")
	}
	if wantVersion != "" && !grafanaVersionsMatch(h.Version, wantVersion) {
		return fmt.Errorf("version %q does not match Grafana %q", h.Version, wantVersion)
	}
	return nil
}

// grafanaVersionsMatch tolerates build/prerelease suffixes ("13.3.0+security-01",
// "13.3.0-123123") but not different releases.
func grafanaVersionsMatch(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return coreVersion(a) == coreVersion(b)
}

func coreVersion(v string) string {
	if i := strings.IndexAny(v, "+-"); i >= 0 {
		return v[:i]
	}
	return v
}
