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
	StatusError         = "STATUS_CODE_ERROR" // spanmetrics Attributes['status.code']
	StatusOK            = "STATUS_CODE_OK"
	StatusUnset         = "STATUS_CODE_UNSET"
	SpanKindServer      = "SPAN_KIND_SERVER" // spanmetrics Attributes['span.kind']
	SpanKindClient      = "SPAN_KIND_CLIENT"
	// otel_traces.SpanKind / StatusCode use short enum labels, not SPAN_KIND_* / STATUS_CODE_*.
	TraceSpanKindServer = "Server"
	TraceSpanKindClient = "Client"
	TraceStatusError    = "Error"
	TraceStatusOK       = "Ok"
	TraceStatusUnset    = "Unset"
	ResCluster          = "cluster"
	ResServiceNamespace = "service.namespace"
	ResK8sNamespace     = "k8s.namespace.name"
	ResSDKLanguage      = "telemetry.sdk.language"
	ColTimeUnix         = "TimeUnix"  // metrics tables
	ColTimestamp        = "Timestamp" // traces / logs / sessions
)

// MetricSpanKindServerFilter matches SERVER in spanmetrics Attributes.
func MetricSpanKindServerFilter() string {
	return fmt.Sprintf("Attributes['%s'] = %s", AttrSpanKind, q(SpanKindServer))
}

// MetricSpanKindClientFilter matches CLIENT in spanmetrics Attributes.
func MetricSpanKindClientFilter() string {
	return fmt.Sprintf("Attributes['%s'] = %s", AttrSpanKind, q(SpanKindClient))
}

// MetricSpanKindFilter matches a spanmetrics Attributes['span.kind'] value.
func MetricSpanKindFilter(kind string) string {
	return fmt.Sprintf("Attributes['%s'] = %s", AttrSpanKind, q(kind))
}

// MetricStatusErrorFilter matches ERROR in spanmetrics Attributes.
func MetricStatusErrorFilter() string {
	return fmt.Sprintf("Attributes['%s'] = %s", AttrStatusCode, q(StatusError))
}

// TraceServerKindFilter matches SERVER spans in otel_traces across both label styles.
func TraceServerKindFilter() string {
	return fmt.Sprintf("SpanKind IN (%s, %s)", q(TraceSpanKindServer), q(SpanKindServer))
}

// TraceStatusErrorFilter matches error spans; dual-encodes short + STATUS_CODE_* defensively.
func TraceStatusErrorFilter() string {
	return fmt.Sprintf("StatusCode IN (%s, %s)", q(TraceStatusError), q(StatusError))
}

// TraceStatusNotErrorFilter excludes error spans (ok + unset).
func TraceStatusNotErrorFilter() string {
	return fmt.Sprintf("StatusCode NOT IN (%s, %s)", q(TraceStatusError), q(StatusError))
}

// TimeBounds returns epoch-ms bounds for SQL literals.
func TimeBounds(from, to time.Time) (fromMs, toMs int64) {
	return from.UnixMilli(), to.UnixMilli()
}

// MetricTimeFilter bounds metrics rows on TimeUnix (DateTime64) via fromUnixTimestamp64Milli.
func MetricTimeFilter(from, to time.Time) string {
	fromMs, toMs := TimeBounds(from, to)
	return fmt.Sprintf("%s >= fromUnixTimestamp64Milli(%d) AND %s <= fromUnixTimestamp64Milli(%d)",
		ColTimeUnix, fromMs, ColTimeUnix, toMs)
}

// TimestampFilter bounds traces/logs/sessions on Timestamp (DateTime64).
func TimestampFilter(from, to time.Time) string {
	fromMs, toMs := TimeBounds(from, to)
	return fmt.Sprintf("%s >= fromUnixTimestamp64Milli(%d) AND %s <= fromUnixTimestamp64Milli(%d)",
		ColTimestamp, fromMs, ColTimestamp, toMs)
}

// BucketMsExpr buckets a DateTime64 column into epoch-ms for sparkline series.
func BucketMsExpr(col string, stepSec int) string {
	if stepSec <= 0 {
		stepSec = 60
	}
	return fmt.Sprintf("toUnixTimestamp(toStartOfInterval(%s, INTERVAL %d SECOND)) * 1000", col, stepSec)
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

// NamespaceDisplayExpr prefers k8s.namespace.name over OTel service.namespace
// (product labels like "sanad" break scrape/runtime filters that expect k8s ns).
func NamespaceDisplayExpr(resourceCol string) string {
	return fmt.Sprintf(
		"coalesce(nullIf(%s['%s'], ''), nullIf(%s['%s'], ''))",
		resourceCol, ResK8sNamespace, resourceCol, ResServiceNamespace,
	)
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
		// Prefer k8s deployment ns; Sanad also sets OTel service.namespace=product
		// ("sanad") which is not useful for inventory filters / scrape joins.
		parts = append(parts, fmt.Sprintf(
			"(%s['%s'] = %s OR %s['%s'] = %s)",
			resourceCol, ResK8sNamespace, q(namespace),
			resourceCol, ResServiceNamespace, q(namespace),
		))
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
