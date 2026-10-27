package chquery

import (
	"fmt"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// ServiceGraphSQL returns client→server edge totals and failures.
func ServiceGraphSQL(from, to time.Time, environment string) string {
	fromMs, toMs := TimeBounds(from, to)
	envFilt := "1"
	if environment != "" {
		envFilt = fmt.Sprintf("ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	return fmt.Sprintf(`
SELECT
  Attributes['client'] AS client,
  Attributes['server'] AS server,
  sumIf(Value, MetricName = %s) AS total,
  sumIf(Value, MetricName = %s) AS failed
FROM %s
WHERE MetricName IN (%s, %s)
  AND TimeUnix >= fromUnixTimestamp64Milli(%d)
  AND TimeUnix <= fromUnixTimestamp64Milli(%d)
  AND Attributes['client'] != ''
  AND Attributes['server'] != ''
  AND (%s)
GROUP BY client, server
HAVING total > 0
ORDER BY total DESC
LIMIT 500`,
		q(MetricSGRequest), q(MetricSGFailed),
		table(queries.CHTableMetricsSum),
		q(MetricSGRequest), q(MetricSGFailed),
		fromMs, toMs,
		envFilt,
	)
}

// ServiceGraphEdge is one parsed dependency edge.
type ServiceGraphEdge struct {
	Client    string
	Server    string
	Total     float64
	Failed    float64
	ErrorRate float64
	Rate      float64
}

// ParseServiceGraphMaps converts CH rows into edges.
func ParseServiceGraphMaps(rows []map[string]any, windowSec float64) []ServiceGraphEdge {
	out := make([]ServiceGraphEdge, 0, len(rows))
	for _, r := range rows {
		client := ParseString(r["client"])
		server := ParseString(r["server"])
		if client == "" || server == "" {
			continue
		}
		total := ParseFloat(r["total"])
		failed := ParseFloat(r["failed"])
		errRate := 0.0
		if total > 0 {
			errRate = failed / total
		}
		out = append(out, ServiceGraphEdge{
			Client: client, Server: server,
			Total: total, Failed: failed,
			ErrorRate: errRate,
			Rate:      total / windowSec,
		})
	}
	return out
}
