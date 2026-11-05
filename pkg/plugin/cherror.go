package plugin

import (
	"net/url"
	"regexp"
	"strings"
)

// grafanaHopMarker prefixes transport failures between the plugin and Grafana
// (queries.ClickHouseClient.execute). Everything else came back from Grafana.
const grafanaHopMarker = "executing clickhouse ds query:"

var (
	dnsHostRe    = regexp.MustCompile(`lookup ([^\s:]+)`)
	dialHostRe   = regexp.MustCompile(`dial tcp ([^\s]+?): `)
	postURLRe    = regexp.MustCompile(`(?:Post|Get) "([^"]+)"`)
	httpStatusRe = regexp.MustCompile(`ds query returned (\d{3})`)
)

// summarizeClickHouseError turns datasource/SDK error blobs into a short
// user-facing reason. Raw Grafana /api/ds/query payloads include nested JSON
// and credentials placeholders; never surface those in the UI.
func summarizeClickHouseError(err error) string {
	if err == nil {
		return "unknown error"
	}
	s := err.Error()
	lower := strings.ToLower(s)

	if i := strings.Index(lower, grafanaHopMarker); i >= 0 {
		if msg := summarizeGrafanaHop(s[i+len(grafanaHopMarker):]); msg != "" {
			return msg
		}
	}

	status := ""
	if m := httpStatusRe.FindStringSubmatch(s); m != nil {
		status = m[1]
	}
	if status == "401" || status == "403" {
		return "Grafana rejected plugin credentials (" + status + "). Check managed service accounts."
	}

	switch {
	case strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "i/o timeout"),
		strings.Contains(lower, "deadline exceeded"),
		strings.Contains(lower, "context deadline"),
		strings.Contains(lower, "no such host"),
		strings.Contains(lower, "server misbehaving"):
		return "Configured data source is unavailable"
	case strings.Contains(lower, "authentication"),
		strings.Contains(lower, "access denied"),
		strings.Contains(lower, "password is incorrect"):
		return "Data source authentication failed. Check credentials in plugin settings."
	case strings.Contains(lower, "datasource not configured"),
		strings.Contains(lower, "uid is empty"):
		return "No data source configured. Select one in plugin settings."
	case status == "400":
		return "Grafana rejected the ClickHouse query (400)"
	}

	// Drop nested JSON / HTTP body dumps.
	if i := strings.Index(s, "{"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	s = strings.TrimSpace(strings.TrimSuffix(s, ":"))
	if s == "" || s == "clickhouse ping failed" {
		return "Configured data source is unavailable"
	}
	if len(s) > 120 {
		return s[:117] + "..."
	}
	return s
}

// summarizeGrafanaHop classifies a plugin → Grafana transport error, or ""
// when it is not a recognised network failure.
func summarizeGrafanaHop(s string) string {
	lower := strings.ToLower(s)
	switch {
	case strings.Contains(lower, "no such host"), strings.Contains(lower, "server misbehaving"):
		if host := firstMatch(dnsHostRe, s); host != "" {
			return "Plugin cannot resolve Grafana host " + host + " (DNS)"
		}
		return "Plugin cannot resolve Grafana host (DNS)"
	case strings.Contains(lower, "x509"), strings.Contains(lower, "tls:"):
		if host := hopHost(s); host != "" {
			return "TLS error calling Grafana at " + host
		}
		return "TLS error calling Grafana"
	case strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "i/o timeout"),
		strings.Contains(lower, "deadline exceeded"):
		if host := hopHost(s); host != "" {
			return "Plugin cannot reach Grafana at " + host
		}
		return "Plugin cannot reach Grafana"
	}
	return ""
}

// hopHost extracts host[:port] only — never path, query or userinfo.
func hopHost(s string) string {
	if raw := firstMatch(postURLRe, s); raw != "" {
		if u, err := url.Parse(raw); err == nil && u.Host != "" {
			return u.Host
		}
	}
	return firstMatch(dialHostRe, s)
}

func firstMatch(re *regexp.Regexp, s string) string {
	if m := re.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}
