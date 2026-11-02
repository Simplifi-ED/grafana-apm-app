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
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "simple eq", in: "SeverityText = 'error'", want: "SeverityText = 'error'"},
		{name: "allowed function", in: "positionCaseInsensitiveUTF8(Body, 'x') > 0", want: "positionCaseInsensitiveUTF8(Body, 'x') > 0"},
		{name: "parens in string literal", in: "Body = 'a) OR (1=1'", want: "Body = 'a) OR (1=1'"},
		{name: "empty", in: "", want: ""},
		{name: "semicolon", in: "1; DROP TABLE otel_logs", wantErr: true},
		{name: "union", in: "1 UNION SELECT 1", wantErr: true},
		{name: "comment", in: "Body LIKE '%x%' -- comment", wantErr: true},
		{name: "scope breakout OR", in: "1=0) OR (ServiceName='other-svc'", wantErr: true},
		{name: "unbalanced close", in: "SeverityText = 'error')", wantErr: true},
		{name: "unbalanced open", in: "(SeverityText = 'error'", wantErr: true},
		{name: "url table function", in: "1=1 OR (SELECT count() FROM url('http://169.254.169.254/','RawBLOB'))>0", wantErr: true},
		{name: "file table function", in: "file('/etc/passwd') = 'x'", wantErr: true},
		{name: "remote table function", in: "remote('h', 'db', 't') = 1", wantErr: true},
		{name: "select subquery", in: "(SELECT count() FROM otel_logs) > 0", wantErr: true},
		{name: "bare select keyword", in: "select 1", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ValidateSQLWhere(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
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
