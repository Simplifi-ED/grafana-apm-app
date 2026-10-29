package plugin

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
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
	Environment          string `json:"environment,omitempty"`
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

// namespaceServiceRef is one inventory row to probe (name + optional env).
type namespaceServiceRef struct {
	Name string
	Env  string
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
	// When probes forward caller auth (no SA token), include caller identity so
	// per-user datasource permissions cannot leak via a shared cache entry.
	caller := ""
	if a.resolveServiceToken(ctx) == "" {
		caller = req.Header.Get("X-Grafana-Id")
		if caller == "" {
			caller = req.Header.Get("X-Grafana-User")
		}
	}
	ck := cacheKey("ns-issues-attention", orgID, caller, namespace, env, roundedUnix(from), roundedUnix(to))
	a.writeCached(w, ck, "computing namespace issues attention failed", func() (any, error) {
		return a.computeNamespaceIssuesAttention(ctx, req, namespace, env, from, to)
	})
}

func (a *App) computeNamespaceIssuesAttention(
	ctx context.Context,
	req *http.Request,
	namespace, env string,
	from, to time.Time,
) (NamespaceIssuesAttentionResponse, error) {
	refs, truncated := a.namespaceServiceRefs(ctx, req, namespace, env, from, to, maxNamespaceAttentionServices)
	if len(refs) == 0 {
		return NamespaceIssuesAttentionResponse{ByService: []NamespaceServiceAttention{}}, nil
	}

	type result struct {
		ref namespaceServiceRef
		att IssuesAttention
	}
	results := make([]result, 0, len(refs))
	var mu sync.Mutex
	sem := make(chan struct{}, namespaceAttentionConcurrency)
	var wg sync.WaitGroup
	var firstErr error
	okCount := 0

	for _, r := range refs {
		wg.Add(1)
		go func(ref namespaceServiceRef) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			probeEnv := probeEnvironment(env, ref.Env)
			att, err := a.probeServiceIssuesAttention(ctx, req, namespace, ref.Name, probeEnv, from, to)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			okCount++
			if att.New == 0 && att.ReturnedAfterResolve == 0 {
				return
			}
			results = append(results, result{ref: ref, att: att})
		}(r)
	}
	wg.Wait()

	// Do not cache an all-zero rollup when every probe failed — that would
	// hide outages for the cache TTL.
	if okCount == 0 && firstErr != nil {
		return NamespaceIssuesAttentionResponse{}, firstErr
	}

	sort.Slice(results, func(i, j int) bool {
		si := results[i].att.New + results[i].att.ReturnedAfterResolve
		sj := results[j].att.New + results[j].att.ReturnedAfterResolve
		if si != sj {
			return si > sj
		}
		if results[i].ref.Name != results[j].ref.Name {
			return results[i].ref.Name < results[j].ref.Name
		}
		return results[i].ref.Env < results[j].ref.Env
	})

	out := NamespaceIssuesAttentionResponse{
		ByService: make([]NamespaceServiceAttention, 0, len(results)),
		Truncated: truncated,
		Scanned:   len(refs),
	}
	for _, r := range results {
		out.Attention.New += r.att.New
		out.Attention.ReturnedAfterResolve += r.att.ReturnedAfterResolve
		out.ByService = append(out.ByService, NamespaceServiceAttention{
			Service:              r.ref.Name,
			Environment:          r.ref.Env,
			New:                  r.att.New,
			ReturnedAfterResolve: r.att.ReturnedAfterResolve,
		})
	}
	return out, nil
}

// probeEnvironment prefers a single request filter; otherwise uses the inventory row's env
// so multi-env / empty filters still hit the correct per-environment Loki UID.
func probeEnvironment(requestEnv, rowEnv string) string {
	requestEnv = strings.TrimSpace(requestEnv)
	if requestEnv == "" {
		return rowEnv
	}
	if !strings.Contains(requestEnv, ",") {
		return requestEnv
	}
	if rowEnv != "" {
		return rowEnv
	}
	return requestEnv
}

func (a *App) namespaceServiceRefs(
	ctx context.Context,
	req *http.Request,
	namespace, env string,
	from, to time.Time,
	cap int,
) (refs []namespaceServiceRef, truncated bool) {
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
	return capNamespaceServiceRefs(summaries, cap)
}

// capNamespaceServiceRefs sorts by rate desc, dedupes name+env, truncates to cap.
func capNamespaceServiceRefs(summaries []queries.ServiceSummary, cap int) (refs []namespaceServiceRef, truncated bool) {
	sorted := append([]queries.ServiceSummary(nil), summaries...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Rate != sorted[j].Rate {
			return sorted[i].Rate > sorted[j].Rate
		}
		if sorted[i].Name != sorted[j].Name {
			return sorted[i].Name < sorted[j].Name
		}
		return sorted[i].Environment < sorted[j].Environment
	})
	seen := make(map[string]bool, len(sorted))
	for _, s := range sorted {
		if s.Name == "" {
			continue
		}
		key := s.Name + "\x00" + s.Environment
		if seen[key] {
			continue
		}
		seen[key] = true
		refs = append(refs, namespaceServiceRef{Name: s.Name, Env: s.Environment})
	}
	if cap > 0 && len(refs) > cap {
		return refs[:cap], true
	}
	return refs, false
}

// capNamespaceServiceNames sorts by rate desc, dedupes by name, truncates to cap.
func capNamespaceServiceNames(summaries []queries.ServiceSummary, cap int) (names []string, truncated bool) {
	refs, truncated := capNamespaceServiceRefs(summaries, 0)
	seen := make(map[string]bool, len(refs))
	for _, r := range refs {
		if seen[r.Name] {
			continue
		}
		seen[r.Name] = true
		names = append(names, r.Name)
	}
	if cap > 0 && len(names) > cap {
		return names[:cap], true
	}
	return names, truncated
}

func (a *App) probeServiceIssuesAttention(
	ctx context.Context,
	req *http.Request,
	namespace, service, env string,
	from, to time.Time,
) (IssuesAttention, error) {
	var resp IssuesResponse
	if a.clickStackMode() {
		ch := a.chForRequest(req)
		resp = a.fetchIssuesClickHouse(ctx, ch, service, env, namespace, from, to)
	} else {
		lokiUID := a.settings.LogsDataSource.Resolve(env).UID
		if lokiUID == "" {
			return IssuesAttention{}, fmt.Errorf("no Loki datasource for environment %q", env)
		}
		dsClient := queries.NewDsQueryClient(a.grafanaURL, a.resolveServiceToken(ctx)).WithAuthHeaders(req.Header)
		resp = a.queryIssues(ctx, dsClient, lokiUID, service, env, namespace, from, to, browserFacets{})
	}
	if resp.Unavailable {
		return IssuesAttention{}, fmt.Errorf("issues unavailable for service %q", service)
	}
	if len(resp.Issues) == 0 {
		return IssuesAttention{}, nil
	}
	var prev map[string]bool
	if span := to.Sub(from); span > 0 {
		var baseline IssuesResponse
		if a.clickStackMode() {
			baseline = a.fetchIssuesClickHouse(ctx, a.chForRequest(req), service, env, namespace, from.Add(-span), from)
		} else {
			lokiUID := a.settings.LogsDataSource.Resolve(env).UID
			dsClient := queries.NewDsQueryClient(a.grafanaURL, a.resolveServiceToken(ctx)).WithAuthHeaders(req.Header)
			baseline = a.queryIssues(ctx, dsClient, lokiUID, service, env, namespace, from.Add(-span), from, browserFacets{})
		}
		if len(baseline.Issues) > 0 {
			prev = issueFingerprintSet(baseline.Issues)
		}
	}
	a.attachIssuesNotifyFlags(ctx, namespace, service, env, from, to, &resp, prev)
	if resp.Attention == nil {
		return IssuesAttention{}, nil
	}
	return *resp.Attention, nil
}
