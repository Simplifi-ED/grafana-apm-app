package chquery

import (
	"strings"
	"testing"
	"time"
)

func TestJobsKubeProbeSQL_MatchesKubeOnly(t *testing.T) {
	sql := JobsKubeProbeSQL(time.Now().Add(-time.Hour), time.Now())
	for _, want := range []string{
		"otel_metrics_gauge",
		JobsKubeMetricNameMatch,
		"kube_cronjob_",
		"kube_job_",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("JobsKubeProbeSQL missing %q\n%s", want, sql)
		}
	}
	if strings.Contains(sql, "app_job_") {
		t.Fatal("probe must not match app_job_*")
	}
}

func TestJobsGaugeLatestSQL_LabelsAndNamespace(t *testing.T) {
	sql := JobsGaugeLatestSQL(MetricKubeCronjobInfo, "prod-shop", time.Now().Add(-time.Hour), time.Now())
	for _, want := range []string{
		MetricKubeCronjobInfo,
		"Attributes['cronjob']",
		"Attributes['namespace']",
		"k8s_cluster_name",
		"argMax(Value, TimeUnix)",
		"prod-shop",
		"otel_metrics_gauge",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("JobsGaugeLatestSQL missing %q\n%s", want, sql)
		}
	}
	unscoped := JobsGaugeLatestSQL(MetricKubeJobOwner, "", time.Now().Add(-time.Hour), time.Now())
	if !strings.Contains(unscoped, "AND (1)") {
		t.Fatalf("empty namespace should not filter: %s", unscoped)
	}
}

func TestParseJobsGaugeRows(t *testing.T) {
	rows := []map[string]any{
		{
			"cronjob": "gfs-backup-retention", "namespace": "prod-shop",
			"k8s_cluster_name": "prod-k3s", "schedule": "0 3 * * *", "value": 1.0,
		},
		{
			"job_name": "gfs-backup-retention-28938400", "namespace": "prod-shop",
			"k8s_cluster_name": "prod-k3s", "owner_kind": "CronJob",
			"owner_name": "gfs-backup-retention", "value": 1.0,
		},
	}
	got := ParseJobsGaugeRows(rows)
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Metric["cronjob"] != "gfs-backup-retention" || got[0].Value.Float() != 1 {
		t.Fatalf("cronjob row: %+v value=%v", got[0].Metric, got[0].Value.Float())
	}
	if got[1].Metric["owner_kind"] != "CronJob" || got[1].Metric["job_name"] == "" {
		t.Fatalf("owner row: %+v", got[1].Metric)
	}
	// Empty labels omitted (not blank keys).
	if _, ok := got[0].Metric["job_name"]; ok {
		t.Fatal("empty job_name should be omitted")
	}
}

func TestJobsKubeMetricNames_CoverContract(t *testing.T) {
	want := map[string]bool{
		MetricKubeCronjobInfo: true, MetricKubeCronjobLastSchedule: true,
		MetricKubeCronjobNextSchedule: true, MetricKubeJobOwner: true,
		MetricKubeJobSucceeded: true, MetricKubeJobFailed: true,
		MetricKubeJobStartTime: true, MetricKubeJobCompletionTime: true,
	}
	if len(JobsKubeMetricNames) != len(want) {
		t.Fatalf("len=%d want %d", len(JobsKubeMetricNames), len(want))
	}
	for _, n := range JobsKubeMetricNames {
		if !want[n] {
			t.Errorf("unexpected metric %q", n)
		}
	}
}
