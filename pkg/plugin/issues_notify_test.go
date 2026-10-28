package plugin

import "testing"

func TestApplyIssuesNotifyFlags_ReturnedAndNew(t *testing.T) {
	issues := []Issue{
		{ExceptionGroup: ExceptionGroup{Fingerprint: "v1:old", Count: 10}},
		{ExceptionGroup: ExceptionGroup{Fingerprint: "v1:new", Count: 5}},
		{ExceptionGroup: ExceptionGroup{Fingerprint: "v1:active", Count: 20}},
	}
	triage := map[string]TriageState{
		"v1:old": {Status: "resolved", UpdatedAt: 1000, UpdatedBy: "a"},
	}
	prev := map[string]bool{"v1:old": true, "v1:active": true}

	att := applyIssuesNotifyFlags(issues, triage, 2000, prev)
	if !issues[0].ReturnedAfterResolve {
		t.Fatal("v1:old should be returned after resolve")
	}
	if issues[0].IsNew {
		t.Fatal("v1:old should not be new")
	}
	if !issues[1].IsNew || issues[1].ReturnedAfterResolve {
		t.Fatalf("v1:new flags = %+v", issues[1])
	}
	if issues[2].IsNew || issues[2].ReturnedAfterResolve {
		t.Fatalf("v1:active flags = %+v", issues[2])
	}
	if att.New != 1 || att.ReturnedAfterResolve != 1 {
		t.Fatalf("attention=%+v", att)
	}
}

func TestApplyIssuesNotifyFlags_EmptyBaselineSkipsIsNew(t *testing.T) {
	issues := []Issue{{ExceptionGroup: ExceptionGroup{Fingerprint: "v1:x", Count: 1}}}
	att := applyIssuesNotifyFlags(issues, nil, 0, map[string]bool{})
	if issues[0].IsNew || att.New != 0 {
		t.Fatalf("empty baseline must not mark new: %+v att=%+v", issues[0], att)
	}
}

func TestApplyIssuesNotifyFlags_NilBaselineSkipsIsNew(t *testing.T) {
	issues := []Issue{{ExceptionGroup: ExceptionGroup{Fingerprint: "v1:x", Count: 1}}}
	att := applyIssuesNotifyFlags(issues, nil, 0, nil)
	if issues[0].IsNew || att.New != 0 {
		t.Fatalf("nil baseline must not mark new: %+v", issues[0])
	}
}

func TestApplyIssuesNotifyFlags_NoDeploySkipsReturned(t *testing.T) {
	issues := []Issue{{ExceptionGroup: ExceptionGroup{Fingerprint: "v1:old", Count: 1}}}
	triage := map[string]TriageState{"v1:old": {Status: "resolved", UpdatedAt: 1000}}
	applyIssuesNotifyFlags(issues, triage, 0, nil)
	if issues[0].ReturnedAfterResolve {
		t.Fatal("no deploy marker → not returned")
	}
}

func TestSortIssuesNotifyFirst(t *testing.T) {
	issues := []Issue{
		{ExceptionGroup: ExceptionGroup{Fingerprint: "v1:c", Count: 100}},
		{ExceptionGroup: ExceptionGroup{Fingerprint: "v1:a", Count: 1}, IsNew: true},
		{ExceptionGroup: ExceptionGroup{Fingerprint: "v1:b", Count: 2}, ReturnedAfterResolve: true},
	}
	sortIssuesNotifyFirst(issues)
	if issues[0].Fingerprint != "v1:b" || issues[1].Fingerprint != "v1:a" || issues[2].Fingerprint != "v1:c" {
		t.Fatalf("order=%v %v %v", issues[0].Fingerprint, issues[1].Fingerprint, issues[2].Fingerprint)
	}
}
