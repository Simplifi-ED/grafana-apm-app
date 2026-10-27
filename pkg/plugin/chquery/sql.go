// Package chquery builds ClickHouse SQL for the ClickStack-backed APM path.
// Handlers call these builders then execute via queries.ClickHouseClient.
package chquery

import (
	"fmt"
	"strings"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// Verified metric / attribute names for this deployment's ClickStack schema
// (docs/clickstack-migration/schema.md, inspected 2026-10-27).
const (
	MetricSpanCalls     = "traces.spanmetrics.calls"
	MetricSpanDuration  = "traces.spanmetrics.duration"
	MetricSGRequest     = "traces_service_graph_request_total"
	MetricSGFailed      = "traces_service_graph_request_failed_total"
	AttrServiceName     = "service.name"
	AttrSpanKind        = "span.kind"
	AttrSpanName        = "span.name"
	AttrStatusCode      = "status.code"
	AttrHTTPMethod      = "http.method"
	AttrHTTPRoute       = "http.route"
	AttrRPCService      = "rpc.service"
	AttrRPCMethod       = "rpc.method"
	AttrDBSystem        = "db.system"
	AttrDBOperation     = "db.operation.name"
	AttrMessagingSystem = "messaging.system"
	StatusError         = "STATUS_CODE_ERROR"
	SpanKindServer      = "SPAN_KIND_SERVER" // spanmetrics Attributes['span.kind']
	SpanKindClient      = "SPAN_KIND_CLIENT"
	// otel_traces.SpanKind uses short enum labels (Server/Client), not SPAN_KIND_*.
	TraceSpanKindServer = "Server"
	TraceSpanKindClient = "Client"
	ResCluster          = "cluster"
	ResServiceNamespace = "service.namespace"
)

// TraceServerKindFilter matches SERVER spans in otel_traces across both label styles.
func TraceServerKindFilter() string {
	return fmt.Sprintf("SpanKind IN (%s, %s)", q(TraceSpanKindServer), q(SpanKindServer))
}

// TimeBounds returns epoch-ms bounds for SQL literals.
func TimeBounds(from, to time.Time) (fromMs, toMs int64) {
	return from.UnixMilli(), to.UnixMilli()
}

// WindowSeconds is the query window length used to convert call counts → rates.
func WindowSeconds(from, to time.Time) float64 {
	s := to.Sub(from).Seconds()
	if s < 1 {
		return 1
	}
	return s
}

func q(s string) string { return queries.QuoteClickHouseString(s) }

func table(name string) string {
	t, err := queries.QualifiedTable(queries.CHDatabaseDefault, name)
	if err != nil {
		return "default.`" + name + "`"
	}
	return t
}

// ServiceExpr prefers Attributes['service.name'], falls back to ServiceName column.
func ServiceExpr(attrCol string) string {
	return fmt.Sprintf("coalesce(nullIf(%s['%s'], ''), ServiceName)", attrCol, AttrServiceName)
}

// FilterClause builds AND-joined filters for namespace / environment / service list.
// environment matches ResourceAttributes['cluster'] (verified HyperDX/OTel layout).
func FilterClause(namespace, environment, service string, resourceCol string) string {
	var parts []string
	if service != "" {
		parts = append(parts, fmt.Sprintf("(%s = %s OR ServiceName = %s)",
			ServiceExpr("Attributes"), q(service), q(service)))
	}
	if namespace != "" {
		parts = append(parts, fmt.Sprintf("%s['%s'] = %s", resourceCol, ResServiceNamespace, q(namespace)))
	}
	if environment != "" {
		envs := strings.Split(environment, ",")
		quoted := make([]string, 0, len(envs))
		for _, e := range envs {
			e = strings.TrimSpace(e)
			if e != "" {
				quoted = append(quoted, q(e))
			}
		}
		if len(quoted) == 1 {
			parts = append(parts, fmt.Sprintf("%s['%s'] = %s", resourceCol, ResCluster, quoted[0]))
		} else if len(quoted) > 1 {
			parts = append(parts, fmt.Sprintf("%s['%s'] IN (%s)", resourceCol, ResCluster, strings.Join(quoted, ", ")))
		}
	}
	if len(parts) == 0 {
		return "1"
	}
	return strings.Join(parts, " AND ")
}

// ParseFloat extracts a float from a ClickHouse frame cell.
func ParseFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int64:
		return float64(t)
	case int:
		return float64(t)
	case string:
		var f float64
		_, _ = fmt.Sscanf(t, "%f", &f)
		return f
	default:
		return 0
	}
}

// ParseString extracts a string from a frame cell.
func ParseString(v any) string {
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

// ParseInt64 extracts an int64 from a frame cell.
func ParseInt64(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case string:
		var n int64
		_, _ = fmt.Sscanf(t, "%d", &n)
		return n
	default:
		return 0
	}
}
