package chquery

import (
	"strings"
	"testing"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

func TestInventorySQL_ContainsVerifiedMetric(t *testing.T) {
	sql := InventorySQL(time.Now().Add(-time.Hour), time.Now(), "sanad", "prod-k3s-omnivya")
	for _, want := range []string{
		MetricSpanCalls,
		"SPAN_KIND_SERVER",
		"STATUS_CODE_ERROR",
		"sanad",
		"prod-k3s-omnivya",
		"LIMIT 500",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("InventorySQL missing %q\n%s", want, sql)
		}
	}
	if strings.Contains(sql, ";") {
		t.Fatal("multi-statement not allowed")
	}
}

func TestInventoryHistogramLatencySQL(t *testing.T) {
	sql := InventoryHistogramLatencySQL(time.Now().Add(-time.Hour), time.Now(), "", "", "", SpanKindServer)
	for _, want := range []string{
		MetricSpanDuration,
		"otel_metrics_histogram",
		"quantilePrometheusHistogram",
		"SPAN_KIND_SERVER",
		"arrayCumSum",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("missing %q\n%s", want, sql)
		}
	}
	if strings.Contains(sql, "otel_traces") {
		t.Fatal("inventory p95 must be histogram-primary, not otel_traces")
	}
}

func TestDiscoverInventorySQL_NoSpanKindFilter(t *testing.T) {
	sql := DiscoverInventorySQL(time.Now().Add(-time.Hour), time.Now(), "sanad", "")
	if strings.Contains(sql, "SPAN_KIND_SERVER") || strings.Contains(sql, "SPAN_KIND_CLIENT") {
		t.Fatalf("discover must include all kinds: %s", sql)
	}
	if !strings.Contains(sql, ResSDKLanguage) {
		t.Fatalf("expected sdk language attr: %s", sql)
	}
}

func TestInventoryClientSQL_ClientKind(t *testing.T) {
	sql := InventoryClientSQL(time.Now().Add(-time.Hour), time.Now(), "", "")
	if !strings.Contains(sql, SpanKindClient) {
		t.Fatalf("expected CLIENT filter: %s", sql)
	}
	if strings.Contains(sql, SpanKindServer) {
		t.Fatalf("CLIENT SQL must not filter SERVER: %s", sql)
	}
}

func TestServiceGraphSQL_BothMetrics(t *testing.T) {
	sql := ServiceGraphSQL(time.Now().Add(-time.Hour), time.Now(), "")
	if !strings.Contains(sql, MetricSGRequest) || !strings.Contains(sql, MetricSGFailed) {
		t.Fatalf("missing SG metrics: %s", sql)
	}
}

func TestParseInventoryMaps_RateAndError(t *testing.T) {
	calls := []map[string]any{
		{"service": "nova-api", "namespace": "sanad", "environment": "prod", "calls": 120.0, "errors": 12.0},
	}
	lat := []map[string]any{
		{"service": "nova-api", "p95_ms": 3.5},
	}
	got := ParseInventoryMaps(calls, lat, 60)
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Rate != 2.0 {
		t.Errorf("rate=%v want 2", got[0].Rate)
	}
	if got[0].ErrorRate != 0.1 {
		t.Errorf("errorRate=%v want 0.1", got[0].ErrorRate)
	}
	if got[0].P95Duration != 3.5 || got[0].DurationUnit != "ms" {
		t.Errorf("p95=%v unit=%s", got[0].P95Duration, got[0].DurationUnit)
	}
	if !got[0].HasServerSpans {
		t.Fatal("SERVER call rows must set HasServerSpans")
	}
}

func TestMergeInventory_ClientFallbackAndFrontend(t *testing.T) {
	discover := []map[string]any{
		{"service": "nova-api", "namespace": "sanad", "environment": "prod", "sdk_language": "go", "calls": 1.0},
		{"service": "frontend", "namespace": "sanad", "environment": "prod", "calls": 1.0},
	}
	server := []map[string]any{
		{"service": "nova-api", "namespace": "sanad", "environment": "prod", "calls": 120.0, "errors": 12.0},
	}
	client := []map[string]any{
		{"service": "frontend", "namespace": "sanad", "environment": "prod", "calls": 60.0, "errors": 0.0},
	}
	serverLat := []map[string]any{{"service": "nova-api", "p95_ms": 10.0}}
	clientLat := []map[string]any{{"service": "frontend", "p95_ms": 5.0}}
	got := MergeInventory(discover, server, client, serverLat, clientLat, nil, 60)
	byName := map[string]queries.ServiceSummary{}
	for _, s := range got {
		byName[s.Name] = s
	}
	api := byName["nova-api"]
	if !api.HasServerSpans || api.Rate != 2.0 || api.P95Duration != 10.0 || api.SDKLanguage != "go" {
		t.Fatalf("nova-api=%+v", api)
	}
	fe := byName["frontend"]
	if fe.HasServerSpans || fe.Rate != 1.0 || fe.P95Duration != 5.0 {
		t.Fatalf("frontend CLIENT fallback=%+v", fe)
	}

	MarkHasFrontend(got, map[string]bool{"nova-frontend": true}, func(inv string) string {
		if inv == "frontend" {
			return "nova-frontend"
		}
		return ""
	})
	for _, s := range got {
		if s.Name == "frontend" && !s.HasFrontend {
			t.Fatal("expected hasFrontend via alias")
		}
	}
}

func TestParseServiceNames(t *testing.T) {
	got := ParseServiceNames("sanad/nova-api,/envoy-gateway,_/admin-api,sanad/nova-api")
	if len(got) != 3 {
		t.Fatalf("got %v", got)
	}
	want := map[string]bool{"nova-api": true, "envoy-gateway": true, "admin-api": true}
	for _, n := range got {
		if !want[n] {
			t.Errorf("unexpected %q", n)
		}
	}
}

func TestInventoryCallsSeriesSQL(t *testing.T) {
	sql := InventoryCallsSeriesSQL(time.Now().Add(-time.Hour), time.Now(), 60, "sanad", "prod", "sanad/nova-api")
	for _, want := range []string{
		"toStartOfInterval",
		"toUnixTimestamp",
		"SPAN_KIND_SERVER",
		"nova-api",
		"error_rate",
		MetricSpanCalls,
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("missing %q\n%s", want, sql)
		}
	}
}

func TestInventoryDurationSeriesSQL_HistogramPrimary(t *testing.T) {
	sql := InventoryDurationSeriesSQL(time.Now().Add(-time.Hour), time.Now(), 60, "sanad", "prod", "sanad/nova-api")
	for _, want := range []string{
		MetricSpanDuration,
		"quantilePrometheusHistogram",
		"otel_metrics_histogram",
		"SPAN_KIND_SERVER",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("missing %q\n%s", want, sql)
		}
	}
	if strings.Contains(sql, "otel_traces") {
		t.Fatal("duration sparklines must not use otel_traces")
	}
}

func TestFrontendServiceNamesSQL(t *testing.T) {
	sql := FrontendServiceNamesSQL(time.Now().Add(-time.Hour), time.Now(), "prod-k3s-omnivya")
	for _, want := range []string{"webvitals", "hyperdx_sessions", "rum.sessionId", "prod-k3s-omnivya"} {
		if !strings.Contains(sql, want) {
			t.Errorf("missing %q\n%s", want, sql)
		}
	}
}

func TestAttachInventorySeries(t *testing.T) {
	sums := []queries.ServiceSummary{{Name: "nova-api", Rate: 1}}
	calls := []map[string]any{
		{"service": "nova-api", "t": int64(1000), "rate": 1.5, "error_rate": 0.02},
		{"service": "nova-api", "t": int64(2000), "rate": 2.0, "error_rate": 0.0},
	}
	durs := []map[string]any{
		{"service": "nova-api", "t": int64(1000), "p95_ms": 3.0},
		{"service": "nova-api", "t": int64(2000), "p95_ms": 4.0},
	}
	AttachInventorySeries(sums, calls, durs)
	if len(sums[0].RateSeries) != 2 || sums[0].RateSeries[1].Value != 2.0 {
		t.Fatalf("rateSeries=%v", sums[0].RateSeries)
	}
	if len(sums[0].ErrorSeries) != 2 || sums[0].ErrorSeries[0].Value != 0.02 {
		t.Fatalf("errorSeries=%v", sums[0].ErrorSeries)
	}
	if len(sums[0].DurationSeries) != 2 || sums[0].DurationSeries[1].Value != 4.0 {
		t.Fatalf("durationSeries=%v", sums[0].DurationSeries)
	}
}

func TestNamespaceDisplayExpr_PrefersK8s(t *testing.T) {
	got := NamespaceDisplayExpr("ResourceAttributes")
	if !strings.Contains(got, ResK8sNamespace) || !strings.Contains(got, ResServiceNamespace) {
		t.Fatalf("expected k8s then service.namespace: %s", got)
	}
	if !strings.Contains(got, "coalesce") {
		t.Fatalf("expected coalesce: %s", got)
	}
}

func TestFilterClause_NamespaceMatchesK8sOrService(t *testing.T) {
	got := FilterClause("prod-nova", "", "", "ResourceAttributes")
	if !strings.Contains(got, "k8s.namespace.name") || !strings.Contains(got, "service.namespace") {
		t.Fatalf("namespace filter must OR k8s + service.namespace: %s", got)
	}
	if !strings.Contains(got, "prod-nova") {
		t.Fatalf("missing namespace value: %s", got)
	}
}

func TestFilterClause_EscapesQuotes(t *testing.T) {
	got := FilterClause("", "", "O'Reilly", "ResourceAttributes")
	if !strings.Contains(got, "O''Reilly") {
		t.Fatalf("expected escaped quote: %s", got)
	}
}

func TestADR0003_ContractHelpers(t *testing.T) {
	from := time.UnixMilli(1_700_000_000_000)
	to := time.UnixMilli(1_700_000_360_000)

	metricTime := MetricTimeFilter(from, to)
	for _, want := range []string{ColTimeUnix, "fromUnixTimestamp64Milli(1700000000000)", "fromUnixTimestamp64Milli(1700000360000)"} {
		if !strings.Contains(metricTime, want) {
			t.Errorf("MetricTimeFilter missing %q: %s", want, metricTime)
		}
	}

	tsTime := TimestampFilter(from, to)
	for _, want := range []string{ColTimestamp, "fromUnixTimestamp64Milli(1700000000000)"} {
		if !strings.Contains(tsTime, want) {
			t.Errorf("TimestampFilter missing %q: %s", want, tsTime)
		}
	}

	if got := MetricSpanKindServerFilter(); !strings.Contains(got, SpanKindServer) || !strings.Contains(got, AttrSpanKind) {
		t.Fatalf("MetricSpanKindServerFilter: %s", got)
	}
	if got := MetricStatusErrorFilter(); !strings.Contains(got, StatusError) || !strings.Contains(got, AttrStatusCode) {
		t.Fatalf("MetricStatusErrorFilter: %s", got)
	}
	if got := TraceServerKindFilter(); !strings.Contains(got, TraceSpanKindServer) || !strings.Contains(got, SpanKindServer) {
		t.Fatalf("TraceServerKindFilter: %s", got)
	}
	if got := TraceStatusErrorFilter(); !strings.Contains(got, TraceStatusError) || !strings.Contains(got, StatusError) {
		t.Fatalf("TraceStatusErrorFilter dual encoding: %s", got)
	}
	if got := TraceStatusNotErrorFilter(); !strings.Contains(got, "NOT IN") {
		t.Fatalf("TraceStatusNotErrorFilter: %s", got)
	}
	if got := BucketMsExpr(ColTimeUnix, 60); !strings.Contains(got, "toStartOfInterval") || !strings.Contains(got, "* 1000") {
		t.Fatalf("BucketMsExpr: %s", got)
	}

	// InventorySQL must use metric helpers (SPAN_KIND_* / STATUS_CODE_* / TimeUnix).
	inv := InventorySQL(from, to, "sanad", "prod")
	for _, want := range []string{SpanKindServer, StatusError, "fromUnixTimestamp64Milli", ColTimeUnix} {
		if !strings.Contains(inv, want) {
			t.Errorf("InventorySQL missing %q", want)
		}
	}
	// ErrorSpansSQL must use dual StatusCode filter, not metrics STATUS_CODE alone.
	errSQL := ErrorSpansSQL("nova-api", from, to, "")
	if !strings.Contains(errSQL, TraceStatusError) || !strings.Contains(errSQL, StatusError) {
		t.Fatalf("ErrorSpansSQL expected dual StatusCode: %s", errSQL)
	}
	if strings.Contains(errSQL, "Attributes['status.code']") {
		t.Fatal("ErrorSpansSQL must not use metrics status attribute on traces")
	}
}

func TestWebVitalsSQL_TargetsHyperDXSpans(t *testing.T) {
	sql := WebVitalsSQL("nova-frontend", "prod-k3s-omnivya", time.Now().Add(-6*time.Hour), time.Now())
	for _, want := range []string{"webvitals", "nova-frontend", "quantileIf(0.75)", "SpanAttributes['lcp']", "prod-k3s-omnivya"} {
		if !strings.Contains(sql, want) {
			t.Errorf("WebVitalsSQL missing %q\n%s", want, sql)
		}
	}
}

func TestBrowserExceptionGroupsSQL_HyperDXSelector(t *testing.T) {
	from := time.Unix(1_700_000_000, 0)
	to := from.Add(time.Hour)
	sql := BrowserExceptionGroupsSQL("nova-frontend", from, to, "prod-k3s-omnivya", 50)
	for _, want := range []string{
		"nova-frontend", "unhandledrejection", "error.object", "error.message",
		"rum.sessionId", "has(Events.Name", "prod-k3s-omnivya",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("BrowserExceptionGroupsSQL missing %q\n%s", want, sql)
		}
	}
}

func TestServerExceptionGroupsSQL_ExcludesBrowserShapes(t *testing.T) {
	from := time.Unix(1_700_000_000, 0)
	to := from.Add(time.Hour)
	sql := ServerExceptionGroupsSQL("nova-api", from, to, "", 50)
	if !strings.Contains(sql, "has(Events.Name") || !strings.Contains(sql, "NOT (SpanName") {
		t.Fatalf("expected Events.exception + browser exclusion: %s", sql)
	}
}

func TestBrowserExceptionOccurrencesSQL_ReplayFields(t *testing.T) {
	from := time.Unix(1_700_000_000, 0)
	to := from.Add(time.Hour)
	sql := BrowserExceptionOccurrencesSQL("nova-frontend", from, to, "", "Error", 50)
	for _, want := range []string{"error.stack", "rum.sessionId", "userId", "location.href", "session_id"} {
		if !strings.Contains(sql, want) {
			t.Errorf("BrowserExceptionOccurrencesSQL missing %q\n%s", want, sql)
		}
	}
}

func TestSessionEnrichmentSQL_AllSpansNotErrorsOnly(t *testing.T) {
	from := time.Unix(1_700_000_000, 0)
	to := from.Add(time.Hour)
	sql := SessionEnrichmentSQL("nova-frontend", from, to, "prod-k3s-omnivya")
	for _, want := range []string{
		"countIf(",
		"unhandledrejection",
		"userId",
		"userEmail",
		"service.version",
		"location.href",
		"http.user_agent",
		"app_version",
		"pages",
		"user_agent",
		"prod-k3s-omnivya",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("SessionEnrichmentSQL missing %q\n%s", want, sql)
		}
	}
	// Must not require the error filter on the whole WHERE (healthy sessions enrich).
	whereIdx := strings.Index(sql, "WHERE")
	if whereIdx < 0 {
		t.Fatal("no WHERE")
	}
	where := sql[whereIdx:]
	if strings.Contains(where, "AND (SpanName = 'unhandledrejection'") {
		t.Fatalf("error filter must not gate entire WHERE:\n%s", where)
	}
}

func TestParseWebVitalsMaps(t *testing.T) {
	vitals, samples := ParseWebVitalsMaps([]map[string]any{
		{"lcp": 1200.0, "fcp": 800.0, "cls": 0.0, "inp": 50.0, "ttfb": 100.0, "samples": 12.0},
	})
	if samples != 12 {
		t.Fatalf("samples=%v", samples)
	}
	if vitals["lcp"] != 1200 || vitals["inp"] != 50 {
		t.Fatalf("%+v", vitals)
	}
	if _, ok := vitals["cls"]; ok {
		t.Fatal("zero cls should be omitted")
	}
}

func TestOperationsToEndpointGroups(t *testing.T) {
	ops := []queries.OperationSummary{
		{SpanName: "GET /api/x", SpanKindRaw: SpanKindServer, Rate: 1, DurationUnit: "ms"},
		{SpanName: "query SELECT", SpanKindRaw: SpanKindClient, Rate: 2, DurationUnit: "ms"},
		{SpanName: "HTTP GET", SpanKindRaw: SpanKindClient, Rate: 3, DurationUnit: "ms"},
		{SpanName: "click", SpanKindRaw: "SPAN_KIND_INTERNAL", Rate: 4, DurationUnit: "ms"},
	}
	g := OperationsToEndpointGroups(ops)
	if len(g.HTTP) != 1 || g.HTTP[0].HTTPMethod != "GET" {
		t.Fatalf("http=%+v", g.HTTP)
	}
	if len(g.Database) != 1 || g.Database[0].SpanName != "query SELECT" {
		t.Fatalf("db=%+v", g.Database)
	}
	if len(g.Client) != 1 || g.Client[0].HTTPMethod != "GET" {
		t.Fatalf("client=%+v", g.Client)
	}
	if len(g.Internal) != 1 {
		t.Fatalf("internal=%+v", g.Internal)
	}
}

func TestParseServiceGraphMaps(t *testing.T) {
	rows := []map[string]any{
		{"client": "nova-api", "server": "postgres", "total": 100.0, "failed": 5.0},
	}
	edges := ParseServiceGraphMaps(rows, 50)
	if len(edges) != 1 || edges[0].ErrorRate != 0.05 || edges[0].Rate != 2.0 {
		t.Fatalf("%+v", edges)
	}
}

func TestLogServiceIdentityFilter_IncludesK8s(t *testing.T) {
	got := LogServiceIdentityFilter("nova-api", "nova-app")
	for _, want := range []string{
		"ServiceName IN ('nova-api', 'nova-app')",
		"k8s.deployment.name",
		"startsWith(ResourceAttributes['k8s.pod.name'], 'nova-app-')",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestLogPatternSQL_NormalizesDigits(t *testing.T) {
	sql := LogPatternSQL("nova-api", time.Now().Add(-time.Hour), time.Now(), "", 20)
	if !strings.Contains(sql, "replaceRegexpAll") {
		t.Fatal(sql)
	}
}

func TestTraceDetailSQL_BindsTraceID(t *testing.T) {
	sql := TraceDetailSQL("abc123", time.Now().Add(-time.Hour), time.Now(), 100)
	if !strings.Contains(sql, "'abc123'") {
		t.Fatal(sql)
	}
}

func TestRuntimeSQL_UsesScrapeName(t *testing.T) {
	from, to := time.Now().Add(-time.Hour), time.Now()
	disc := RuntimeDiscoverSQL("nova-app", "prod-nova", from, to)
	for _, want := range []string{"nova-app", "go_", "process_", "otel_metrics_gauge"} {
		if !strings.Contains(disc, want) {
			t.Errorf("discover missing %q\n%s", want, disc)
		}
	}
	if strings.Contains(disc, "nova-api") {
		t.Fatal("must not query inventory name")
	}
	gauges := RuntimeGaugeAvgsSQL("nova-app", "prod-nova", []string{"go_goroutines"}, from, to)
	if !strings.Contains(gauges, "go_goroutines") || !strings.Contains(gauges, "nova-app") {
		t.Fatalf("gauges: %s", gauges)
	}
	cpu := RuntimeProcessCPURateSQL("admin-app", "", from, to)
	if !strings.Contains(cpu, "process_cpu_seconds_total") || !strings.Contains(cpu, "admin-app") {
		t.Fatalf("cpu: %s", cpu)
	}
}

func TestParseGoRuntimeMaps(t *testing.T) {
	got := ParseGoRuntimeMaps(
		[]map[string]any{
			{"MetricName": "go_goroutines", "value": 73.0, "pods": 2.0},
			{"MetricName": "go_memstats_alloc_bytes", "value": 1.6e7, "pods": 2.0},
		},
		[]map[string]any{{"version": "go1.27.0", "c": 2.0}},
		[]map[string]any{{"cpu_rate": 0.012}},
		[]map[string]any{{"gc_pause_avg": 0.0001, "gc_rate": 0.5}},
	)
	if got == nil || got.Goroutines != 73 || got.PodCount != 2 || len(got.Versions) != 1 {
		t.Fatalf("%+v", got)
	}
}

func TestCustomMetricsDiscoverSQL_DenylistAndScrape(t *testing.T) {
	deny := `jvm_.*|go_.*|process_.*|up|promhttp_.*`
	sql := CustomMetricsDiscoverSQL("nova-app", "prod-nova", deny, time.Now().Add(-time.Hour), time.Now(), 50)
	for _, want := range []string{
		"nova-app", "NOT match", "otel_metrics_sum", "otel_metrics_histogram", "^(jvm_.*",
		"GROUP BY MetricName, kind", // CH code 215 without outer agg keys
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("missing %q\n%s", want, sql)
		}
	}
	if strings.Contains(sql, "nova-api") {
		t.Fatal("must not use inventory name")
	}
	pat := DenylistMatchPattern(deny)
	if !strings.HasPrefix(pat, "^(") || !strings.HasSuffix(pat, ")$") {
		t.Fatalf("pattern=%q", pat)
	}
}

func TestChartHintForKind(t *testing.T) {
	if ChartHintForKind("sum") != "rate" || ChartHintForKind("histogram") != "p95" || ChartHintForKind("gauge") != "gauge" {
		t.Fatal("chart hints")
	}
	if TypeForKind("sum") != "counter" || TypeForKind("gauge") != "gauge" {
		t.Fatal("types")
	}
}

func TestScrapeServiceFilter_EmitsAliasName(t *testing.T) {
	got := ScrapeServiceFilter("nova-app", "prod-nova")
	if !strings.Contains(got, "nova-app") || !strings.Contains(got, "prod-nova") {
		t.Fatal(got)
	}
}

func TestScorecardProbeSQL(t *testing.T) {
	from, to := time.Now().Add(-time.Hour), time.Now()
	span := SpanMetricsProbeSQL("nova-api", "prod", from, to)
	if !strings.Contains(span, MetricSpanCalls) || !strings.Contains(span, "nova-api") {
		t.Fatal(span)
	}
	browser := BrowserScorecardProbeSQL("nova-frontend", "prod", from, to)
	for _, want := range []string{"hyperdx_sessions", "webvitals", "unhandledrejection", "rum.sessionId"} {
		if !strings.Contains(browser, want) {
			t.Errorf("browser probe missing %q\n%s", want, browser)
		}
	}
}

func TestTopDBQueriesSQL(t *testing.T) {
	sql := TopDBQueriesSQL("nova-api", "prod", time.Now().Add(-time.Hour), time.Now(), 100)
	for _, want := range []string{"db.statement", "db.system", "nova-api", "duration_ms"} {
		if !strings.Contains(sql, want) {
			t.Errorf("missing %q\n%s", want, sql)
		}
	}
}

func TestBrowserVersionsSQL_UsesServiceVersionNotRumVersion(t *testing.T) {
	from, to := time.Now().Add(-time.Hour), time.Now()
	sql := BrowserVersionsSQL("nova-frontend", from, to, "prod", 20)
	for _, want := range []string{
		"service.version",
		"@hyperdx/otel-web",
		"telemetry.sdk.name",
		"rum.sessionId",
		"nova-frontend",
		"unhandledrejection",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("versions SQL missing %q\n%s", want, sql)
		}
	}
	if strings.Contains(sql, "rum.version") {
		t.Fatalf("must never query rum.version (SDK): %s", sql)
	}
	probe := BrowserVersionsProbeSQL("nova-frontend", "prod", from, to)
	if !strings.Contains(probe, "service.version") || strings.Contains(probe, "rum.version") {
		t.Fatalf("probe=%s", probe)
	}
	parsed := ParseVersionGroupsMaps([]map[string]any{
		{"version": "1.72.1", "sessions": 10.0, "err_sessions": 1.0, "exceptions": 3.0},
		{"version": "", "sessions": 5.0},
	})
	if len(parsed) != 1 || parsed[0].Version != "1.72.1" || parsed[0].ErrSessions != 1 {
		t.Fatalf("%+v", parsed)
	}
}

func TestBrowserFeedbackSQL_FaroShapedContract(t *testing.T) {
	from, to := time.Now().Add(-time.Hour), time.Now()
	sql := BrowserFeedbackSQL("nova-frontend", from, to, "prod", "sess-1", "v1:abc", 50)
	for _, want := range []string{
		"faro.feedback",
		"feedback.message",
		"feedback.category",
		"feedback.email",
		"feedback.fingerprint",
		"location.href",
		"service.version",
		"rum.sessionId",
		"sess-1",
		"v1:abc",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("feedback SQL missing %q\n%s", want, sql)
		}
	}
	if strings.Contains(sql, "/api/feedback") {
		t.Fatal("must not bind to product HTTP /api/feedback")
	}
	probe := BrowserFeedbackProbeSQL("nova-frontend", "", from, to)
	if !strings.Contains(probe, "faro.feedback") {
		t.Fatalf("probe=%s", probe)
	}
	rows := ParseFeedbackRowsMaps([]map[string]any{
		{"time_ms": int64(1000), "message": "hi", "category": "bug", "session_id": "s1", "app_version": "1.0"},
	})
	if len(rows) != 1 || rows[0].Message != "hi" || rows[0].AppVersion != "1.0" {
		t.Fatalf("%+v", rows)
	}
}

func TestErrorRateAlertSQL_UsesTimeFilterMacro(t *testing.T) {
	sql := ErrorRateAlertSQL("nova-api", "ns", "prod")
	for _, want := range []string{"$__timeFilter(TimeUnix)", MetricSpanCalls, "STATUS_CODE_ERROR", "SPAN_KIND_SERVER"} {
		if !strings.Contains(sql, want) {
			t.Errorf("missing %q\n%s", want, sql)
		}
	}
}
