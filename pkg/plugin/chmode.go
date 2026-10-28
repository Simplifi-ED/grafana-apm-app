package plugin

import (
	"context"
	"fmt"
	"net/http"
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
// metric names in ClickHouse and lists services/environments. Pyroscope is
// probed independently (ticket 18 / letter J) so Profiling works in ClickStack
// mode even when ClickHouse is down.
func (a *App) detectClickStackCapabilities(ctx context.Context, ch *queries.ClickHouseClient, headers http.Header, serviceToken string) queries.Capabilities {
	caps := queries.Capabilities{
		Tempo: DataSourceStatusUnavailable(),
		Loki:  DataSourceStatusUnavailable(),
	}
	// Profiling DS detection does not depend on ClickHouse.
	caps.Pyroscope = a.detectPyroscope(ctx, headers, serviceToken)

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
	applyFrameworkFromSDKLanguage(summaries)
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
	// Histogram-primary (ADR 0003); traces quantile is fallback when hist empty.
	latRes, err := ch.Query(ctx, chquery.OperationLatencyHistogramSQL(service, from, to, environment), opts)
	if err != nil {
		logger.Warn("operations histogram latency failed", "error", err)
		latRes = &queries.ClickHouseResult{}
	}
	if latRes.RowCount() == 0 {
		traceLat, terr := ch.Query(ctx, chquery.OperationLatencySQL(service, from, to, environment), opts)
		if terr != nil {
			logger.Warn("operations traces latency fallback failed", "error", terr)
		} else {
			latRes = traceLat
		}
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
	window := chquery.WindowSeconds(from, to)
	var calls, errors, maxP95 float64
	for _, op := range ops {
		c := op.Rate * window
		calls += c
		errors += op.ErrorRate * c
		if op.P95Duration > maxP95 {
			maxP95 = op.P95Duration
		}
	}
	if calls > 0 {
		summary.Rate = roundTo(calls/window, 3)
		summary.ErrorRate = roundTo(errors/calls*100, 1)
	}
	summary.P95Duration = roundTo(maxP95, 2)

	// Previous-period baseline (same window length ending at `from`).
	prevFrom := from.Add(-to.Sub(from))
	prevOps := a.fetchOperationsClickHouse(ctx, ch, service, environment, prevFrom, from)
	var prevCalls, prevErrors, prevMaxP95 float64
	prevByName := make(map[string]queries.OperationSummary, len(prevOps))
	for _, op := range prevOps {
		prevByName[op.SpanName+"\x00"+op.SpanKind] = op
		c := op.Rate * window
		prevCalls += c
		prevErrors += op.ErrorRate * c
		if op.P95Duration > prevMaxP95 {
			prevMaxP95 = op.P95Duration
		}
	}
	if prevCalls > 0 {
		v := roundTo(prevCalls/window, 3)
		summary.PrevRate = &v
		errPct := roundTo(prevErrors/prevCalls*100, 1)
		summary.PrevErrorRate = &errPct
	}
	if prevMaxP95 > 0 {
		v := roundTo(prevMaxP95, 2)
		summary.PrevP95Duration = &v
	}

	// Degraded operations vs previous period (absolute error ≥5% or delta).
	for _, op := range ops {
		if op.Rate < anomalyMinRate {
			continue
		}
		prev := prevByName[op.SpanName+"\x00"+op.SpanKind]
		curErrPct := roundTo(op.ErrorRate*100, 1)
		prevErrPct := roundTo(prev.ErrorRate*100, 1)
		isErrorDelta := prev.Rate > 0 &&
			curErrPct > 0 &&
			(curErrPct-prevErrPct) >= anomalyMinErrorIncrease &&
			(prevErrPct == 0 || curErrPct/prevErrPct >= anomalyRelativeFactor)
		isErrorAbsolute := curErrPct >= absoluteErrorCritical
		isLatency := prev.P95Duration > 0 && op.P95Duration > 0 &&
			(op.P95Duration-prev.P95Duration) >= anomalyMinLatencyIncreaseMs &&
			op.P95Duration/prev.P95Duration >= anomalyRelativeFactor
		if isErrorDelta || isErrorAbsolute || isLatency {
			summary.DegradedOps = append(summary.DegradedOps, DegradedOperation{
				SpanName:        op.SpanName,
				SpanKind:        op.SpanKind,
				Rate:            roundTo(op.Rate, 3),
				ErrorRate:       curErrPct,
				P95Duration:     roundTo(op.P95Duration, 2),
				DurationUnit:    "ms",
				PrevErrorRate:   prevErrPct,
				PrevP95Duration: roundTo(prev.P95Duration, 2),
				ErrorAnomaly:    isErrorDelta || isErrorAbsolute,
				LatencyAnomaly:  isLatency,
			})
		}
	}

	// Degraded dependencies from service-graph edges.
	curDeps := a.fetchDependenciesClickHouse(ctx, ch, service, environment, from, to)
	prevDeps := a.fetchDependenciesClickHouse(ctx, ch, service, environment, prevFrom, from)
	prevDepByName := make(map[string]DependencySummary, len(prevDeps))
	for _, d := range prevDeps {
		prevDepByName[d.Name] = d
	}
	for _, d := range curDeps {
		if d.Rate < anomalyMinRate {
			continue
		}
		prev := prevDepByName[d.Name]
		curErrPct := roundTo(d.ErrorRate*100, 1)
		prevErrPct := roundTo(prev.ErrorRate*100, 1)
		isErrorDelta := prev.Rate > 0 &&
			curErrPct > 0 &&
			(curErrPct-prevErrPct) >= anomalyMinErrorIncrease &&
			(prevErrPct == 0 || curErrPct/prevErrPct >= anomalyRelativeFactor)
		isErrorAbsolute := curErrPct >= absoluteErrorCritical
		isLatency := prev.P95Duration > 0 && d.P95Duration > 0 &&
			(d.P95Duration-prev.P95Duration) >= anomalyMinLatencyIncreaseMs &&
			d.P95Duration/prev.P95Duration >= anomalyRelativeFactor
		if isErrorDelta || isErrorAbsolute || isLatency {
			summary.DegradedDeps = append(summary.DegradedDeps, DegradedDependency{
				Name:            d.Name,
				Type:            d.Type,
				Rate:            roundTo(d.Rate, 3),
				ErrorRate:       curErrPct,
				P95Duration:     roundTo(d.P95Duration, 2),
				DurationUnit:    "ms",
				PrevErrorRate:   prevErrPct,
				PrevP95Duration: roundTo(prev.P95Duration, 2),
				ErrorAnomaly:    isErrorDelta || isErrorAbsolute,
				LatencyAnomaly:  isLatency,
			})
		}
	}

	summary.CauseCategory = classifyCause(summary)
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
	clientSet := make(map[string]struct{})
	for _, e := range edges {
		clientSet[e.Client] = struct{}{}
	}
	out := make([]DependencySummary, 0)
	for _, e := range edges {
		if e.Client != service {
			continue
		}
		depType, _ := inferClickStackNodeType(e.Server, clientSet)
		out = append(out, DependencySummary{
			Name:         e.Server,
			Type:         depType,
			Rate:         e.Rate,
			ErrorRate:    e.ErrorRate,
			DurationUnit: "ms",
			Impact:       e.Rate * e.ErrorRate,
		})
	}
	return out
}

// fetchGlobalDependenciesClickHouse aggregates all service-graph servers as
// fleet-wide external dependencies (infra peers preferred over pure services).
func (a *App) fetchGlobalDependenciesClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	environment string,
	from, to time.Time,
) []DependencySummary {
	edges := a.serviceGraphEdgesClickHouse(ctx, ch, environment, from, to)
	clientSet := make(map[string]struct{})
	for _, e := range edges {
		clientSet[e.Client] = struct{}{}
	}
	type agg struct {
		rate, errW float64
		depType    string
	}
	byServer := make(map[string]*agg)
	for _, e := range edges {
		depType, _ := inferClickStackNodeType(e.Server, clientSet)
		if depType == "service" {
			continue
		}
		d := byServer[e.Server]
		if d == nil {
			d = &agg{depType: depType}
			byServer[e.Server] = d
		}
		d.rate += e.Rate
		d.errW += e.Rate * e.ErrorRate
	}
	out := make([]DependencySummary, 0, len(byServer))
	for name, d := range byServer {
		errRate := 0.0
		if d.rate > 0 {
			errRate = d.errW / d.rate
		}
		out = append(out, DependencySummary{
			Name: name, Type: d.depType, Rate: d.rate, ErrorRate: errRate,
			DurationUnit: "ms", Impact: d.rate * errRate,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Impact != out[j].Impact {
			return out[i].Impact > out[j].Impact
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// fetchNamespaceDependenciesClickHouse keeps outbound edges whose client has
// ResourceAttributes service.namespace matching namespace. When CH service-graph
// rows lack namespace, falls back to clients whose inventory name prefix or
// exact ResourceAttributes probe isn't available — use Attribute-less heuristic:
// include all non-service targets called by any client (namespace filter soft).
func (a *App) fetchNamespaceDependenciesClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	namespace, environment string,
	from, to time.Time,
) []NamespaceDependency {
	// Prefer inventory discover filtered by namespace to know which clients belong.
	nsClients := map[string]struct{}{}
	if ch != nil && namespace != "" {
		opts := queries.ClickHouseQueryOptions{From: from, To: to, Limit: 500}
		if res, err := ch.Query(ctx, chquery.DiscoverInventorySQL(from, to, namespace, environment), opts); err == nil {
			for _, m := range res.Maps() {
				if s := chquery.ParseString(m["service"]); s != "" {
					nsClients[s] = struct{}{}
				}
			}
		}
	}
	edges := a.serviceGraphEdgesClickHouse(ctx, ch, environment, from, to)
	clientSet := make(map[string]struct{})
	for _, e := range edges {
		clientSet[e.Client] = struct{}{}
	}
	type agg struct {
		callers    map[string]struct{}
		rate, errW float64
		depType    string
	}
	byServer := make(map[string]*agg)
	for _, e := range edges {
		if len(nsClients) > 0 {
			if _, ok := nsClients[e.Client]; !ok {
				continue
			}
			if _, ok := nsClients[e.Server]; ok {
				continue // in-namespace peer
			}
		}
		depType, _ := inferClickStackNodeType(e.Server, clientSet)
		if depType == "service" && len(nsClients) == 0 {
			continue
		}
		d := byServer[e.Server]
		if d == nil {
			d = &agg{callers: make(map[string]struct{}), depType: depType}
			byServer[e.Server] = d
		}
		d.callers[e.Client] = struct{}{}
		d.rate += e.Rate
		d.errW += e.Rate * e.ErrorRate
	}
	out := make([]NamespaceDependency, 0, len(byServer))
	for name, d := range byServer {
		errRate := 0.0
		if d.rate > 0 {
			errRate = d.errW / d.rate
		}
		out = append(out, NamespaceDependency{
			Name: name, Type: d.depType, CallerCount: len(d.callers),
			Rate: d.rate, ErrorRate: errRate, DurationUnit: "ms",
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CallerCount != out[j].CallerCount {
			return out[i].CallerCount > out[j].CallerCount
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func (a *App) fetchDependencyDetailClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	depName, environment string,
	from, to time.Time,
) DependencyDetailResponse {
	edges := a.serviceGraphEdgesClickHouse(ctx, ch, environment, from, to)
	clientSet := make(map[string]struct{})
	for _, e := range edges {
		clientSet[e.Client] = struct{}{}
	}
	depType, _ := inferClickStackNodeType(depName, clientSet)
	var rate, errW float64
	upstreams := make([]DependencySummary, 0)
	for _, e := range edges {
		if e.Server != depName {
			continue
		}
		rate += e.Rate
		errW += e.Rate * e.ErrorRate
		upstreams = append(upstreams, DependencySummary{
			Name: e.Client, Type: "service", Rate: e.Rate, ErrorRate: e.ErrorRate,
			DurationUnit: "ms", Impact: e.Rate * e.ErrorRate,
		})
	}
	errRate := 0.0
	if rate > 0 {
		errRate = errW / rate
	}
	sort.Slice(upstreams, func(i, j int) bool {
		if upstreams[i].Impact != upstreams[j].Impact {
			return upstreams[i].Impact > upstreams[j].Impact
		}
		return upstreams[i].Name < upstreams[j].Name
	})
	return DependencyDetailResponse{
		Dependency: DependencySummary{
			Name: depName, Type: depType, Rate: rate, ErrorRate: errRate,
			DurationUnit: "ms", Impact: rate * errRate,
		},
		Upstreams:  upstreams,
		Operations: []queries.DependencyOperation{},
	}
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
	scrape, _ := a.ResolveServiceAlias(service)
	res, err := ch.Query(ctx, chquery.LogPatternSQLFiltered(service, scrape, from, to, environment, severity, 50), opts)
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
			chquery.LogPatternSQLFiltered(service, scrape, from.Add(-span), from, environment, severity, 50),
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
	logger := log.DefaultLogger.With("handler", "issues-clickhouse")
	opts := queries.ClickHouseQueryOptions{From: from, To: to, Limit: 50, SkipLimitEnforcement: true}

	resp := IssuesResponse{
		FingerprintVersion: fingerprint.Version,
		Issues:             []Issue{},
	}

	// Browser Issues: ResolveServiceAlias → HyperDX error spans (ticket 19/20).
	browserName, aliasOK := a.ResolveServiceAlias(service)
	if !aliasOK {
		resp.Hint = AliasMissingHint(service)
	} else {
		probeRes, probeErr := ch.Query(ctx, chquery.BrowserIssuesProbeSQL(browserName, environment, from, to), opts)
		if probeErr != nil {
			logger.Warn("browser issues probe failed", "error", probeErr)
		} else if probeRes.RowCount() > 0 && chquery.ParseFloat(probeRes.Maps()[0]["c"]) >= 1 {
			resp.Sources.Browser = true
		}
		if resp.Sources.Browser {
			bRes, bErr := ch.Query(ctx, chquery.BrowserExceptionGroupsSQL(browserName, from, to, environment, 50), opts)
			if bErr != nil {
				logger.Warn("browser issues CH query failed", "error", bErr)
				resp.Sources.Browser = false
			} else {
				for _, row := range chquery.ParseTraceExceptionGroupsMaps(bRes.Maps()) {
					fp := fingerprint.Compute(fingerprint.Event{Type: row.ExceptionType, Value: row.Message})
					issue := Issue{
						Fingerprint:  fp.Value,
						Tier:         int(fp.Tier),
						Title:        fp.Title,
						Types:        nil,
						Count:        float64(row.Occurrences),
						Sessions:     float64(row.Sessions),
						MemberHashes: []string{}, // CH path: drawer uses /issues/occurrences
						Source:       issueSourceBrowser,
					}
					if row.ExceptionType != "" {
						issue.Types = []string{row.ExceptionType}
					}
					resp.Issues = append(resp.Issues, issue)
				}
			}
		} else if aliasOK {
			// Alias OK but no HyperDX exception signal in range — operator/product copy (ticket 13).
			resp.Hint = "No HyperDX browser exception signals for this service in range. Ensure @hyperdx/browser (or Faro-shaped) exceptions are exported to ClickHouse."
		}
	}

	// Server secondary feed: inventory ServiceName + Events.Name=exception, badged server.
	sRes, sErr := ch.Query(ctx, chquery.ServerExceptionGroupsSQL(service, from, to, environment, 50), opts)
	if sErr != nil {
		logger.Warn("server issues CH query failed", "error", sErr)
	} else {
		resp.Sources.ServerLogs = true
		for _, row := range chquery.ParseTraceExceptionGroupsMaps(sRes.Maps()) {
			fp := fingerprint.Compute(fingerprint.Event{Type: row.ExceptionType, Value: row.Message})
			issue := Issue{
				Fingerprint:  fp.Value,
				Tier:         int(fp.Tier),
				Title:        fp.Title,
				Types:        nil,
				Count:        float64(row.Occurrences),
				MemberHashes: []string{},
				Source:       issueSourceServer,
				Impact:       &IssueImpact{Pods: 0, Versions: []string{}},
			}
			if row.ExceptionType != "" {
				issue.Types = []string{row.ExceptionType}
			}
			resp.Issues = append(resp.Issues, issue)
		}
	}

	sort.Slice(resp.Issues, func(i, j int) bool {
		if resp.Issues[i].Count != resp.Issues[j].Count {
			return resp.Issues[i].Count > resp.Issues[j].Count
		}
		if resp.Issues[i].Fingerprint != resp.Issues[j].Fingerprint {
			return resp.Issues[i].Fingerprint < resp.Issues[j].Fingerprint
		}
		return resp.Issues[i].Source < resp.Issues[j].Source
	})
	if len(resp.Issues) > maxGroups {
		resp.Issues = resp.Issues[:maxGroups]
	}
	resp.Unavailable = !resp.Sources.Browser && !resp.Sources.ServerLogs && sErr != nil
	// Clear enablement hint when we already have server issues to show — empty-state
	// variants only apply when the list is empty (frontend).
	if len(resp.Issues) > 0 && strings.Contains(resp.Hint, "HyperDX browser exception") {
		resp.Hint = ""
	}
	return resp
}

func (a *App) fetchFrontendSessionsClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	service, environment string,
	from, to time.Time,
	q string,
) FrontendSessionsResponse {
	browserName, ok := a.ResolveServiceAlias(service)
	if !ok {
		return FrontendSessionsResponse{
			Sessions: []SessionSummary{},
			Hint:     AliasMissingHint(service),
		}
	}
	if ch == nil {
		return FrontendSessionsResponse{Sessions: []SessionSummary{}, Unavailable: true}
	}
	// HyperDX sessions with rum.sessionId + error/user enrichment from otel_traces.
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

	opts := queries.ClickHouseQueryOptions{From: from, To: to, Limit: 50}
	res, err := ch.Query(ctx, sql, opts)
	if err != nil {
		log.DefaultLogger.Warn("frontend sessions CH query failed", "error", err)
		return FrontendSessionsResponse{Sessions: []SessionSummary{}, Unavailable: true}
	}

	type enrich struct {
		errors     float64
		userID     string
		userEmail  string
		appVersion string
		pages      int
		browser    string
		os         string
	}
	bySession := map[string]enrich{}
	enrichRes, enrichErr := ch.Query(ctx, chquery.SessionEnrichmentSQL(browserName, from, to, environment), opts)
	if enrichErr != nil {
		log.DefaultLogger.Warn("frontend sessions enrichment failed", "error", enrichErr)
	} else {
		for _, m := range enrichRes.Maps() {
			sid := chquery.ParseString(m["session_id"])
			if sid == "" {
				continue
			}
			browser, osName := parseUserAgent(chquery.ParseString(m["user_agent"]))
			bySession[sid] = enrich{
				errors:     chquery.ParseFloat(m["errors"]),
				userID:     chquery.ParseString(m["user_id"]),
				userEmail:  chquery.ParseString(m["user_email"]),
				appVersion: chquery.ParseString(m["app_version"]),
				pages:      int(chquery.ParseFloat(m["pages"])),
				browser:    browser,
				os:         osName,
			}
		}
	}

	out := make([]SessionSummary, 0, res.RowCount())
	for _, m := range res.Maps() {
		sid := chquery.ParseString(m["session_id"])
		e := bySession[sid]
		out = append(out, SessionSummary{
			SessionID:   sid,
			FirstSeenMs: chquery.ParseInt64(m["first_ms"]),
			LastSeenMs:  chquery.ParseInt64(m["last_ms"]),
			Events:      chquery.ParseFloat(m["events"]),
			Errors:      e.errors,
			UserID:      e.userID,
			UserEmail:   e.userEmail,
			Browser:     e.browser,
			OS:          e.os,
			AppVersion:  e.appVersion,
			Pages:       e.pages,
		})
	}
	// Prefer sessions with errors, then recency (M5 / HyperDX polish).
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Errors != out[j].Errors {
			return out[i].Errors > out[j].Errors
		}
		return out[i].LastSeenMs > out[j].LastSeenMs
	})
	if q = strings.TrimSpace(q); q != "" {
		filtered := out[:0]
		for i := range out {
			if matchesSessionQuery(&out[i], q) {
				filtered = append(filtered, out[i])
			}
		}
		out = filtered
	}
	return FrontendSessionsResponse{Sessions: out}
}

func (a *App) fetchFrontendVersionsClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	service, environment string,
	from, to time.Time,
) FrontendVersionsResponse {
	browserName, ok := a.ResolveServiceAlias(service)
	if !ok {
		return FrontendVersionsResponse{
			Versions: []VersionStat{},
			Hint:     AliasMissingHint(service),
		}
	}
	if ch == nil {
		return FrontendVersionsResponse{Versions: []VersionStat{}, Unavailable: true}
	}
	logger := log.DefaultLogger.With("handler", "frontend-versions-clickhouse")
	opts := queries.ClickHouseQueryOptions{From: from, To: to, Limit: 20, SkipLimitEnforcement: true}

	probeRes, probeErr := ch.Query(ctx, chquery.BrowserVersionsProbeSQL(browserName, environment, from, to), opts)
	if probeErr != nil {
		logger.Warn("browser versions probe failed", "error", probeErr)
		return FrontendVersionsResponse{Versions: []VersionStat{}, Unavailable: true}
	}
	if probeRes.RowCount() == 0 || chquery.ParseFloat(probeRes.Maps()[0]["c"]) < 1 {
		return FrontendVersionsResponse{
			Versions: []VersionStat{},
			Hint:     versionsEnablementHint,
		}
	}

	res, err := ch.Query(ctx, chquery.BrowserVersionsSQL(browserName, from, to, environment, maxVersions), opts)
	if err != nil {
		logger.Warn("browser versions CH query failed", "error", err)
		return FrontendVersionsResponse{Versions: []VersionStat{}, Unavailable: true}
	}

	groups := chquery.ParseVersionGroupsMaps(res.Maps())
	var totalSessions float64
	for _, g := range groups {
		totalSessions += g.Sessions
	}

	deploys, latestVersion := a.fetchDeployAnnotations(ctx, service, environment, from, to)

	out := make([]VersionStat, 0, len(groups))
	for _, g := range groups {
		stat := VersionStat{
			Version:    g.Version,
			Sessions:   g.Sessions,
			Exceptions: g.Exceptions,
		}
		if totalSessions > 0 {
			stat.Adoption = roundTo(g.Sessions/totalSessions, 4)
		}
		if g.Sessions > 0 {
			stat.ErrorFreeRate = roundTo(1-min(g.ErrSessions/g.Sessions, 1), 4)
		}
		if t, ok := deploys[g.Version]; ok {
			stat.DeployedAtMs = t
		}
		out = append(out, stat)
	}
	return FrontendVersionsResponse{Versions: out, LatestVersion: latestVersion}
}

func (a *App) fetchFeedbackClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	service, environment string,
	from, to time.Time,
	sessionID, fingerprint string,
) FeedbackResponse {
	browserName, ok := a.ResolveServiceAlias(service)
	if !ok {
		return FeedbackResponse{
			Feedback: []FeedbackEntry{},
			Hint:     AliasMissingHint(service),
		}
	}
	if ch == nil {
		return FeedbackResponse{Feedback: []FeedbackEntry{}, Unavailable: true}
	}
	logger := log.DefaultLogger.With("handler", "feedback-clickhouse")
	opts := queries.ClickHouseQueryOptions{From: from, To: to, Limit: maxFeedback, SkipLimitEnforcement: true}

	// Probe without session/fingerprint filters so enablement copy is about
	// signal presence, not an over-narrow filter.
	probeRes, probeErr := ch.Query(ctx, chquery.BrowserFeedbackProbeSQL(browserName, environment, from, to), opts)
	if probeErr != nil {
		logger.Warn("browser feedback probe failed", "error", probeErr)
		return FeedbackResponse{Feedback: []FeedbackEntry{}, Unavailable: true}
	}
	if probeRes.RowCount() == 0 || chquery.ParseFloat(probeRes.Maps()[0]["c"]) < 1 {
		return FeedbackResponse{
			Feedback: []FeedbackEntry{},
			Hint:     feedbackEnablementHint,
		}
	}

	sid := sanitizeSessionID(sessionID)
	fp := sanitizeFingerprintFilter(fingerprint)
	res, err := ch.Query(ctx, chquery.BrowserFeedbackSQL(browserName, from, to, environment, sid, fp, maxFeedback), opts)
	if err != nil {
		logger.Warn("browser feedback CH query failed", "error", err)
		return FeedbackResponse{Feedback: []FeedbackEntry{}, Unavailable: true}
	}

	rows := chquery.ParseFeedbackRowsMaps(res.Maps())
	out := make([]FeedbackEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, FeedbackEntry{
			TimeMs:      r.TimeMs,
			Message:     r.Message,
			Category:    r.Category,
			Email:       r.Email,
			SessionID:   r.SessionID,
			Fingerprint: r.Fingerprint,
			PageURL:     r.PageURL,
			AppVersion:  r.AppVersion,
		})
	}
	return FeedbackResponse{Feedback: out}
}

func patternSeen(patterns map[string]struct{}, pattern string) bool {
	_, ok := patterns[pattern]
	return ok
}
