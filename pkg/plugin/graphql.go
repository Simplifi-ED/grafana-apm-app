package plugin

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/nais/grafana-otel-plugin/pkg/plugin/chquery"
	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// GraphQL metric framework identifiers.
const (
	frameworkDGS          = "DGS"
	frameworkMicroProfile = "MicroProfile"
	frameworkCustomTimer  = "Custom"
)

// graphqlProbe defines how to detect and query a specific GraphQL metric pattern.
type graphqlProbe struct {
	framework   string
	countMetric string // metric to check for existence and query rates
	sumMetric   string // metric for latency (rate of sum / rate of count)
	opLabel     string // label that contains the operation/field name
	typeLabel   string // label for operation type (query/mutation), empty if N/A
	errorFilter string // label filter for errors, empty if not available
	latencyUnit string // "s" or "ms" — what unit the raw metric values are in
}

// Ordered list of probes. First match wins.
var graphqlProbes = []graphqlProbe{
	{
		framework:   frameworkDGS,
		countMetric: "graphql_request_seconds_count",
		sumMetric:   "graphql_request_seconds_sum",
		opLabel:     "graphql_operation_type",
		errorFilter: `graphql_outcome="REQUEST_ERROR"`,
		latencyUnit: "s",
	},
	{
		framework:   frameworkMicroProfile,
		countMetric: "mp_graphql_seconds_count",
		sumMetric:   "mp_graphql_seconds_sum",
		opLabel:     "name",
		typeLabel:   "type",
		latencyUnit: "s",
	},
	{
		framework:   frameworkCustomTimer + " (notifikasjon)",
		countMetric: "graphql_timer_seconds_count",
		sumMetric:   "graphql_timer_seconds_sum",
		opLabel:     "queryName",
		latencyUnit: "s",
	},
	{
		framework:   frameworkCustomTimer + " (spesialist)",
		countMetric: "graphql_responstider_count",
		sumMetric:   "graphql_responstider_sum",
		opLabel:     "operationName",
		latencyUnit: "ms",
	},
	{
		framework:   frameworkCustomTimer + " (custom)",
		countMetric: "graphql_query_timed_seconds_count",
		sumMetric:   "graphql_query_timed_seconds_sum",
		opLabel:     "query",
		errorFilter: `exception!="none"`,
		latencyUnit: "s",
	},
	{
		framework:   frameworkCustomTimer + " (custom)",
		countMetric: "requests_graphql_duration_seconds_count",
		sumMetric:   "requests_graphql_duration_seconds_sum",
		opLabel:     "operation",
		latencyUnit: "s",
	},
}

// Known metric prefixes to exclude from per-query-name per-query-name discovery.
var knownGraphQLPrefixes = []string{
	"graphql_request_", "graphql_datafetcher_", "graphql_dataloader_",
	"graphql_timer_", "graphql_responstider", "graphql_query_timed_",
	"graphql_context_", "graphql_consumer_errors_", "graphql_token_",
	"graphql_field_fetch_", "graphql_mutation_", "graphql_response_errors_",
	"graphql_bruker_",
}

// GraphQLOperation, GraphQLMetricsResponse → models.go

func (a *App) handleGraphQLMetrics(w http.ResponseWriter, req *http.Request) {
	if !requireGET(w, req) {
		return
	}
	ctx := a.requestContext(req)

	namespace, service := parseServiceRef(req)
	environment := parseEnvironment(req)
	if !requireServiceParam(w, service) {
		return
	}

	from, to := parseTimeRange(req)

	if a.clickStackMode() {
		ch := a.chForRequest(req)
		writeJSON(w, a.queryGraphQLMetricsClickHouse(ctx, ch, namespace, service, environment, from, to))
		return
	}

	orgID := req.Header.Get("X-Grafana-Org-Id")
	ck := cacheKey("graphql", orgID, roundedUnix(from), roundedUnix(to), namespace, service, environment)
	a.writeCached(w, ck, "querying graphql metrics failed", func() (any, error) {
		return a.queryGraphQLMetrics(ctx, namespace, service, environment, from, to), nil
	})
}

// queryGraphQLMetricsClickHouse detects GraphQL via Nais metric probes in
// otel_metrics_* OR OTel span attrs on otel_traces (ticket 16 / letter H).
func (a *App) queryGraphQLMetricsClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	namespace, service, environment string,
	from, to time.Time,
) GraphQLMetricsResponse {
	logger := log.DefaultLogger.With("handler", "graphql-clickhouse")
	if ch == nil || service == "" {
		return GraphQLMetricsResponse{Hint: chquery.GraphQLHintUndetected}
	}

	// Metrics use scrape identity when aliased; spans use inventory ServiceName
	// (fixture graphql-demo matches inventory).
	metricService := service
	if scrape, ok := a.ResolveServiceAlias(service); ok && scrape != "" {
		metricService = scrape
	}

	lookbackFrom := to.Add(-5 * time.Minute)
	if lookbackFrom.Before(from) {
		lookbackFrom = from
	}
	opts := queries.ClickHouseQueryOptions{From: lookbackFrom, To: to, Limit: 200}

	if resp := a.queryGraphQLFromCHMetrics(ctx, ch, logger, namespace, metricService, lookbackFrom, to, opts); resp.Detected {
		return resp
	}
	if resp := a.queryGraphQLFromCHSpans(ctx, ch, logger, namespace, service, environment, from, to, opts); resp.Detected {
		return resp
	}
	return GraphQLMetricsResponse{Hint: chquery.GraphQLHintUndetected}
}

func (a *App) queryGraphQLFromCHMetrics(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	logger log.Logger,
	namespace, scrapeService string,
	from, to time.Time,
	opts queries.ClickHouseQueryOptions,
) GraphQLMetricsResponse {
	disc, err := ch.Query(ctx, chquery.GraphQLMetricDiscoverSQL(scrapeService, namespace, from, to), opts)
	if err != nil {
		logger.Warn("graphql CH metric discovery failed", "error", err, "service", scrapeService)
		return GraphQLMetricsResponse{}
	}
	metricNames := chquery.ParseGraphQLMetricNames(disc.Maps())
	if len(metricNames) == 0 {
		return GraphQLMetricsResponse{}
	}

	for _, probe := range graphqlProbes {
		if !metricNames[probe.countMetric] {
			continue
		}
		logger.Debug("detected GraphQL framework (CH metrics)", "framework", probe.framework, "service", scrapeService)
		opsRes, qerr := ch.Query(ctx, chquery.GraphQLMetricOpsSQL(
			scrapeService, namespace,
			probe.countMetric, probe.sumMetric, probe.opLabel, probe.typeLabel, probe.errorFilter,
			from, to, 50,
		), opts)
		if qerr != nil {
			logger.Warn("graphql CH metric ops failed", "error", qerr, "framework", probe.framework)
			continue
		}
		ops := mapCHGraphQLOps(chquery.ParseGraphQLOpRows(opsRes.Maps(), probe.latencyUnit))
		resp := GraphQLMetricsResponse{
			Detected:   true,
			Framework:  probe.framework,
			Source:     "metrics",
			Operations: ops,
		}
		if probe.framework == frameworkDGS && metricNames["graphql_datafetcher_seconds_count"] {
			fRes, ferr := ch.Query(ctx, chquery.GraphQLMetricOpsSQL(
				scrapeService, namespace,
				"graphql_datafetcher_seconds_count", "graphql_datafetcher_seconds_sum",
				"graphql_field_name", "", `graphql_outcome!="SUCCESS"`,
				from, to, 50,
			), opts)
			if ferr == nil {
				resp.Fetchers = mapCHGraphQLOps(chquery.ParseGraphQLOpRows(fRes.Maps(), "s"))
			}
		}
		return resp
	}

	// per-query-name per-query-name metrics: graphql_{QueryName}_seconds_count
	pdlOps := a.queryPDLOperationsClickHouse(ctx, ch, logger, scrapeService, namespace, metricNames, from, to, opts)
	if len(pdlOps) > 0 {
		return GraphQLMetricsResponse{
			Detected:   true,
			Framework:  "Custom (per-query)",
			Source:     "metrics",
			Operations: pdlOps,
		}
	}
	return GraphQLMetricsResponse{}
}

func (a *App) queryPDLOperationsClickHouse(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	logger log.Logger,
	scrapeService, namespace string,
	metricNames map[string]bool,
	from, to time.Time,
	opts queries.ClickHouseQueryOptions,
) []GraphQLOperation {
	var countMetrics []string
	for name := range metricNames {
		if !strings.HasPrefix(name, "graphql_") || !strings.HasSuffix(name, "_seconds_count") {
			continue
		}
		excluded := false
		for _, prefix := range knownGraphQLPrefixes {
			if strings.HasPrefix(name, prefix) {
				excluded = true
				break
			}
		}
		if !excluded {
			countMetrics = append(countMetrics, name)
		}
	}
	if len(countMetrics) == 0 {
		return nil
	}
	sort.Strings(countMetrics)

	ops := make([]GraphQLOperation, 0, len(countMetrics))
	for _, countMetric := range countMetrics {
		name := strings.TrimPrefix(countMetric, "graphql_")
		name = strings.TrimSuffix(name, "_seconds_count")
		sumMetric := strings.Replace(countMetric, "_count", "_sum", 1)
		res, err := ch.Query(ctx, chquery.GraphQLPDLOpSQL(
			scrapeService, namespace, name, countMetric, sumMetric, from, to,
		), opts)
		if err != nil {
			logger.Debug("per-query CH op failed", "metric", countMetric, "error", err)
			continue
		}
		ops = append(ops, mapCHGraphQLOps(chquery.ParseGraphQLOpRows(res.Maps(), "s"))...)
	}
	sort.Slice(ops, func(i, j int) bool { return ops[i].Rate > ops[j].Rate })
	return ops
}

func (a *App) queryGraphQLFromCHSpans(
	ctx context.Context,
	ch *queries.ClickHouseClient,
	logger log.Logger,
	namespace, service, environment string,
	from, to time.Time,
	opts queries.ClickHouseQueryOptions,
) GraphQLMetricsResponse {
	det, err := ch.Query(ctx, chquery.GraphQLSpanDetectSQL(service, namespace, environment, from, to), opts)
	if err != nil {
		logger.Warn("graphql CH span detect failed", "error", err, "service", service)
		return GraphQLMetricsResponse{}
	}
	if len(det.Maps()) == 0 || chquery.ParseFloat(det.Maps()[0]["c"]) <= 0 {
		return GraphQLMetricsResponse{}
	}
	opsRes, err := ch.Query(ctx, chquery.GraphQLSpanOpsSQL(service, namespace, environment, from, to, 50), opts)
	if err != nil {
		logger.Warn("graphql CH span ops failed", "error", err, "service", service)
		return GraphQLMetricsResponse{Detected: true, Framework: "OpenTelemetry spans", Source: "spans"}
	}
	ops := mapCHGraphQLOps(chquery.ParseGraphQLOpRows(opsRes.Maps(), "s"))
	return GraphQLMetricsResponse{
		Detected:   true,
		Framework:  "OpenTelemetry spans",
		Source:     "spans",
		Operations: ops,
	}
}

func mapCHGraphQLOps(rows []chquery.GraphQLOpRow) []GraphQLOperation {
	if len(rows) == 0 {
		return nil
	}
	ops := make([]GraphQLOperation, 0, len(rows))
	for _, r := range rows {
		op := GraphQLOperation{
			Name:        r.Name,
			Type:        r.Type,
			Rate:        r.Rate,
			AvgLatency:  r.AvgLatency,
			LatencyUnit: r.LatencyUnit,
		}
		if r.HasErrorRate {
			er := r.ErrorRate
			op.ErrorRate = &er
		}
		ops = append(ops, op)
	}
	return ops
}

func (a *App) queryGraphQLMetrics(ctx context.Context, namespace, service, environment string, _, to time.Time) GraphQLMetricsResponse {
	logger := log.DefaultLogger.With("handler", "graphql")
	client := a.prom(ctx)
	if client == nil {
		return GraphQLMetricsResponse{}
	}

	// Build the service filter — try both app and service_name labels.
	// Most Nav services have app=service_name and namespace=service_namespace.
	svcFilter := fmt.Sprintf(`app="%s", namespace="%s"`, service, namespace)
	if environment != "" {
		svcFilter += fmt.Sprintf(`, %s="%s"`, a.otelCfg.Labels.DeploymentEnv, environment)
	}

	// Single discovery query: find all graphql-related metric names for this service
	discoveryQuery := fmt.Sprintf(
		`count by (__name__) ({__name__=~"graphql_.*|mp_graphql_.*|requests_graphql_.*", %s})`,
		svcFilter,
	)
	results, err := client.InstantQuery(ctx, discoveryQuery, to)
	if err != nil {
		logger.Warn("graphql discovery query failed", "error", err)
		return GraphQLMetricsResponse{}
	}

	if len(results) == 0 {
		// No GraphQL metrics with app label — nothing to show
		return GraphQLMetricsResponse{}
	}

	// Collect discovered metric names
	metricNames := make(map[string]bool, len(results))
	for _, r := range results {
		if name := r.Metric["__name__"]; name != "" {
			metricNames[name] = true
		}
	}

	// Try each probe against discovered metrics
	for _, probe := range graphqlProbes {
		if !metricNames[probe.countMetric] {
			continue
		}
		logger.Debug("detected GraphQL framework", "framework", probe.framework, "service", service)
		ops := queryProbeOperations(ctx, client, probe, svcFilter, to, logger)
		resp := GraphQLMetricsResponse{
			Detected:   true,
			Framework:  probe.framework,
			Operations: ops,
		}
		// DGS enrichment: fetch resolver/datafetcher data
		if probe.framework == frameworkDGS && metricNames["graphql_datafetcher_seconds_count"] {
			resp.Fetchers = queryDGSFetchers(ctx, client, svcFilter, to, logger)
		}
		return resp
	}

	// Check for per-query-name per-query-name metrics: graphql_{QueryName}_seconds_count
	pdlOps := queryPDLOperations(ctx, client, svcFilter, metricNames, to, logger)
	if len(pdlOps) > 0 {
		return GraphQLMetricsResponse{
			Detected:   true,
			Framework:  "Custom (per-query)",
			Operations: pdlOps,
		}
	}

	return GraphQLMetricsResponse{}
}

// queryProbeOperations runs rate + latency queries for a matched probe.
func queryProbeOperations(
	ctx context.Context, client *queries.PrometheusClient,
	probe graphqlProbe, svcFilter string, at time.Time,
	logger log.Logger,
) []GraphQLOperation {
	rangeStr := "[5m]"

	// Rate per operation
	rateQuery := fmt.Sprintf(
		`sum by (%s) (rate(%s{%s}%s))`,
		probe.opLabel, probe.countMetric, svcFilter, rangeStr,
	)
	if probe.typeLabel != "" {
		rateQuery = fmt.Sprintf(
			`sum by (%s, %s) (rate(%s{%s}%s))`,
			probe.opLabel, probe.typeLabel, probe.countMetric, svcFilter, rangeStr,
		)
	}

	// Latency per operation (avg = rate(sum) / rate(count))
	latQuery := fmt.Sprintf(
		`sum by (%s) (rate(%s{%s}%s)) / sum by (%s) (rate(%s{%s}%s))`,
		probe.opLabel, probe.sumMetric, svcFilter, rangeStr,
		probe.opLabel, probe.countMetric, svcFilter, rangeStr,
	)
	if probe.typeLabel != "" {
		latQuery = fmt.Sprintf(
			`sum by (%s, %s) (rate(%s{%s}%s)) / sum by (%s, %s) (rate(%s{%s}%s))`,
			probe.opLabel, probe.typeLabel, probe.sumMetric, svcFilter, rangeStr,
			probe.opLabel, probe.typeLabel, probe.countMetric, svcFilter, rangeStr,
		)
	}

	rateResults, err := client.InstantQuery(ctx, rateQuery, at)
	if err != nil {
		logger.Warn("graphql rate query failed", "error", err)
		return nil
	}

	latResults, err := client.InstantQuery(ctx, latQuery, at)
	if err != nil {
		logger.Warn("graphql latency query failed", "error", err)
	}

	// Build latency lookup
	latMap := make(map[string]float64)
	for _, r := range latResults {
		key := r.Metric[probe.opLabel]
		if probe.typeLabel != "" {
			key += "|" + r.Metric[probe.typeLabel]
		}
		latMap[key] = r.Value.Float()
	}

	// Error rate query (optional)
	var errMap map[string]float64
	if probe.errorFilter != "" {
		errQuery := fmt.Sprintf(
			`sum by (%s) (rate(%s{%s, %s}%s))`,
			probe.opLabel, probe.countMetric, svcFilter, probe.errorFilter, rangeStr,
		)
		errResults, err := client.InstantQuery(ctx, errQuery, at)
		if err == nil {
			errMap = make(map[string]float64)
			for _, r := range errResults {
				errMap[r.Metric[probe.opLabel]] = r.Value.Float()
			}
		}
	}

	ops := make([]GraphQLOperation, 0, len(rateResults))
	for _, r := range rateResults {
		name := r.Metric[probe.opLabel]
		if name == "" {
			continue
		}
		rate := r.Value.Float()
		if rate == 0 || !isValidMetricValue(rate) {
			continue
		}

		key := name
		opType := ""
		if probe.typeLabel != "" {
			opType = strings.ToLower(r.Metric[probe.typeLabel])
			key += "|" + r.Metric[probe.typeLabel]
		}

		lat := safeFloat(latMap[key])

		op := GraphQLOperation{
			Name:        name,
			Type:        opType,
			Rate:        math.Round(rate*1000) / 1000,
			AvgLatency:  lat,
			LatencyUnit: probe.latencyUnit,
		}

		if errMap != nil {
			if rate > 0 {
				pct := calculateErrorRate(errMap[name], rate)
				op.ErrorRate = &pct
			}
		}

		ops = append(ops, op)
	}

	sort.Slice(ops, func(i, j int) bool { return ops[i].Rate > ops[j].Rate })
	return ops
}

// queryDGSFetchers returns DGS datafetcher/resolver metrics.
func queryDGSFetchers(
	ctx context.Context, client *queries.PrometheusClient,
	svcFilter string, at time.Time, logger log.Logger,
) []GraphQLOperation {
	rangeStr := "[5m]"
	opLabel := "graphql_field_name"

	rateQuery := fmt.Sprintf(
		`sum by (%s) (rate(graphql_datafetcher_seconds_count{%s}%s))`,
		opLabel, svcFilter, rangeStr,
	)
	latQuery := fmt.Sprintf(
		`sum by (%s) (rate(graphql_datafetcher_seconds_sum{%s}%s)) / sum by (%s) (rate(graphql_datafetcher_seconds_count{%s}%s))`,
		opLabel, svcFilter, rangeStr, opLabel, svcFilter, rangeStr,
	)
	errQuery := fmt.Sprintf(
		`sum by (%s) (rate(graphql_datafetcher_seconds_count{%s, graphql_outcome!="SUCCESS"}%s))`,
		opLabel, svcFilter, rangeStr,
	)

	rateResults, err := client.InstantQuery(ctx, rateQuery, at)
	if err != nil {
		logger.Warn("DGS fetcher rate query failed", "error", err)
		return nil
	}

	latResults, _ := client.InstantQuery(ctx, latQuery, at)
	errResults, _ := client.InstantQuery(ctx, errQuery, at)

	latMap := make(map[string]float64)
	for _, r := range latResults {
		latMap[r.Metric[opLabel]] = r.Value.Float()
	}
	errMap := make(map[string]float64)
	for _, r := range errResults {
		errMap[r.Metric[opLabel]] = r.Value.Float()
	}

	fetchers := make([]GraphQLOperation, 0, len(rateResults))
	for _, r := range rateResults {
		name := r.Metric[opLabel]
		rate := r.Value.Float()
		if name == "" || rate == 0 || !isValidMetricValue(rate) {
			continue
		}
		lat := safeFloat(latMap[name])
		errRate := calculateErrorRate(errMap[name], rate)
		fetchers = append(fetchers, GraphQLOperation{
			Name:        name,
			Rate:        math.Round(rate*1000) / 1000,
			ErrorRate:   &errRate,
			AvgLatency:  lat,
			LatencyUnit: "s",
		})
	}

	sort.Slice(fetchers, func(i, j int) bool { return fetchers[i].Rate > fetchers[j].Rate })
	return fetchers
}

// queryPDLOperations detects per-query-name metrics where the query name is in the metric name:
// graphql_{queryName}_seconds_{count,sum}
func queryPDLOperations(
	ctx context.Context, client *queries.PrometheusClient,
	svcFilter string, metricNames map[string]bool,
	at time.Time, logger log.Logger,
) []GraphQLOperation {
	rangeStr := "[5m]"

	// Find metrics matching graphql_*_seconds_count that aren't known frameworks
	var countMetrics []string
	for name := range metricNames {
		if !strings.HasPrefix(name, "graphql_") {
			continue
		}
		if !strings.HasSuffix(name, "_seconds_count") {
			continue
		}
		excluded := false
		for _, prefix := range knownGraphQLPrefixes {
			if strings.HasPrefix(name, prefix) {
				excluded = true
				break
			}
		}
		if !excluded {
			countMetrics = append(countMetrics, name)
		}
	}

	if len(countMetrics) == 0 {
		return nil
	}

	sort.Strings(countMetrics)

	ops := make([]GraphQLOperation, 0, len(countMetrics))
	for _, countMetric := range countMetrics {
		// Extract query name: graphql_{name}_seconds_count → {name}
		name := strings.TrimPrefix(countMetric, "graphql_")
		name = strings.TrimSuffix(name, "_seconds_count")

		sumMetric := strings.Replace(countMetric, "_count", "_sum", 1)

		rateQuery := fmt.Sprintf(`sum(rate(%s{%s}%s))`, countMetric, svcFilter, rangeStr)
		latQuery := fmt.Sprintf(
			`sum(rate(%s{%s}%s)) / sum(rate(%s{%s}%s))`,
			sumMetric, svcFilter, rangeStr, countMetric, svcFilter, rangeStr,
		)

		rateResults, err := client.InstantQuery(ctx, rateQuery, at)
		if err != nil {
			logger.Debug("per-query rate query failed", "metric", countMetric, "error", err)
			continue
		}

		var rate float64
		for _, r := range rateResults {
			rate = r.Value.Float()
		}
		if rate == 0 || !isValidMetricValue(rate) {
			continue
		}

		var lat float64
		latResults, err := client.InstantQuery(ctx, latQuery, at)
		if err == nil {
			for _, r := range latResults {
				lat = r.Value.Float()
			}
		}
		lat = safeFloat(lat)

		ops = append(ops, GraphQLOperation{
			Name:        name,
			Type:        "query",
			Rate:        math.Round(rate*1000) / 1000,
			AvgLatency:  lat,
			LatencyUnit: "s",
		})
	}

	sort.Slice(ops, func(i, j int) bool { return ops[i].Rate > ops[j].Rate })
	return ops
}
