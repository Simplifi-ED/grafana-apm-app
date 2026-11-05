package plugin

import (
	"errors"
	"strings"
	"testing"
)

func TestSummarizeClickHouseError_ConnectionRefused(t *testing.T) {
	raw := errors.New(`clickhouse ping failed: clickhouse ds query returned 400: {"results":{"A":{"error":"dial tcp 0.250.250.254:8123: connect: connection refused","status":502}}}`)
	got := summarizeClickHouseError(raw)
	if got != "Configured data source is unavailable" {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, "ClickHouse") || strings.Contains(got, "clickstack:tunnel") || strings.Contains(got, "—") {
		t.Fatalf("leaked internal or AI-style copy: %q", got)
	}
	if strings.Contains(got, "{") || strings.Contains(got, "0.250.250.254") {
		t.Fatalf("leaked raw blob: %q", got)
	}
}

func TestSummarizeClickHouseError_Timeout(t *testing.T) {
	got := summarizeClickHouseError(errors.New("context deadline exceeded"))
	if got != "Configured data source is unavailable" {
		t.Fatalf("got %q", got)
	}
}

func TestSummarizeClickHouseError_Nil(t *testing.T) {
	if summarizeClickHouseError(nil) != "unknown error" {
		t.Fatal("nil")
	}
}

// One case per row of the spec table (Grafana hop vs datasource).
func TestSummarizeClickHouseError_Categories(t *testing.T) {
	cases := []struct {
		name string
		err  string
		want string
	}{
		{
			name: "grafana DNS",
			err:  "clickhouse ping failed: executing clickhouse ds query: dial tcp: lookup monitoring-clickstack.tailec7cf.ts.net on 127.0.0.53:53: no such host",
			want: "Plugin cannot resolve Grafana host monitoring-clickstack.tailec7cf.ts.net (DNS)",
		},
		{
			name: "grafana DNS server misbehaving",
			err:  "executing clickhouse ds query: dial tcp: lookup grafana.internal: server misbehaving",
			want: "Plugin cannot resolve Grafana host grafana.internal (DNS)",
		},
		{
			name: "grafana connection refused",
			err:  "clickhouse ping failed: executing clickhouse ds query: dial tcp 127.0.0.1:3000: connect: connection refused",
			want: "Plugin cannot reach Grafana at 127.0.0.1:3000",
		},
		{
			name: "grafana dial timeout",
			err:  "executing clickhouse ds query: dial tcp 10.0.0.5:443: i/o timeout",
			want: "Plugin cannot reach Grafana at 10.0.0.5:443",
		},
		{
			name: "grafana TLS",
			err:  `executing clickhouse ds query: Post "https://grafana.example/api/ds/query": tls: failed to verify certificate: x509: certificate signed by unknown authority`,
			want: "TLS error calling Grafana at grafana.example",
		},
		{
			name: "grafana 401",
			err:  `clickhouse ping failed: clickhouse ds query returned 401: {"message":"Unauthorized"}`,
			want: "Grafana rejected plugin credentials (401). Check managed service accounts.",
		},
		{
			name: "grafana 403",
			err:  `clickhouse ds query returned 403: {"message":"Forbidden"}`,
			want: "Grafana rejected plugin credentials (403). Check managed service accounts.",
		},
		{
			name: "grafana 400 without datasource network error",
			err:  `clickhouse ds query returned 400: {"results":{"A":{"error":"Syntax error: failed at position 1","status":400}}}`,
			want: "Grafana rejected the ClickHouse query (400)",
		},
		{
			name: "clickhouse auth unchanged",
			err:  "clickhouse query error: code: 516, message: default: Authentication failed: password is incorrect",
			want: "Data source authentication failed. Check credentials in plugin settings.",
		},
		{
			name: "datasource not configured unchanged",
			err:  "clickhouse datasource not configured",
			want: "No data source configured. Select one in plugin settings.",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := summarizeClickHouseError(errors.New(c.err))
			if got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
			if strings.Contains(got, "{") {
				t.Fatalf("leaked raw blob: %q", got)
			}
		})
	}
}
