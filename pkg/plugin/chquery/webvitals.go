package chquery

import (
	"fmt"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// SpanNameWebVitals is the HyperDX / @hyperdx/otel-web Internal span name used
// once per measurement; the vital name is a SpanAttributes key
// (lcp/fcp/cls/inp/ttfb/fid) with the numeric value as a string.
const SpanNameWebVitals = "webvitals"

// WebVitalsSQL returns p75 for each Core Web Vital from otel_traces webvitals spans.
func WebVitalsSQL(service, environment string, from, to time.Time) string {
	fromMs, toMs := TimeBounds(from, to)
	where := fmt.Sprintf(`Timestamp >= fromUnixTimestamp64Milli(%d)
  AND Timestamp <= fromUnixTimestamp64Milli(%d)
  AND ServiceName = %s
  AND SpanName = %s`, fromMs, toMs, q(service), q(SpanNameWebVitals))
	if environment != "" {
		where += fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	return fmt.Sprintf(`
SELECT
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['lcp']), SpanAttributes['lcp'] != '') AS lcp,
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['fcp']), SpanAttributes['fcp'] != '') AS fcp,
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['cls']), SpanAttributes['cls'] != '') AS cls,
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['inp']), SpanAttributes['inp'] != '') AS inp,
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['ttfb']), SpanAttributes['ttfb'] != '') AS ttfb,
  count() AS samples
FROM %s
WHERE %s
LIMIT 1`, table(queries.CHTableTraces), where)
}

// BrowserTelemetryProbeSQL detects HyperDX / OTel browser SDK traffic for a service.
func BrowserTelemetryProbeSQL(service, environment string, from, to time.Time) string {
	fromMs, toMs := TimeBounds(from, to)
	where := fmt.Sprintf(`Timestamp >= fromUnixTimestamp64Milli(%d)
  AND Timestamp <= fromUnixTimestamp64Milli(%d)
  AND ServiceName = %s
  AND (SpanName = %s OR ResourceAttributes['process.runtime.name'] = 'browser'
       OR ResourceAttributes['telemetry.sdk.language'] = 'webjs')`,
		fromMs, toMs, q(service), q(SpanNameWebVitals))
	if environment != "" {
		where += fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	return fmt.Sprintf(`
SELECT count() AS c
FROM %s
WHERE %s
LIMIT 1`, table(queries.CHTableTraces), where)
}

// ParseWebVitalsMaps extracts p75 vitals (ms except cls) from a CH result row.
func ParseWebVitalsMaps(rows []map[string]any) (vitals map[string]float64, samples float64) {
	if len(rows) == 0 {
		return nil, 0
	}
	r := rows[0]
	samples = ParseFloat(r["samples"])
	vitals = make(map[string]float64)
	for _, k := range []string{"lcp", "fcp", "cls", "inp", "ttfb"} {
		v := ParseFloat(r[k])
		if v > 0 {
			vitals[k] = round2(v)
		}
	}
	if len(vitals) == 0 {
		return nil, samples
	}
	return vitals, samples
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
