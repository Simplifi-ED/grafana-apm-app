package chquery

import (
	"strings"
	"testing"
	"time"
)

func TestParseFieldParams_LogAllowlist(t *testing.T) {
	got := ParseFieldParams([]string{
		"k8s.pod.name=api-7f",
		"http.route=/x", // traces-only — ignored for logs
		"evil=1",
		"k8s.pod.name=dup", // duplicate key ignored
		"=novalue",
		"bad key=1",
	}, "log")
	if len(got) != 1 || got[0].Key != "k8s.pod.name" || got[0].Value != "api-7f" {
		t.Fatalf("got=%+v", got)
	}
}

func TestParseFieldParams_TraceAllowlist(t *testing.T) {
	got := ParseFieldParams([]string{"http.route=/api", "exception.type=Boom"}, "trace")
	if len(got) != 1 || got[0].Key != "http.route" {
		t.Fatalf("got=%+v", got)
	}
}

func TestFieldFiltersSQL_Escapes(t *testing.T) {
	sql := FieldFiltersSQL([]FieldFilter{{Key: "k8s.pod.name", Value: "O'Reilly"}}, "log")
	if !strings.Contains(sql, "ResourceAttributes['k8s.pod.name']") {
		t.Fatalf("map lookup: %s", sql)
	}
	if !strings.Contains(sql, "O''Reilly") {
		t.Fatalf("escape: %s", sql)
	}
}

func TestValidateSQLWhere(t *testing.T) {
	ok, err := ValidateSQLWhere("SeverityText = 'error'")
	if err != nil || ok != "SeverityText = 'error'" {
		t.Fatalf("ok=%q err=%v", ok, err)
	}
	if _, err := ValidateSQLWhere("1; DROP TABLE otel_logs"); err == nil {
		t.Fatal("expected reject semicolon")
	}
	if _, err := ValidateSQLWhere("1 UNION SELECT 1"); err == nil {
		t.Fatal("expected reject union")
	}
	if _, err := ValidateSQLWhere("Body LIKE '%x%' -- comment"); err == nil {
		t.Fatal("expected reject comment")
	}
}

func TestLogSearchSQLFiltered_FieldsAndWhere(t *testing.T) {
	from := time.UnixMilli(1_700_000_000_000)
	to := time.UnixMilli(1_700_000_360_000)
	sql := LogSearchSQLFiltered("svc", "svc-app", from, to, "", "error", "boom",
		[]FieldFilter{{Key: "exception.type", Value: "Timeout"}},
		"positionCaseInsensitiveUTF8(Body, 'x') > 0",
		50,
	)
	for _, want := range []string{
		"ServiceName IN ('svc', 'svc-app')",
		"k8s.deployment.name",
		"k8s.pod.name",
		"lower(SeverityText) = 'error'",
		"positionCaseInsensitiveUTF8(Body, 'boom')",
		"LogAttributes['exception.type'] = 'Timeout'",
		"positionCaseInsensitiveUTF8(Body, 'x') > 0",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("missing %q\n%s", want, sql)
		}
	}
}

func TestTraceSearchSQLFiltered_FieldsAndWhere(t *testing.T) {
	from := time.UnixMilli(1_700_000_000_000)
	to := time.UnixMilli(1_700_000_360_000)
	sql := TraceSearchSQLFiltered("svc", from, to, "", "error", "GET", 10, 0,
		[]FieldFilter{{Key: "http.route", Value: "/api"}},
		"Duration > 1000000",
		20,
	)
	for _, want := range []string{
		"SpanAttributes['http.route'] = '/api'",
		"Duration > 1000000",
		"positionCaseInsensitiveUTF8(SpanName, 'GET')",
		"Duration >= 10 * 1000000",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("missing %q\n%s", want, sql)
		}
	}
}
