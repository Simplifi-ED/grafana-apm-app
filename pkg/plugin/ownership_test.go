package plugin

import (
	"testing"
	"time"
)

func TestMergeOwnership_TelemetryWins(t *testing.T) {
	now := time.Now()
	tel := OwnershipFields{Team: "platform", Contact: "#platform"}
	ann := OwnershipFields{Team: "other", Contact: "#other"}
	got := mergeOwnership(tel, now.UnixMilli(), true, ann, true, nil, now)
	if got.Source != OwnershipSourceTelemetry || got.Team != "platform" {
		t.Fatalf("got %+v", got)
	}
	if got.Conflict == nil || got.Conflict.Losers[0].Fields.Team != "other" {
		t.Fatalf("expected conflict, got %+v", got.Conflict)
	}
	if got.Stale || got.EmptyHint != nil || got.Unavailable {
		t.Fatalf("unexpected stale/empty/unavailable: %+v", got)
	}
}

func TestMergeOwnership_AnnotationFallback(t *testing.T) {
	now := time.Now()
	got := mergeOwnership(OwnershipFields{}, 0, true, OwnershipFields{Team: "sre"}, true, nil, now)
	if got.Source != OwnershipSourceAnnotation || got.Team != "sre" {
		t.Fatalf("got %+v", got)
	}
}

func TestMergeOwnership_Unowned(t *testing.T) {
	got := mergeOwnership(OwnershipFields{}, 0, true, OwnershipFields{}, true, nil, time.Now())
	if got.hasTeam() || got.EmptyHint == nil || got.Unavailable {
		t.Fatalf("expected empty hint, got %+v", got)
	}
}

func TestMergeOwnership_Unavailable(t *testing.T) {
	got := mergeOwnership(OwnershipFields{}, 0, false, OwnershipFields{}, false, nil, time.Now())
	if !got.Unavailable || got.EmptyHint != nil {
		t.Fatalf("expected unavailable, got %+v", got)
	}
}

func TestMergeOwnership_AnnotationWhenTelemetryPartial(t *testing.T) {
	got := mergeOwnership(
		OwnershipFields{Repo: "https://github.com/o/r"},
		time.Now().UnixMilli(),
		true,
		OwnershipFields{Team: "platform"},
		true,
		nil,
		time.Now(),
	)
	if got.Source != OwnershipSourceAnnotation || got.Team != "platform" {
		t.Fatalf("got %+v", got)
	}
}

func TestMergeOwnership_Stale(t *testing.T) {
	now := time.Now()
	old := now.Add(-10 * 24 * time.Hour).UnixMilli()
	got := mergeOwnership(OwnershipFields{Team: "platform"}, old, true, OwnershipFields{}, true, nil, now)
	if !got.Stale {
		t.Fatalf("expected stale, got %+v", got)
	}
}

func TestMergeOwnership_KnownTeams(t *testing.T) {
	now := time.Now()
	got := mergeOwnership(OwnershipFields{Team: "rogue"}, now.UnixMilli(), true, OwnershipFields{}, true, []string{"platform", "sre"}, now)
	if got.TeamKnown {
		t.Fatalf("expected unknown team flag, got %+v", got)
	}
	got2 := mergeOwnership(OwnershipFields{Team: "platform"}, now.UnixMilli(), true, OwnershipFields{}, true, []string{"platform"}, now)
	if !got2.TeamKnown {
		t.Fatalf("expected known, got %+v", got2)
	}
}

func TestParseOwnerAnnotation(t *testing.T) {
	ann := grafanaAnnotation{
		Text: "owner.team=platform\nowner.contact=#ops\nvcs.repository.url=https://github.com/o/r",
		Tags: []string{"clickhouse-apm:owner", "service:payments", "runbook:https://wiki/r"},
	}
	f := parseOwnerAnnotation(ann)
	if f.Team != "platform" || f.Contact != "#ops" || f.Repo == "" || f.Runbook == "" {
		t.Fatalf("got %+v", f)
	}
}

func TestApplyHasOwnerCheck(t *testing.T) {
	checks := scorecardBaseChecks()
	applyHasOwnerCheck(checks, Ownership{OwnershipFields: OwnershipFields{Team: "platform"}, Source: OwnershipSourceTelemetry})
	c := scorecardCheckByKey(checks, "hasOwner")
	if c == nil || !c.OK {
		t.Fatalf("expected OK hasOwner, got %+v", c)
	}

	checks = scorecardBaseChecks()
	applyHasOwnerCheck(checks, Ownership{Unavailable: true})
	c = scorecardCheckByKey(checks, "hasOwner")
	if c == nil || !c.NotApplicable || c.OK {
		t.Fatalf("expected N/A unavailable, got %+v", c)
	}

	checks = scorecardBaseChecks()
	applyHasOwnerCheck(checks, Ownership{})
	c = scorecardCheckByKey(checks, "hasOwner")
	if c == nil || c.OK || c.NotApplicable || c.Hint == "" {
		t.Fatalf("expected fail hint, got %+v", c)
	}
}
