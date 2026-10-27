# ClickStack session replay vertical slice

**Date:** 2026-10-27  
**Status:** Implemented (unit-tested). End-to-end play-in-Grafana requires a provisioned Grafana ClickHouse datasource UID.

## Goal

Play HyperDX/`@hyperdx/browser` session recordings inside this Grafana app using data already in ClickHouse (`hyperdx_sessions`). No Loki/Tempo/Mimir, no new ingestion, no SDK change.

## Verified storage format

See [schema.md](schema.md) § `hyperdx_sessions`. Summary:

- Body = raw rrweb JSON (or a **string fragment** of that JSON when chunked).
- Chunk metadata in `LogAttributes`: `rr-web.event`, `rr-web.chunk`, `rr-web.total-chunks`, `rr-web.offset`.
- Session id: `ResourceAttributes['rum.sessionId']`.
- **Not** the NAIS Faro Loki format (`event_data_enc=gzip+b64`).

Reassembly matches HyperDX's `rrwebChunkAssembler` (sort chunks by index, concatenate, `JSON.parse`).

## API

| Method | Path | Role |
|--------|------|------|
| GET | `/services/{ns}/{svc}/frontend/sessions/recordings` | List sessions with recordings |
| GET | `/services/{ns}/{svc}/frontend/sessions/{sessionId}/replay/probe` | Existence / mode probe |
| GET | `/services/{ns}/{svc}/frontend/sessions/{sessionId}/replay` | Fetch + reassemble events |

Requires `jsonData.clickHouseDataSource.uid`. Auth = Grafana datasource auth (same as Step 1 client). Limits: 24h range, 5000 rows/fetch, 50 MiB assembled.

## Frontend

- `fetchReplay` / `probeReplay` call the plugin resource API first.
- If the backend returns `unavailable` (no ClickHouse UID), they fall back to the legacy Loki Faro path when `logsUid` is set.
- `ReplayPlayer` / `@grafana/rrweb-player` unchanged.

## Changed files (this slice)

**Backend:** `pkg/plugin/replay.go`, `replay_assemble.go`, `*_test.go`, `app.go` routes  
**Frontend:** `src/pages/tabs/frontend/replay/fetchReplay.ts` (+ tests), `ReplaySection.tsx`, `ExceptionDrawer.tsx` (+ replay test)  
**Docs:** `docs/api/openapi.yaml`, `docs/api/frontend.md`, `schema.md`, this file  
**Config (Step 1):** `clickHouseDataSource` + `queries/clickhouse.go`

## Tests

| Layer | What | How |
|-------|------|-----|
| Unit | Chunk reassembly, duplicates, malformed, size limit | `go test ./pkg/plugin/ -run Assemble` |
| Unit | HTTP handlers + SQL wiring (mocked `/api/ds/query`) | `go test ./pkg/plugin/ -run HandleSession` |
| Unit | Frontend backend path + Loki fallback | `fetchReplay.test.ts`, `ExceptionDrawer.replay.test.tsx` |
| Integration | Live ClickHouse format inspection | ClickStack MCP read-only queries (Phase A) — **not** through Grafana datasource |
| Integration | Live multi-chunk reassembly | CH SQL concat of event `206` chunks → valid JSON `type=2` + `timestamp` (no Body logged) |

Fixtures are **synthetic** (example.test URLs, empty DOM nodes). Do not commit live Bodies.

## Local verification (Grafana → ClickHouse)

1. Provision `grafana-clickhouse-datasource` pointed at the same CH as ClickStack (`provisioning/datasources/clickhouse.yaml.example`).
2. Set plugin `jsonData.clickHouseDataSource.uid` (`provisioning/plugins/apps-clickhouse.yaml.example`).
3. `mise run deploy` (or build backend + restart Grafana).
4. Health: plugin CheckHealth should show ClickHouse OK.
5. Pick a known `ServiceName` + `rum.sessionId` from CH (e.g. via HyperDX UI or MCP).
6. Open Exception Drawer for that service/session **or** call:

```bash
curl -sS -H "Authorization: Bearer $TOKEN" \
  "$GRAFANA/api/plugins/nais-apm-app/resources/services/_/nova-frontend/frontend/sessions/SESSION_ID/replay/probe?from=FROM&to=TO"
```

7. Click **Play replay** — player should mount with ≥2 events (Meta + FullSnapshot).

## Remaining limitations

- Other APM screens still on LGTM path (out of scope).
- Faro/Loki replay remains as frontend fallback only; not ported into the Go handler.
- rrweb recorder (HyperDX) vs `@grafana/rrweb-player` 2.x may skew; retrieval is correct regardless.
- Large sessions may truncate at 5000 rows / 50 MiB — response sets `truncated: true`.
- Environment filter uses `ResourceAttributes['cluster']` (verified); other env attr keys are not applied.
- True E2E “play inside Grafana” still needs the ClickHouse datasource provisioned in the local Grafana stack (not verified in this session’s Grafana process).
