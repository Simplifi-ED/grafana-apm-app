package chquery

import (
	"fmt"
	"regexp"
	"strings"
)

// FieldFilter is one Nais-like attribute chip (key=value) applied to search.
type FieldFilter struct {
	Key   string
	Value string
}

// Curated log field keys → ClickHouse map column (service already scoped).
var logFieldMaps = map[string]string{
	"k8s.pod.name":       "ResourceAttributes",
	"k8s.namespace.name": "ResourceAttributes",
	"host.name":          "ResourceAttributes",
	"exception.type":     "LogAttributes",
	"exception.message":  "LogAttributes",
}

// Curated trace field keys → ClickHouse map column.
var traceFieldMaps = map[string]string{
	"http.route":         "SpanAttributes",
	"http.method":        "SpanAttributes",
	"http.status_code":   "SpanAttributes",
	"db.system":          "SpanAttributes",
	"db.operation.name":  "SpanAttributes",
	"rpc.method":         "SpanAttributes",
	"k8s.pod.name":       "ResourceAttributes",
	"k8s.namespace.name": "ResourceAttributes",
}

// attrKeyRe restricts chip keys to OTel-like dotted names.
var attrKeyRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._/-]{0,127}$`)

// sqlWhereSafeRe allows simple predicate characters only (no statements).
var sqlWhereSafeRe = regexp.MustCompile(`^[a-zA-Z0-9_\s\[\]'\.=<>!\(\),%*/+-]+$`)

var sqlWhereBanned = []string{
	";", "--", "/*", "*/",
	" drop ", " delete ", " insert ", " alter ", " create ",
	" truncate ", " grant ", " system ", " union ", " into ",
	" format ", " outfile ", " infile ", " attach ", " detach ",
	" kill ", " optimize ", " rename ", " exchange ",
}

// LogFieldKeys returns curated keys for the logs filter-chip picker.
func LogFieldKeys() []string {
	keys := make([]string, 0, len(logFieldMaps))
	for k := range logFieldMaps {
		keys = append(keys, k)
	}
	return keys
}

// TraceFieldKeys returns curated keys for the traces filter-chip picker.
func TraceFieldKeys() []string {
	keys := make([]string, 0, len(traceFieldMaps))
	for k := range traceFieldMaps {
		keys = append(keys, k)
	}
	return keys
}

// ParseFieldParams parses repeated field=key=value query values into chips.
// Invalid or unknown keys are skipped (optional filters).
func ParseFieldParams(raw []string, kind string) []FieldFilter {
	allow := logFieldMaps
	if kind == "trace" {
		allow = traceFieldMaps
	}
	out := make([]FieldFilter, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		key, value, ok := strings.Cut(strings.TrimSpace(item), "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" || !attrKeyRe.MatchString(key) {
			continue
		}
		if _, known := allow[key]; !known {
			continue
		}
		if len(value) > 256 {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, FieldFilter{Key: key, Value: value})
	}
	return out
}

// FieldFiltersSQL builds AND clauses for curated attribute chips.
func FieldFiltersSQL(fields []FieldFilter, kind string) string {
	allow := logFieldMaps
	if kind == "trace" {
		allow = traceFieldMaps
	}
	var parts []string
	for _, f := range fields {
		col, ok := allow[f.Key]
		if !ok || f.Value == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s[%s] = %s", col, q(f.Key), q(f.Value)))
	}
	return strings.Join(parts, " AND ")
}

// ValidateSQLWhere accepts a power-user WHERE fragment for the Advanced SQL hatch.
// Empty input is OK. Rejects multi-statement / DDL-ish input; returns sanitized fragment.
func ValidateSQLWhere(fragment string) (string, error) {
	fragment = strings.TrimSpace(fragment)
	if fragment == "" {
		return "", nil
	}
	if len(fragment) > 512 {
		return "", fmt.Errorf("sql where too long (max 512)")
	}
	if !sqlWhereSafeRe.MatchString(fragment) {
		return "", fmt.Errorf("sql where has disallowed characters")
	}
	lower := strings.ToLower(fragment)
	for _, bad := range []string{";", "--", "/*", "*/"} {
		if strings.Contains(lower, bad) {
			return "", fmt.Errorf("sql where contains disallowed token")
		}
	}
	padded := " " + lower + " "
	for _, bad := range sqlWhereBanned {
		if strings.HasPrefix(bad, " ") && strings.Contains(padded, bad) {
			return "", fmt.Errorf("sql where contains disallowed token")
		}
	}
	return fragment, nil
}
