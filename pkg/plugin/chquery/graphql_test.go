package chquery

import (
	"strings"
	"testing"
	"time"
)

func TestGraphQLSpanSQL_MatchesFixtureShape(t *testing.T) {
	from := time.UnixMilli(1_700_000_000_000)
	to := time.UnixMilli(1_700_000_360_000)
	detect := GraphQLSpanDetectSQL("graphql-demo", "demo", "ci", from, to)
	for _, want := range []string{
		"graphql-demo",
		"graphql.operation.name",
		"graphql.operation.type",
		"graphql.document",
		"mapContains",
		"otel_traces",
		"demo",
		"ci",
	} {
		if !strings.Contains(detect, want) {
			t.Errorf("detect missing %q\n%s", want, detect)
		}
	}
	if strings.Contains(detect, ";") {
		t.Fatal("multi-statement not allowed")
	}

	ops := GraphQLSpanOpsSQL("graphql-demo", "demo", "ci", from, to, 50)
	for _, want := range []string{
		"op_name",
		"op_type",
		"avg_latency_s",
		"error_frac",
		"Duration / 1e9",
		"StatusCode IN",
		"LIMIT 50",
	} {
		if !strings.Contains(ops, want) {
			t.Errorf("ops missing %q\n%s", want, ops)
		}
	}
}

func TestGraphQLMetricDiscoverSQL_ProbePrefixes(t *testing.T) {
	from, to := time.Now().Add(-time.Hour), time.Now()
	sql := GraphQLMetricDiscoverSQL("graphql-demo", "demo", from, to)
	for _, want := range []string{
		"otel_metrics_sum",
		"otel_metrics_summary",
		"graphql_|mp_graphql_|requests_graphql_",
		"graphql-demo",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("discover missing %q\n%s", want, sql)
		}
	}
}

func TestGraphQLMetricOpsSQL_LabelsAndErrorFilter(t *testing.T) {
	from, to := time.Now().Add(-5*time.Minute), time.Now()
	sql := GraphQLMetricOpsSQL(
		"svc", "ns",
		"graphql_request_seconds_count", "graphql_request_seconds_sum",
		"graphql_operation_type", "", `graphql_outcome="REQUEST_ERROR"`,
		from, to, 50,
	)
	for _, want := range []string{
		"graphql_request_seconds_count",
		"graphql_request_seconds_sum",
		"Attributes['graphql_operation_type']",
		"Attributes['graphql_outcome'] = 'REQUEST_ERROR'",
		"error_frac",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("ops missing %q\n%s", want, sql)
		}
	}

	mp := GraphQLMetricOpsSQL(
		"svc", "",
		"mp_graphql_seconds_count", "mp_graphql_seconds_sum",
		"name", "type", "",
		from, to, 20,
	)
	if !strings.Contains(mp, "Attributes['name']") || !strings.Contains(mp, "Attributes['type']") {
		t.Fatalf("microprofile labels: %s", mp)
	}
	if strings.Contains(mp, "error_delta") {
		t.Fatal("no error filter → no error join")
	}
}

func TestParseGraphQLOpRows_FixtureLikeSpans(t *testing.T) {
	// Two ops as produced by GraphQLSpanOpsSQL against scripts/fixtures/graphql-otel-spans.sql
	rows := ParseGraphQLOpRows([]map[string]any{
		{"op_name": "GetInvoice", "op_type": "query", "rate": 0.003, "avg_latency_s": 0.0125, "error_frac": 0.0},
		{"op_name": "CreateInvoice", "op_type": "mutation", "rate": 0.003, "avg_latency_s": 0.042, "error_frac": 1.0},
		{"op_name": "", "op_type": "", "rate": 1.0}, // dropped
		{"op_name": "Zero", "rate": 0.0},             // dropped
	}, "s")
	if len(rows) != 2 {
		t.Fatalf("got %d rows: %+v", len(rows), rows)
	}
	if rows[0].Name != "GetInvoice" || rows[0].Type != "query" || rows[0].LatencyUnit != "s" {
		t.Fatalf("row0=%+v", rows[0])
	}
	if !rows[1].HasErrorRate || rows[1].ErrorRate != 100 {
		t.Fatalf("CreateInvoice errorRate=%v has=%v", rows[1].ErrorRate, rows[1].HasErrorRate)
	}
}

func TestParseGraphQLMetricNames(t *testing.T) {
	got := ParseGraphQLMetricNames([]map[string]any{
		{"MetricName": "graphql_request_seconds_count", "kind": "sum"},
		{"metricname": "mp_graphql_seconds_sum"},
		{"MetricName": ""},
	})
	if !got["graphql_request_seconds_count"] || !got["mp_graphql_seconds_sum"] || len(got) != 2 {
		t.Fatalf("%v", got)
	}
}

func TestPromLabelFilterToCH(t *testing.T) {
	eq := promLabelFilterToCH(`graphql_outcome="REQUEST_ERROR"`)
	if eq != "Attributes['graphql_outcome'] = 'REQUEST_ERROR'" {
		t.Fatalf("eq=%q", eq)
	}
	neg := promLabelFilterToCH(`graphql_outcome!="SUCCESS"`)
	if neg != "Attributes['graphql_outcome'] != 'SUCCESS'" {
		t.Fatalf("neg=%q", neg)
	}
	if promLabelFilterToCH("") != "" || promLabelFilterToCH("nonsense") != "" {
		t.Fatal("expected empty for bad filters")
	}
}

func TestGraphQLPDLOpSQL(t *testing.T) {
	from, to := time.Now().Add(-time.Minute), time.Now()
	sql := GraphQLPDLOpSQL("svc", "ns", "HentPerson", "graphql_HentPerson_seconds_count", "graphql_HentPerson_seconds_sum", from, to)
	for _, want := range []string{"HentPerson", "graphql_HentPerson_seconds_count", "graphql_HentPerson_seconds_sum", "op_type"} {
		if !strings.Contains(sql, want) {
			t.Errorf("missing %q\n%s", want, sql)
		}
	}
}
