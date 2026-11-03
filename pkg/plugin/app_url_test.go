package plugin

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/config"
)

func TestResolveGrafanaURL(t *testing.T) {
	logger := log.DefaultLogger

	t.Run("plugin override wins over SDK", func(t *testing.T) {
		ctx := config.WithGrafanaConfig(context.Background(), config.NewGrafanaCfg(map[string]string{
			config.AppURL: "https://external.example.test",
		}))
		got := resolveGrafanaURL(ctx, "http://localhost:3000/grafana", logger)
		if got != "http://localhost:3000/grafana" {
			t.Fatalf("got %q, want http://localhost:3000/grafana", got)
		}
	})

	t.Run("invalid override falls through to SDK", func(t *testing.T) {
		ctx := config.WithGrafanaConfig(context.Background(), config.NewGrafanaCfg(map[string]string{
			config.AppURL: "https://grafana.example.test",
		}))
		got := resolveGrafanaURL(ctx, "http://evil@attacker.example/", logger)
		if got != "https://grafana.example.test" {
			t.Fatalf("got %q, want https://grafana.example.test", got)
		}
	})

	t.Run("app URL set", func(t *testing.T) {
		ctx := config.WithGrafanaConfig(context.Background(), config.NewGrafanaCfg(map[string]string{
			config.AppURL: "https://grafana.example.test",
		}))
		got := resolveGrafanaURL(ctx, "", logger)
		if got != "https://grafana.example.test" {
			t.Fatalf("got %q, want https://grafana.example.test", got)
		}
	})

	t.Run("app URL with sub-path and trailing slash", func(t *testing.T) {
		ctx := config.WithGrafanaConfig(context.Background(), config.NewGrafanaCfg(map[string]string{
			config.AppURL: "https://grafana.example.test/grafana/",
		}))
		got := resolveGrafanaURL(ctx, "", logger)
		if got != "https://grafana.example.test/grafana" {
			t.Fatalf("got %q, want https://grafana.example.test/grafana", got)
		}
	})

	t.Run("app URL missing returns empty", func(t *testing.T) {
		// Empty AppURL in the config map disables the SDK's getenv fallback,
		// so this case stays deterministic regardless of the host environment.
		ctx := config.WithGrafanaConfig(context.Background(), config.NewGrafanaCfg(map[string]string{
			config.AppURL: "",
		}))
		got := resolveGrafanaURL(ctx, "", logger)
		if got != "" {
			t.Fatalf("got %q, want empty (no invented listen port)", got)
		}
	})

	t.Run("IPv6 SDK AppURL preserved", func(t *testing.T) {
		ctx := config.WithGrafanaConfig(context.Background(), config.NewGrafanaCfg(map[string]string{
			config.AppURL: "http://[::1]:3000/grafana/",
		}))
		got := resolveGrafanaURL(ctx, "", logger)
		if got != "http://[::1]:3000/grafana" {
			t.Fatalf("got %q, want http://[::1]:3000/grafana", got)
		}
	})

	t.Run("percent-encoded path preserved", func(t *testing.T) {
		ctx := config.WithGrafanaConfig(context.Background(), config.NewGrafanaCfg(map[string]string{
			config.AppURL: "https://grafana.example/my%20org/",
		}))
		got := resolveGrafanaURL(ctx, "", logger)
		if got != "https://grafana.example/my%20org" {
			t.Fatalf("got %q, want https://grafana.example/my%%20org", got)
		}
	})
}
