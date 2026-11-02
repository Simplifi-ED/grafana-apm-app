package plugin

import (
	"errors"
	"strings"
	"testing"
)

func TestSummarizeClickHouseError_ConnectionRefused(t *testing.T) {
	raw := errors.New(`clickhouse ping failed: clickhouse ds query returned 400: {"results":{"A":{"error":"dial tcp 0.250.250.254:8123: connect: connection refused","status":502}}}`)
	got := summarizeClickHouseError(raw)
	if !strings.Contains(got, "cannot reach ClickHouse") || !strings.Contains(got, "datasource") {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, "clickstack:tunnel") || strings.Contains(got, "SSH") || strings.Contains(got, "8123") {
		t.Fatalf("leaked internal setup hint: %q", got)
	}
	if strings.Contains(got, "{") || strings.Contains(got, "0.250.250.254") {
		t.Fatalf("leaked raw blob: %q", got)
	}
}

func TestSummarizeClickHouseError_Timeout(t *testing.T) {
	got := summarizeClickHouseError(errors.New("context deadline exceeded"))
	if !strings.Contains(got, "timed out reaching ClickHouse") || !strings.Contains(got, "datasource") {
		t.Fatalf("got %q", got)
	}
}

func TestSummarizeClickHouseError_Nil(t *testing.T) {
	if summarizeClickHouseError(nil) != "unknown error" {
		t.Fatal("nil")
	}
}
