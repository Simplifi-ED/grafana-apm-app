package plugin

import (
	"strings"
)

// summarizeClickHouseError turns datasource/SDK error blobs into a short
// user-facing reason. Raw Grafana /api/ds/query payloads include nested JSON
// and credentials placeholders; never surface those in the UI.
func summarizeClickHouseError(err error) string {
	if err == nil {
		return "unknown error"
	}
	s := err.Error()
	lower := strings.ToLower(s)

	switch {
	case strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "i/o timeout"),
		strings.Contains(lower, "deadline exceeded"),
		strings.Contains(lower, "context deadline"),
		strings.Contains(lower, "no such host"),
		strings.Contains(lower, "server misbehaving"):
		return "Configured data source is unavailable"
	case strings.Contains(lower, "authentication"),
		strings.Contains(lower, "access denied"),
		strings.Contains(lower, "password is incorrect"):
		return "Data source authentication failed. Check credentials in plugin settings."
	case strings.Contains(lower, "datasource not configured"),
		strings.Contains(lower, "uid is empty"):
		return "No data source configured. Select one in plugin settings."
	}

	// Drop nested JSON / HTTP body dumps.
	if i := strings.Index(s, "{"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	s = strings.TrimSpace(strings.TrimSuffix(s, ":"))
	if s == "" || s == "clickhouse ping failed" || s == "clickhouse ds query returned 400" {
		return "Configured data source is unavailable"
	}
	if len(s) > 120 {
		return s[:117] + "..."
	}
	return s
}
