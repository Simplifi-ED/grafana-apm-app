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

func TestLatencySQL_UsesTraces(t *testing.T) {
	sql := LatencySQL(time.Now().Add(-time.Hour), time.Now(), "", "")
	if !strings.Contains(sql, "otel_traces") {
		t.Fatalf("expected otel_traces: %s", sql)
	}
	if !strings.Contains(sql, "quantile(0.95)") {
		t.Fatal("expected p95 quantile")
	}
	if !strings.Contains(sql, "Server") || !strings.Contains(sql, "SPAN_KIND_SERVER") {
		t.Fatalf("expected dual SpanKind match: %s", sql)
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
