package chquery

import (
	"strings"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// OperationsToEndpointGroups maps ClickHouse operation RED rows into the
// protocol-grouped shape the Backend / Endpoints UI expects.
func OperationsToEndpointGroups(ops []queries.OperationSummary) queries.EndpointGroups {
	groups := queries.EndpointGroups{
		HTTP:         make([]queries.EndpointSummary, 0),
		GRPC:         make([]queries.EndpointSummary, 0),
		Database:     make([]queries.EndpointSummary, 0),
		Messaging:    make([]queries.EndpointSummary, 0),
		Internal:     make([]queries.EndpointSummary, 0),
		Client:       make([]queries.EndpointSummary, 0),
		DurationUnit: "ms",
	}
	for _, op := range ops {
		ep := queries.EndpointSummary{
			SpanName:     op.SpanName,
			Rate:         op.Rate,
			ErrorRate:    op.ErrorRate,
			P50Duration:  op.P50Duration,
			P95Duration:  op.P95Duration,
			P99Duration:  op.P99Duration,
			DurationUnit: op.DurationUnit,
		}
		if ep.DurationUnit == "" {
			ep.DurationUnit = "ms"
		}
		switch op.SpanKindRaw {
		case SpanKindServer, TraceSpanKindServer:
			ep.HTTPMethod = op.HTTPMethod
			ep.HTTPRoute = op.HTTPRoute
			if op.RPCService != "" || op.RPCMethod != "" {
				ep.RPCService = op.RPCService
				ep.RPCMethod = op.RPCMethod
				groups.GRPC = append(groups.GRPC, ep)
			} else {
				method, route := splitHTTPName(op.SpanName)
				if ep.HTTPMethod == "" {
					ep.HTTPMethod = method
				}
				if ep.HTTPRoute == "" {
					ep.HTTPRoute = route
				}
				groups.HTTP = append(groups.HTTP, ep)
			}
		case SpanKindClient, TraceSpanKindClient:
			switch {
			case op.DBSystem != "" || looksLikeDBSpan(op.SpanName):
				ep.DBSystem = op.DBSystem
				groups.Database = append(groups.Database, ep)
			case op.MessagingSystem != "":
				ep.MessagingKind = "Client"
				groups.Messaging = append(groups.Messaging, ep)
			default:
				method, route := splitHTTPName(op.SpanName)
				ep.HTTPMethod = op.HTTPMethod
				ep.HTTPRoute = op.HTTPRoute
				if ep.HTTPMethod == "" {
					ep.HTTPMethod = method
				}
				if ep.HTTPRoute == "" {
					ep.HTTPRoute = route
				}
				groups.Client = append(groups.Client, ep)
			}
		case "SPAN_KIND_INTERNAL", "Internal":
			groups.Internal = append(groups.Internal, ep)
		case "SPAN_KIND_CONSUMER", "Consumer":
			ep.MessagingKind = "Consumer"
			groups.Messaging = append(groups.Messaging, ep)
		case "SPAN_KIND_PRODUCER", "Producer":
			ep.MessagingKind = "Producer"
			groups.Messaging = append(groups.Messaging, ep)
		default:
			// Unknown kind — keep visible under Internal rather than dropping.
			groups.Internal = append(groups.Internal, ep)
		}
	}
	return groups
}

func splitHTTPName(name string) (method, route string) {
	if idx := strings.IndexByte(name, ' '); idx > 0 && idx < 10 {
		candidate := name[:idx]
		switch candidate {
		case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "CONNECT", "TRACE":
			return candidate, strings.TrimSpace(name[idx+1:])
		}
	}
	switch name {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "CONNECT", "TRACE":
		return name, ""
	}
	// HyperDX browser client spans: "HTTP GET"
	if after, ok := strings.CutPrefix(name, "HTTP "); ok {
		rest := after
		if idx := strings.IndexByte(rest, ' '); idx > 0 {
			return rest[:idx], strings.TrimSpace(rest[idx+1:])
		}
		return rest, ""
	}
	return "", name
}

func looksLikeDBSpan(spanName string) bool {
	n := strings.ToLower(strings.TrimSpace(spanName))
	for _, p := range []string{
		"query", "prepare", "pool.", "exec ", "execute",
		"commit", "begin", "rollback", "evalsha",
		"get ", "set ", "del ", "hget", "hset", "exists",
	} {
		if n == strings.TrimSpace(p) || strings.HasPrefix(n, p) {
			return true
		}
	}
	return false
}
