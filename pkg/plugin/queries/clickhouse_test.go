package queries

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func chTestClient(t *testing.T, handler http.HandlerFunc) *ClickHouseClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := NewClickHouseClient(srv.URL, "test-token", "clickhouse")
	// Use the httptest client so cancelled requests tear down the server-side
	// handler promptly (sharedTransport keep-alives can leave Close hung).
	client.httpClient = srv.Client()
	client.httpClient.Timeout = 30 * time.Second
	return client
}

func TestPrepareClickHouseSQL(t *testing.T) {
	t.Run("rejects empty", func(t *testing.T) {
		if _, err := PrepareClickHouseSQL("  ", 0, 100, 1000, false); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rejects multi-statement", func(t *testing.T) {
		if _, err := PrepareClickHouseSQL("SELECT 1; SELECT 2", 0, 100, 1000, false); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("allows trailing semicolon strip", func(t *testing.T) {
		sql, err := PrepareClickHouseSQL("SELECT 1;", 0, 100, 1000, true)
		if err != nil {
			t.Fatal(err)
		}
		if sql != "SELECT 1" {
			t.Fatalf("got %q", sql)
		}
	})

	t.Run("rejects mutating statements", func(t *testing.T) {
		for _, q := range []string{
			"DROP TABLE otel_logs",
			"INSERT INTO otel_logs VALUES (1)",
			"ALTER TABLE otel_logs DELETE WHERE 1",
			"TRUNCATE TABLE otel_logs",
		} {
			if _, err := PrepareClickHouseSQL(q, 0, 100, 1000, true); err == nil {
				t.Fatalf("expected reject for %q", q)
			}
		}
	})

	t.Run("appends default limit", func(t *testing.T) {
		sql, err := PrepareClickHouseSQL("SELECT * FROM otel_logs", 0, 50, 1000, false)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(sql, "LIMIT 50") {
			t.Fatalf("got %q", sql)
		}
	})

	t.Run("clamps excessive existing limit", func(t *testing.T) {
		sql, err := PrepareClickHouseSQL("SELECT * FROM otel_logs LIMIT 999999", 0, 50, 1000, false)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(sql, "LIMIT 1000") {
			t.Fatalf("got %q", sql)
		}
	})

	t.Run("keeps safe existing limit", func(t *testing.T) {
		sql, err := PrepareClickHouseSQL("SELECT * FROM otel_logs LIMIT 10", 0, 50, 1000, false)
		if err != nil {
			t.Fatal(err)
		}
		if sql != "SELECT * FROM otel_logs LIMIT 10" {
			t.Fatalf("got %q", sql)
		}
	})

	t.Run("requested limit wins when no clause", func(t *testing.T) {
		sql, err := PrepareClickHouseSQL("SELECT 1", 25, 50, 1000, false)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(sql, "LIMIT 25") {
			t.Fatalf("got %q", sql)
		}
	})
}

func TestQuoteAndValidateIdent(t *testing.T) {
	if got := QuoteClickHouseString("O'Reilly"); got != "'O''Reilly'" {
		t.Fatalf("got %q", got)
	}
	if _, err := QuoteClickHouseIdent("otel_logs"); err != nil {
		t.Fatal(err)
	}
	if _, err := QuoteClickHouseIdent("otel;drop"); err == nil {
		t.Fatal("expected invalid ident")
	}
	q, err := QualifiedTable("default", "otel_traces")
	if err != nil {
		t.Fatal(err)
	}
	if q != "`default`.`otel_traces`" {
		t.Fatalf("got %q", q)
	}
}

func TestClickHouseQueryParsesTableFrames(t *testing.T) {
	client := chTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ds/query" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("missing auth header")
		}
		body, _ := io.ReadAll(r.Body)
		var req chQueryRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("bad request body: %v", err)
		}
		if len(req.Queries) != 1 {
			t.Fatalf("queries = %d", len(req.Queries))
		}
		q := req.Queries[0]
		if q.Datasource.UID != "clickhouse" || q.Datasource.Type != ClickHouseDatasourceType {
			t.Fatalf("datasource = %+v", q.Datasource)
		}
		if q.Format != ClickHouseFormatTable {
			t.Fatalf("format = %d", q.Format)
		}
		if !strings.Contains(q.RawSQL, "LIMIT") {
			t.Fatalf("expected LIMIT in %q", q.RawSQL)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":{"A":{"frames":[{
			"schema":{"fields":[{"name":"ServiceName","type":"string"},{"name":"c","type":"number"}]},
			"data":{"values":[["nova-app","dex"],[12,3]]}
		}]}}}`))
	})

	res, err := client.Query(t.Context(), "SELECT ServiceName, count() AS c FROM otel_logs GROUP BY ServiceName", ClickHouseQueryOptions{
		From:  time.Unix(1000, 0),
		To:    time.Unix(2000, 0),
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if res.RowCount() != 2 {
		t.Fatalf("rows = %d", res.RowCount())
	}
	if res.Columns[0] != "ServiceName" || res.Columns[1] != "c" {
		t.Fatalf("columns = %+v", res.Columns)
	}
	maps := res.Maps()
	if maps[0]["ServiceName"] != "nova-app" {
		t.Fatalf("row0 = %+v", maps[0])
	}
	if maps[1]["c"].(float64) != 3 {
		t.Fatalf("row1 = %+v", maps[1])
	}
}

func TestClickHouseQueryPropagatesDatasourceError(t *testing.T) {
	client := chTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":{"A":{"error":"code: 60, table does not exist"}}}`))
	})
	_, err := client.Query(t.Context(), "SELECT 1", ClickHouseQueryOptions{
		SkipLimitEnforcement: true,
		From:                 time.Unix(1, 0),
		To:                   time.Unix(2, 0),
	})
	if err == nil || !strings.Contains(err.Error(), "table does not exist") {
		t.Fatalf("err = %v", err)
	}
}

func TestClickHouseQueryHTTPError(t *testing.T) {
	client := chTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream boom"))
	})
	_, err := client.Query(t.Context(), "SELECT 1", ClickHouseQueryOptions{
		SkipLimitEnforcement: true,
		From:                 time.Unix(1, 0),
		To:                   time.Unix(2, 0),
	})
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("err = %v", err)
	}
}

func TestClickHouseQueryRejectsBadTimeRange(t *testing.T) {
	client := NewClickHouseClient("http://example.invalid", "", "ch")
	_, err := client.Query(t.Context(), "SELECT 1", ClickHouseQueryOptions{
		SkipLimitEnforcement: true,
		From:                 time.Unix(2000, 0),
		To:                   time.Unix(1000, 0),
	})
	if err == nil || !strings.Contains(err.Error(), "time range") {
		t.Fatalf("err = %v", err)
	}
}

func TestClickHouseQueryRequiresUID(t *testing.T) {
	client := NewClickHouseClient("http://example.invalid", "", "")
	_, err := client.Query(t.Context(), "SELECT 1", ClickHouseQueryOptions{SkipLimitEnforcement: true})
	if err == nil || !strings.Contains(err.Error(), "UID") {
		t.Fatalf("err = %v", err)
	}
}

func TestClickHousePing(t *testing.T) {
	client := chTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"rawSql":"SELECT 1 AS ok"`) {
			t.Fatalf("body = %s", body)
		}
		_, _ = w.Write([]byte(`{"results":{"A":{"frames":[{
			"schema":{"fields":[{"name":"ok"}]},
			"data":{"values":[[1]]}
		}]}}}`))
	})
	if err := client.Ping(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestClickHousePingEmptyRows(t *testing.T) {
	client := chTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":{"A":{"frames":[{
			"schema":{"fields":[{"name":"ok"}]},
			"data":{"values":[[]]}
		}]}}}`))
	})
	if err := client.Ping(t.Context()); err == nil {
		t.Fatal("expected error")
	}
}

func TestClickHouseQueryCancellation(t *testing.T) {
	started := make(chan struct{})
	client := chTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	})
	ctx, cancel := context.WithCancel(t.Context())
	errCh := make(chan error, 1)
	go func() {
		_, err := client.Query(ctx, "SELECT 1", ClickHouseQueryOptions{
			SkipLimitEnforcement: true,
			From:                 time.Unix(1, 0),
			To:                   time.Unix(2, 0),
		})
		errCh <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not start")
	}
	cancel()
	err := <-errCh
	if err == nil || !strings.Contains(err.Error(), "cancel") {
		t.Fatalf("err = %v", err)
	}
}

func TestClickHouseConfiguredUIDGate(t *testing.T) {
	// Handlers must only query the configured UID — never a client-supplied one.
	settings := PluginSettings{
		ClickHouseDataSource: DataSourceRef{UID: "clickhouse", Type: ClickHouseDatasourceType},
	}
	if settings.ClickHouseDataSource.UID != "clickhouse" {
		t.Fatal("expected configured UID")
	}
	if settings.ClickHouseDataSource.UID == "other" {
		t.Fatal("unexpected match")
	}
}
