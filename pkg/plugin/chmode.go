package plugin

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/chquery"
	"github.com/nais/grafana-otel-plugin/pkg/plugin/fingerprint"
	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// clickStackMode is true when a ClickHouse datasource is configured. In this
// mode the plugin serves APM from ClickStack tables and does not require
// Mimir/Tempo/Loki.
func (a *App) clickStackMode() bool {
	return a != nil && a.chClient != nil
}

func (a *App) chForRequest(req *http.Request) *queries.ClickHouseClient {
	if !a.clickStackMode() {
		return nil
	}
	return a.clickHouseClient(req)
}

// detectClickStackCapabilities probes verified spanmetrics / service-graph
// metric names in ClickHouse and lists services/environments.
func (a *App) detectClickStackCapabilities(ctx context.Context, ch *queries.ClickHouseClient) queries.Capabilities {
	caps := queries.Capabilities{
		Tempo: DataSourceStatusUnavailable(),
		Loki:  DataSourceStatusUnavailable(),
	}
	if ch == nil {
		st := queries.DataSourceStatus{Available: false, Error: "clickhouse datasource not configured"}
		caps.ClickHouse = &st
		return caps
	}
	if err := ch.Ping(ctx); err != nil {
		st := queries.DataSourceStatus{Available: false, Error: summarizeClickHouseError(err)}
		caps.ClickHouse = &st
		return caps
	}
	st := queries.DataSourceStatus{Available: true}
	caps.ClickHouse = &st

	now := time.Now()
	opts := queries.ClickHouseQueryOptions{From: now.Add(-time.Hour), To: now, SkipLimitEnforcement: true}

	if res, err := ch.Query(ctx, chquery.DetectSpanMetricsSQL(), opts); err == nil && res.RowCount() > 0 {
		c := chquery.ParseFloat(res.Maps()[0]["c"])
		if c > 0 {
			caps.SpanMetrics = queries.SpanMetricsCapability{
				Detected:       true,
				Namespace:      "traces.spanmetrics",
				CallsMetric:    chquery.MetricSpanCalls,
				DurationMetric: chquery.MetricSpanDuration,
				DurationUnit:   "ms",
			}
		}
	}
	if res, err := ch.Query(ctx, chquery.DetectServiceGraphSQL(), opts); err == nil && res.RowCount() > 0 {
		c := chquery.ParseFloat(res.Maps()[0]["c"])
		if c > 0 {
			caps.ServiceGraph = queries.ServiceGraphCapability{
				Detected: true,
				Prefix:   "traces_service_graph",
			}
		}
	}

	dayOpts := queries.ClickHouseQueryOptions{From: now.Add(-24 * time.Hour), To: now, Limit: 500}
	if res, err := ch.Query(ctx, chquery.ListServicesSQL(), dayOpts); err == nil {
		for _, m := range res.Maps() {
			if s := chquery.ParseString(m["service"]); s != "" {
				caps.Services = append(caps.Services, s)
			}
		}
	}
	if res, err := ch.Query(ctx, chquery.ListEnvironmentsSQL(), dayOpts); err == nil {
		for _, m := range res.Maps() {
			if e := chquery.ParseString(m["env"]); e != "" {
				caps.Environments = append(caps.Environments, e)
			}
		}
	}
	return caps
}

// DataSourceStatusUnavailable is a helper for ClickStack-only responses.
func DataSourceStatusUnavailable() queries.DataSourceStatus {
	return queries.DataSourceStatus{Available: false, Error: "not required in ClickStack mode"}
}

func (a *App) fetchServiceSummariesClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	from, to time.Time,
	filterNamespace, filterEnvironment string,
) []queries.ServiceSummary {
	logger := log.DefaultLogger.With("handler", "services-clickhouse")
	window := chquery.WindowSeconds(from, to)
	opts := queries.ClickHouseQueryOptions{From: from, To: to, Limit: 500}

	callRes, err := ch.Query(ctx, chquery.InventorySQL(from, to, filterNamespace, filterEnvironment), opts)
	if err != nil {
		logger.Warn("inventory calls query failed", "error", err)
		return nil
	}
	latRes, err := ch.Query(ctx, chquery.LatencySQL(from, to, filterNamespace, filterEnvironment), opts)
	if err != nil {
		logger.Warn("inventory latency query failed", "error", err)
		latRes = &queries.ClickHouseResult{}
	}
	return chquery.ParseInventoryMaps(callRes.Maps(), latRes.Maps(), window)
}

func (a *App) fetchOperationsClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	service, environment string,
	from, to time.Time,
) []queries.OperationSummary {
	logger := log.DefaultLogger.With("handler", "operations-clickhouse")
	if ch == nil {
		return nil
	}
	window := chquery.WindowSeconds(from, to)
	opts := queries.ClickHouseQueryOptions{From: from, To: to, Limit: 200}

	callRes, err := ch.Query(ctx, chquery.OperationsSQL(service, from, to, environment), opts)
	if err != nil {
		logger.Warn("operations calls query failed", "error", err)
		return nil
	}
	latRes, err := ch.Query(ctx, chquery.OperationLatencySQL(service, from, to, environment), opts)
	if err != nil {
		logger.Warn("operations latency query failed", "error", err)
		latRes = &queries.ClickHouseResult{}
	}
	return chquery.ParseOperations(callRes.Maps(), latRes.Maps(), window)
}

func (a *App) fetchEndpointsClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	service, environment string,
	from, to time.Time,
) queries.EndpointGroups {
	ops := a.fetchOperationsClickHouse(ctx, ch, service, environment, from, to)
	if len(ops) == 0 {
		return emptyEndpointGroups()
	}
	return chquery.OperationsToEndpointGroups(ops)
}

func (a *App) fetchFrontendMetricsClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	service, environment string,
	from, to time.Time,
) FrontendMetricsResponse {
	logger := log.DefaultLogger.With("handler", "frontend-clickhouse")
	if ch == nil {
		return FrontendMetricsResponse{Available: false, Source: "clickhouse"}
	}
	opts := queries.ClickHouseQueryOptions{From: from, To: to, Limit: 1, SkipLimitEnforcement: true}

	probe, err := ch.Query(ctx, chquery.BrowserTelemetryProbeSQL(service, environment, from, to), opts)
	if err != nil {
		logger.Warn("browser telemetry probe failed", "error", err)
		return FrontendMetricsResponse{Available: false, Source: "clickhouse"}
	}
	if probe.RowCount() == 0 || chquery.ParseFloat(probe.Maps()[0]["c"]) < 1 {
		return FrontendMetricsResponse{Available: false, Source: "clickhouse"}
	}

	vitalsRes, err := ch.Query(ctx, chquery.WebVitalsSQL(service, environment, from, to), opts)
	if err != nil {
		logger.Warn("web vitals query failed", "error", err)
		return FrontendMetricsResponse{Available: true, Source: "clickhouse-traces"}
	}
	vitals, _ := chquery.ParseWebVitalsMaps(vitalsRes.Maps())
	return FrontendMetricsResponse{
		Available: true,
		Source:    "clickhouse-traces",
		Vitals:    vitals,
	}
}

func (a *App) fetchHealthClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	service, environment string,
	from, to time.Time,
) HealthSummary {
	ops := a.fetchOperationsClickHouse(ctx, ch, service, environment, from, to)
	summary := HealthSummary{DurationUnit: "ms"}
	if len(ops) == 0 {
		return summary
	}
	var calls, errors, maxP95 float64
	for _, op := range ops {
		// Reconstruct approximate call volume from rate × window.
		window := chquery.WindowSeconds(from, to)
		c := op.Rate * window
		calls += c
		errors += op.ErrorRate * c
		if op.P95Duration > maxP95 {
			maxP95 = op.P95Duration
		}
	}
	if calls > 0 {
		summary.Rate = calls / chquery.WindowSeconds(from, to)
		summary.ErrorRate = errors / calls
	}
	summary.P95Duration = maxP95
	return summary
}

func (a *App) fetchConnectedClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	service, environment string,
	from, to time.Time,
) ConnectedServicesResponse {
	edges := a.serviceGraphEdgesClickHouse(ctx, ch, environment, from, to)
	out := ConnectedServicesResponse{
		Inbound:  []ConnectedService{},
		Outbound: []ConnectedService{},
	}
	for _, e := range edges {
		cs := ConnectedService{
			Name:         "",
			Rate:         e.Rate,
			ErrorRate:    e.ErrorRate,
			DurationUnit: "ms",
		}
		switch {
		case e.Client == service:
			cs.Name = e.Server
			out.Outbound = append(out.Outbound, cs)
		case e.Server == service:
			cs.Name = e.Client
			out.Inbound = append(out.Inbound, cs)
		}
	}
	return out
}

func (a *App) fetchDependenciesClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	service, environment string,
	from, to time.Time,
) []DependencySummary {
	edges := a.serviceGraphEdgesClickHouse(ctx, ch, environment, from, to)
	out := make([]DependencySummary, 0)
	for _, e := range edges {
		if e.Client != service {
			continue
		}
		out = append(out, DependencySummary{
			Name:         e.Server,
			Type:         "service",
			Rate:         e.Rate,
			ErrorRate:    e.ErrorRate,
			DurationUnit: "ms",
			Impact:       e.Rate * e.ErrorRate,
		})
	}
	return out
}

func (a *App) serviceGraphEdgesClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	environment string,
	from, to time.Time,
) []chquery.ServiceGraphEdge {
	if ch == nil {
		return nil
	}
	window := chquery.WindowSeconds(from, to)
	opts := queries.ClickHouseQueryOptions{From: from, To: to, Limit: 500}
	res, err := ch.Query(ctx, chquery.ServiceGraphSQL(from, to, environment), opts)
	if err != nil {
		log.DefaultLogger.Warn("service graph CH query failed", "error", err)
		return nil
	}
	return chquery.ParseServiceGraphMaps(res.Maps(), window)
}

func (a *App) fetchServiceMapClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	from, to time.Time,
	environment string,
) ServiceMapResponse {
	logger := log.DefaultLogger.With("handler", "servicemap-clickhouse")
	window := chquery.WindowSeconds(from, to)
	opts := queries.ClickHouseQueryOptions{From: from, To: to, Limit: 500}

	res, err := ch.Query(ctx, chquery.ServiceGraphSQL(from, to, environment), opts)
	if err != nil {
		logger.Warn("service graph query failed", "error", err)
		return ServiceMapResponse{Nodes: []ServiceMapNode{}, Edges: []ServiceMapEdge{}}
	}
	parsed := chquery.ParseServiceGraphMaps(res.Maps(), window)
	nodeSet := make(map[string]*ServiceMapNode)
	ensure := func(name string) {
		if _, ok := nodeSet[name]; !ok {
			nodeSet[name] = &ServiceMapNode{
				ID:    name,
				Title: name,
			}
		}
	}
	var edges []ServiceMapEdge
	for _, e := range parsed {
		ensure(e.Client)
		ensure(e.Server)
		nodeSet[e.Server].ErrorRate = e.ErrorRate
		edges = append(edges, ServiceMapEdge{
			ID:            e.Client + "->" + e.Server,
			Source:        e.Client,
			Target:        e.Server,
			MainStat:      fmt.Sprintf("%.2f/s", e.Rate),
			SecondaryStat: fmt.Sprintf("%.1f%% err", e.ErrorRate*100),
		})
	}
	nodes := make([]ServiceMapNode, 0, len(nodeSet))
	for _, n := range nodeSet {
		n.ArcErrors = n.ErrorRate
		n.ArcOK = 1 - n.ErrorRate
		nodes = append(nodes, *n)
	}
	return ServiceMapResponse{Nodes: nodes, Edges: edges}
}

func (a *App) fetchLogPatternsClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	service, environment string,
	from, to time.Time,
) LogPatternsResponse {
	opts := queries.ClickHouseQueryOptions{From: from, To: to, Limit: 50}
	res, err := ch.Query(ctx, chquery.LogPatternSQL(service, from, to, environment, 50), opts)
	if err != nil {
		log.DefaultLogger.Warn("log patterns CH query failed", "error", err)
		return LogPatternsResponse{Mode: "unavailable", Patterns: []LogPattern{}, Note: "clickhouse query failed"}
	}
	parsed := chquery.ParseLogPatternMaps(res.Maps())
	out := make([]LogPattern, 0, len(parsed))
	for _, p := range parsed {
		out = append(out, LogPattern{
			Pattern: p.Pattern,
			Count:   p.Count,
			Level:   p.Severity,
		})
	}
	return LogPatternsResponse{Mode: "sampled", Patterns: out, Note: "ClickHouse Body clustering"}
}

func (a *App) fetchTraceBreakdownClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	service, environment string,
	from, to time.Time,
) TraceBreakdownResponse {
	opts := queries.ClickHouseQueryOptions{From: from, To: to, Limit: 100}
	res, err := ch.Query(ctx, chquery.TraceBreakdownSQL(service, from, to, environment, 100), opts)
	if err != nil {
		log.DefaultLogger.Warn("trace breakdown CH query failed", "error", err)
		return TraceBreakdownResponse{Mode: "unavailable", Note: "clickhouse query failed"}
	}
	window := chquery.WindowSeconds(from, to)
	rows := chquery.ParseTraceBreakdownMaps(res.Maps())
	out := make([]TraceBreakdownRow, 0, len(rows))
	for _, r := range rows {
		errRate := 0.0
		if r.Spans > 0 {
			errRate = float64(r.Errors) / float64(r.Spans)
		}
		out = append(out, TraceBreakdownRow{
			Value:     r.SpanName,
			Rate:      float64(r.Spans) / window,
			ErrorRate: errRate,
			P95Ms:     r.P95Ms,
			P99Ms:     r.P95Ms, // p99 not queried separately; reuse p95 until dedicated query
		})
	}
	return TraceBreakdownResponse{
		Mode:       "spanmetrics",
		Dimension:  "span_name",
		Dimensions: []string{"span_name"},
		Rows:       out,
		Note:       "ClickHouse otel_traces aggregates",
	}
}

func (a *App) fetchIssuesClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	service, environment string,
	from, to time.Time,
) IssuesResponse {
	opts := queries.ClickHouseQueryOptions{From: from, To: to, Limit: 50}
	res, err := ch.Query(ctx, chquery.ErrorSpansSQL(service, from, to, environment), opts)
	if err != nil {
		log.DefaultLogger.Warn("issues CH query failed", "error", err)
		return IssuesResponse{
			FingerprintVersion: fingerprint.Version,
			Issues:             []Issue{},
			Unavailable:        true,
			Sources:            IssueSources{},
		}
	}
	issues := make([]Issue, 0, res.RowCount())
	for _, m := range res.Maps() {
		name := chquery.ParseString(m["span_name"])
		msg := chquery.ParseString(m["status_message"])
		title := name
		if msg != "" {
			title = name + ": " + msg
		}
		count := chquery.ParseFloat(m["errors"])
		fp := "ch:span:" + name
		issues = append(issues, Issue{
			ExceptionGroup: ExceptionGroup{
				Fingerprint:  fp,
				Tier:         1,
				Title:        title,
				Types:        []string{name},
				Count:        count,
				MemberHashes: []string{fp},
			},
			Source: "server",
		})
	}
	return IssuesResponse{
		FingerprintVersion: fingerprint.Version,
		Sources:            IssueSources{ServerLogs: true},
		Issues:             issues,
	}
}

func (a *App) fetchFrontendSessionsClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	service, environment string,
	from, to time.Time,
) FrontendSessionsResponse {
	// Reuse recordings list: HyperDX sessions with rum.sessionId.
	table, err := queries.QualifiedTable(queries.CHDatabaseDefault, queries.CHTableSessions)
	if err != nil {
		return FrontendSessionsResponse{Sessions: []SessionSummary{}, Unavailable: true}
	}
	fromMs, toMs := from.UnixMilli(), to.UnixMilli()
	where := fmt.Sprintf("ServiceName = %s AND ResourceAttributes['rum.sessionId'] != ''", queries.QuoteClickHouseString(service))
	if environment != "" {
		where += fmt.Sprintf(" AND ResourceAttributes['cluster'] = %s", queries.QuoteClickHouseString(environment))
	}
	sql := fmt.Sprintf(`
SELECT
  ResourceAttributes['rum.sessionId'] AS session_id,
  toUnixTimestamp64Milli(min(Timestamp)) AS first_ms,
  toUnixTimestamp64Milli(max(Timestamp)) AS last_ms,
  count() AS events
FROM %s
WHERE %s
  AND Timestamp >= fromUnixTimestamp64Milli(%d)
  AND Timestamp <= fromUnixTimestamp64Milli(%d)
GROUP BY session_id
ORDER BY last_ms DESC
LIMIT 50`, table, where, fromMs, toMs)

	res, err := ch.Query(ctx, sql, queries.ClickHouseQueryOptions{From: from, To: to, Limit: 50})
	if err != nil {
		log.DefaultLogger.Warn("frontend sessions CH query failed", "error", err)
		return FrontendSessionsResponse{Sessions: []SessionSummary{}, Unavailable: true}
	}
	out := make([]SessionSummary, 0, res.RowCount())
	for _, m := range res.Maps() {
		out = append(out, SessionSummary{
			SessionID:   chquery.ParseString(m["session_id"]),
			FirstSeenMs: chquery.ParseInt64(m["first_ms"]),
			LastSeenMs:  chquery.ParseInt64(m["last_ms"]),
			Events:      chquery.ParseFloat(m["events"]),
		})
	}
	return FrontendSessionsResponse{Sessions: out}
}
