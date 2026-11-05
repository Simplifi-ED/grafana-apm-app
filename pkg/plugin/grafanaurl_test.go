package plugin

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/useragent"
	"github.com/grafana/grafana-plugin-sdk-go/config"
)

// grafanaHealthServer serves Grafana-shaped /api/health under pathPrefix.
func grafanaHealthServer(t *testing.T, version, pathPrefix string, hits *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits != nil {
			hits.Add(1)
		}
		if r.URL.Path != pathPrefix+"/api/health" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "" {
			t.Errorf("health probe must not send credentials")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"commit":"abc","database":"ok","version":"` + version + `"}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// deadBase returns a loopback URL with nothing listening.
func deadBase(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	return "http://" + addr
}

func ctxWith(appURL, grafanaVersion string) context.Context {
	ctx := config.WithGrafanaConfig(context.Background(), config.NewGrafanaCfg(map[string]string{
		config.AppURL: appURL,
	}))
	if grafanaVersion != "" {
		ua, err := useragent.New(grafanaVersion, "linux", "amd64")
		if err != nil {
			panic(err)
		}
		ctx = useragent.WithUserAgent(ctx, ua)
	}
	return ctx
}

func TestResolveGrafanaURL_Probe(t *testing.T) {
	logger := log.DefaultLogger

	t.Run("healthy override wins even when loopback is healthy", func(t *testing.T) {
		override := grafanaHealthServer(t, "13.3.0", "", nil)
		loop := grafanaHealthServer(t, "13.3.0", "", nil)
		got := resolveGrafanaURLWith(ctxWith("https://grafana.invalid", "13.3.0"), override.URL, loop.URL, logger)
		if got != override.URL {
			t.Fatalf("got %q, want override %q", got, override.URL)
		}
	})

	t.Run("loopback chosen when AppURL host does not resolve", func(t *testing.T) {
		loop := grafanaHealthServer(t, "13.3.0", "", nil)
		got := resolveGrafanaURLWith(ctxWith("https://monitoring.tailnet.invalid/", "13.3.0"), "", loop.URL, logger)
		if got != loop.URL {
			t.Fatalf("got %q, want loopback %q", got, loop.URL)
		}
	})

	t.Run("AppURL chosen when loopback is down", func(t *testing.T) {
		app := grafanaHealthServer(t, "13.3.0", "", nil)
		got := resolveGrafanaURLWith(ctxWith(app.URL, "13.3.0"), "", deadBase(t), logger)
		if got != app.URL {
			t.Fatalf("got %q, want AppURL %q", got, app.URL)
		}
	})

	t.Run("version mismatch rejected", func(t *testing.T) {
		impostor := grafanaHealthServer(t, "9.9.9", "", nil)
		app := grafanaHealthServer(t, "13.3.0", "", nil)
		got := resolveGrafanaURLWith(ctxWith(app.URL, "13.3.0"), "", impostor.URL, logger)
		if got != app.URL {
			t.Fatalf("got %q, want AppURL %q (loopback reports wrong version)", got, app.URL)
		}
	})

	t.Run("build suffix on version tolerated", func(t *testing.T) {
		loop := grafanaHealthServer(t, "13.3.0+security-01", "", nil)
		got := resolveGrafanaURLWith(ctxWith("https://grafana.invalid", "13.3.0"), "", loop.URL, logger)
		if got != loop.URL {
			t.Fatalf("got %q, want loopback %q", got, loop.URL)
		}
	})

	t.Run("unknown SDK version skips version check", func(t *testing.T) {
		loop := grafanaHealthServer(t, "13.3.0", "", nil)
		got := resolveGrafanaURLWith(ctxWith("https://grafana.invalid", ""), "", loop.URL, logger)
		if got != loop.URL {
			t.Fatalf("got %q, want loopback %q", got, loop.URL)
		}
	})

	t.Run("200 with non-Grafana body rejected", func(t *testing.T) {
		other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}))
		t.Cleanup(other.Close)
		app := grafanaHealthServer(t, "13.3.0", "", nil)
		got := resolveGrafanaURLWith(ctxWith(app.URL, "13.3.0"), "", other.URL, logger)
		if got != app.URL {
			t.Fatalf("got %q, want AppURL %q", got, app.URL)
		}
	})

	t.Run("redirecting health endpoint rejected", func(t *testing.T) {
		redir := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://elsewhere.invalid/api/health", http.StatusFound)
		}))
		t.Cleanup(redir.Close)
		app := grafanaHealthServer(t, "13.3.0", "", nil)
		got := resolveGrafanaURLWith(ctxWith(app.URL, "13.3.0"), "", redir.URL, logger)
		if got != app.URL {
			t.Fatalf("got %q, want AppURL %q", got, app.URL)
		}
	})

	t.Run("nothing healthy falls back to AppURL, never to guessed loopback", func(t *testing.T) {
		got := resolveGrafanaURLWith(ctxWith("https://grafana.invalid/", "13.3.0"), "", deadBase(t), logger)
		if got != "https://grafana.invalid" {
			t.Fatalf("got %q, want https://grafana.invalid", got)
		}
	})

	t.Run("nothing healthy and no AppURL returns empty", func(t *testing.T) {
		got := resolveGrafanaURLWith(ctxWith("", "13.3.0"), "", deadBase(t), logger)
		if got != "" {
			t.Fatalf("got %q, want empty", got)
		}
	})

	t.Run("loopback keeps AppURL sub-path when Grafana serves it", func(t *testing.T) {
		loop := grafanaHealthServer(t, "13.3.0", "/grafana", nil)
		got := resolveGrafanaURLWith(ctxWith("https://grafana.invalid/grafana/", "13.3.0"), "", loop.URL, logger)
		if got != loop.URL+"/grafana" {
			t.Fatalf("got %q, want %q", got, loop.URL+"/grafana")
		}
	})

	t.Run("loopback drops sub-path when Grafana serves from root", func(t *testing.T) {
		loop := grafanaHealthServer(t, "13.3.0", "", nil)
		got := resolveGrafanaURLWith(ctxWith("https://grafana.invalid/grafana/", "13.3.0"), "", loop.URL, logger)
		if got != loop.URL {
			t.Fatalf("got %q, want %q", got, loop.URL)
		}
	})

	t.Run("duplicate candidates probed once", func(t *testing.T) {
		var hits atomic.Int32
		loop := grafanaHealthServer(t, "13.3.0", "", &hits)
		// AppURL == loopback; force both to fail the version check so every
		// candidate is probed, then count requests.
		got := resolveGrafanaURLWith(ctxWith(loop.URL, "1.0.0"), loop.URL, loop.URL, logger)
		if got != loop.URL {
			t.Fatalf("got %q, want fallback %q", got, loop.URL)
		}
		if n := hits.Load(); n != 1 {
			t.Fatalf("probed %d times, want 1", n)
		}
	})
}

func TestGrafanaVersionsMatch(t *testing.T) {
	cases := []struct {
		health, sdk string
		want        bool
	}{
		{"13.3.0", "13.3.0", true},
		{"13.3.0+security-01", "13.3.0", true},
		{"13.3.0", "13.3.0-123123", true},
		{"13.3.1", "13.3.0", false},
		{"13.3.0", "13.3.01", false},
		{"", "13.3.0", false},
	}
	for _, c := range cases {
		if got := grafanaVersionsMatch(c.health, c.sdk); got != c.want {
			t.Errorf("grafanaVersionsMatch(%q, %q) = %v, want %v", c.health, c.sdk, got, c.want)
		}
	}
}
