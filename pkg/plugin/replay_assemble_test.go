package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestAssembleHyperDXReplay_UnchunkedOrdered(t *testing.T) {
	rows := []ReplayChunkRow{
		{Body: `{"type":4,"data":{"href":"https://example.test/","width":800,"height":600},"timestamp":1000}`, EventID: 1, Chunk: 1, TotalChunks: 1, Offset: 1},
		{Body: `{"type":2,"data":{"node":{"type":0,"childNodes":[]}},"timestamp":1010}`, EventID: 2, Chunk: 1, TotalChunks: 1, Offset: 2},
		{Body: `{"type":3,"data":{"source":1},"timestamp":1020}`, EventID: 3, Chunk: 1, TotalChunks: 1, Offset: 3},
	}
	got, err := AssembleHyperDXReplay(rows)
	if err != nil {
		t.Fatal(err)
	}
	if got.EventCount != 3 {
		t.Fatalf("EventCount=%d, want 3", got.EventCount)
	}
	if len(got.Drops) != 0 {
		t.Fatalf("unexpected drops: %+v", got.Drops)
	}
	var first struct {
		Type      int   `json:"type"`
		Timestamp int64 `json:"timestamp"`
	}
	if err := json.Unmarshal(got.Events[0], &first); err != nil {
		t.Fatal(err)
	}
	if first.Type != 4 || first.Timestamp != 1000 {
		t.Fatalf("first event = %+v", first)
	}
}

func TestAssembleHyperDXReplay_MultiChunkOutOfOrder(t *testing.T) {
	full := `{"type":2,"data":{"node":{"type":0,"childNodes":[{"type":2,"tagName":"div","attributes":{},"childNodes":[]}]}},"timestamp":2000}`
	mid := len(full) / 2
	part1 := full[:mid]
	part2 := full[mid:]

	// Chunk 2 arrives before chunk 1 — assembler must sort by chunk index.
	rows := []ReplayChunkRow{
		{Body: `{"type":4,"data":{"href":"https://example.test/a"},"timestamp":1990}`, EventID: 1, Chunk: 1, TotalChunks: 1, Offset: 1},
		{Body: part2, EventID: 10, Chunk: 2, TotalChunks: 2, Offset: 3, TimestampMs: 2000},
		{Body: part1, EventID: 10, Chunk: 1, TotalChunks: 2, Offset: 2, TimestampMs: 2000},
		{Body: `{"type":3,"data":{"source":2},"timestamp":2010}`, EventID: 11, Chunk: 1, TotalChunks: 1, Offset: 4},
	}
	got, err := AssembleHyperDXReplay(rows)
	if err != nil {
		t.Fatal(err)
	}
	if got.EventCount != 3 {
		t.Fatalf("EventCount=%d want 3; drops=%+v", got.EventCount, got.Drops)
	}
	var snap struct {
		Type      int   `json:"type"`
		Timestamp int64 `json:"timestamp"`
	}
	if err := json.Unmarshal(got.Events[1], &snap); err != nil {
		t.Fatal(err)
	}
	if snap.Type != 2 || snap.Timestamp != 2000 {
		t.Fatalf("reassembled snapshot = %+v", snap)
	}
}

func TestAssembleHyperDXReplay_DuplicateChunkDropsPending(t *testing.T) {
	full := `{"type":2,"data":{"x":1},"timestamp":3000}`
	part1 := full[:len(full)/2]
	rows := []ReplayChunkRow{
		{Body: part1, EventID: 5, Chunk: 1, TotalChunks: 2},
		{Body: part1, EventID: 5, Chunk: 1, TotalChunks: 2}, // colliding chunk → drop first pending
		// No chunk 2 — second pending stays incomplete.
	}
	got, err := AssembleHyperDXReplay(rows)
	if err != nil {
		t.Fatal(err)
	}
	if got.EventCount != 0 {
		t.Fatalf("expected 0 events, got %d", got.EventCount)
	}
	foundDup := false
	foundIncomplete := false
	for _, d := range got.Drops {
		if d.Reason == ReplayDropDuplicateChunk {
			foundDup = true
		}
		if d.Reason == ReplayDropIncomplete {
			foundIncomplete = true
		}
	}
	if !foundDup || !foundIncomplete {
		t.Fatalf("drops=%+v, want duplicate-chunk and incomplete-event", got.Drops)
	}
}

func TestAssembleHyperDXReplay_MalformedSkipped(t *testing.T) {
	rows := []ReplayChunkRow{
		{Body: ``, EventID: 1, Chunk: 1, TotalChunks: 1},
		{Body: `not-json`, EventID: 2, Chunk: 1, TotalChunks: 1},
		{Body: `{"type":4,"data":{},"timestamp":1}`, EventID: 3, Chunk: 1, TotalChunks: 1},
	}
	got, err := AssembleHyperDXReplay(rows)
	if err != nil {
		t.Fatal(err)
	}
	if got.EventCount != 1 {
		t.Fatalf("EventCount=%d want 1; drops=%+v", got.EventCount, got.Drops)
	}
}

func TestAssembleHyperDXReplay_Empty(t *testing.T) {
	got, err := AssembleHyperDXReplay(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.EventCount != 0 || got.RowCount != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestAssembleHyperDXReplay_SizeLimit(t *testing.T) {
	// Build an event just over the assembled-bytes cap via many medium events.
	const n = 60
	body := `{"type":3,"data":{"pad":"` + strings.Repeat("x", maxAssembledReplayBytes/n) + `"},"timestamp":1}`
	rows := make([]ReplayChunkRow, 0, n)
	for i := range n {
		rows = append(rows, ReplayChunkRow{
			Body:        strings.Replace(body, `"timestamp":1`, `"timestamp":`+strconv.Itoa(i+1), 1),
			EventID:     i + 1,
			Chunk:       1,
			TotalChunks: 1,
			Offset:      i + 1,
		})
	}
	got, err := AssembleHyperDXReplay(rows)
	if err != nil {
		t.Fatal(err)
	}
	if got.EventCount >= n {
		t.Fatalf("expected size-limit drops, EventCount=%d", got.EventCount)
	}
	found := false
	for _, d := range got.Drops {
		if d.Reason == ReplayDropSizeLimit {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected size-limit drop, drops=%+v", got.Drops)
	}
}

func TestAssembleHyperDXReplay_SanitizedFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "hyperdx_replay", "sanitized_chunks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Rows []struct {
			Body        string `json:"body"`
			EventID     int    `json:"eventId"`
			Chunk       int    `json:"chunk"`
			TotalChunks int    `json:"totalChunks"`
			Offset      int    `json:"offset"`
			TimestampMs int64  `json:"timestampMs"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}

	// Split the FullSnapshot across chunks 1+2 like the HyperDX recorder.
	full := `{"type":2,"data":{"node":{"type":0,"childNodes":[{"type":2,"tagName":"div","attributes":{"id":"root"},"childNodes":[]}]}},"timestamp":1010}`
	mid := len(full) / 2
	rows := make([]ReplayChunkRow, 0, len(fixture.Rows))
	for _, r := range fixture.Rows {
		body := r.Body
		if r.EventID == 2 && r.Chunk == 1 {
			body = full[:mid]
		}
		if r.EventID == 2 && r.Chunk == 2 {
			body = full[mid:]
		}
		rows = append(rows, ReplayChunkRow{
			Body: body, EventID: r.EventID, Chunk: r.Chunk,
			TotalChunks: r.TotalChunks, Offset: r.Offset, TimestampMs: r.TimestampMs,
		})
	}

	got, err := AssembleHyperDXReplay(rows)
	if err != nil {
		t.Fatal(err)
	}
	if got.EventCount != 3 {
		t.Fatalf("EventCount=%d drops=%+v", got.EventCount, got.Drops)
	}
	var types []int
	for _, e := range got.Events {
		var m struct {
			Type int `json:"type"`
		}
		if err := json.Unmarshal(e, &m); err != nil {
			t.Fatal(err)
		}
		types = append(types, m.Type)
	}
	if types[0] != 4 || types[1] != 2 || types[2] != 3 {
		t.Fatalf("types=%v want [4 2 3]", types)
	}
}

func TestParseReplayInt(t *testing.T) {
	cases := []struct {
		in   any
		want int
	}{
		{nil, 0},
		{"", 0},
		{"12", 12},
		{12.0, 12},
		{int64(7), 7},
		{json.Number("9"), 9},
	}
	for _, c := range cases {
		if got := parseReplayInt(c.in); got != c.want {
			t.Errorf("parseReplayInt(%v)=%d want %d", c.in, got, c.want)
		}
	}
}
