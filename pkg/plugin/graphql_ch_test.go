package plugin

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/chquery"
	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

func TestHandleGraphQL_ClickStackHintWhenEmpty(t *testing.T) {
	ch := chHandlerClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		// Metric discover + span detect both empty.
		if strings.Contains(string(body), "MetricName") {
			_, _ = w.Write([]byte(`{"results":{"A":{"frames":[{
				"schema":{"fields":[{"name":"MetricName","type":"string"},{"name":"kind","type":"string"}]},
				"data":{"values":[[],[]]}
			}]}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"results":{"A":{"frames":[{
			"schema":{"fields":[{"name":"c","type":"number"}]},
			"data":{"values":[[0]]}
		}]}}}`))
	})
	app := &App{
		settings: queries.PluginSettings{
			ClickHouseDataSource: queries.DataSourceRef{UID: "clickhouse"},
		},
		chClient: ch,
	}
	req := httptest.NewRequest(http.MethodGet, "/services/demo/graphql-demo/graphql?from=1000&to=2000&environment=ci", nil)
	req.SetPathValue("namespace", "demo")
	req.SetPathValue("service", "graphql-demo")
	w := httptest.NewRecorder()
	app.handleGraphQLMetrics(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp GraphQLMetricsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Detected {
		t.Fatal("expected detected=false")
	}
	if resp.Hint == "" || !strings.Contains(resp.Hint, "ticket 16") {
		t.Fatalf("hint=%q", resp.Hint)
	}
	if len(resp.Operations) != 0 {
		t.Fatalf("no fake zeros: %+v", resp.Operations)
	}
}

func TestQueryGraphQLMetricsClickHouse_SpansFromFixtureShape(t *testing.T) {
	ch := chHandlerClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		sql := string(body)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(sql, "MetricName"):
			_, _ = w.Write([]byte(`{"results":{"A":{"frames":[{
				"schema":{"fields":[{"name":"MetricName","type":"string"},{"name":"kind","type":"string"}]},
				"data":{"values":[[],[]]}
			}]}}}`))
		case strings.Contains(sql, "count() AS c") && strings.Contains(sql, "graphql.operation.name"):
			if !strings.Contains(sql, "graphql-demo") {
				t.Fatalf("span detect must scope service: %s", sql)
			}
			_, _ = w.Write([]byte(`{"results":{"A":{"frames":[{
				"schema":{"fields":[{"name":"c","type":"number"}]},
				"data":{"values":[[2]]}
			}]}}}`))
		case strings.Contains(sql, "avg_latency_s"):
			for _, want := range []string{
				"graphql.operation.name",
				"graphql.operation.type",
				"graphql.document",
				"Duration / 1e9",
			} {
				if !strings.Contains(sql, want) {
					t.Fatalf("span ops missing %q in %s", want, sql)
				}
			}
			_, _ = w.Write([]byte(`{"results":{"A":{"frames":[{
				"schema":{"fields":[
					{"name":"op_name","type":"string"},
					{"name":"op_type","type":"string"},
					{"name":"rate","type":"number"},
					{"name":"avg_latency_s","type":"number"},
					{"name":"error_frac","type":"number"}
				]},
				"data":{"values":[
					["GetInvoice","CreateInvoice"],
					["query","mutation"],
					[0.003,0.003],
					[0.0125,0.042],
					[0,1]
				]}
			}]}}}`))
		default:
			t.Fatalf("unexpected SQL: %s", sql)
		}
	})
	app := &App{chClient: ch}
	from, to := time.Unix(1, 0), time.Unix(3600, 0)
	resp := app.queryGraphQLMetricsClickHouse(t.Context(), ch, "demo", "graphql-demo", "ci", from, to)
	if !resp.Detected || resp.Source != "spans" || resp.Framework != "OpenTelemetry spans" {
		t.Fatalf("resp=%+v", resp)
	}
	if resp.Hint != "" {
		t.Fatalf("unexpected hint: %q", resp.Hint)
	}
	if len(resp.Operations) != 2 {
		t.Fatalf("ops=%+v", resp.Operations)
	}
	if resp.Operations[0].Name != "GetInvoice" || resp.Operations[0].Type != "query" {
		t.Fatalf("op0=%+v", resp.Operations[0])
	}
	if resp.Operations[1].ErrorRate == nil || *resp.Operations[1].ErrorRate != 100 {
		t.Fatalf("CreateInvoice errorRate=%v", resp.Operations[1].ErrorRate)
	}
}

func TestQueryGraphQLMetricsClickHouse_MetricsProbeWins(t *testing.T) {
	ch := chHandlerClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		sql := string(body)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(sql, "UNION ALL") && strings.Contains(sql, "MetricName"):
			_, _ = w.Write([]byte(`{"results":{"A":{"frames":[{
				"schema":{"fields":[{"name":"MetricName","type":"string"},{"name":"kind","type":"string"}]},
				"data":{"values":[
					["graphql_request_seconds_count","graphql_request_seconds_sum"],
					["sum","sum"]
				]}
			}]}}}`))
		case strings.Contains(sql, "graphql_request_seconds_count") && strings.Contains(sql, "count_delta"):
			_, _ = w.Write([]byte(`{"results":{"A":{"frames":[{
				"schema":{"fields":[
					{"name":"op_name","type":"string"},
					{"name":"op_type","type":"string"},
					{"name":"rate","type":"number"},
					{"name":"avg_latency","type":"number"},
					{"name":"error_frac","type":"number"}
				]},
				"data":{"values":[
					["QUERY"],
					[""],
					[1.5],
					[0.02],
					[0.1]
				]}
			}]}}}`))
		default:
			t.Fatalf("unexpected SQL: %s", sql)
		}
	})
	app := &App{chClient: ch}
	from, to := time.Unix(1, 0), time.Unix(301, 0)
	resp := app.queryGraphQLMetricsClickHouse(t.Context(), ch, "demo", "svc", "", from, to)
	if !resp.Detected || resp.Source != "metrics" || resp.Framework != frameworkDGS {
		t.Fatalf("resp=%+v", resp)
	}
	if len(resp.Operations) != 1 || resp.Operations[0].Name != "QUERY" {
		t.Fatalf("ops=%+v", resp.Operations)
	}
	if resp.Operations[0].ErrorRate == nil || *resp.Operations[0].ErrorRate != 10 {
		t.Fatalf("errorRate=%v", resp.Operations[0].ErrorRate)
	}
}

func TestMapCHGraphQLOps(t *testing.T) {
	ops := mapCHGraphQLOps([]chquery.GraphQLOpRow{
		{Name: "A", Type: "query", Rate: 1, AvgLatency: 0.01, LatencyUnit: "s", ErrorRate: 5, HasErrorRate: true},
	})
	if len(ops) != 1 || ops[0].ErrorRate == nil || *ops[0].ErrorRate != 5 {
		t.Fatalf("%+v", ops)
	}
}
