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

func TestFilterClause_EscapesQuotes(t *testing.T) {
	got := FilterClause("", "", "O'Reilly", "ResourceAttributes")
	if !strings.Contains(got, "O''Reilly") {
		t.Fatalf("expected escaped quote: %s", got)
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
