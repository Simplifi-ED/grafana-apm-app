package plugin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/chquery"
	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// ClickStackTraceSummary is the bounded trace-list row returned by the
// ClickHouse-backed traces tab.
type ClickStackTraceSummary struct {
	TraceID   string  `json:"traceId"`
	StartMs   int64   `json:"startMs"`
	EndMs     int64   `json:"endMs"`
	SpanCount int64   `json:"spanCount"`
	Errors    int64   `json:"errors"`
	TotalMs   float64 `json:"totalMs"`
}

// ClickStackTraceSearchResponse is the JSON payload for handleTraceSearch.
type ClickStackTraceSearchResponse struct {
	Mode   string                   `json:"mode"`
	Traces []ClickStackTraceSummary `json:"traces"`
	Note   string                   `json:"note,omitempty"`
}

// ClickStackTraceSpan is one span in the ClickStack waterfall.
type ClickStackTraceSpan struct {
	TimestampMs   int64   `json:"timestampMs"`
	TraceID       string  `json:"traceId"`
	SpanID        string  `json:"spanId"`
	ParentSpanID  string  `json:"parentSpanId"`
	SpanName      string  `json:"spanName"`
	SpanKind      string  `json:"spanKind"`
	Service       string  `json:"service"`
	Status        string  `json:"status"`
	DurationMs    float64 `json:"durationMs"`
	StatusMessage string  `json:"statusMessage,omitempty"`
}

// ClickStackTraceDetailResponse is the JSON payload for handleTraceDetail.
type ClickStackTraceDetailResponse struct {
	Mode    string                `json:"mode"`
	TraceID string                `json:"traceId"`
	Spans   []ClickStackTraceSpan `json:"spans"`
	Note    string                `json:"note,omitempty"`
}

// handleTraceSearch serves the ClickStack trace list without requiring a
// Tempo datasource. The Tempo path intentionally remains owned by the
// existing TraceQL scene.
func (a *App) handleTraceSearch(w http.ResponseWriter, req *http.Request) {
	if !requireGET(w, req) {
		return
	}
	_, service := parseServiceRef(req)
	if !requireServiceParam(w, service) {
		return
	}
	if !a.clickStackMode() {
		writeJSON(w, ClickStackTraceSearchResponse{
			Mode:   "unavailable",
			Traces: []ClickStackTraceSummary{},
			Note:   "ClickHouse datasource not configured",
		})
		return
	}

	from, to := parseTimeRange(req)
	environment := parseEnvironment(req)
	status := strings.ToLower(strings.TrimSpace(req.URL.Query().Get("status")))
	if status != "error" && status != "ok" {
		status = ""
	}
	spanSearch := strings.TrimSpace(req.URL.Query().Get("span"))
	minDurationMs := parseOptionalFloat(req.URL.Query().Get("minDurationMs"))
	maxDurationMs := parseOptionalFloat(req.URL.Query().Get("maxDurationMs"))

	ch := a.chForRequest(req)
	if ch == nil {
		writeJSON(w, ClickStackTraceSearchResponse{
			Mode:   "unavailable",
			Traces: []ClickStackTraceSummary{},
			Note:   "ClickHouse datasource not configured",
		})
		return
	}
	res, err := ch.Query(req.Context(), chquery.TraceSearchSQLFiltered(
		service, from, to, environment, status, spanSearch, minDurationMs, maxDurationMs, 50,
	), queries.ClickHouseQueryOptions{From: from, To: to, Limit: 50})
	if err != nil {
		writeJSON(w, ClickStackTraceSearchResponse{
			Mode:   "unavailable",
			Traces: []ClickStackTraceSummary{},
			Note:   "ClickHouse trace query failed",
		})
		return
	}

	traces := make([]ClickStackTraceSummary, 0, res.RowCount())
	for _, row := range res.Maps() {
		traces = append(traces, ClickStackTraceSummary{
			TraceID:   chquery.ParseString(row["trace_id"]),
			StartMs:   chquery.ParseInt64(row["start_ts"]),
			EndMs:     chquery.ParseInt64(row["end_ts"]),
			SpanCount: chquery.ParseInt64(row["spans"]),
			Errors:    chquery.ParseInt64(row["errors"]),
			TotalMs:   chquery.ParseFloat(row["total_ms"]),
		})
	}
	writeJSON(w, ClickStackTraceSearchResponse{Mode: "clickhouse", Traces: traces})
}

// handleTraceDetail returns spans for one ClickStack trace so the frontend
// can render a parent/child waterfall locally.
func (a *App) handleTraceDetail(w http.ResponseWriter, req *http.Request) {
	if !requireGET(w, req) {
		return
	}
	traceID := strings.TrimSpace(req.PathValue("traceId"))
	if traceID == "" {
		http.Error(w, "missing trace id", http.StatusBadRequest)
		return
	}
	if !a.clickStackMode() {
		writeJSON(w, ClickStackTraceDetailResponse{
			Mode:    "unavailable",
			TraceID: traceID,
			Spans:   []ClickStackTraceSpan{},
			Note:    "ClickHouse datasource not configured",
		})
		return
	}
	from, to := parseTimeRange(req)
	ch := a.chForRequest(req)
	if ch == nil {
		writeJSON(w, ClickStackTraceDetailResponse{
			Mode:    "unavailable",
			TraceID: traceID,
			Spans:   []ClickStackTraceSpan{},
			Note:    "ClickHouse datasource not configured",
		})
		return
	}
	res, err := ch.Query(req.Context(), chquery.TraceDetailSQL(traceID, from, to, 2000),
		queries.ClickHouseQueryOptions{From: from, To: to, Limit: 2000})
	if err != nil {
		writeJSON(w, ClickStackTraceDetailResponse{
			Mode:    "unavailable",
			TraceID: traceID,
			Spans:   []ClickStackTraceSpan{},
			Note:    "ClickHouse trace query failed",
		})
		return
	}

	spans := make([]ClickStackTraceSpan, 0, res.RowCount())
	for _, row := range res.Maps() {
		spans = append(spans, ClickStackTraceSpan{
			TimestampMs:   chquery.ParseInt64(row["ts_ms"]),
			TraceID:       chquery.ParseString(row["trace_id"]),
			SpanID:        chquery.ParseString(row["span_id"]),
			ParentSpanID:  chquery.ParseString(row["parent_span_id"]),
			SpanName:      chquery.ParseString(row["span_name"]),
			SpanKind:      chquery.ParseString(row["span_kind"]),
			Service:       chquery.ParseString(row["service"]),
			Status:        chquery.ParseString(row["status"]),
			DurationMs:    chquery.ParseFloat(row["duration_ns"]) / 1e6,
			StatusMessage: chquery.ParseString(row["status_message"]),
		})
	}
	writeJSON(w, ClickStackTraceDetailResponse{Mode: "clickhouse", TraceID: traceID, Spans: spans})
}

func parseOptionalFloat(value string) float64 {
	if value == "" {
		return 0
	}
	v, err := strconv.ParseFloat(value, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}
