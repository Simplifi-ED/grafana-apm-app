package queries

import (
	"context"
	"net/url"
	"testing"
	"time"
)

func TestSafeBaseURL(t *testing.T) {
	for in, want := range map[string]string{
		"": "",
		"http://localhost:3000/api/datasources/proxy/uid/abc-1_X/":    "http://localhost:3000/api/datasources/proxy/uid/abc-1_X",
		"http://127.0.0.1:3000/api/prometheus/grafana":                "http://127.0.0.1:3000/api/prometheus/grafana",
		"https://grafana.example.com/api/prometheus/grafana":          "https://grafana.example.com/api/prometheus/grafana",
		"http://localhost:3000/grafana/api/datasources/proxy/uid/abc": "http://localhost:3000/grafana/api/datasources/proxy/uid/abc",
		"http://[::1]:3000":                                           "http://[::1]:3000",
		"http://[::1]:3000/grafana/":                                  "http://[::1]:3000/grafana",
		"https://grafana.example/my%20org/":                           "https://grafana.example/my%20org",
		"https://grafana.example/path%2Fkeep":                         "https://grafana.example/path%2Fkeep",
		"http://localhost:3000/api/datasources/proxy/uid/x?y=1":       "",
		"http://evil@attacker.example/":                               "",
		"http://localhost:3000/api/datasources/proxy/uid/a#b":         "",
		"http://localhost:3000/api/../x":                              "",
		"http://localhost:3000/api/%2e%2e/x":                          "",
		"file:///etc/passwd":                                          "",
	} {
		if got := SafeBaseURL(in); got != want {
			t.Errorf("SafeBaseURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDatasourceProxyURL(t *testing.T) {
	got := DatasourceProxyURL("http://[::1]:3000/grafana/", "prom-1")
	if want := "http://[::1]:3000/grafana/api/datasources/proxy/uid/prom-1"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	got = DatasourceProxyURL("https://grafana.example/my%20org/", "abc_1-X")
	if want := "https://grafana.example/my%20org/api/datasources/proxy/uid/abc_1-X"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if DatasourceProxyURL("", "prom") != "" {
		t.Fatal("expected empty proxy URL when base is empty")
	}
	if DatasourceProxyURL("http://localhost:3000", "../x") != "" {
		t.Fatal("expected empty proxy URL for invalid uid")
	}
	if DatasourceProxyURL("http://localhost:3000", "a/b") != "" {
		t.Fatal("expected empty proxy URL for uid with slash")
	}
}

func TestProxyRequestURL(t *testing.T) {
	q := mapToValues(map[string]string{"q": `{service="x"}`, "limit": "10"})
	got, err := ProxyRequestURL("http://[::1]:3000/grafana/", q, "api", "search")
	if err != nil {
		t.Fatal(err)
	}
	if want := "http://[::1]:3000/grafana/api/search?limit=10&q=%7Bservice%3D%22x%22%7D"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if _, err := ProxyRequestURL("http://localhost:3000", q, "..", "x"); err == nil {
		t.Fatal("expected error for traversal segment")
	}
}

func mapToValues(m map[string]string) url.Values {
	q := make(url.Values, len(m))
	for k, v := range m {
		q.Set(k, v)
	}
	return q
}

func TestDsQueryEndpoint(t *testing.T) {
	got, err := DsQueryEndpoint("http://localhost:3000/grafana/")
	if err != nil {
		t.Fatal(err)
	}
	if want := "http://localhost:3000/grafana/api/ds/query"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if _, err := DsQueryEndpoint("http://evil@attacker.example/"); err == nil {
		t.Fatal("expected error for invalid base URL")
	}
	if _, err := DsQueryEndpoint(""); err == nil {
		t.Fatal("expected error for empty base URL")
	}
}

func TestLokiMetricClientBaseURL(t *testing.T) {
	c := NewLokiMetricClient("http://localhost:3000/api/datasources/proxy/uid/abc/", "")
	if want := "http://localhost:3000/api/datasources/proxy/uid/abc/loki"; c.baseURL != want {
		t.Errorf("baseURL = %q, want %q", c.baseURL, want)
	}
}

func TestRejectedClientQueryFails(t *testing.T) {
	c := NewPrometheusClient("http://evil@attacker.example/", "")
	if _, err := c.InstantQuery(context.Background(), "up", time.Now()); err == nil {
		t.Fatal("expected error from client with rejected base URL")
	}
}
