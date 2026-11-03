package plugin

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/config"
)

func TestResolveGrafanaURL(t *testing.T) {
	logger := log.DefaultLogger

	t.Run("app URL set", func(t *testing.T) {
		ctx := config.WithGrafanaConfig(context.Background(), config.NewGrafanaCfg(map[string]string{
			config.AppURL: "https://grafana.example.test",
		}))
		got := resolveGrafanaURL(ctx, logger)
		if got != "https://grafana.example.test" {
			t.Fatalf("got %q, want https://grafana.example.test", got)
		}
	})

	t.Run("app URL with sub-path and trailing slash", func(t *testing.T) {
		ctx := config.WithGrafanaConfig(context.Background(), config.NewGrafanaCfg(map[string]string{
			config.AppURL: "https://grafana.example.test/grafana/",
		}))
		got := resolveGrafanaURL(ctx, logger)
		if got != "https://grafana.example.test/grafana" {
			t.Fatalf("got %q, want https://grafana.example.test/grafana", got)
		}
	})

	t.Run("app URL missing falls back to localhost", func(t *testing.T) {
		// Empty AppURL in the config map disables the SDK's getenv fallback,
		// so this case stays deterministic regardless of the host environment.
		ctx := config.WithGrafanaConfig(context.Background(), config.NewGrafanaCfg(map[string]string{
			config.AppURL: "",
		}))
		got := resolveGrafanaURL(ctx, logger)
		if got != "http://localhost:3000" {
			t.Fatalf("got %q, want http://localhost:3000", got)
		}
	})
}
