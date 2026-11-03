package chquery

import "testing"

func TestParseServiceNamespaceMap(t *testing.T) {
	got := ParseServiceNamespaceMap([]map[string]any{
		{"service": "nova-api", "namespace": "prod-nova"},
		{"service": "envoy-gateway", "namespace": "envoy-gateway-system"},
		{"service": "valkey", "namespace": ""},
		{"service": "", "namespace": "prod-nova"},
		{"service": "nova-api", "namespace": "other"}, // first wins
	})
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d: %#v", len(got), got)
	}
	if got["nova-api"] != "prod-nova" {
		t.Errorf("nova-api ns = %q, want prod-nova", got["nova-api"])
	}
	if _, ok := got["valkey"]; ok {
		t.Error("empty namespace must be skipped")
	}
}
