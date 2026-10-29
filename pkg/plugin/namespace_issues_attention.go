package plugin

import (
	"context"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

const (
	// maxNamespaceAttentionServices caps fan-out so a large namespace cannot
	// stampede ClickHouse / Loki with one /issues pair per service.
	maxNamespaceAttentionServices = 25
	// namespaceAttentionConcurrency bounds parallel per-service probes.
	namespaceAttentionConcurrency = 4
)

// NamespaceServiceAttention is per-service attention inside a namespace rollup.
type NamespaceServiceAttention struct {
	Service              string `json:"service"`
	New                  int    `json:"new"`
	ReturnedAfterResolve int    `json:"returnedAfterResolve"`
}

// NamespaceIssuesAttentionResponse is GET /namespaces/{namespace}/issues-attention.
type NamespaceIssuesAttentionResponse struct {
	Attention IssuesAttention             `json:"attention"`
	ByService []NamespaceServiceAttention `json:"byService"`
	// Truncated is true when the namespace had more services than the probe cap.
	Truncated bool `json:"truncated,omitempty"`
	Scanned   int  `json:"scanned"`
}

// handleNamespaceIssuesAttention aggregates new / returned-after-resolve issue
// counts across services in a namespace without the UI doing N+1 /issues calls.
// GET /namespaces/{namespace}/issues-attention?from=&to=&environment=
func (a *App) handleNamespaceIssuesAttention(w http.ResponseWriter, req *http.Request) {
	if !requireGET(w, req) {
		return
	}
	ctx := a.requestContext(req)
	namespace := queries.MustSanitizeLabel(req.PathValue("namespace"))
	if namespace == "" {
		http.Error(w, `{"error":"missing or invalid namespace"}`, http.StatusBadRequest)
		return
	}
	env := parseEnvironment(req)
	from, to := parseTimeRange(req)

	orgID := req.Header.Get("X-Grafana-Org-Id")
	ck := cacheKey("ns-issues-attention", orgID, namespace, env, roundedUnix(from), roundedUnix(to))
	a.writeCached(w, ck, "computing namespace issues attention failed", func() (any, error) {
		return a.computeNamespaceIssuesAttention(ctx, req, namespace, env, from, to), nil
	})
}

func (a *App) computeNamespaceIssuesAttention(
	ctx context.Context,
	req *http.Request,
	namespace, env string,
	from, to time.Time,
) NamespaceIssuesAttentionResponse {
	names, truncated := a.namespaceServiceNames(ctx, req, namespace, env, from, to)
	if len(names) == 0 {
		return NamespaceIssuesAttentionResponse{ByService: []NamespaceServiceAttention{}}
	}

	type result struct {
		svc string
		att IssuesAttention
	}
	results := make([]result, 0, len(names))
	var mu sync.Mutex
	sem := make(chan struct{}, namespaceAttentionConcurrency)
	var wg sync.WaitGroup

	for _, svc := range names {
		wg.Add(1)
		go func(service string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			att := a.probeServiceIssuesAttention(ctx, req, namespace, service, env, from, to)
			if att.New == 0 && att.ReturnedAfterResolve == 0 {
				return
			}
			mu.Lock()
			results = append(results, result{svc: service, att: att})
			mu.Unlock()
		}(svc)
	}
	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		si := results[i].att.New + results[i].att.ReturnedAfterResolve
		sj := results[j].att.New + results[j].att.ReturnedAfterResolve
		if si != sj {
			return si > sj
		}
		return results[i].svc < results[j].svc
	})

	out := NamespaceIssuesAttentionResponse{
		ByService: make([]NamespaceServiceAttention, 0, len(results)),
		Truncated: truncated,
		Scanned:   len(names),
	}
	for _, r := range results {
		out.Attention.New += r.att.New
		out.Attention.ReturnedAfterResolve += r.att.ReturnedAfterResolve
		out.ByService = append(out.ByService, NamespaceServiceAttention{
			Service:              r.svc,
			New:                  r.att.New,
			ReturnedAfterResolve: r.att.ReturnedAfterResolve,
		})
	}
	return out
}

func (a *App) namespaceServiceNames(
	ctx context.Context,
	req *http.Request,
	namespace, env string,
	from, to time.Time,
) (names []string, truncated bool) {
	step := 60 * time.Second
	var summaries []queries.ServiceSummary
	if a.clickStackMode() {
		summaries = a.fetchServiceSummariesClickHouse(ctx, a.chForRequest(req), from, to, step, false, namespace, env, "")
	} else {
		caps := a.cachedOrDetectCapabilities(ctx)
		if !caps.SpanMetrics.Detected {
			return nil, false
		}
		summaries = a.fetchServiceSummaries(ctx, caps, from, to, step, false, namespace, env, "", req.Header)
	}
	return capNamespaceServiceNames(summaries, maxNamespaceAttentionServices)
}

// capNamespaceServiceNames sorts by rate desc, dedupes, and truncates to cap.
func capNamespaceServiceNames(summaries []queries.ServiceSummary, cap int) (names []string, truncated bool) {
	sorted := append([]queries.ServiceSummary(nil), summaries...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Rate != sorted[j].Rate {
			return sorted[i].Rate > sorted[j].Rate
		}
		return sorted[i].Name < sorted[j].Name
	})
	seen := make(map[string]bool, len(sorted))
	for _, s := range sorted {
		if s.Name == "" || seen[s.Name] {
			continue
		}
		seen[s.Name] = true
		names = append(names, s.Name)
	}
	if cap > 0 && len(names) > cap {
		return names[:cap], true
	}
	return names, false
}

func (a *App) probeServiceIssuesAttention(
	ctx context.Context,
	req *http.Request,
	namespace, service, env string,
	from, to time.Time,
) IssuesAttention {
	var resp IssuesResponse
	if a.clickStackMode() {
		ch := a.chForRequest(req)
		resp = a.fetchIssuesClickHouse(ctx, ch, service, env, from, to)
	} else {
		lokiUID := a.settings.LogsDataSource.Resolve(env).UID
		if lokiUID == "" {
			return IssuesAttention{}
		}
		dsClient := queries.NewDsQueryClient(a.grafanaURL, a.resolveServiceToken(ctx)).WithAuthHeaders(req.Header)
		resp = a.queryIssues(ctx, dsClient, lokiUID, service, env, from, to, browserFacets{})
	}
	if len(resp.Issues) == 0 {
		return IssuesAttention{}
	}
	var prev map[string]bool
	if span := to.Sub(from); span > 0 {
		var baseline IssuesResponse
		if a.clickStackMode() {
			baseline = a.fetchIssuesClickHouse(ctx, a.chForRequest(req), service, env, from.Add(-span), from)
		} else {
			lokiUID := a.settings.LogsDataSource.Resolve(env).UID
			dsClient := queries.NewDsQueryClient(a.grafanaURL, a.resolveServiceToken(ctx)).WithAuthHeaders(req.Header)
			baseline = a.queryIssues(ctx, dsClient, lokiUID, service, env, from.Add(-span), from, browserFacets{})
		}
		if len(baseline.Issues) > 0 {
			prev = issueFingerprintSet(baseline.Issues)
		}
	}
	a.attachIssuesNotifyFlags(ctx, namespace, service, env, from, to, &resp, prev)
	if resp.Attention == nil {
		return IssuesAttention{}
	}
	return *resp.Attention
}
