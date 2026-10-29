package plugin

import (
	"reflect"
	"testing"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

func TestCapNamespaceServiceNames(t *testing.T) {
	summaries := make([]queries.ServiceSummary, 0, 30)
	for i := range 30 {
		summaries = append(summaries, queries.ServiceSummary{
			Name: "svc-" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
			Rate: float64(30 - i),
		})
	}
	// Force duplicate name — should dedupe.
	summaries[5].Name = summaries[0].Name

	names, truncated := capNamespaceServiceNames(summaries, 25)
	if !truncated {
		t.Fatal("expected truncated=true for 30 unique-ish names over cap 25")
	}
	if len(names) != 25 {
		t.Fatalf("len=%d want 25", len(names))
	}
	// Expect the 25 highest-rate unique names, in rate order.
	// summaries[5] duplicates summaries[0], so index 5 is skipped.
	want := make([]string, 0, 25)
	for i := 0; i < len(summaries); i++ {
		if i == 5 {
			continue
		}
		want = append(want, summaries[i].Name)
		if len(want) == 25 {
			break
		}
	}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("names=%v\nwant=%v", names, want)
	}
}

func TestCapNamespaceServiceNames_UnderCap(t *testing.T) {
	summaries := []queries.ServiceSummary{
		{Name: "b", Rate: 1},
		{Name: "a", Rate: 2},
		{Name: "", Rate: 99},
	}
	names, truncated := capNamespaceServiceNames(summaries, 25)
	if truncated {
		t.Fatal("expected truncated=false")
	}
	if len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Fatalf("got %#v", names)
	}
}

func TestCapNamespaceServiceRefs_KeepsEnvPairs(t *testing.T) {
	summaries := []queries.ServiceSummary{
		{Name: "api", Environment: "prod", Rate: 10},
		{Name: "api", Environment: "staging", Rate: 5},
		{Name: "api", Environment: "prod", Rate: 1}, // dup
	}
	refs, truncated := capNamespaceServiceRefs(summaries, 25)
	if truncated {
		t.Fatal("expected truncated=false")
	}
	if len(refs) != 2 {
		t.Fatalf("got %#v", refs)
	}
	if refs[0].Name != "api" || refs[0].Env != "prod" {
		t.Fatalf("highest-rate first: %#v", refs[0])
	}
	if refs[1].Env != "staging" {
		t.Fatalf("expected staging second: %#v", refs[1])
	}
}

func TestProbeEnvironment(t *testing.T) {
	if got := probeEnvironment("prod", "staging"); got != "prod" {
		t.Fatalf("single filter: got %q", got)
	}
	if got := probeEnvironment("prod,staging", "staging"); got != "staging" {
		t.Fatalf("multi filter prefers row: got %q", got)
	}
	if got := probeEnvironment("", "dev"); got != "dev" {
		t.Fatalf("empty filter uses row: got %q", got)
	}
	if got := probeEnvironment("a,b", ""); got != "a,b" {
		t.Fatalf("multi with empty row keeps filter: got %q", got)
	}
}
