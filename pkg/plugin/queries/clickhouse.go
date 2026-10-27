package queries

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

const (
	// ClickHouseDatasourceType is the Grafana plugin id for the official
	// ClickHouse datasource (grafana-clickhouse-datasource).
	ClickHouseDatasourceType = "grafana-clickhouse-datasource"

	// ClickHouseFormatTable asks the datasource for tabular data frames
	// (format=1). Time-series format is 2; this foundation uses table mode.
	ClickHouseFormatTable = 1

	// DefaultClickHouseLimit is applied when a query has no LIMIT clause.
	DefaultClickHouseLimit = 1000

	// MaxClickHouseLimit caps any LIMIT (existing or appended).
	MaxClickHouseLimit = 10000

	// clickHouseMaxResponseBytes bounds how much of a /api/ds/query body we read.
	clickHouseMaxResponseBytes = 20 << 20 // 20 MiB
)

// Verified ClickStack / OpenTelemetry ClickHouse table names (database: default).
// Confirmed against the deployed ClickStack instance on 2026-10-27 — see
// docs/clickstack-migration/schema.md.
const (
	CHTableLogs                = "otel_logs"
	CHTableTraces              = "otel_traces"
	CHTableTracesTraceIDTS     = "otel_traces_trace_id_ts"
	CHTableMetricsGauge        = "otel_metrics_gauge"
	CHTableMetricsSum          = "otel_metrics_sum"
	CHTableMetricsHistogram    = "otel_metrics_histogram"
	CHTableMetricsExpHistogram = "otel_metrics_exponential_histogram"
	CHTableMetricsSummary      = "otel_metrics_summary"
	CHTableSessions            = "hyperdx_sessions"
	CHDatabaseDefault          = "default"
)

var (
	limitClauseRe = regexp.MustCompile(`(?i)\bLIMIT\s+(\d+)\b`)
	identRe       = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// ClickHouseClient executes SQL against a grafana-clickhouse-datasource via
// Grafana's POST /api/ds/query API. Credentials stay in Grafana's datasource
// config — this client never stores ClickHouse passwords.
type ClickHouseClient struct {
	grafanaURL   string
	serviceToken string
	dsUID        string
	httpClient   *http.Client
	headers      http.Header
	logger       log.Logger
	defaultLimit int
	maxLimit     int
}

// NewClickHouseClient builds a client bound to a single datasource UID.
// grafanaURL is the internal Grafana base (e.g. http://localhost:3000).
func NewClickHouseClient(grafanaURL, serviceToken, dsUID string) *ClickHouseClient {
	return &ClickHouseClient{
		grafanaURL:   strings.TrimRight(grafanaURL, "/"),
		serviceToken: serviceToken,
		dsUID:        dsUID,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: sharedTransport,
		},
		logger:       log.DefaultLogger.With("component", "clickhouseClient", "dsUID", dsUID),
		defaultLimit: DefaultClickHouseLimit,
		maxLimit:     MaxClickHouseLimit,
	}
}

// DatasourceUID returns the configured Grafana ClickHouse datasource UID.
func (c *ClickHouseClient) DatasourceUID() string {
	return c.dsUID
}

// WithAuthHeaders returns a shallow copy that forwards browser auth headers
// when no service account token is configured.
func (c *ClickHouseClient) WithAuthHeaders(h http.Header) *ClickHouseClient {
	cp := *c
	cp.headers = h
	return &cp
}

// WithServiceToken returns a shallow copy with an overridden service token.
func (c *ClickHouseClient) WithServiceToken(token string) *ClickHouseClient {
	cp := *c
	cp.serviceToken = token
	return &cp
}

// ClickHouseQueryOptions controls a single SQL execution.
type ClickHouseQueryOptions struct {
	// From/To populate the /api/ds/query time window (also used by $__timeFilter macros).
	From time.Time
	To   time.Time
	// Limit is the max rows. Zero uses DefaultClickHouseLimit. Values above
	// MaxClickHouseLimit are clamped.
	Limit int
	// SkipLimitEnforcement is for health probes that must not rewrite SQL
	// (e.g. SELECT 1). Prefer leaving this false for all data queries.
	SkipLimitEnforcement bool
}

// ClickHouseResult is a tabular view of one data frame from /api/ds/query.
type ClickHouseResult struct {
	Columns []string
	// Rows are column-aligned; Rows[i][j] is the value of column Columns[j]
	// in row i. Values are JSON-decoded (float64, string, bool, nil, nested).
	Rows [][]any
	// RawSQL is the SQL after limit enforcement (what was sent).
	RawSQL string
}

// RowCount returns the number of data rows.
func (r *ClickHouseResult) RowCount() int {
	if r == nil {
		return 0
	}
	return len(r.Rows)
}

// Maps returns rows as maps keyed by column name (convenient for tests/handlers).
func (r *ClickHouseResult) Maps() []map[string]any {
	if r == nil {
		return nil
	}
	out := make([]map[string]any, 0, len(r.Rows))
	for _, row := range r.Rows {
		m := make(map[string]any, len(r.Columns))
		for j, col := range r.Columns {
			if j < len(row) {
				m[col] = row[j]
			}
		}
		out = append(out, m)
	}
	return out
}

type chQueryRequest struct {
	From    string          `json:"from"`
	To      string          `json:"to"`
	Queries []chQueryTarget `json:"queries"`
}

type chQueryTarget struct {
	RefID      string       `json:"refId"`
	Datasource chQueryDsRef `json:"datasource"`
	RawSQL     string       `json:"rawSql"`
	Format     int          `json:"format"`
	QueryType  string       `json:"queryType,omitempty"`
}

type chQueryDsRef struct {
	UID  string `json:"uid"`
	Type string `json:"type"`
}

type chQueryFrame struct {
	Schema struct {
		Fields []struct {
			Name string `json:"name"`
			Type string `json:"type,omitempty"`
		} `json:"fields"`
	} `json:"schema"`
	Data struct {
		Values []json.RawMessage `json:"values"`
	} `json:"data"`
}

type chQueryResponse struct {
	Results map[string]struct {
		Error  string         `json:"error,omitempty"`
		Frames []chQueryFrame `json:"frames"`
	} `json:"results"`
}

// Query runs rawSQL against the configured ClickHouse datasource and returns
// tabular rows. The SQL is validated (no multi-statements) and a LIMIT is
// enforced unless SkipLimitEnforcement is set.
func (c *ClickHouseClient) Query(ctx context.Context, rawSQL string, opts ClickHouseQueryOptions) (*ClickHouseResult, error) {
	if c == nil {
		return nil, fmt.Errorf("clickhouse client is nil")
	}
	if c.dsUID == "" {
		return nil, fmt.Errorf("clickhouse datasource UID is not configured")
	}

	sql, err := PrepareClickHouseSQL(rawSQL, opts.Limit, c.defaultLimit, c.maxLimit, opts.SkipLimitEnforcement)
	if err != nil {
		return nil, err
	}

	from, to := opts.From, opts.To
	if to.IsZero() {
		to = time.Now()
	}
	if from.IsZero() {
		from = to.Add(-time.Hour)
	}
	if !from.Before(to) {
		return nil, fmt.Errorf("invalid time range: from (%s) must be before to (%s)", from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339))
	}

	c.logger.Debug("Executing ClickHouse query",
		"sqlPreview", truncate(sql, 200),
		"from", from.UTC().Format(time.RFC3339),
		"to", to.UTC().Format(time.RFC3339),
	)

	frames, err := c.execute(ctx, sql, from, to)
	if err != nil {
		c.logger.Warn("ClickHouse query failed", "error", err, "sqlPreview", truncate(sql, 200))
		return nil, err
	}

	result, err := framesToClickHouseResult(frames, sql)
	if err != nil {
		return nil, err
	}
	c.logger.Debug("ClickHouse query ok", "rows", result.RowCount(), "cols", len(result.Columns))
	return result, nil
}

// Ping verifies ClickHouse connectivity with a trivial SELECT.
func (c *ClickHouseClient) Ping(ctx context.Context) error {
	res, err := c.Query(ctx, "SELECT 1 AS ok", ClickHouseQueryOptions{
		SkipLimitEnforcement: true,
		From:                 time.Now().Add(-time.Minute),
		To:                   time.Now(),
	})
	if err != nil {
		return fmt.Errorf("clickhouse ping failed: %w", err)
	}
	if res.RowCount() == 0 {
		return fmt.Errorf("clickhouse ping returned no rows")
	}
	return nil
}

func (c *ClickHouseClient) execute(ctx context.Context, rawSQL string, from, to time.Time) ([]chQueryFrame, error) {
	body, err := json.Marshal(chQueryRequest{
		From: strconv.FormatInt(from.UnixMilli(), 10),
		To:   strconv.FormatInt(to.UnixMilli(), 10),
		Queries: []chQueryTarget{{
			RefID: "A",
			Datasource: chQueryDsRef{
				UID:  c.dsUID,
				Type: ClickHouseDatasourceType,
			},
			RawSQL:    rawSQL,
			Format:    ClickHouseFormatTable,
			QueryType: "sql",
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("marshaling clickhouse ds query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.grafanaURL+"/api/ds/query", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating clickhouse ds query request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("clickhouse query canceled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("executing clickhouse ds query: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	raw, err := io.ReadAll(io.LimitReader(resp.Body, clickHouseMaxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("reading clickhouse ds query response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("clickhouse ds query returned %d: %s", resp.StatusCode, truncate(string(raw), 512))
	}

	var envelope chQueryResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("unmarshaling clickhouse ds query response: %w", err)
	}
	result, ok := envelope.Results["A"]
	if !ok {
		return nil, fmt.Errorf("clickhouse ds query response missing refId A")
	}
	if result.Error != "" {
		return nil, fmt.Errorf("clickhouse query error: %s", result.Error)
	}
	return result.Frames, nil
}

func (c *ClickHouseClient) applyAuth(req *http.Request) {
	if c.serviceToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.serviceToken)
		if c.headers != nil {
			if vals := c.headers.Values("X-Grafana-Org-Id"); len(vals) > 0 {
				for _, v := range vals {
					req.Header.Add("X-Grafana-Org-Id", v)
				}
			}
		}
		return
	}
	if c.headers == nil {
		return
	}
	for _, key := range []string{"Cookie", "Authorization", "X-Grafana-Org-Id", "X-Grafana-Id"} {
		if vals := c.headers.Values(key); len(vals) > 0 {
			for _, v := range vals {
				req.Header.Add(key, v)
			}
		}
	}
}

func framesToClickHouseResult(frames []chQueryFrame, sql string) (*ClickHouseResult, error) {
	out := &ClickHouseResult{RawSQL: sql}
	if len(frames) == 0 {
		return out, nil
	}
	// Use the first non-empty frame; CH table queries typically return one.
	frame := frames[0]
	for _, f := range frames {
		if len(f.Schema.Fields) > 0 {
			frame = f
			break
		}
	}

	cols := make([]string, len(frame.Schema.Fields))
	for i, f := range frame.Schema.Fields {
		cols[i] = f.Name
	}
	out.Columns = cols

	if len(frame.Data.Values) == 0 || len(cols) == 0 {
		return out, nil
	}

	colValues := make([][]any, len(frame.Data.Values))
	rowCount := 0
	for i, raw := range frame.Data.Values {
		var vals []any
		if err := json.Unmarshal(raw, &vals); err != nil {
			return nil, fmt.Errorf("decoding clickhouse column %d (%q): %w", i, safeColName(cols, i), err)
		}
		colValues[i] = vals
		rowCount = max(rowCount, len(vals))
	}

	rows := make([][]any, rowCount)
	for r := range rowCount {
		row := make([]any, len(colValues))
		for c := range colValues {
			if r < len(colValues[c]) {
				row[c] = colValues[c][r]
			}
		}
		rows[r] = row
	}
	out.Rows = rows
	return out, nil
}

func safeColName(cols []string, i int) string {
	if i >= 0 && i < len(cols) {
		return cols[i]
	}
	return "?"
}

// PrepareClickHouseSQL validates and optionally enforces a LIMIT on rawSQL.
func PrepareClickHouseSQL(rawSQL string, requestedLimit, defaultLimit, maxLimit int, skipLimit bool) (string, error) {
	sql := strings.TrimSpace(rawSQL)
	if sql == "" {
		return "", fmt.Errorf("empty clickhouse SQL")
	}
	// Strip a single trailing semicolon; reject any other semicolons (multi-statement).
	sql = strings.TrimSuffix(sql, ";")
	sql = strings.TrimSpace(sql)
	if strings.Contains(sql, ";") {
		return "", fmt.Errorf("multi-statement clickhouse SQL is not allowed")
	}
	if looksLikeMutatingSQL(sql) {
		return "", fmt.Errorf("mutating clickhouse SQL is not allowed")
	}
	if skipLimit {
		return sql, nil
	}
	return enforceClickHouseLimit(sql, requestedLimit, defaultLimit, maxLimit), nil
}

func looksLikeMutatingSQL(sql string) bool {
	upper := strings.ToUpper(strings.TrimLeftFunc(sql, unicode.IsSpace))
	for _, prefix := range []string{
		"INSERT ", "ALTER ", "DROP ", "CREATE ", "TRUNCATE ", "DELETE ",
		"RENAME ", "ATTACH ", "DETACH ", "OPTIMIZE ", "SYSTEM ", "GRANT ",
		"REVOKE ", "SET ", "EXCHANGE ",
	} {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}
	return false
}

func enforceClickHouseLimit(sql string, requested, defaultLimit, maxLimit int) string {
	if defaultLimit <= 0 {
		defaultLimit = DefaultClickHouseLimit
	}
	if maxLimit <= 0 {
		maxLimit = MaxClickHouseLimit
	}
	limit := requested
	if limit <= 0 {
		limit = defaultLimit
	}
	limit = min(limit, maxLimit)

	if m := limitClauseRe.FindStringSubmatchIndex(sql); m != nil {
		existing, _ := strconv.Atoi(sql[m[2]:m[3]])
		if existing > maxLimit {
			return sql[:m[0]] + fmt.Sprintf("LIMIT %d", maxLimit) + sql[m[1]:]
		}
		return sql
	}
	return fmt.Sprintf("%s LIMIT %d", sql, limit)
}

// QuoteClickHouseString escapes a value for use as a ClickHouse string literal.
func QuoteClickHouseString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// QuoteClickHouseIdent wraps an identifier in backticks after validating it.
func QuoteClickHouseIdent(name string) (string, error) {
	if err := ValidateClickHouseIdent(name); err != nil {
		return "", err
	}
	return "`" + name + "`", nil
}

// ValidateClickHouseIdent rejects identifiers that are not simple SQL names.
func ValidateClickHouseIdent(name string) error {
	if name == "" {
		return fmt.Errorf("empty identifier")
	}
	if len(name) > 128 {
		return fmt.Errorf("identifier too long")
	}
	if !identRe.MatchString(name) {
		return fmt.Errorf("invalid identifier %q", name)
	}
	return nil
}

// QualifiedTable returns database.table with validated identifiers.
func QualifiedTable(database, table string) (string, error) {
	database = cmp.Or(database, CHDatabaseDefault)
	db, err := QuoteClickHouseIdent(database)
	if err != nil {
		return "", fmt.Errorf("database: %w", err)
	}
	tbl, err := QuoteClickHouseIdent(table)
	if err != nil {
		return "", fmt.Errorf("table: %w", err)
	}
	return db + "." + tbl, nil
}
