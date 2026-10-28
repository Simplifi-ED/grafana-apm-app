package plugin

import (
	"strings"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// FrameworkFromSDKLanguage maps OTel telemetry.sdk.language values to inventory
// Type badge labels. Empty / unknown languages return "".
//
// Specific metric probes (Ktor, Spring Boot, …) still win when present; this is
// the fallback used by both Prometheus and ClickStack inventory paths.
func FrameworkFromSDKLanguage(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "java":
		return "Java"
	case "go":
		return "Go"
	case "nodejs", "node":
		return "Node.js"
	case "python":
		return "Python"
	case "dotnet", "csharp":
		return ".NET"
	case "rust":
		return "Rust"
	case "cpp", "c++", "cxx":
		return "C++"
	case "c":
		return "C"
	case "webjs", "javascript", "js", "browser":
		return "Browser"
	case "typescript", "ts":
		return "TypeScript"
	case "php":
		return "PHP"
	case "ruby":
		return "Ruby"
	case "swift":
		return "Swift"
	case "kotlin":
		return "Kotlin"
	default:
		return ""
	}
}

// applyFrameworkFromSDKLanguage fills Framework from SDKLanguage when unset.
func applyFrameworkFromSDKLanguage(summaries []queries.ServiceSummary) {
	for i := range summaries {
		if summaries[i].Framework == "" && summaries[i].SDKLanguage != "" {
			summaries[i].Framework = FrameworkFromSDKLanguage(summaries[i].SDKLanguage)
		}
	}
}
