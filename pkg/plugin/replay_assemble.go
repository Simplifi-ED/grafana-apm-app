package plugin

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// HyperDX / @hyperdx/browser stores each rrweb event as one or more rows in
// hyperdx_sessions. Large events are JSON-stringified then split (~950KB) into
// chunks identified by LogAttributes:
//
//	rr-web.event         — event counter shared by all chunks of one event
//	rr-web.chunk         — 1-based chunk index
//	rr-web.total-chunks  — number of chunks for the event
//	rr-web.offset        — stream offset (ordering hint)
//
// Body is raw JSON (or a JSON fragment for chunk > 1). There is no gzip+b64
// encoding — that is the NAIS/Faro Loki format, not HyperDX.
//
// Reassembly follows HyperDX's rrwebChunkAssembler
// (https://github.com/hyperdxio/hyperdx/blob/main/packages/app/src/utils/rrwebChunkAssembler.ts):
// buffer by event id, join sorted by chunk index, JSON.parse.

const (
	// maxPendingReplayEvents caps concurrent incomplete multi-chunk events.
	maxPendingReplayEvents = 20

	// maxAssembledReplayBytes rejects recordings that would exceed this after
	// chunk concatenation (defense in depth on top of row limits).
	maxAssembledReplayBytes = 50 << 20 // 50 MiB
)

// ReplayChunkRow is one hyperdx_sessions row used for reassembly.
type ReplayChunkRow struct {
	Body        string
	EventID     int
	Chunk       int
	TotalChunks int
	Offset      int
	// TimestampMs is the ClickHouse Timestamp column (ingest / event time).
	TimestampMs int64
}

// ReplayDropReason explains why a chunked event was discarded.
type ReplayDropReason string

// Drop reasons returned in SessionReplayResponse.drops.
const (
	ReplayDropParseError     ReplayDropReason = "parse-error"
	ReplayDropIncomplete     ReplayDropReason = "incomplete-event"
	ReplayDropDuplicateChunk ReplayDropReason = "duplicate-chunk"
	ReplayDropBufferOverflow ReplayDropReason = "buffer-overflow"
	ReplayDropSizeLimit      ReplayDropReason = "size-limit"
	ReplayDropMalformed      ReplayDropReason = "malformed"
)

// ReplayDrop describes a dropped event during assembly.
type ReplayDrop struct {
	Reason         ReplayDropReason `json:"reason"`
	EventID        int              `json:"eventId,omitempty"`
	ChunksReceived int              `json:"chunksReceived"`
	TotalChunks    int              `json:"totalChunks"`
	Detail         string           `json:"detail,omitempty"`
}

// AssembledReplay is the result of reassembly.
type AssembledReplay struct {
	// Events are parsed rrweb eventWithTime objects, ordered by timestamp then offset.
	Events []json.RawMessage `json:"events"`
	// RowCount is the number of input rows considered.
	RowCount int `json:"rowCount"`
	// EventCount is the number of successfully parsed events.
	EventCount int `json:"eventCount"`
	// Drops lists events that could not be reassembled.
	Drops []ReplayDrop `json:"drops,omitempty"`
	// Truncated is set when input was cut by a row or byte limit before assembly.
	Truncated bool `json:"truncated,omitempty"`
}

type pendingReplayEvent struct {
	totalChunks int
	parts       map[int]string
	offset      int
	tsMs        int64
}

type assembledEvent struct {
	raw    json.RawMessage
	ts     int64
	offset int
	seq    int // input order for stable sort
}

// AssembleHyperDXReplay reassembles HyperDX chunked rrweb rows into events.
// rows should already be roughly time-ordered; chunk order within an event is
// enforced here regardless of input order.
func AssembleHyperDXReplay(rows []ReplayChunkRow) (AssembledReplay, error) {
	out := AssembledReplay{RowCount: len(rows)}
	if len(rows) == 0 {
		return out, nil
	}

	pending := make(map[int]*pendingReplayEvent) // insertion order tracked separately
	pendingOrder := make([]int, 0, 8)
	assembled := make([]assembledEvent, 0, len(rows))
	totalBytes := 0
	seq := 0

	emit := func(body string, eventID, chunksReceived, totalChunks, offset int, tsMs int64) {
		if totalBytes+len(body) > maxAssembledReplayBytes {
			out.Drops = append(out.Drops, ReplayDrop{
				Reason:         ReplayDropSizeLimit,
				EventID:        eventID,
				ChunksReceived: chunksReceived,
				TotalChunks:    totalChunks,
				Detail:         fmt.Sprintf("assembled payload exceeds %d bytes", maxAssembledReplayBytes),
			})
			return
		}
		if !json.Valid([]byte(body)) {
			out.Drops = append(out.Drops, ReplayDrop{
				Reason:         ReplayDropParseError,
				EventID:        eventID,
				ChunksReceived: chunksReceived,
				TotalChunks:    totalChunks,
				Detail:         "body is not valid JSON",
			})
			return
		}
		var meta struct {
			Type      int   `json:"type"`
			Timestamp int64 `json:"timestamp"`
		}
		if err := json.Unmarshal([]byte(body), &meta); err != nil {
			out.Drops = append(out.Drops, ReplayDrop{
				Reason:         ReplayDropParseError,
				EventID:        eventID,
				ChunksReceived: chunksReceived,
				TotalChunks:    totalChunks,
				Detail:         err.Error(),
			})
			return
		}
		if meta.Timestamp <= 0 && tsMs > 0 {
			// Prefer event timestamp; fall back to row time for ordering only.
			meta.Timestamp = tsMs
		}
		totalBytes += len(body)
		assembled = append(assembled, assembledEvent{
			raw:    json.RawMessage(body),
			ts:     meta.Timestamp,
			offset: offset,
			seq:    seq,
		})
		seq++
	}

	dropPending := func(eventID int, entry *pendingReplayEvent, reason ReplayDropReason) {
		delete(pending, eventID)
		out.Drops = append(out.Drops, ReplayDrop{
			Reason:         reason,
			EventID:        eventID,
			ChunksReceived: len(entry.parts),
			TotalChunks:    entry.totalChunks,
		})
		// Rebuild order slice without eventID.
		filtered := pendingOrder[:0]
		for _, id := range pendingOrder {
			if id != eventID {
				filtered = append(filtered, id)
			}
		}
		pendingOrder = filtered
	}

	for _, row := range rows {
		if row.Body == "" {
			out.Drops = append(out.Drops, ReplayDrop{
				Reason:         ReplayDropMalformed,
				EventID:        row.EventID,
				ChunksReceived: 0,
				TotalChunks:    row.TotalChunks,
				Detail:         "empty body",
			})
			continue
		}

		totalChunks := row.TotalChunks
		chunk := row.Chunk
		eventID := row.EventID

		// Unchunked (or missing chunk metadata): Body is a complete event.
		if totalChunks <= 1 || chunk == 0 {
			emit(row.Body, eventID, 1, 1, row.Offset, row.TimestampMs)
			continue
		}

		entry, ok := pending[eventID]
		if ok && entry.parts[chunk] != "" {
			// Event counter can reset on reload; a colliding chunk id means a
			// new event started before the previous completed.
			dropPending(eventID, entry, ReplayDropDuplicateChunk)
			ok = false
		}
		if !ok {
			entry = &pendingReplayEvent{
				totalChunks: totalChunks,
				parts:       make(map[int]string, totalChunks),
				offset:      row.Offset,
				tsMs:        row.TimestampMs,
			}
			pending[eventID] = entry
			pendingOrder = append(pendingOrder, eventID)
		}
		entry.parts[chunk] = row.Body
		if entry.offset == 0 || (row.Offset > 0 && row.Offset < entry.offset) {
			entry.offset = row.Offset
		}
		if entry.tsMs == 0 || (row.TimestampMs > 0 && row.TimestampMs < entry.tsMs) {
			entry.tsMs = row.TimestampMs
		}

		if len(entry.parts) >= entry.totalChunks {
			parts := make([]string, 0, entry.totalChunks)
			complete := true
			for i := 1; i <= entry.totalChunks; i++ {
				part, has := entry.parts[i]
				if !has {
					dropPending(eventID, entry, ReplayDropIncomplete)
					complete = false
					break
				}
				parts = append(parts, part)
			}
			if complete {
				body := strings.Join(parts, "")
				off, ts := entry.offset, entry.tsMs
				delete(pending, eventID)
				filtered := pendingOrder[:0]
				for _, id := range pendingOrder {
					if id != eventID {
						filtered = append(filtered, id)
					}
				}
				pendingOrder = filtered
				emit(body, eventID, entry.totalChunks, entry.totalChunks, off, ts)
			}
		}

		if len(pending) > maxPendingReplayEvents {
			oldestID := pendingOrder[0]
			dropPending(oldestID, pending[oldestID], ReplayDropBufferOverflow)
		}
	}

	for eventID, entry := range pending {
		out.Drops = append(out.Drops, ReplayDrop{
			Reason:         ReplayDropIncomplete,
			EventID:        eventID,
			ChunksReceived: len(entry.parts),
			TotalChunks:    entry.totalChunks,
		})
	}

	sort.SliceStable(assembled, func(i, j int) bool {
		if assembled[i].ts != assembled[j].ts {
			return assembled[i].ts < assembled[j].ts
		}
		if assembled[i].offset != assembled[j].offset {
			return assembled[i].offset < assembled[j].offset
		}
		return assembled[i].seq < assembled[j].seq
	})

	out.Events = make([]json.RawMessage, len(assembled))
	for i, e := range assembled {
		out.Events[i] = e.raw
	}
	out.EventCount = len(out.Events)
	return out, nil
}

// parseReplayInt coerces ClickHouse string/number attribute values to int.
func parseReplayInt(v any) int {
	switch t := v.(type) {
	case nil:
		return 0
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	case string:
		if t == "" {
			return 0
		}
		n, err := strconv.Atoi(t)
		if err != nil {
			f, ferr := strconv.ParseFloat(t, 64)
			if ferr != nil {
				return 0
			}
			return int(f)
		}
		return n
	default:
		return 0
	}
}

// parseReplayInt64 coerces values to int64 (timestamps).
func parseReplayInt64(v any) int64 {
	switch t := v.(type) {
	case nil:
		return 0
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case json.Number:
		n, _ := t.Int64()
		return n
	case string:
		if t == "" {
			return 0
		}
		n, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			f, ferr := strconv.ParseFloat(t, 64)
			if ferr != nil {
				return 0
			}
			return int64(f)
		}
		return n
	default:
		return 0
	}
}

// anyToString coerces a frame cell to string.
func anyToString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return fmt.Sprint(t)
	}
}
