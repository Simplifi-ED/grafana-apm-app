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
