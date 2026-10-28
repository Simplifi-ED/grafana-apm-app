package plugin

import (
	"context"
	"sort"
	"time"
)

// IssuesAttention summarizes new / returned-after-resolve issue counts for
// Overview attention strips (post-parity letter D / ticket 07).
type IssuesAttention struct {
	New                  int `json:"new"`
	ReturnedAfterResolve int `json:"returnedAfterResolve"`
}

// applyIssuesNotifyFlags sets on-read IsNew / ReturnedAfterResolve on each
// issue and returns attention counts. Stateless: no issue store.
//
// ReturnedAfterResolve mirrors docs/triage.md Phase 1: folded status is
// resolved, resolve time predates the newest deploy annotation, and the
// fingerprint still appears in the current result set.
//
// IsNew mirrors log-patterns P-1: only when previousFingerprints is non-nil
// AND non-empty (baseline window actually held issues). Empty previous =
// retention gap, not "everything is new".
func applyIssuesNotifyFlags(
	issues []Issue,
	triage map[string]TriageState,
	latestDeployMs int64,
	previousFingerprints map[string]bool,
) IssuesAttention {
	var att IssuesAttention
	baselineOK := len(previousFingerprints) > 0
	for i := range issues {
		fp := issues[i].Fingerprint
		if baselineOK && !previousFingerprints[fp] {
			issues[i].IsNew = true
			att.New++
		}
		st, ok := triage[fp]
		if ok && st.Status == "resolved" && latestDeployMs > 0 && st.UpdatedAt < latestDeployMs {
			issues[i].ReturnedAfterResolve = true
			att.ReturnedAfterResolve++
		}
	}
	return att
}

// sortIssuesNotifyFirst puts New and ReturnedAfterResolve ahead of count order
// so the default Issues view surfaces attention without client re-derivation.
func sortIssuesNotifyFirst(issues []Issue) {
	sort.SliceStable(issues, func(i, j int) bool {
		score := func(x Issue) int {
			s := 0
			if x.ReturnedAfterResolve {
				s += 2
			}
			if x.IsNew {
				s++
			}
			return s
		}
		si, sj := score(issues[i]), score(issues[j])
		if si != sj {
			return si > sj
		}
		if issues[i].Count != issues[j].Count {
			return issues[i].Count > issues[j].Count
		}
		return issues[i].Fingerprint < issues[j].Fingerprint
	})
}

func issueFingerprintSet(issues []Issue) map[string]bool {
	out := make(map[string]bool, len(issues))
	for _, iss := range issues {
		if iss.Fingerprint != "" {
			out[iss.Fingerprint] = true
		}
	}
	return out
}

// attachIssuesNotifyFlags loads triage + newest deploy annotation and applies
// notify flags. previousFingerprints may be nil (skip IsNew) or empty (also
// skip — baseline missing).
func (a *App) attachIssuesNotifyFlags(
	ctx context.Context,
	namespace, service, env string,
	from, to time.Time,
	resp *IssuesResponse,
	previousFingerprints map[string]bool,
) {
	if resp == nil || len(resp.Issues) == 0 {
		return
	}
	var triage map[string]TriageState
	if store := a.triageStore(ctx); store != nil && a.grafanaURL != "" {
		if states, err := store.States(ctx, namespace, service); err == nil {
			triage = states
		}
	}
	_, _, latestDeployMs := a.fetchDeployAnnotations(ctx, service, env, from, to)
	att := applyIssuesNotifyFlags(resp.Issues, triage, latestDeployMs, previousFingerprints)
	resp.Attention = &att
	sortIssuesNotifyFirst(resp.Issues)
}
