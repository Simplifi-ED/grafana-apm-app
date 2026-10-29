package chquery

import (
	"strings"
	"testing"
	"time"
)

func TestOwnershipAttrsSQL(t *testing.T) {
	sql := OwnershipAttrsSQL("payments", "prod", time.Unix(0, 0), time.Unix(3600, 0))
	for _, want := range []string{
		"owner.team", "owner.contact", "vcs.repository.url", "owner.runbook.url",
		"ServiceName = 'payments'", "otel_traces", "last_seen_ms",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("SQL missing %q:\n%s", want, sql)
		}
	}
}

func TestNamespaceOwnershipSQL(t *testing.T) {
	sql := NamespaceOwnershipSQL("sanad", time.Unix(0, 0), time.Unix(3600, 0))
	if !strings.Contains(sql, "GROUP BY ServiceName") {
		t.Fatalf("expected group by:\n%s", sql)
	}
}

func TestFleetOwnershipSQL_GroupsByNamespace(t *testing.T) {
	sql := FleetOwnershipSQL(time.Unix(0, 0), time.Unix(3600, 0))
	if !strings.Contains(sql, "GROUP BY namespace, ServiceName") {
		t.Fatalf("expected ns+service group by:\n%s", sql)
	}
}
