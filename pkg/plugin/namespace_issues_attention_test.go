package plugin

import (
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
	// Highest rate first after sort.
	if names[0] != summaries[0].Name && names[0] != summaries[5].Name {
		// index 0 has highest rate; after dedupe still first
		t.Fatalf("expected highest-rate service first, got %q", names[0])
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
