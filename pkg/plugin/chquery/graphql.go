package chquery

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// OTel GraphQL span attribute keys (ticket 16 / grilling 24).
const (
	AttrGraphQLOperationName = "graphql.operation.name"
	AttrGraphQLOperationType = "graphql.operation.type"
	AttrGraphQLDocument      = "graphql.document"
)

// GraphQLMetricNameMatch discovers Nais-style GraphQL metric families in CH.
const GraphQLMetricNameMatch = "(?i)(graphql_|mp_graphql_|requests_graphql_)"

// GraphQLHintUndetected is returned when neither metrics nor spans are present.
const GraphQLHintUndetected = "No GraphQL metrics or spans detected in ClickHouse for this service. " +
	"Export Nais probe metrics (graphql_request_seconds_*, mp_graphql_*, graphql_timer_*, …) into otel_metrics_*, " +
	"or OTel spans with graphql.operation.name / .type / .document (ticket 16)."

// GraphQLOpRow is one aggregated GraphQL operation from CH metrics or spans.
type GraphQLOpRow struct {
	Name         string
	Type         string
	Rate         float64
	AvgLatency   float64
	LatencyUnit  string
	ErrorRate    float64 // 0–100 when HasErrorRate
	HasErrorRate bool
}

// graphqlSpanAttrFilter matches fixture + live OTel GraphQL spans.
func graphqlSpanAttrFilter() string {
	return fmt.Sprintf(
		`(mapContains(SpanAttributes, %s) OR mapContains(SpanAttributes, %s) OR mapContains(SpanAttributes, %s))`,
		q(AttrGraphQLOperationName), q(AttrGraphQLOperationType), q(AttrGraphQLDocument),
	)
}

func graphqlTraceServiceWhere(service, namespace, environment string) string {
	where := fmt.Sprintf("(ServiceName = %s OR ResourceAttributes['%s'] = %s)",
		q(service), AttrServiceName, q(service))
	if namespace != "" {
		where += fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResServiceNamespace, q(namespace))
	}
	if environment != "" {
		where += fmt.Sprintf(" AND ResourceAttributes['%s'] = %s", ResCluster, q(environment))
	}
	return where
}

// GraphQLSpanDetectSQL returns a cheap presence probe for GraphQL span attrs.
func GraphQLSpanDetectSQL(service, namespace, environment string, from, to time.Time) string {
	where := fmt.Sprintf("%s AND %s AND %s",
		TimestampFilter(from, to),
		graphqlTraceServiceWhere(service, namespace, environment),
		graphqlSpanAttrFilter(),
	)
	return fmt.Sprintf(`
SELECT count() AS c
FROM %s
WHERE %s
LIMIT 1`,
		table(queries.CHTableTraces), where,
	)
}

// GraphQLSpanOpsSQL aggregates operations from otel_traces GraphQL attrs
// (matches scripts/fixtures/graphql-otel-spans.sql shape).
func GraphQLSpanOpsSQL(service, namespace, environment string, from, to time.Time, limit int) string {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	window := WindowSeconds(from, to)
	where := fmt.Sprintf("%s AND %s AND %s",
		TimestampFilter(from, to),
		graphqlTraceServiceWhere(service, namespace, environment),
		graphqlSpanAttrFilter(),
	)
	return fmt.Sprintf(`
SELECT
  nullIf(SpanAttributes[%s], '') AS op_name,
  lower(nullIf(SpanAttributes[%s], '')) AS op_type,
  count() / %g AS rate,
  avg(Duration / 1e9) AS avg_latency_s,
  countIf(%s) / nullIf(count(), 0) AS error_frac
FROM %s
WHERE %s
GROUP BY op_name, op_type
HAVING op_name != '' OR op_type != ''
ORDER BY rate DESC
LIMIT %d`,
		q(AttrGraphQLOperationName),
		q(AttrGraphQLOperationType),
		window,
		TraceStatusErrorFilter(),
		table(queries.CHTableTraces),
		where,
		limit,
	)
}

// GraphQLMetricDiscoverSQL lists GraphQL-related MetricNames across CH metric tables.
func GraphQLMetricDiscoverSQL(service, namespace string, from, to time.Time) string {
	svc := ScrapeServiceFilter(service, namespace)
	tf := MetricTimeFilter(from, to)
	pat := q(GraphQLMetricNameMatch)
	return fmt.Sprintf(`
SELECT MetricName, kind FROM (
  SELECT MetricName, 'sum' AS kind
  FROM %s
  WHERE match(MetricName, %s) AND %s AND (%s)
  GROUP BY MetricName
  UNION ALL
  SELECT MetricName, 'summary' AS kind
  FROM %s
  WHERE match(MetricName, %s) AND %s AND (%s)
  GROUP BY MetricName
  UNION ALL
  SELECT MetricName, 'gauge' AS kind
  FROM %s
  WHERE match(MetricName, %s) AND %s AND (%s)
  GROUP BY MetricName
  UNION ALL
  SELECT MetricName, 'histogram' AS kind
  FROM %s
  WHERE match(MetricName, %s) AND %s AND (%s)
  GROUP BY MetricName
)
ORDER BY MetricName
LIMIT 200`,
		table(queries.CHTableMetricsSum), pat, tf, svc,
		table(queries.CHTableMetricsSummary), pat, tf, svc,
		table(queries.CHTableMetricsGauge), pat, tf, svc,
		table(queries.CHTableMetricsHistogram), pat, tf, svc,
	)
}

// GraphQLMetricOpsSQL aggregates rate + avg latency from Prom-style count/sum
// MetricNames in otel_metrics_sum (cumulative counters → Δ/window).
// errorFilter is optional Prom-style `label="value"` / `label!="value"`.
func GraphQLMetricOpsSQL(
	service, namespace, countMetric, sumMetric, opLabel, typeLabel, errorFilter string,
	from, to time.Time,
	limit int,
) string {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	svc := ScrapeServiceFilter(service, namespace)
	tf := MetricTimeFilter(from, to)
	window := WindowSeconds(from, to)

	nameExpr := fmt.Sprintf("Attributes[%s]", q(opLabel))
	typeExpr := "''"
	groupBy := "op_name"
	joinExtra := ""
	if typeLabel != "" {
		typeExpr = fmt.Sprintf("lower(Attributes[%s])", q(typeLabel))
		groupBy = "op_name, op_type"
		joinExtra = " AND c.op_type = s.op_type"
	}

	errSelect := "CAST(NULL AS Nullable(Float64)) AS error_frac"
	errJoin := ""
	if attrPred := promLabelFilterToCH(errorFilter); attrPred != "" {
		errSelect = "e.error_delta / nullIf(c.count_delta, 0) AS error_frac"
		errOn := "c.op_name = e.op_name"
		if typeLabel != "" {
			errOn += " AND c.op_type = e.op_type"
		}
		errJoin = fmt.Sprintf(`
LEFT JOIN (
  SELECT
    %s AS op_name,
    %s AS op_type,
    greatest(max(Value) - min(Value), 0) AS error_delta
  FROM %s
  WHERE MetricName = %s AND %s AND (%s) AND (%s)
  GROUP BY %s
) AS e ON %s`,
			nameExpr, typeExpr,
			table(queries.CHTableMetricsSum),
			q(countMetric), tf, svc, attrPred, groupBy, errOn,
		)
	}

	return fmt.Sprintf(`
SELECT
  c.op_name AS op_name,
  c.op_type AS op_type,
  c.count_delta / %g AS rate,
  if(c.count_delta > 0, s.sum_delta / c.count_delta, 0) AS avg_latency,
  %s
FROM (
  SELECT
    %s AS op_name,
    %s AS op_type,
    greatest(max(Value) - min(Value), 0) AS count_delta
  FROM %s
  WHERE MetricName = %s AND %s AND (%s)
  GROUP BY %s
) AS c
LEFT JOIN (
  SELECT
    %s AS op_name,
    %s AS op_type,
    greatest(max(Value) - min(Value), 0) AS sum_delta
  FROM %s
  WHERE MetricName = %s AND %s AND (%s)
  GROUP BY %s
) AS s ON c.op_name = s.op_name%s
%s
WHERE c.op_name != '' AND c.count_delta > 0
ORDER BY rate DESC
LIMIT %d`,
		window,
		errSelect,
		nameExpr, typeExpr, table(queries.CHTableMetricsSum), q(countMetric), tf, svc, groupBy,
		nameExpr, typeExpr, table(queries.CHTableMetricsSum), q(sumMetric), tf, svc, groupBy,
		joinExtra,
		errJoin,
		limit,
	)
}

// promLabelFilterToCH converts a simple Prom label filter `key="value"` or
// `key!="value"` into a ClickHouse Attributes predicate. Returns "" if unparseable.
func promLabelFilterToCH(filter string) string {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return ""
	}
	neg := false
	var key, val string
	switch {
	case strings.Contains(filter, "!="):
		neg = true
		parts := strings.SplitN(filter, "!=", 2)
		if len(parts) != 2 {
			return ""
		}
		key, val = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	case strings.Contains(filter, "="):
		parts := strings.SplitN(filter, "=", 2)
		if len(parts) != 2 {
			return ""
		}
		key, val = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	default:
		return ""
	}
	val = strings.Trim(val, `"`)
	if key == "" {
		return ""
	}
	if neg {
		return fmt.Sprintf("Attributes[%s] != %s", q(key), q(val))
	}
	return fmt.Sprintf("Attributes[%s] = %s", q(key), q(val))
}

// GraphQLPDLOpSQL rates a single pdl-style count/sum MetricName pair
// (graphql_{QueryName}_seconds_{count,sum}) with no operation label.
func GraphQLPDLOpSQL(service, namespace, opName, countMetric, sumMetric string, from, to time.Time) string {
	svc := ScrapeServiceFilter(service, namespace)
	tf := MetricTimeFilter(from, to)
	window := WindowSeconds(from, to)
	return fmt.Sprintf(`
SELECT
  %s AS op_name,
  'query' AS op_type,
  greatest(maxIf(Value, MetricName = %s) - minIf(Value, MetricName = %s), 0) / %g AS rate,
  if(
    greatest(maxIf(Value, MetricName = %s) - minIf(Value, MetricName = %s), 0) > 0,
    greatest(maxIf(Value, MetricName = %s) - minIf(Value, MetricName = %s), 0)
      / greatest(maxIf(Value, MetricName = %s) - minIf(Value, MetricName = %s), 0),
    0
  ) AS avg_latency,
  CAST(NULL AS Nullable(Float64)) AS error_frac
FROM %s
WHERE MetricName IN (%s, %s) AND %s AND (%s)
LIMIT 1`,
		q(opName),
		q(countMetric), q(countMetric), window,
		q(countMetric), q(countMetric),
		q(sumMetric), q(sumMetric),
		q(countMetric), q(countMetric),
		table(queries.CHTableMetricsSum),
		q(countMetric), q(sumMetric), tf, svc,
	)
}

// ParseGraphQLMetricNames returns discovered MetricName set.
func ParseGraphQLMetricNames(rows []map[string]any) map[string]bool {
	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		name := ParseString(r["MetricName"])
		if name == "" {
			name = ParseString(r["metricname"])
		}
		if name != "" {
			out[name] = true
		}
	}
	return out
}

// ParseGraphQLOpRows maps CH aggregation rows into GraphQLOpRow.
// Latency prefers avg_latency_s (spans, seconds); falls back to avg_latency (metrics).
// error_frac (0–1) is converted to a 0–100 percentage.
func ParseGraphQLOpRows(rows []map[string]any, latencyUnit string) []GraphQLOpRow {
	out := make([]GraphQLOpRow, 0, len(rows))
	for _, r := range rows {
		name := ParseString(r["op_name"])
		opType := strings.ToLower(ParseString(r["op_type"]))
		rate := ParseFloat(r["rate"])
		if name == "" && opType == "" {
			continue
		}
		if name == "" {
			name = opType
		}
		if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
			continue
		}
		lat := ParseFloat(r["avg_latency_s"])
		if lat == 0 {
			lat = ParseFloat(r["avg_latency"])
		}
		if math.IsNaN(lat) || math.IsInf(lat, 0) {
			lat = 0
		}
		row := GraphQLOpRow{
			Name:        name,
			Type:        opType,
			Rate:        math.Round(rate*1000) / 1000,
			AvgLatency:  lat,
			LatencyUnit: latencyUnit,
		}
		if _, ok := r["error_frac"]; ok && r["error_frac"] != nil {
			er := ParseFloat(r["error_frac"])
			if !math.IsNaN(er) && !math.IsInf(er, 0) {
				pct := math.Min(math.Round(er*10000)/100, 100)
				row.ErrorRate = pct
				row.HasErrorRate = true
			}
		}
		out = append(out, row)
	}
	return out
}
