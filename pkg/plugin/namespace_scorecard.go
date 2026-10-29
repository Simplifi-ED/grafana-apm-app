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
	// Slightly tighter than issues-attention — each service runs a full scorecard.
	maxNamespaceScorecardServices = 20
	namespaceScorecardConcurrency = 4
)

// NamespaceServiceScorecard is one service's readiness inside a namespace rollup.
type NamespaceServiceScorecard struct {
	Service     string `json:"service"`
	Environment string `json:"environment,omitempty"`
	Score       int    `json:"score"`
	Total       int    `json:"total"`
}

// NamespaceScorecardResponse is GET /namespaces/{namespace}/scorecard.
// ClickStack-native team rollup (namespace ≈ team).
type NamespaceScorecardResponse struct {
	Score     int                         `json:"score"`
	Total     int                         `json:"total"`
	Services  []NamespaceServiceScorecard `json:"services"`
	Scanned   int                         `json:"scanned"`
	Truncated bool                        `json:"truncated,omitempty"`
}

// handleNamespaceScorecard aggregates per-service scorecards for a namespace.
// GET /namespaces/{namespace}/scorecard?environment=
func (a *App) handleNamespaceScorecard(w http.ResponseWriter, req *http.Request) {
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
	caller := ""
	if a.resolveServiceToken(ctx) == "" {
		caller = req.Header.Get("X-Grafana-Id")
		if caller == "" {
			caller = req.Header.Get("X-Grafana-User")
		}
	}
	ck := cacheKey("ns-scorecard", orgID, caller, namespace, env, roundedUnix(from), roundedUnix(to))
	cache := a.scorecardResponseCache()
	if cached, ok := cache.get(ck); ok {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		_, _ = w.Write(cached)
		return
	}
	data, err := cache.getOrCompute(ck, func() (any, error) {
		return a.computeNamespaceScorecard(ctx, req, namespace, env, from, to), nil
	})
	if err != nil {
		http.Error(w, "computing namespace scorecard failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func (a *App) computeNamespaceScorecard(
	ctx context.Context,
	req *http.Request,
	namespace, env string,
	from, to time.Time,
) NamespaceScorecardResponse {
	refs, truncated := a.namespaceServiceRefs(ctx, req, namespace, env, from, to, maxNamespaceScorecardServices)
	out := NamespaceScorecardResponse{
		Services:  make([]NamespaceServiceScorecard, 0, len(refs)),
		Truncated: truncated,
		Scanned:   len(refs),
	}
	if len(refs) == 0 {
		return out
	}

	type result struct {
		ref          namespaceServiceRef
		score, total int
	}
	results := make([]result, 0, len(refs))
	var mu sync.Mutex
	sem := make(chan struct{}, namespaceScorecardConcurrency)
	var wg sync.WaitGroup
	headers := req.Header.Clone()

	for _, r := range refs {
		wg.Add(1)
		go func(ref namespaceServiceRef) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			probeEnv := probeEnvironment(env, ref.Env)
			readiness := a.computeScorecard(ctx, headers, namespace, ref.Name, probeEnv).Readiness
			mu.Lock()
			results = append(results, result{ref: ref, score: readiness.Score, total: readiness.Total})
			mu.Unlock()
		}(r)
	}
	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		fi, fj := 0.0, 0.0
		if results[i].total > 0 {
			fi = float64(results[i].score) / float64(results[i].total)
		}
		if results[j].total > 0 {
			fj = float64(results[j].score) / float64(results[j].total)
		}
		if fi != fj {
			return fi < fj // weakest first — attention order
		}
		if results[i].ref.Name != results[j].ref.Name {
			return results[i].ref.Name < results[j].ref.Name
		}
		return results[i].ref.Env < results[j].ref.Env
	})

	for _, r := range results {
		out.Score += r.score
		out.Total += r.total
		out.Services = append(out.Services, NamespaceServiceScorecard{
			Service:     r.ref.Name,
			Environment: r.ref.Env,
			Score:       r.score,
			Total:       r.total,
		})
	}
	return out
}
