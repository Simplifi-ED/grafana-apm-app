package plugin

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"
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
	step time.Duration,
	withSeries bool,
	filterNamespace, filterEnvironment, filterServices string,
) []queries.ServiceSummary {
	logger := log.DefaultLogger.With("handler", "services-clickhouse")
	if ch == nil {
		return nil
	}
	window := chquery.WindowSeconds(from, to)
	opts := queries.ClickHouseQueryOptions{From: from, To: to, Limit: 500}

	// All-kind discover (CLIENT-only rows included) + SERVER RED + CLIENT RED fallback.
	discoverRes, err := ch.Query(ctx, chquery.DiscoverInventorySQL(from, to, filterNamespace, filterEnvironment), opts)
	if err != nil {
		logger.Warn("inventory discover query failed", "error", err)
		return nil
	}
	serverCalls, err := ch.Query(ctx, chquery.InventorySQL(from, to, filterNamespace, filterEnvironment), opts)
	if err != nil {
		logger.Warn("inventory SERVER calls query failed", "error", err)
		serverCalls = &queries.ClickHouseResult{}
	}
	clientCalls, err := ch.Query(ctx, chquery.InventoryClientSQL(from, to, filterNamespace, filterEnvironment), opts)
	if err != nil {
		logger.Warn("inventory CLIENT calls query failed", "error", err)
		clientCalls = &queries.ClickHouseResult{}
	}
	serverLat, err := ch.Query(ctx, chquery.InventoryHistogramLatencySQL(from, to, filterNamespace, filterEnvironment, "", chquery.SpanKindServer), opts)
	if err != nil {
		logger.Warn("inventory SERVER histogram latency failed", "error", err)
		serverLat = &queries.ClickHouseResult{}
	}
	clientLat, err := ch.Query(ctx, chquery.InventoryHistogramLatencySQL(from, to, filterNamespace, filterEnvironment, "", chquery.SpanKindClient), opts)
	if err != nil {
		logger.Warn("inventory CLIENT histogram latency failed", "error", err)
		clientLat = &queries.ClickHouseResult{}
	}

	frontendNames := map[string]bool{}
	if frontRes, ferr := ch.Query(ctx, chquery.FrontendServiceNamesSQL(from, to, filterEnvironment), opts); ferr != nil {
		logger.Warn("inventory hasFrontend probe failed", "error", ferr)
	} else {
		frontendNames = chquery.ParseFrontendServiceNames(frontRes.Maps())
	}

	summaries := chquery.MergeInventory(
		discoverRes.Maps(),
		serverCalls.Maps(),
		clientCalls.Maps(),
		serverLat.Maps(),
		clientLat.Maps(),
		nil,
		window,
	)
	chquery.MarkHasFrontend(summaries, frontendNames, func(inventory string) string {
		if scrape, ok := a.ResolveServiceAlias(inventory); ok {
			return scrape
		}
		return ""
	})
	for i := range summaries {
		summaries[i].IsSidecar = isSidecar(summaries[i].Name)
	}
	if !withSeries || len(summaries) == 0 {
		return summaries
	}

	seriesFrom, seriesTo := chquery.SparklineBounds(from, to)
	stepSec := int64(step.Seconds())
	if stepSec < 1 {
		stepSec = 60
	}
	// When the UI asks for sparklines for the visible page only, restrict series
	// SQL to those service names (avoids fleet-wide range scans).
	seriesOpts := queries.ClickHouseQueryOptions{From: seriesFrom, To: seriesTo, Limit: 5000}
	callSeries, err := ch.Query(ctx, chquery.InventoryCallsSeriesSQL(seriesFrom, seriesTo, stepSec, filterNamespace, filterEnvironment, filterServices), seriesOpts)
	if err != nil {
		logger.Warn("inventory SERVER rate/error series failed", "error", err)
		return summaries
	}
	durSeries, err := ch.Query(ctx, chquery.InventoryDurationSeriesSQL(seriesFrom, seriesTo, stepSec, filterNamespace, filterEnvironment, filterServices), seriesOpts)
	if err != nil {
		logger.Warn("inventory SERVER duration series failed", "error", err)
		durSeries = &queries.ClickHouseResult{}
	}
	chquery.AttachInventorySeries(summaries, callSeries.Maps(), durSeries.Maps())

	// CLIENT sparklines only for rows that still lack SERVER series.
	needClient := false
	for _, s := range summaries {
		if !s.HasServerSpans {
			needClient = true
			break
		}
	}
	if needClient {
		clientCallSeries, cerr := ch.Query(ctx, chquery.InventoryClientCallsSeriesSQL(seriesFrom, seriesTo, stepSec, filterNamespace, filterEnvironment, filterServices), seriesOpts)
		if cerr != nil {
			logger.Warn("inventory CLIENT rate/error series failed", "error", cerr)
		} else {
			clientDurSeries, derr := ch.Query(ctx, chquery.InventoryClientDurationSeriesSQL(seriesFrom, seriesTo, stepSec, filterNamespace, filterEnvironment, filterServices), seriesOpts)
			if derr != nil {
				logger.Warn("inventory CLIENT duration series failed", "error", derr)
				clientDurSeries = &queries.ClickHouseResult{}
			}
			chquery.AttachInventorySeriesSelective(summaries, clientCallSeries.Maps(), clientDurSeries.Maps(), false)
		}
	}
	return summaries
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
	browserName, ok := a.ResolveServiceAlias(service)
	if !ok {
		return FrontendMetricsResponse{
			Available: false,
			Source:    "clickhouse",
			Hint:      AliasMissingHint(service),
		}
	}
	if ch == nil {
		return FrontendMetricsResponse{Available: false, Source: "clickhouse"}
	}
	opts := queries.ClickHouseQueryOptions{From: from, To: to, Limit: 1, SkipLimitEnforcement: true}

	probe, err := ch.Query(ctx, chquery.BrowserTelemetryProbeSQL(browserName, environment, from, to), opts)
	if err != nil {
		logger.Warn("browser telemetry probe failed", "error", err)
		return FrontendMetricsResponse{Available: false, Source: "clickhouse"}
	}
	if probe.RowCount() == 0 || chquery.ParseFloat(probe.Maps()[0]["c"]) < 1 {
		return FrontendMetricsResponse{Available: false, Source: "clickhouse"}
	}

	vitalsRes, err := ch.Query(ctx, chquery.WebVitalsSQL(browserName, environment, from, to), opts)
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

	type nodeAgg struct {
		inRate   float64
		errRateW float64 // weighted error numerator (rate * errRate)
		callers  map[string]struct{}
	}
	aggs := make(map[string]*nodeAgg)
	clientSet := make(map[string]struct{})
	ensure := func(name string) *nodeAgg {
		if a, ok := aggs[name]; ok {
			return a
		}
		a := &nodeAgg{callers: make(map[string]struct{})}
		aggs[name] = a
		return a
	}

	var edges []ServiceMapEdge
	for _, e := range parsed {
		clientSet[e.Client] = struct{}{}
		ensure(e.Client)
		srv := ensure(e.Server)
		srv.inRate += e.Rate
		srv.errRateW += e.Rate * e.ErrorRate
		srv.callers[e.Client] = struct{}{}
		edges = append(edges, ServiceMapEdge{
			ID:            e.Client + "->" + e.Server,
			Source:        e.Client,
			Target:        e.Server,
			MainStat:      fmt.Sprintf("%.1f req/s", e.Rate),
			SecondaryStat: fmt.Sprintf("%.1f%% err", e.ErrorRate*100),
		})
	}

	nodes := make([]ServiceMapNode, 0, len(aggs))
	for name, agg := range aggs {
		errPct := 0.0
		if agg.inRate > 0 {
			errPct = agg.errRateW / agg.inRate
			if errPct > 1 {
				errPct = 1
			}
		}
		nType, subtitle := inferClickStackNodeType(name, clientSet)
		main := ""
		sec := ""
		if agg.inRate > 0 {
			main = fmt.Sprintf("%.1f req/s", agg.inRate)
			sec = fmt.Sprintf("%.1f%% err", errPct*100)
		}
		nodes = append(nodes, ServiceMapNode{
			ID:            name,
			Title:         name,
			SubTitle:      subtitle,
			MainStat:      main,
			SecondaryStat: sec,
			ArcErrors:     errPct,
			ArcOK:         1 - errPct,
			NodeType:      nType,
			CallerCount:   len(agg.callers),
			ErrorRate:     errPct,
		})
	}
	return ServiceMapResponse{Nodes: nodes, Edges: edges}
}

// inferClickStackNodeType classifies peers from service-graph names when
// connection_type dimensions are absent (common in ClickStack).
func inferClickStackNodeType(name string, clients map[string]struct{}) (nodeType, subtitle string) {
	lower := strings.ToLower(name)
	_, isClient := clients[name]
	isInfraName := false
	switch {
	case strings.Contains(lower, "postgres"), strings.Contains(lower, "postgresql"),
		strings.Contains(lower, "mysql"), strings.Contains(lower, "mariadb"),
		strings.Contains(lower, "mongo"), strings.Contains(lower, "redis"),
		strings.Contains(lower, "valkey"), strings.Contains(lower, "clickhouse"),
		strings.Contains(lower, "elastic"), strings.Contains(lower, "cassandra"):
		isInfraName = true
		nodeType, subtitle = "database", name
	case strings.Contains(lower, "kafka"), strings.Contains(lower, "rabbit"),
		strings.Contains(lower, "nats"), strings.Contains(lower, "pubsub"),
		strings.Contains(lower, "sqs"), strings.Contains(lower, "queue"):
		isInfraName = true
		nodeType, subtitle = "messaging", name
	case lower == "user" || lower == "unknown" || lower == "client" || lower == "world":
		return "external", ""
	default:
		return "service", ""
	}
	// Nodes that also initiate calls are real services, not pure infra.
	if isClient && isInfraName {
		return "service", ""
	}
	return nodeType, subtitle
}

func (a *App) fetchLogPatternsClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	service, environment string,
	from, to time.Time,
) LogPatternsResponse {
	if ch == nil {
		return LogPatternsResponse{Mode: "unavailable", Patterns: []LogPattern{}, Note: "ClickHouse datasource not configured"}
	}
	opts := queries.ClickHouseQueryOptions{From: from, To: to, Limit: 50}
	severity := "error,fatal,critical"
	res, err := ch.Query(ctx, chquery.LogPatternSQLFiltered(service, from, to, environment, severity, 50), opts)
	if err != nil {
		log.DefaultLogger.Warn("log patterns CH query failed", "error", err)
		return LogPatternsResponse{Mode: "unavailable", Patterns: []LogPattern{}, Note: "clickhouse query failed"}
	}
	parsed := chquery.ParseLogPatternMaps(res.Maps())
	span := to.Sub(from)
	previous := make(map[string]struct{})
	if span > 0 {
		prevRes, prevErr := ch.Query(
			ctx,
			chquery.LogPatternSQLFiltered(service, from.Add(-span), from, environment, severity, 50),
			queries.ClickHouseQueryOptions{From: from.Add(-span), To: from, Limit: 50},
		)
		if prevErr == nil {
			for _, pattern := range chquery.ParseLogPatternMaps(prevRes.Maps()) {
				previous[pattern.Pattern] = struct{}{}
			}
		}
	}
	out := make([]LogPattern, 0, len(parsed))
	for _, p := range parsed {
		out = append(out, LogPattern{
			Pattern:       p.Pattern,
			Count:         p.Count,
			Level:         p.Severity,
			FirstSeenMs:   p.FirstSeenMs,
			LastSeenMs:    p.LastSeenMs,
			IsNew:         len(previous) > 0 && !patternSeen(previous, p.Pattern),
			FilterLiteral: longestLiteral(p.Pattern),
		})
	}
	return LogPatternsResponse{Mode: "sampled", Patterns: out, Note: "ClickHouse error-log clustering"}
}

func (a *App) fetchTraceBreakdownClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	service, environment, dimension string,
	from, to time.Time,
) TraceBreakdownResponse {
	opts := queries.ClickHouseQueryOptions{From: from, To: to, Limit: 100}
	res, err := ch.Query(ctx, chquery.TraceBreakdownSQLForDimension(service, from, to, environment, dimension, 100), opts)
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
			Value:     r.Value,
			Rate:      float64(r.Spans) / window,
			ErrorRate: errRate,
			P95Ms:     r.P95Ms,
			P99Ms:     r.P99Ms,
		})
	}
	return TraceBreakdownResponse{
		Mode:       "spanmetrics",
		Dimension:  dimension,
		Dimensions: []string{"name", "http.route", "http.status_code", "db.system", "db.operation", "messaging.system", "rpc.method"},
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
	if ch == nil {
		return IssuesResponse{
			FingerprintVersion: fingerprint.Version,
			Sources:            IssueSources{},
			Issues:             []Issue{},
			Unavailable:        true,
		}
	}
	// Server exception spans/logs use inventory ServiceName (spanmetrics identity).
	// Browser Issues (letter F) must ResolveServiceAlias before querying HyperDX.
	opts := queries.ClickHouseQueryOptions{From: from, To: to, Limit: 50}
	logRes, logErr := ch.Query(ctx, chquery.ExceptionGroupsSQL(service, from, to, environment, 50), opts)
	if logErr != nil {
		log.DefaultLogger.Warn("issues CH log query failed", "error", logErr)
	}
	spanRes, spanErr := ch.Query(ctx, chquery.ErrorSpansSQL(service, from, to, environment), opts)
	if spanErr != nil {
		log.DefaultLogger.Warn("issues CH span query failed", "error", spanErr)
	}

	byFingerprint := make(map[string]*Issue)
	add := func(event fingerprint.Event, count float64, issueType, member string, sourceTitle string) {
		result := fingerprint.Compute(event)
		issue, ok := byFingerprint[result.Value]
		if !ok {
			issue = &Issue{
				Fingerprint:  result.Value,
				Tier:         int(result.Tier),
				Title:        result.Title,
				Types:        []string{},
				MemberHashes: []string{},
				Source:       issueSourceServer,
			}
			byFingerprint[result.Value] = issue
		}
		issue.Count += count
		if issueType != "" && !containsString(issue.Types, issueType) {
			issue.Types = append(issue.Types, issueType)
		}
		if member != "" && !containsString(issue.MemberHashes, member) {
			issue.MemberHashes = append(issue.MemberHashes, member)
		}
		if issue.Title == "" && sourceTitle != "" {
			issue.Title = sourceTitle
		}
	}

	if logErr == nil {
		for _, row := range chquery.ParseExceptionGroupsMaps(logRes.Maps()) {
			add(
				fingerprint.Event{Type: row.ExceptionType, Value: row.Message},
				float64(row.Occurrences),
				row.ExceptionType,
				"ch:log:"+row.Fingerprint,
				row.Message,
			)
		}
	}
	if spanErr == nil {
		for _, row := range spanRes.Maps() {
			name := chquery.ParseString(row["span_name"])
			message := chquery.ParseString(row["status_message"])
			add(
				fingerprint.Event{Type: name, Value: message},
				chquery.ParseFloat(row["errors"]),
				name,
				"ch:span:"+name,
				name,
			)
		}
	}

	issues := make([]Issue, 0, len(byFingerprint))
	for _, issue := range byFingerprint {
		issues = append(issues, *issue)
	}
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Count != issues[j].Count {
			return issues[i].Count > issues[j].Count
		}
		return issues[i].Fingerprint < issues[j].Fingerprint
	})
	if len(issues) > maxGroups {
		issues = issues[:maxGroups]
	}
	return IssuesResponse{
		FingerprintVersion: fingerprint.Version,
		Sources:            IssueSources{ServerLogs: logErr == nil || spanErr == nil},
		Issues:             issues,
		Unavailable:        logErr != nil && spanErr != nil,
	}
}

func (a *App) fetchFrontendSessionsClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	service, environment string,
	from, to time.Time,
) FrontendSessionsResponse {
	browserName, ok := a.ResolveServiceAlias(service)
	if !ok {
		return FrontendSessionsResponse{
			Sessions: []SessionSummary{},
			Hint:     AliasMissingHint(service),
		}
	}
	// Reuse recordings list: HyperDX sessions with rum.sessionId.
	table, err := queries.QualifiedTable(queries.CHDatabaseDefault, queries.CHTableSessions)
	if err != nil {
		return FrontendSessionsResponse{Sessions: []SessionSummary{}, Unavailable: true}
	}
	where := fmt.Sprintf("ServiceName = %s AND ResourceAttributes['rum.sessionId'] != ''", queries.QuoteClickHouseString(browserName))
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
  AND %s
GROUP BY session_id
ORDER BY last_ms DESC
LIMIT 50`, table, where, chquery.TimestampFilter(from, to))

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

func containsString(values []string, want string) bool {
	return slices.Contains(values, want)
}

func patternSeen(patterns map[string]struct{}, pattern string) bool {
	_, ok := patterns[pattern]
	return ok
}
