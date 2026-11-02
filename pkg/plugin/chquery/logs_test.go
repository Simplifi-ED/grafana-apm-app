package chquery

import "testing"

func TestParseLogSearchMaps_SeverityFallsBackToJSONBodyLevel(t *testing.T) {
	rows := []map[string]any{
		{
			"ts_ms":    int64(1),
			"severity": "",
			"body":     `{"time":"2026-11-02T15:22:46Z","level":"INFO","msg":"request"}`,
			"trace_id": "",
			"span_id":  "",
		},
		{
			"ts_ms":    int64(2),
			"severity": "",
			"body":     `{"level": "WARN","msg":"recurring series generate failed"}`,
			"trace_id": "",
			"span_id":  "",
		},
		{
			"ts_ms":    int64(3),
			"severity": "error",
			"body":     `{"level":"INFO","msg":"should keep SeverityText"}`,
			"trace_id": "",
			"span_id":  "",
		},
		{
			"ts_ms":    int64(4),
			"severity": "",
			"body":     `plain text without level`,
			"trace_id": "",
			"span_id":  "",
		},
	}
	got := ParseLogSearchMaps(rows)
	if got[0].Severity != "INFO" {
		t.Fatalf("row0 severity=%q, want INFO", got[0].Severity)
	}
	if got[1].Severity != "WARN" {
		t.Fatalf("row1 severity=%q, want WARN", got[1].Severity)
	}
	if got[2].Severity != "error" {
		t.Fatalf("row2 severity=%q, want error (SeverityText wins)", got[2].Severity)
	}
	if got[3].Severity != "" {
		t.Fatalf("row3 severity=%q, want empty", got[3].Severity)
	}
}

func TestParseLogSearchMaps_SeverityFallsBackToNumericJSONLevel(t *testing.T) {
	rows := []map[string]any{
		{
			"ts_ms":    int64(1),
			"severity": "",
			"body":     `{"level":30,"msg":"pino info"}`,
			"trace_id": "",
			"span_id":  "",
		},
		{
			"ts_ms":    int64(2),
			"severity": "",
			"body":     `{"level":50,"msg":"pino error"}`,
			"trace_id": "",
			"span_id":  "",
		},
		{
			"ts_ms":    int64(3),
			"severity": "",
			"body":     `{"level":7,"msg":"unknown numeric"}`,
			"trace_id": "",
			"span_id":  "",
		},
	}
	got := ParseLogSearchMaps(rows)
	if got[0].Severity != "info" {
		t.Fatalf("row0 severity=%q, want info", got[0].Severity)
	}
	if got[1].Severity != "error" {
		t.Fatalf("row1 severity=%q, want error", got[1].Severity)
	}
	if got[2].Severity != "7" {
		t.Fatalf("row2 severity=%q, want 7 (unmapped numeric)", got[2].Severity)
	}
}
