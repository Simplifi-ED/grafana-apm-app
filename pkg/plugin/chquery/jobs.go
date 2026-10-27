package chquery

import (
	"fmt"
	"strconv"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// KSM metric names for Jobs UI (letter I / ticket 17). Same families as
// pkg/plugin/jobs.go PromQL path — expected in otel_metrics_gauge when Path B
// scrapes kube-state-metrics into ClickHouse.
const (
	MetricKubeCronjobInfo         = "kube_cronjob_info"
	MetricKubeCronjobLastSchedule = "kube_cronjob_status_last_schedule_time"
	MetricKubeCronjobNextSchedule = "kube_cronjob_next_schedule_time"
	MetricKubeJobOwner            = "kube_job_owner"
	MetricKubeJobSucceeded        = "kube_job_status_succeeded"
	MetricKubeJobFailed           = "kube_job_status_failed"
	MetricKubeJobStartTime        = "kube_job_status_start_time"
	MetricKubeJobCompletionTime   = "kube_job_status_completion_time"

	// JobsKubeMetricNameMatch probes for any KSM job/cronjob family (not nova_job_*).
	JobsKubeMetricNameMatch = "^(kube_cronjob_|kube_job_)"
)

// JobsKubeMetricNames is the ordered set of gauges Jobs needs (Path B).
var JobsKubeMetricNames = []string{
	MetricKubeCronjobInfo,
	MetricKubeCronjobLastSchedule,
	MetricKubeCronjobNextSchedule,
	MetricKubeJobOwner,
	MetricKubeJobSucceeded,
	MetricKubeJobFailed,
	MetricKubeJobStartTime,
	MetricKubeJobCompletionTime,
}

// JobsKubeProbeSQL is true when any kube_cronjob_* / kube_job_* gauge exists
// in the lookback window. Does not match nova_job_* (research 09).
func JobsKubeProbeSQL(from, to time.Time) string {
	return fmt.Sprintf(`
SELECT count() AS c FROM (
  SELECT 1
  FROM %s
  WHERE match(MetricName, %s)
    AND %s
  LIMIT 1
)
LIMIT 1`,
		table(queries.CHTableMetricsGauge),
		q(JobsKubeMetricNameMatch),
		MetricTimeFilter(from, to),
	)
}

func jobsNamespaceFilter(namespace string) string {
	if namespace == "" {
		return "1"
	}
	return fmt.Sprintf("Attributes['namespace'] = %s", q(namespace))
}

// JobsGaugeLatestSQL returns the latest Value per KSM label set for one metric.
// Labels mirror the Prom path (cronjob/job_name/namespace/k8s_cluster_name/…).
func JobsGaugeLatestSQL(metricName, namespace string, from, to time.Time) string {
	clusterExpr := `coalesce(
    nullIf(Attributes['k8s_cluster_name'], ''),
    nullIf(ResourceAttributes['k8s.cluster.name'], ''),
    nullIf(ResourceAttributes['cluster'], ''),
    ''
  )`
	return fmt.Sprintf(`
SELECT
  Attributes['cronjob'] AS cronjob,
  Attributes['job_name'] AS job_name,
  Attributes['namespace'] AS namespace,
  %s AS k8s_cluster_name,
  Attributes['schedule'] AS schedule,
  Attributes['timezone'] AS timezone,
  Attributes['owner_kind'] AS owner_kind,
  Attributes['owner_name'] AS owner_name,
  Attributes['reason'] AS reason,
  argMax(Value, TimeUnix) AS value
FROM %s
WHERE MetricName = %s
  AND %s
  AND (%s)
GROUP BY
  cronjob, job_name, namespace, k8s_cluster_name,
  schedule, timezone, owner_kind, owner_name, reason
LIMIT 5000`,
		clusterExpr,
		table(queries.CHTableMetricsGauge),
		q(metricName),
		MetricTimeFilter(from, to),
		jobsNamespaceFilter(namespace),
	)
}

// ParseJobsGaugeRows maps CH gauge rows into PromResult so Jobs can reuse
// aggregateJobs without duplicating streak/status join logic.
func ParseJobsGaugeRows(rows []map[string]any) []queries.PromResult {
	out := make([]queries.PromResult, 0, len(rows))
	keys := []string{
		"cronjob", "job_name", "namespace", "k8s_cluster_name",
		"schedule", "timezone", "owner_kind", "owner_name", "reason",
	}
	for _, r := range rows {
		m := make(map[string]string, len(keys))
		for _, k := range keys {
			if v := ParseString(r[k]); v != "" {
				m[k] = v
			}
		}
		val := ParseFloat(r["value"])
		out = append(out, queries.PromResult{
			Metric: m,
			Value:  queries.NewPromValue(0, strconv.FormatFloat(val, 'f', -1, 64)),
		})
	}
	return out
}
