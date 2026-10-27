package chquery

import (
	"fmt"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// Browser SDK / version resource attrs (ticket 20/21). Never use rum.version —
// that is the HyperDX SDK version, not the app release.
const (
	ResTelemetrySDKName = "telemetry.sdk.name"
	ResServiceVersion   = "service.version"
	SDKNameHyperDXWeb   = "@hyperdx/otel-web"
	// ResRumVersion is documented only so callers can assert it is NOT queried.
	ResRumVersion = "rum.version"
)

// BrowserOtelWebFilter matches @hyperdx/otel-web resources.
func BrowserOtelWebFilter() string {
	return fmt.Sprintf("ResourceAttributes[%s] = %s", q(ResTelemetrySDKName), q(SDKNameHyperDXWeb))
}

// BrowserVersionsProbeSQL is true when any non-empty service.version exists on
// @hyperdx/otel-web browser resources for the aliased ServiceName.
func BrowserVersionsProbeSQL(service, environment string, from, to time.Time) string {
	where := fmt.Sprintf(`%s
  AND ServiceName = %s
  AND %s
  AND ResourceAttributes[%s] != ''%s`,
		TimestampFilter(from, to), q(service), BrowserOtelWebFilter(),
		q(ResServiceVersion), tracesEnvFilter(environment))
	return fmt.Sprintf(`
SELECT count() AS c
FROM %s
WHERE %s
LIMIT 1`, table(queries.CHTableTraces), where)
}

// BrowserVersionsSQL groups release health by ResourceAttributes['service.version'].
// service must already be the resolved browser ServiceName (via serviceAliases).
// Does NOT reference rum.version.
func BrowserVersionsSQL(service string, from, to time.Time, environment string, limit int) string {
	if limit <= 0 || limit > maxVersions {
		limit = maxVersions
	}
	errFilter := BrowserHyperDXErrorFilter()
	where := fmt.Sprintf(`%s
  AND ServiceName = %s
  AND %s
  AND ResourceAttributes[%s] != ''%s`,
		TimestampFilter(from, to), q(service), BrowserOtelWebFilter(),
		q(ResServiceVersion), tracesEnvFilter(environment))
	return fmt.Sprintf(`
SELECT
  ResourceAttributes[%s] AS version,
  uniqExactIf(ResourceAttributes[%s], ResourceAttributes[%s] != '') AS sessions,
  uniqExactIf(ResourceAttributes[%s],
    ResourceAttributes[%s] != '' AND (%s)) AS err_sessions,
  countIf(%s) AS exceptions
FROM %s
WHERE %s
GROUP BY version
ORDER BY sessions DESC, version ASC
LIMIT %d`,
		q(ResServiceVersion),
		q(ResRumSessionID), q(ResRumSessionID),
		q(ResRumSessionID), q(ResRumSessionID), errFilter,
		errFilter,
		table(queries.CHTableTraces), where, limit)
}

const maxVersions = 20

// VersionGroup is one release row from BrowserVersionsSQL.
type VersionGroup struct {
	Version     string
	Sessions    float64
	ErrSessions float64
	Exceptions  float64
}

// ParseVersionGroupsMaps parses BrowserVersionsSQL rows.
func ParseVersionGroupsMaps(rows []map[string]any) []VersionGroup {
	out := make([]VersionGroup, 0, len(rows))
	for _, r := range rows {
		ver := ParseString(r["version"])
		if ver == "" {
			continue
		}
		out = append(out, VersionGroup{
			Version:     ver,
			Sessions:    ParseFloat(r["sessions"]),
			ErrSessions: ParseFloat(r["err_sessions"]),
			Exceptions:  ParseFloat(r["exceptions"]),
		})
	}
	return out
}
