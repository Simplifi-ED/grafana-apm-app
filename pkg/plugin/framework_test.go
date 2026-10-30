package plugin

import (
	"testing"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

func TestFrameworkFromSDKLanguage(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"go":         "Go",
		"GO":         "Go",
		"java":       "Java",
		"nodejs":     "Node.js",
		"node":       "Node.js",
		"python":     "Python",
		"dotnet":     ".NET",
		"csharp":     ".NET",
		"rust":       "Rust",
		"cpp":        "C++",
		"c++":        "C++",
		"c":          "C",
		"webjs":      "Browser",
		"javascript": "Browser",
		"browser":    "Browser",
		"typescript": "TypeScript",
		"":           "",
		"cobol":      "",
	}
	for in, want := range cases {
		if got := FrameworkFromSDKLanguage(in); got != want {
			t.Errorf("FrameworkFromSDKLanguage(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestApplyFrameworkFromSDKLanguage(t *testing.T) {
	t.Parallel()
	summaries := []queries.ServiceSummary{
		{Name: "checkout-api", SDKLanguage: "go"},
		{Name: "ingress-gateway", SDKLanguage: "cpp"},
		{Name: "storefront", SDKLanguage: "webjs"},
		{Name: "ktor-svc", Framework: "Ktor", SDKLanguage: "java"},
	}
	applyFrameworkFromSDKLanguage(summaries)
	if summaries[0].Framework != "Go" {
		t.Fatalf("checkout-api framework = %q, want Go", summaries[0].Framework)
	}
	if summaries[1].Framework != "C++" {
		t.Fatalf("ingress-gateway framework = %q, want C++", summaries[1].Framework)
	}
	if summaries[2].Framework != "Browser" {
		t.Fatalf("frontend framework = %q, want Browser", summaries[2].Framework)
	}
	if summaries[3].Framework != "Ktor" {
		t.Fatalf("ktor framework overwritten: %q", summaries[3].Framework)
	}
}
