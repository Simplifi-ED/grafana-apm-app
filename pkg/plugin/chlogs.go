package plugin

import (
	"net/http"
	"strings"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/chquery"
	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// ClickStackLogEntry is one row in the ClickHouse-backed log search table.
type ClickStackLogEntry struct {
	TimestampMs int64  `json:"timestampMs"`
	Severity    string `json:"severity"`
	Body        string `json:"body"`
	TraceID     string `json:"traceId,omitempty"`
	SpanID      string `json:"spanId,omitempty"`
}

// ClickStackLogsResponse is the JSON payload for handleLogSearch.
type ClickStackLogsResponse struct {
	Mode string               `json:"mode"`
	Logs []ClickStackLogEntry `json:"logs"`
	Note string               `json:"note,omitempty"`
}

// handleLogSearch serves the in-app ClickStack log table. Grafana's Loki
// scene remains the LGTM path; ClickHouse mode queries otel_logs directly.
func (a *App) handleLogSearch(w http.ResponseWriter, req *http.Request) {
	if !requireGET(w, req) {
		return
	}
	_, service := parseServiceRef(req)
	if !requireServiceParam(w, service) {
		return
	}
	if !a.clickStackMode() {
		writeJSON(w, ClickStackLogsResponse{
			Mode: "unavailable",
			Logs: []ClickStackLogEntry{},
			Note: "ClickHouse datasource not configured",
		})
		return
	}

	from, to := parseTimeRange(req)
	environment := parseEnvironment(req)
	severity := strings.TrimSpace(req.URL.Query().Get("severity"))
	search := strings.TrimSpace(req.URL.Query().Get("search"))
	ch := a.chForRequest(req)
	if ch == nil {
		writeJSON(w, ClickStackLogsResponse{
			Mode: "unavailable",
			Logs: []ClickStackLogEntry{},
			Note: "ClickHouse datasource not configured",
		})
		return
	}
	res, err := ch.Query(req.Context(), chquery.LogSearchSQLFiltered(
		service, from, to, environment, severity, search, 200,
	), queries.ClickHouseQueryOptions{From: from, To: to, Limit: 200})
	if err != nil {
		writeJSON(w, ClickStackLogsResponse{
			Mode: "unavailable",
			Logs: []ClickStackLogEntry{},
			Note: "ClickHouse log query failed",
		})
		return
	}

	logs := make([]ClickStackLogEntry, 0, res.RowCount())
	for _, row := range res.Maps() {
		logs = append(logs, ClickStackLogEntry{
			TimestampMs: chquery.ParseInt64(row["ts_ms"]),
			Severity:    chquery.ParseString(row["severity"]),
			Body:        chquery.ParseString(row["body"]),
			TraceID:     chquery.ParseString(row["trace_id"]),
			SpanID:      chquery.ParseString(row["span_id"]),
		})
	}
	writeJSON(w, ClickStackLogsResponse{Mode: "clickhouse", Logs: logs})
}
