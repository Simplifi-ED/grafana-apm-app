package plugin

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/nais/grafana-otel-plugin/pkg/plugin/chquery"
	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// ---------------------------------------------------------------------------
// Cron/Naisjob monitoring view (#74 / letter I).
//
// Built from the standard kube-state-metrics (KSM) job/cronjob families. These
// label and metric names are the KSM public contract, NOT OTel span metrics, so
// they are hardcoded here rather than sourced from otelconfig.
//
// Path A (preferred): Prometheus/Mimir metricsDataSource with:
//
//	kube_cronjob_info{cronjob, namespace, k8s_cluster_name, schedule, timezone}
//	kube_cronjob_status_last_schedule_time{cronjob, ...}   value = unix seconds
//	kube_cronjob_next_schedule_time{cronjob, ...}          value = unix seconds
//	kube_job_owner{job_name, owner_kind, owner_name, ...}
//	kube_job_status_succeeded{job_name, ...}               value = succeeded pods
//	kube_job_status_failed{job_name, reason, ...}          value = failed pods
//	kube_job_status_start_time{job_name, ...}              value = unix seconds
//	kube_job_status_completion_time{job_name, ...}         value = unix seconds
//
// Path B: same MetricNames in ClickHouse otel_metrics_gauge (KSM → OTel → CH).
// Application job metrics (e.g. app_job_*) are NOT a substitute.
//
// When neither path has series the endpoint returns {available:false} with an
// enablement note — no fake zeros.
// ---------------------------------------------------------------------------

// jobsCHLookback is the gauge snapshot window ending at `to` for Path B.
const jobsCHLookback = 2 * time.Hour

// jobsEnablementNote is the soft-unavailable copy when KSM is not on Prom or CH.
const jobsEnablementNote = "CronJob / Job monitoring needs kube-state-metrics " +
	"(kube_cronjob_*, kube_job_*) on a Prometheus metricsDataSource (Path A) " +
	"or the same series in ClickHouse otel_metrics_gauge (Path B). " +
	"Deploy kube-state-metrics in your cluster. " +
	"Application metrics such as app_job_* are not a substitute."

// Standard kube-state-metrics label names used to join and scope job series.
const (
	ksmLabelCronjob   = "cronjob"
	ksmLabelJobName   = "job_name"
	ksmLabelNamespace = "namespace"
	ksmLabelCluster   = "k8s_cluster_name"
	ksmLabelSchedule  = "schedule"
	ksmLabelTimezone  = "timezone"
	ksmLabelOwnerKind = "owner_kind"
	ksmLabelOwnerName = "owner_name"
	ksmLabelReason    = "reason"

	ownerKindCronJob = "CronJob"
)

// JobRun is the newest observed execution of a job (or CronJob's newest child).
type JobRun struct {
	// Outcome is one of: succeeded, failed, running, unknown.
	Outcome      string `json:"outcome"`
	StartMs      int64  `json:"startMs,omitempty"`
	CompletionMs int64  `json:"completionMs,omitempty"`
	// DurationSec is completion-start, only when both timestamps are present.
	DurationSec int64  `json:"durationSec,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

// JobEntry is one row in the Jobs list: a CronJob (scheduled Naisjob) enriched
// with its newest child job, or a standalone (one-shot) Job.
type JobEntry struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Cluster   string `json:"cluster"`
	// Kind is CronJob or Job.
	Kind           string  `json:"kind"`
	Schedule       string  `json:"schedule,omitempty"`
	Timezone       string  `json:"timezone,omitempty"`
	LastRun        *JobRun `json:"lastRun,omitempty"`
	LastScheduleMs int64   `json:"lastScheduleMs,omitempty"`
	NextScheduleMs int64   `json:"nextScheduleMs,omitempty"`
	// FailureStreak counts consecutive failed runs from newest backwards,
	// stopping at the first success (in-flight/unknown runs are skipped).
	FailureStreak int `json:"failureStreak"`
	// Status is one of: ok, failing, unknown (derived from the newest run).
	Status string `json:"status"`
	// RunCount is the number of child jobs observed for a CronJob.
	RunCount int `json:"runCount"`
}

// JobsResponse is the /jobs payload. Available is false when the platform does
// not expose kube-state-metrics job families (see the package doc above).
type JobsResponse struct {
	Available bool       `json:"available"`
	Jobs      []JobEntry `json:"jobs"`
	Note      string     `json:"note,omitempty"`
}

// handleJobs returns the Cron/Naisjob monitoring list.
// GET /jobs?namespace=&from=&to=
func (a *App) handleJobs(w http.ResponseWriter, req *http.Request) {
	if !requireGET(w, req) {
		return
	}
	ctx := a.requestContext(req)
	// KSM series are current-state gauges; the range only scopes caching. Query
	// the snapshot at `to`.
	_, to := parseTimeRange(req)
	namespace := queries.MustSanitizeLabel(req.URL.Query().Get("namespace"))

	orgID := req.Header.Get("X-Grafana-Org-Id")
	ck := cacheKey("jobs", orgID, roundedUnix(to), namespace)
	a.writeCached(w, ck, "querying jobs failed", func() (any, error) {
		return a.resolveJobs(ctx, req, namespace, to), nil
	})
}

// resolveJobs prefers Path A (Prom/KSM), then Path B (CH kube_*), else soft-unavailable.
func (a *App) resolveJobs(ctx context.Context, req *http.Request, namespace string, at time.Time) JobsResponse {
	if a.prom(ctx) != nil {
		resp := a.queryJobs(ctx, namespace, at)
		if resp.Available {
			return resp
		}
		// Prom configured but no KSM series — try CH before surfacing unavailable.
		if ch := a.chForRequest(req); ch != nil {
			if chResp := a.queryJobsClickHouse(ctx, ch, namespace, at); chResp.Available {
				return chResp
			}
		}
		if resp.Note == "" {
			resp.Note = jobsEnablementNote
		}
		return resp
	}

	if ch := a.chForRequest(req); ch != nil {
		return a.queryJobsClickHouse(ctx, ch, namespace, at)
	}

	return JobsResponse{
		Available: false,
		Jobs:      []JobEntry{},
		Note:      jobsEnablementNote,
	}
}

// ksmScope builds a KSM namespace label matcher, e.g. `{namespace="team-a"}`,
// or an empty string when no namespace filter is applied.
func ksmScope(namespace string) string {
	if namespace == "" {
		return ""
	}
	return fmt.Sprintf(`{%s="%s"}`, ksmLabelNamespace, namespace)
}

func (a *App) queryJobs(ctx context.Context, namespace string, at time.Time) JobsResponse {
	logger := log.DefaultLogger.With("handler", "jobs")
	scope := ksmScope(namespace)

	resultMap := a.runInstantQueries(ctx, at, []QueryJob{
		{"cronjobInfo", "kube_cronjob_info" + scope},
		{"lastSchedule", "kube_cronjob_status_last_schedule_time" + scope},
		{"nextSchedule", "kube_cronjob_next_schedule_time" + scope},
		{"jobOwner", "kube_job_owner" + scope},
		{"jobSucceeded", "kube_job_status_succeeded" + scope},
		{"jobFailed", "kube_job_status_failed" + scope},
		{"jobStart", "kube_job_status_start_time" + scope},
		{"jobCompletion", "kube_job_status_completion_time" + scope},
	}, logger)

	return aggregateJobs(resultMap)
}

// queryJobsClickHouse implements Path B: latest kube_* gauges from CH → aggregateJobs.
func (a *App) queryJobsClickHouse(ctx context.Context, ch *queries.ClickHouseClient, namespace string, at time.Time) JobsResponse {
	logger := log.DefaultLogger.With("handler", "jobs-clickhouse")
	if ch == nil {
		return JobsResponse{Available: false, Jobs: []JobEntry{}, Note: jobsEnablementNote}
	}

	from := at.Add(-jobsCHLookback)
	if from.After(at) {
		from = at.Add(-time.Hour)
	}
	opts := queries.ClickHouseQueryOptions{From: from, To: at, Limit: 5000, SkipLimitEnforcement: true}

	probe, err := ch.Query(ctx, chquery.JobsKubeProbeSQL(from, at), opts)
	if err != nil {
		logger.Warn("jobs CH kube_* probe failed", "error", err)
		return JobsResponse{Available: false, Jobs: []JobEntry{}, Note: jobsEnablementNote}
	}
	if probe.RowCount() == 0 || chquery.ParseFloat(probe.Maps()[0]["c"]) <= 0 {
		return JobsResponse{Available: false, Jobs: []JobEntry{}, Note: jobsEnablementNote}
	}

	keyByMetric := map[string]string{
		chquery.MetricKubeCronjobInfo:         "cronjobInfo",
		chquery.MetricKubeCronjobLastSchedule: "lastSchedule",
		chquery.MetricKubeCronjobNextSchedule: "nextSchedule",
		chquery.MetricKubeJobOwner:            "jobOwner",
		chquery.MetricKubeJobSucceeded:        "jobSucceeded",
		chquery.MetricKubeJobFailed:           "jobFailed",
		chquery.MetricKubeJobStartTime:        "jobStart",
		chquery.MetricKubeJobCompletionTime:   "jobCompletion",
	}
	resultMap := make(map[string][]queries.PromResult, len(keyByMetric))
	for _, metric := range chquery.JobsKubeMetricNames {
		key := keyByMetric[metric]
		res, qerr := ch.Query(ctx, chquery.JobsGaugeLatestSQL(metric, namespace, from, at), opts)
		if qerr != nil {
			logger.Warn("jobs CH gauge query failed", "metric", metric, "error", qerr)
			resultMap[key] = nil
			continue
		}
		resultMap[key] = chquery.ParseJobsGaugeRows(res.Maps())
	}

	resp := aggregateJobs(resultMap)
	if !resp.Available && resp.Note == "" {
		resp.Note = jobsEnablementNote
	}
	return resp
}

// jobRec is the joined per-job record assembled from the KSM job families.
type jobRec struct {
	cluster      string
	namespace    string
	name         string
	ownerKind    string
	ownerName    string
	succeeded    float64
	failed       float64
	reason       string
	startMs      int64
	completionMs int64
}

// outcome classifies a job from its status counters. Success takes precedence
// over failure: a job with failed pods that ultimately completed succeeded.
func (r *jobRec) outcome() string {
	switch {
	case r.succeeded >= 1:
		return "succeeded"
	case r.failed >= 1:
		return "failed"
	case r.startMs > 0 && r.completionMs == 0:
		return "running"
	default:
		return "unknown"
	}
}

func (r *jobRec) run() *JobRun {
	jr := &JobRun{Outcome: r.outcome(), StartMs: r.startMs, CompletionMs: r.completionMs}
	if r.outcome() == "failed" {
		jr.Reason = r.reason
	}
	if r.startMs > 0 && r.completionMs > 0 {
		jr.DurationSec = (r.completionMs - r.startMs) / 1000
	}
	return jr
}

// aggregateJobs joins the KSM job/cronjob result sets into the Jobs list.
func aggregateJobs(resultMap map[string][]queries.PromResult) JobsResponse {
	cronjobInfo := resultMap["cronjobInfo"]
	jobOwner := resultMap["jobOwner"]

	// Capability gate: neither family present → KSM job metrics not exposed.
	if len(cronjobInfo) == 0 && len(jobOwner) == 0 {
		return JobsResponse{
			Available: false,
			Jobs:      []JobEntry{},
			Note:      jobsEnablementNote,
		}
	}

	// 1. Assemble joined job records keyed by cluster/namespace/job_name.
	jobs := map[string]*jobRec{}
	getJob := func(r queries.PromResult) *jobRec {
		key := r.Metric[ksmLabelCluster] + "\x00" + r.Metric[ksmLabelNamespace] + "\x00" + r.Metric[ksmLabelJobName]
		jr, ok := jobs[key]
		if !ok {
			jr = &jobRec{
				cluster:   r.Metric[ksmLabelCluster],
				namespace: r.Metric[ksmLabelNamespace],
				name:      r.Metric[ksmLabelJobName],
			}
			jobs[key] = jr
		}
		return jr
	}
	for _, r := range jobOwner {
		jr := getJob(r)
		jr.ownerKind = r.Metric[ksmLabelOwnerKind]
		jr.ownerName = r.Metric[ksmLabelOwnerName]
	}
	for _, r := range resultMap["jobSucceeded"] {
		if v := r.Value.Float(); v > getJob(r).succeeded {
			getJob(r).succeeded = v
		}
	}
	for _, r := range resultMap["jobFailed"] {
		// kube_job_status_failed carries a `reason` label — one series per
		// reason. Keep the max count and record the reason that carried it.
		jr := getJob(r)
		if v := r.Value.Float(); v > jr.failed {
			jr.failed = v
		}
		if r.Value.Float() >= 1 && r.Metric[ksmLabelReason] != "" {
			jr.reason = r.Metric[ksmLabelReason]
		}
	}
	for _, r := range resultMap["jobStart"] {
		getJob(r).startMs = int64(r.Value.Float()) * 1000
	}
	for _, r := range resultMap["jobCompletion"] {
		getJob(r).completionMs = int64(r.Value.Float()) * 1000
	}

	// 2. Build CronJob entries from kube_cronjob_info, keyed by cluster/ns/cronjob.
	entries := map[string]*JobEntry{}
	cronKey := func(cluster, ns, name string) string {
		return cluster + "\x00" + ns + "\x00" + name
	}
	getCron := func(cluster, ns, name string) *JobEntry {
		key := cronKey(cluster, ns, name)
		e, ok := entries[key]
		if !ok {
			e = &JobEntry{Name: name, Namespace: ns, Cluster: cluster, Kind: ownerKindCronJob, Status: "unknown"}
			entries[key] = e
		}
		return e
	}
	for _, r := range cronjobInfo {
		e := getCron(r.Metric[ksmLabelCluster], r.Metric[ksmLabelNamespace], r.Metric[ksmLabelCronjob])
		e.Schedule = r.Metric[ksmLabelSchedule]
		e.Timezone = r.Metric[ksmLabelTimezone]
	}
	for _, r := range resultMap["lastSchedule"] {
		getCron(r.Metric[ksmLabelCluster], r.Metric[ksmLabelNamespace], r.Metric[ksmLabelCronjob]).LastScheduleMs = int64(r.Value.Float()) * 1000
	}
	for _, r := range resultMap["nextSchedule"] {
		getCron(r.Metric[ksmLabelCluster], r.Metric[ksmLabelNamespace], r.Metric[ksmLabelCronjob]).NextScheduleMs = int64(r.Value.Float()) * 1000
	}

	// 3. Group child jobs under their owning CronJob; collect standalone jobs.
	cronChildren := map[string][]*jobRec{}
	var standalone []*jobRec
	for _, jr := range jobs {
		if jr.ownerKind == ownerKindCronJob && jr.ownerName != "" {
			key := cronKey(jr.cluster, jr.namespace, jr.ownerName)
			// Synthesize a CronJob entry when the info series is absent (e.g. a
			// scrape gap or a deleted CronJob whose jobs still linger) so runs
			// stay visible.
			getCron(jr.cluster, jr.namespace, jr.ownerName)
			cronChildren[key] = append(cronChildren[key], jr)
		} else {
			standalone = append(standalone, jr)
		}
	}

	// 4. Fold each CronJob's children into last-run / streak / status.
	for key, e := range entries {
		children := cronChildren[key]
		e.RunCount = len(children)
		if len(children) == 0 {
			continue
		}
		// Newest first by start time.
		sort.Slice(children, func(i, j int) bool { return children[i].startMs > children[j].startMs })
		e.LastRun = children[0].run()
		e.FailureStreak, e.Status = streakAndStatus(children)
	}

	// 5. Standalone one-shot jobs become their own rows.
	for _, jr := range standalone {
		e := &JobEntry{
			Name:      jr.name,
			Namespace: jr.namespace,
			Cluster:   jr.cluster,
			Kind:      "Job",
			RunCount:  1,
			LastRun:   jr.run(),
		}
		streak, status := streakAndStatus([]*jobRec{jr})
		e.FailureStreak, e.Status = streak, status
		entries[cronKey(jr.cluster, jr.namespace, "job/"+jr.name)] = e
	}

	// 6. Emit as a stable, sensibly ordered slice (failing first, then name).
	out := make([]JobEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, *e)
	}
	sort.Slice(out, func(i, j int) bool {
		fi, fj := out[i].Status == "failing", out[j].Status == "failing"
		if fi != fj {
			return fi
		}
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Cluster < out[j].Cluster
	})

	return JobsResponse{Available: true, Jobs: out}
}

// streakAndStatus counts consecutive failed runs from newest backwards and
// derives the entry status. Children must be sorted newest-first. Running and
// unknown runs are skipped so an in-flight execution doesn't reset the streak
// or mask a prior failure.
func streakAndStatus(children []*jobRec) (streak int, status string) {
	status = "unknown"
	decided := false
	for _, jr := range children {
		switch jr.outcome() {
		case "failed":
			streak++
			if !decided {
				status = "failing"
				decided = true
			}
		case "succeeded":
			if !decided {
				status = "ok"
			}
			return streak, status
		default:
			// running / unknown — skip, keep scanning history.
			continue
		}
	}
	return streak, status
}
