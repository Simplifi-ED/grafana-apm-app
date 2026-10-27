package plugin

import (
	"errors"
	"strings"
	"testing"
)

func TestSummarizeClickHouseError_ConnectionRefused(t *testing.T) {
	raw := errors.New(`clickhouse ping failed: clickhouse ds query returned 400: {"results":{"A":{"error":"dial tcp 0.250.250.254:8123: connect: connection refused","status":502}}}`)
	got := summarizeClickHouseError(raw)
	if !strings.Contains(got, "connection refused") || !strings.Contains(got, "clickstack:tunnel") {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, "{") || strings.Contains(got, "0.250.250.254") {
		t.Fatalf("leaked raw blob: %q", got)
	}
}

func TestSummarizeClickHouseError_Timeout(t *testing.T) {
	got := summarizeClickHouseError(errors.New("context deadline exceeded"))
	if got != "timed out reaching ClickHouse" {
		t.Fatalf("got %q", got)
	}
}

func TestSummarizeClickHouseError_Nil(t *testing.T) {
	if summarizeClickHouseError(nil) != "unknown error" {
		t.Fatal("nil")
	}
}
