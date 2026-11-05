package plugin

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// URL normalisation when no candidate is reachable: the resolver falls back to
// the first override/AppURL (loopback is never used unverified).
func TestResolveGrafanaURL(t *testing.T) {
	logger := log.DefaultLogger

	t.Run("plugin override wins over SDK", func(t *testing.T) {
		got := resolveGrafanaURLWith(ctxWith("https://external.example.test", ""), "http://localhost:1/grafana", deadBase(t), logger)
		if got != "http://localhost:1/grafana" {
			t.Fatalf("got %q, want http://localhost:1/grafana", got)
		}
	})

	t.Run("invalid override falls through to SDK", func(t *testing.T) {
		got := resolveGrafanaURLWith(ctxWith("https://grafana.example.test", ""), "http://evil@attacker.example/", deadBase(t), logger)
		if got != "https://grafana.example.test" {
			t.Fatalf("got %q, want https://grafana.example.test", got)
		}
	})

	t.Run("app URL set", func(t *testing.T) {
		got := resolveGrafanaURLWith(ctxWith("https://grafana.example.test", ""), "", deadBase(t), logger)
		if got != "https://grafana.example.test" {
			t.Fatalf("got %q, want https://grafana.example.test", got)
		}
	})

	t.Run("app URL with sub-path and trailing slash", func(t *testing.T) {
		got := resolveGrafanaURLWith(ctxWith("https://grafana.example.test/grafana/", ""), "", deadBase(t), logger)
		if got != "https://grafana.example.test/grafana" {
			t.Fatalf("got %q, want https://grafana.example.test/grafana", got)
		}
	})

	t.Run("app URL missing returns empty", func(t *testing.T) {
		// Empty AppURL in the config map disables the SDK's getenv fallback,
		// so this case stays deterministic regardless of the host environment.
		got := resolveGrafanaURLWith(ctxWith("", ""), "", deadBase(t), logger)
		if got != "" {
			t.Fatalf("got %q, want empty (unverified loopback never used)", got)
		}
	})

	t.Run("IPv6 SDK AppURL preserved", func(t *testing.T) {
		got := resolveGrafanaURLWith(ctxWith("http://[::1]:1/grafana/", ""), "", deadBase(t), logger)
		if got != "http://[::1]:1/grafana" {
			t.Fatalf("got %q, want http://[::1]:1/grafana", got)
		}
	})

	t.Run("percent-encoded path preserved", func(t *testing.T) {
		got := resolveGrafanaURLWith(ctxWith("https://grafana.example/my%20org/", ""), "", deadBase(t), logger)
		if got != "https://grafana.example/my%20org" {
			t.Fatalf("got %q, want https://grafana.example/my%%20org", got)
		}
	})
}
