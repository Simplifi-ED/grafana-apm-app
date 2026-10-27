package plugin

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

const (
	// maxReplayQueryRows bounds how many hyperdx_sessions rows we pull for one
	// session fetch. FullSnapshots are large; 5k rows covers multi-hour sessions
	// in observed ClickStack data while keeping response size manageable.
	maxReplayQueryRows = 5000

	// maxReplayListSessions caps the recordings discovery list.
	maxReplayListSessions = 50

	// maxReplayTimeRange rejects unbounded historical pulls.
	maxReplayTimeRange = 24 * time.Hour
)

// SessionRecordingSummary is one session that has HyperDX replay rows.
type SessionRecordingSummary struct {
	SessionID   string `json:"sessionId"`
	FirstSeenMs int64  `json:"firstSeenMs"`
	LastSeenMs  int64  `json:"lastSeenMs"`
	RowCount    int64  `json:"rowCount"`
	EventCount  int64  `json:"eventCount"`
}

// SessionRecordingsResponse is GET .../frontend/sessions/recordings.
type SessionRecordingsResponse struct {
	Sessions    []SessionRecordingSummary `json:"sessions"`
	Truncated   bool                      `json:"truncated,omitempty"`
	Unavailable bool                      `json:"unavailable,omitempty"`
	Error       string                    `json:"error,omitempty"`
}

// SessionReplayProbeResponse mirrors the frontend ReplayProbeResult contract.
// HyperDX has no snapshot/recording Faro modes — recordings are always "recording".
type SessionReplayProbeResponse struct {
	HasChunks   bool   `json:"hasChunks"`
	Mode        string `json:"mode"` // "recording" | "" when absent
	ChunkCount  int64  `json:"chunkCount"`
	EventCount  int64  `json:"eventCount,omitempty"`
	Unavailable bool   `json:"unavailable,omitempty"`
	Error       string `json:"error,omitempty"`
}

// SessionReplayResponse is the reassembled rrweb event stream for the player.
type SessionReplayResponse struct {
	Events      []any        `json:"events"`
	Mode        string       `json:"mode"`
	ChunkCount  int          `json:"chunkCount"`
	EventCount  int          `json:"eventCount"`
	Drops       []ReplayDrop `json:"drops,omitempty"`
	Truncated   bool         `json:"truncated,omitempty"`
	Unavailable bool         `json:"unavailable,omitempty"`
	Error       string       `json:"error,omitempty"`
}

// handleSessionRecordings lists sessions that have HyperDX recordings in range.
// GET /services/{namespace}/{service}/frontend/sessions/recordings
func (a *App) handleSessionRecordings(w http.ResponseWriter, req *http.Request) {
	if !requireGET(w, req) {
		return
	}
	ctx := a.requestContext(req)
	_, service := parseServiceRef(req)
	env := parseEnvironment(req)
	if !requireServiceParam(w, service) {
		return
	}
	from, to := parseTimeRange(req)
	if err := validateReplayTimeRange(from, to); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
		return
	}

	ch := a.clickHouseClient(req)
	if ch == nil {
		writeJSON(w, SessionRecordingsResponse{Sessions: []SessionRecordingSummary{}, Unavailable: true})
		return
	}

	resp, err := a.querySessionRecordings(ctx, ch, service, env, from, to)
	if err != nil {
		log.DefaultLogger.Warn("Session recordings query failed", "error", err, "service", service)
		writeJSON(w, SessionRecordingsResponse{
			Sessions:    []SessionRecordingSummary{},
			Unavailable: true,
			Error:       "failed to query session recordings",
		})
		return
	}
	writeJSON(w, resp)
}

// handleSessionReplayProbe checks whether a session has recording rows.
// GET /services/{namespace}/{service}/frontend/sessions/{sessionId}/replay/probe
func (a *App) handleSessionReplayProbe(w http.ResponseWriter, req *http.Request) {
	if !requireGET(w, req) {
		return
	}
	ctx := a.requestContext(req)
	_, service := parseServiceRef(req)
	env := parseEnvironment(req)
	sessionID := sanitizeSessionID(req.PathValue("sessionId"))
	if !requireServiceParam(w, service) {
		return
	}
	if sessionID == "" {
		http.Error(w, `{"error":"missing or invalid sessionId"}`, http.StatusBadRequest)
		return
	}
	from, to := parseTimeRange(req)
	if err := validateReplayTimeRange(from, to); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
		return
	}

	ch := a.clickHouseClient(req)
	if ch == nil {
		writeJSON(w, SessionReplayProbeResponse{Unavailable: true})
		return
	}

	resp, err := a.querySessionReplayProbe(ctx, ch, service, env, sessionID, from, to)
	if err != nil {
		log.DefaultLogger.Warn("Session replay probe failed", "error", err, "service", service, "sessionId", sessionID)
		writeJSON(w, SessionReplayProbeResponse{Unavailable: true, Error: "failed to probe session replay"})
		return
	}
	writeJSON(w, resp)
}

// handleSessionReplay fetches and reassembles rrweb events for a session.
// GET /services/{namespace}/{service}/frontend/sessions/{sessionId}/replay
func (a *App) handleSessionReplay(w http.ResponseWriter, req *http.Request) {
	if !requireGET(w, req) {
		return
	}
	ctx := a.requestContext(req)
	_, service := parseServiceRef(req)
	env := parseEnvironment(req)
	sessionID := sanitizeSessionID(req.PathValue("sessionId"))
	if !requireServiceParam(w, service) {
		return
	}
	if sessionID == "" {
		http.Error(w, `{"error":"missing or invalid sessionId"}`, http.StatusBadRequest)
		return
	}
	from, to := parseTimeRange(req)
	if err := validateReplayTimeRange(from, to); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
		return
	}

	ch := a.clickHouseClient(req)
	if ch == nil {
		writeJSON(w, SessionReplayResponse{Events: []any{}, Unavailable: true})
		return
	}

	resp, err := a.querySessionReplay(ctx, ch, service, env, sessionID, from, to)
	if err != nil {
		log.DefaultLogger.Warn("Session replay fetch failed", "error", err, "service", service, "sessionId", sessionID)
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
		return
	}
	writeJSON(w, resp)
}

func (a *App) clickHouseClient(req *http.Request) *queries.ClickHouseClient {
	if a == nil || a.chClient == nil {
		return nil
	}
	return a.chClient.WithServiceToken(a.resolveServiceToken(a.requestContext(req))).WithAuthHeaders(req.Header)
}

func validateReplayTimeRange(from, to time.Time) error {
	if !from.Before(to) {
		return fmt.Errorf("invalid time range: from must be before to")
	}
	if to.Sub(from) > maxReplayTimeRange {
		return fmt.Errorf("time range exceeds maximum of %s for session replay", maxReplayTimeRange)
	}
	return nil
}

func (a *App) querySessionRecordings(ctx context.Context, ch *queries.ClickHouseClient, service, env string, from, to time.Time) (SessionRecordingsResponse, error) {
	table, err := queries.QualifiedTable(queries.CHDatabaseDefault, queries.CHTableSessions)
	if err != nil {
		return SessionRecordingsResponse{}, err
	}
	where := buildReplayWhere(service, env, "")
	// Fetch one extra row to detect truncation.
	sql := fmt.Sprintf(`
SELECT
  ResourceAttributes['rum.sessionId'] AS session_id,
  toUnixTimestamp64Milli(min(Timestamp)) AS first_ms,
  toUnixTimestamp64Milli(max(Timestamp)) AS last_ms,
  count() AS row_count,
  uniqExact(LogAttributes['rr-web.event']) AS event_count
FROM %s
WHERE %s
  AND Timestamp >= fromUnixTimestamp64Milli(%d)
  AND Timestamp <= fromUnixTimestamp64Milli(%d)
  AND ResourceAttributes['rum.sessionId'] != ''
GROUP BY session_id
ORDER BY last_ms DESC
LIMIT %d`, table, where, from.UnixMilli(), to.UnixMilli(), maxReplayListSessions+1)

	res, err := ch.Query(ctx, sql, queries.ClickHouseQueryOptions{
		From:  from,
		To:    to,
		Limit: maxReplayListSessions + 1,
	})
	if err != nil {
		return SessionRecordingsResponse{}, err
	}

	out := SessionRecordingsResponse{Sessions: []SessionRecordingSummary{}}
	for _, row := range res.Maps() {
		sid := anyToString(row["session_id"])
		if sid == "" {
			continue
		}
		out.Sessions = append(out.Sessions, SessionRecordingSummary{
			SessionID:   sid,
			FirstSeenMs: parseReplayInt64(row["first_ms"]),
			LastSeenMs:  parseReplayInt64(row["last_ms"]),
			RowCount:    parseReplayInt64(row["row_count"]),
			EventCount:  parseReplayInt64(row["event_count"]),
		})
	}
	if len(out.Sessions) > maxReplayListSessions {
		out.Sessions = out.Sessions[:maxReplayListSessions]
		out.Truncated = true
	}
	return out, nil
}

func (a *App) querySessionReplayProbe(ctx context.Context, ch *queries.ClickHouseClient, service, env, sessionID string, from, to time.Time) (SessionReplayProbeResponse, error) {
	table, err := queries.QualifiedTable(queries.CHDatabaseDefault, queries.CHTableSessions)
	if err != nil {
		return SessionReplayProbeResponse{}, err
	}
	where := buildReplayWhere(service, env, sessionID)
	sql := fmt.Sprintf(`
SELECT
  count() AS row_count,
  uniqExact(LogAttributes['rr-web.event']) AS event_count
FROM %s
WHERE %s
  AND Timestamp >= fromUnixTimestamp64Milli(%d)
  AND Timestamp <= fromUnixTimestamp64Milli(%d)`, table, where, from.UnixMilli(), to.UnixMilli())

	res, err := ch.Query(ctx, sql, queries.ClickHouseQueryOptions{
		From:                 from,
		To:                   to,
		SkipLimitEnforcement: true, // aggregate returns one row
	})
	if err != nil {
		return SessionReplayProbeResponse{}, err
	}
	if res.RowCount() == 0 {
		return SessionReplayProbeResponse{HasChunks: false, Mode: ""}, nil
	}
	row := res.Maps()[0]
	rows := parseReplayInt64(row["row_count"])
	events := parseReplayInt64(row["event_count"])
	if rows <= 0 {
		return SessionReplayProbeResponse{HasChunks: false, Mode: ""}, nil
	}
	return SessionReplayProbeResponse{
		HasChunks:  true,
		Mode:       "recording",
		ChunkCount: rows,
		EventCount: events,
	}, nil
}

func (a *App) querySessionReplay(ctx context.Context, ch *queries.ClickHouseClient, service, env, sessionID string, from, to time.Time) (SessionReplayResponse, error) {
	table, err := queries.QualifiedTable(queries.CHDatabaseDefault, queries.CHTableSessions)
	if err != nil {
		return SessionReplayResponse{}, err
	}
	where := buildReplayWhere(service, env, sessionID)
	sql := fmt.Sprintf(`
SELECT
  toUnixTimestamp64Milli(Timestamp) AS ts_ms,
  Body AS body,
  LogAttributes['rr-web.event'] AS event_id,
  LogAttributes['rr-web.chunk'] AS chunk,
  LogAttributes['rr-web.total-chunks'] AS total_chunks,
  LogAttributes['rr-web.offset'] AS offset
FROM %s
WHERE %s
  AND Timestamp >= fromUnixTimestamp64Milli(%d)
  AND Timestamp <= fromUnixTimestamp64Milli(%d)
ORDER BY
  Timestamp ASC,
  toUInt32OrZero(LogAttributes['rr-web.event']) ASC,
  toUInt32OrZero(LogAttributes['rr-web.chunk']) ASC
LIMIT %d`, table, where, from.UnixMilli(), to.UnixMilli(), maxReplayQueryRows)

	res, err := ch.Query(ctx, sql, queries.ClickHouseQueryOptions{
		From:  from,
		To:    to,
		Limit: maxReplayQueryRows,
	})
	if err != nil {
		return SessionReplayResponse{}, err
	}

	rows := make([]ReplayChunkRow, 0, res.RowCount())
	for _, m := range res.Maps() {
		rows = append(rows, ReplayChunkRow{
			Body:        anyToString(m["body"]),
			EventID:     parseReplayInt(m["event_id"]),
			Chunk:       parseReplayInt(m["chunk"]),
			TotalChunks: parseReplayInt(m["total_chunks"]),
			Offset:      parseReplayInt(m["offset"]),
			TimestampMs: parseReplayInt64(m["ts_ms"]),
		})
	}

	truncated := len(rows) >= maxReplayQueryRows
	assembled, err := AssembleHyperDXReplay(rows)
	if err != nil {
		return SessionReplayResponse{}, err
	}
	assembled.Truncated = assembled.Truncated || truncated

	if assembled.EventCount == 0 {
		if len(assembled.Drops) > 0 {
			return SessionReplayResponse{}, fmt.Errorf("recording data is malformed or incomplete (%d dropped events)", len(assembled.Drops))
		}
		return SessionReplayResponse{Events: []any{}, Mode: "recording", ChunkCount: 0}, nil
	}

	events := make([]any, len(assembled.Events))
	for i, raw := range assembled.Events {
		events[i] = raw
	}
	return SessionReplayResponse{
		Events:     events,
		Mode:       "recording",
		ChunkCount: assembled.RowCount,
		EventCount: assembled.EventCount,
		Drops:      assembled.Drops,
		Truncated:  assembled.Truncated,
	}, nil
}

// buildReplayWhere constructs a safe WHERE clause for hyperdx_sessions filters.
// sessionID empty → list/probe-all-sessions path (still requires service).
func buildReplayWhere(service, env, sessionID string) string {
	parts := []string{
		"ServiceName = " + queries.QuoteClickHouseString(service),
	}
	if sessionID != "" {
		parts = append(parts, "ResourceAttributes['rum.sessionId'] = "+queries.QuoteClickHouseString(sessionID))
	}
	if env != "" {
		// Verified: HyperDX browser SDK stores cluster in ResourceAttributes['cluster'].
		envs := strings.Split(env, ",")
		if len(envs) == 1 {
			parts = append(parts, "ResourceAttributes['cluster'] = "+queries.QuoteClickHouseString(envs[0]))
		} else {
			quoted := make([]string, 0, len(envs))
			for _, e := range envs {
				if e == "" {
					continue
				}
				quoted = append(quoted, queries.QuoteClickHouseString(e))
			}
			if len(quoted) > 0 {
				parts = append(parts, "ResourceAttributes['cluster'] IN ("+strings.Join(quoted, ", ")+")")
			}
		}
	}
	return strings.Join(parts, " AND ")
}
