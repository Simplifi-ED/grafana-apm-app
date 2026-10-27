package plugin

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

func TestResolveServiceAlias_HitMiss(t *testing.T) {
	aliases := map[string]string{
		"nova-api":      "nova-app",
		"admin-api":     "admin-app",
		"nova-frontend": "nova-frontend",
		"blank":         "  ",
	}

	tests := []struct {
		name    string
		in      string
		want    string
		wantOK  bool
		aliases map[string]string
	}{
		{name: "hit scrape", in: "nova-api", want: "nova-app", wantOK: true, aliases: aliases},
		{name: "hit identity", in: "nova-frontend", want: "nova-frontend", wantOK: true, aliases: aliases},
		{name: "miss", in: "unknown-api", want: "", wantOK: false, aliases: aliases},
		{name: "blank value treated as miss", in: "blank", want: "", wantOK: false, aliases: aliases},
		{name: "empty inventory", in: "  ", want: "", wantOK: false, aliases: aliases},
		{name: "nil map", in: "nova-api", want: "", wantOK: false, aliases: nil},
		{name: "trims inventory key lookup only exact", in: " nova-api ", want: "nova-app", wantOK: true, aliases: aliases},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ResolveServiceAlias(tc.aliases, tc.in)
			if ok != tc.wantOK || got != tc.want {
				t.Fatalf("ResolveServiceAlias(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestAppResolveServiceAlias(t *testing.T) {
	app := &App{settings: queries.PluginSettings{
		ServiceAliases: map[string]string{"nova-api": "nova-app"},
	}}
	got, ok := app.ResolveServiceAlias("nova-api")
	if !ok || got != "nova-app" {
		t.Fatalf("got (%q, %v)", got, ok)
	}
	if _, ok := app.ResolveServiceAlias("missing"); ok {
		t.Fatal("expected miss")
	}
	var nilApp *App
	if _, ok := nilApp.ResolveServiceAlias("nova-api"); ok {
		t.Fatal("nil app should miss")
	}
}

func TestAliasMissingHint(t *testing.T) {
	hint := AliasMissingHint("nova-api")
	for _, want := range []string{"nova-api", "serviceAliases"} {
		if !strings.Contains(hint, want) {
			t.Fatalf("hint missing %q: %s", want, hint)
		}
	}
}

func TestPluginSettingsServiceAliasesRoundTrip(t *testing.T) {
	raw := []byte(`{
		"clickHouseDataSource": {"uid": "clickhouse", "type": "grafana-clickhouse-datasource"},
		"serviceAliases": {
			"nova-api": "nova-app",
			"admin-api": "admin-app",
			"nova-frontend": "nova-frontend"
		}
	}`)
	var settings queries.PluginSettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatal(err)
	}
	if settings.ClickHouseDataSource.UID != "clickhouse" {
		t.Fatalf("uid=%q", settings.ClickHouseDataSource.UID)
	}
	got, ok := ResolveServiceAlias(settings.ServiceAliases, "admin-api")
	if !ok || got != "admin-app" {
		t.Fatalf("got (%q, %v)", got, ok)
	}
	if _, ok := ResolveServiceAlias(settings.ServiceAliases, "ghost"); ok {
		t.Fatal("expected miss")
	}
}
