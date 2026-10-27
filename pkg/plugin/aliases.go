package plugin

import (
	"fmt"
	"strings"
)

// AliasMissingHint is the operator-facing message when a non-spanmetrics
// ClickStack path has no jsonData.serviceAliases entry for the inventory name.
func AliasMissingHint(inventoryName string) string {
	name := strings.TrimSpace(inventoryName)
	if name == "" {
		name = "<service>"
	}
	return fmt.Sprintf(
		"No serviceAliases entry for %q. Add jsonData.serviceAliases[%q] = \"<scrapeOrBrowserName>\" in plugin config (inventory → scrape/browser ServiceName).",
		name, name,
	)
}

// ResolveServiceAlias maps an inventory / spanmetrics service name to the
// scrape or browser ServiceName used by Runtime, Custom Metrics, HyperDX
// sessions/webvitals, and browser Issues. Missing → ("", false); callers must
// return structured empty + AliasMissingHint and must not query under the
// inventory name.
func (a *App) ResolveServiceAlias(inventoryName string) (scrapeName string, ok bool) {
	if a == nil {
		return "", false
	}
	return ResolveServiceAlias(a.settings.ServiceAliases, inventoryName)
}

// ResolveServiceAlias looks up inventoryName in aliases. Empty keys/values are ignored.
func ResolveServiceAlias(aliases map[string]string, inventoryName string) (scrapeName string, ok bool) {
	name := strings.TrimSpace(inventoryName)
	if name == "" || len(aliases) == 0 {
		return "", false
	}
	mapped, found := aliases[name]
	mapped = strings.TrimSpace(mapped)
	if !found || mapped == "" {
		return "", false
	}
	return mapped, true
}
