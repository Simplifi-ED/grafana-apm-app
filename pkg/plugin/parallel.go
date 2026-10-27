package plugin

import (
	"context"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// QueryJob defines a named Prometheus instant query to run in parallel.
type QueryJob struct {
	Name  string
	Query string
}

// runInstantQueries executes multiple Prometheus instant queries in parallel
// and returns a map of query name → results. Failed queries are logged and
// omitted from the result map.
//
// ClickStack-primary installs have no Mimir client — return empty rather than
// panicking on a nil *PrometheusClient (was crashing the plugin process).
func (a *App) runInstantQueries(ctx context.Context, at time.Time, jobs []QueryJob, logger log.Logger) map[string][]queries.PromResult {
	client := a.prom(ctx)
	if client == nil {
		logger.Debug("skipping Prometheus instant queries — no metrics datasource")
		return map[string][]queries.PromResult{}
	}

	type queryResult struct {
		name    string
		results []queries.PromResult
		err     error
	}

	var wg sync.WaitGroup
	ch := make(chan queryResult, len(jobs))

	for _, job := range jobs {
		wg.Add(1)
		go func(n, query string) {
			defer wg.Done()
			results, err := client.InstantQuery(ctx, query, at)
			ch <- queryResult{name: n, results: results, err: err}
		}(job.Name, job.Query)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	resultMap := make(map[string][]queries.PromResult)
	for qr := range ch {
		if qr.err != nil {
			logger.Warn("Query failed", "query", qr.name, "error", qr.err)
			continue
		}
		resultMap[qr.name] = qr.results
	}
	return resultMap
}
