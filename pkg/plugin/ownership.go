package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/nais/grafana-otel-plugin/pkg/plugin/chquery"
	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

const (
	ownerAnnotationTag     = "clickhouse-apm:owner"
	ownershipStaleAfter    = 7 * 24 * time.Hour
	ownershipTelemetryLook = 30 * 24 * time.Hour
	ownershipDocPath       = "docs/ownership.md"
)

// OwnershipSource identifies which directory source won.
type OwnershipSource string

// Ownership source identifiers (docs/ownership.md precedence).
const (
	OwnershipSourceTelemetry  OwnershipSource = "telemetry"
	OwnershipSourceAnnotation OwnershipSource = "annotation"
	OwnershipSourceClaimed    OwnershipSource = "claimed" // reserved; not written in v1
)

// OwnershipFields are the four directory fields.
type OwnershipFields struct {
	Team    string `json:"team,omitempty"`
	Contact string `json:"contact,omitempty"`
	Repo    string `json:"repo,omitempty"`
	Runbook string `json:"runbook,omitempty"`
}

func (f OwnershipFields) hasTeam() bool { return strings.TrimSpace(f.Team) != "" }

func (f OwnershipFields) equal(o OwnershipFields) bool {
	return f.Team == o.Team && f.Contact == o.Contact && f.Repo == o.Repo && f.Runbook == o.Runbook
}

func (f OwnershipFields) empty() bool {
	return f.Team == "" && f.Contact == "" && f.Repo == "" && f.Runbook == ""
}

// OwnershipConflict describes a losing source when fields disagree.
type OwnershipConflict struct {
	Winner OwnershipSource          `json:"winner"`
	Losers []OwnershipConflictLoser `json:"losers"`
}

// OwnershipConflictLoser is one disagreeing source.
type OwnershipConflictLoser struct {
	Source OwnershipSource `json:"source"`
	Fields OwnershipFields `json:"fields"`
}

// OwnershipEmptyHint tells operators how to fix an unowned service.
type OwnershipEmptyHint struct {
	Attrs   []string `json:"attrs"`
	Snippet string   `json:"snippet"`
	DocPath string   `json:"docPath"`
}

// Ownership is the single DTO for all ownership surfaces.
type Ownership struct {
	OwnershipFields
	Source      OwnershipSource     `json:"source,omitempty"`
	LastSeenMs  int64               `json:"lastSeenMs,omitempty"`
	Stale       bool                `json:"stale"`
	TeamKnown   bool                `json:"teamKnown"`
	Unavailable bool                `json:"unavailable,omitempty"` // lookup failed (not the same as unowned)
	Conflict    *OwnershipConflict  `json:"conflict,omitempty"`
	EmptyHint   *OwnershipEmptyHint `json:"emptyHint,omitempty"`
}

// NamespaceOwnershipResponse is GET /namespaces/{ns}/ownership.
type NamespaceOwnershipResponse struct {
	Services  map[string]Ownership `json:"services"`
	Owned     int                  `json:"owned"`
	Total     int                  `json:"total"`
	Coverage  float64              `json:"coverage"` // owned/total, 0 when total=0
	Truncated bool                 `json:"truncated,omitempty"`
	Scanned   int                  `json:"scanned"`
}

// FleetOwnershipResponse is GET /ownership/coverage (inventory hygiene).
type FleetOwnershipResponse struct {
	Owned    int                   `json:"owned"`
	Total    int                   `json:"total"`
	Coverage float64               `json:"coverage"`
	Unowned  []FleetUnownedService `json:"unowned"`
	ByTeam   map[string]int        `json:"byTeam,omitempty"`
	Services map[string]Ownership  `json:"services,omitempty"` // key: ns/service
}

// FleetUnownedService is one inventory service lacking owner.team.
type FleetUnownedService struct {
	Namespace string `json:"namespace,omitempty"`
	Service   string `json:"service"`
}

func ownershipEmptyHint() *OwnershipEmptyHint {
	return &OwnershipEmptyHint{
		Attrs:   []string{chquery.ResOwnerTeam, chquery.ResOwnerContact, chquery.ResVCSRepoURL, chquery.ResOwnerRunbook},
		Snippet: `export OTEL_RESOURCE_ATTRIBUTES="owner.team=YOUR_TEAM,owner.contact=#YOUR_CHANNEL"`,
		DocPath: ownershipDocPath,
	}
}

// mergeOwnership applies precedence telemetry > claimed > annotation.
// A source "wins" when it has owner.team; otherwise the next source with a team
// is used. Partial telemetry (no team) does not block an annotation team.
// telOK/annOK are false when the underlying lookup failed (vs empty success).
func mergeOwnership(tel OwnershipFields, telLastMs int64, telOK bool, ann OwnershipFields, annOK bool, knownTeams []string, now time.Time) Ownership {
	out := Ownership{TeamKnown: true}
	var winner OwnershipFields
	var source OwnershipSource

	switch {
	case telOK && tel.hasTeam():
		winner, source = tel, OwnershipSourceTelemetry
		out.LastSeenMs = telLastMs
	case annOK && ann.hasTeam():
		winner, source = ann, OwnershipSourceAnnotation
	case telOK && !tel.empty():
		winner, source = tel, OwnershipSourceTelemetry
		out.LastSeenMs = telLastMs
	case annOK && !ann.empty():
		winner, source = ann, OwnershipSourceAnnotation
	default:
		if !telOK || !annOK {
			out.Unavailable = true
			return out
		}
		out.EmptyHint = ownershipEmptyHint()
		return out
	}

	out.OwnershipFields = winner
	out.Source = source
	if source == OwnershipSourceTelemetry && telLastMs > 0 {
		out.Stale = now.Sub(time.UnixMilli(telLastMs)) > ownershipStaleAfter
	}

	if !winner.hasTeam() {
		if !telOK || !annOK {
			out.Unavailable = true
			return out
		}
		out.EmptyHint = ownershipEmptyHint()
		return out
	}

	if telOK && annOK && tel.hasTeam() && ann.hasTeam() && !tel.equal(ann) {
		loserSrc := OwnershipSourceAnnotation
		loserFields := ann
		if source == OwnershipSourceAnnotation {
			loserSrc = OwnershipSourceTelemetry
			loserFields = tel
		}
		out.Conflict = &OwnershipConflict{
			Winner: source,
			Losers: []OwnershipConflictLoser{{Source: loserSrc, Fields: loserFields}},
		}
	}

	out.TeamKnown = teamIsKnown(winner.Team, knownTeams)
	return out
}

func teamIsKnown(team string, known []string) bool {
	if len(known) == 0 || team == "" {
		return true
	}
	for _, k := range known {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(team)) {
			return true
		}
	}
	return false
}

func (a *App) knownTeams() []string {
	return a.settings.KnownTeams
}

func (a *App) resolveOwnership(ctx context.Context, headers http.Header, namespace, service string, now time.Time) Ownership {
	from := now.Add(-ownershipTelemetryLook)
	tel, telMs, telOK := a.fetchOwnershipTelemetry(ctx, headers, namespace, service, from, now)
	ann, annOK := a.fetchOwnershipAnnotation(ctx, headers, service)
	return mergeOwnership(tel, telMs, telOK, ann, annOK, a.knownTeams(), now)
}

func (a *App) fetchOwnershipTelemetry(
	ctx context.Context,
	headers http.Header,
	namespace, service string,
	from, to time.Time,
) (OwnershipFields, int64, bool) {
	if a.chClient == nil {
		// No CH configured — treat as successful empty (annotation may still win).
		return OwnershipFields{}, 0, true
	}
	token := a.resolveServiceToken(ctx)
	ch := a.chClient.WithServiceToken(token).WithAuthHeaders(headers)
	res, err := ch.Query(ctx, chquery.OwnershipAttrsSQL(service, namespace, from, to), queries.ClickHouseQueryOptions{
		From: from, To: to,
	})
	if err != nil {
		log.DefaultLogger.Debug("ownership telemetry query failed", "error", err, "service", service)
		return OwnershipFields{}, 0, false
	}
	if res.RowCount() == 0 {
		return OwnershipFields{}, 0, true
	}
	m := res.Maps()[0]
	return OwnershipFields{
		Team:    chquery.ParseString(m["team"]),
		Contact: chquery.ParseString(m["contact"]),
		Repo:    chquery.ParseString(m["repo"]),
		Runbook: chquery.ParseString(m["runbook"]),
	}, int64(chquery.ParseFloat(m["last_seen_ms"])), true
}

func (a *App) fetchOwnershipAnnotation(ctx context.Context, headers http.Header, service string) (OwnershipFields, bool) {
	if a.healthClient == nil || a.grafanaURL == "" {
		return OwnershipFields{}, true
	}
	params := url.Values{}
	params.Add("tags", ownerAnnotationTag)
	params.Add("tags", "service:"+service)
	params.Set("limit", "20")
	apiReq, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/api/annotations?%s", a.grafanaURL, params.Encode()), nil)
	if err != nil {
		return OwnershipFields{}, false
	}
	applyServiceAuth(apiReq, headers, a.resolveServiceToken(ctx))
	resp, err := a.healthClient.Do(apiReq)
	if err != nil {
		log.DefaultLogger.Debug("ownership annotation request failed", "error", err, "service", service)
		return OwnershipFields{}, false
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		log.DefaultLogger.Debug("ownership annotation request failed", "status", resp.StatusCode, "service", service)
		return OwnershipFields{}, false
	}
	var anns []grafanaAnnotation
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&anns); err != nil {
		return OwnershipFields{}, false
	}
	if len(anns) == 0 {
		return OwnershipFields{}, true
	}
	best := OwnershipFields{}
	for _, ann := range anns {
		f := parseOwnerAnnotation(ann)
		if f.hasTeam() {
			return f, true
		}
		if best.empty() && !f.empty() {
			best = f
		}
	}
	return best, true
}

func parseOwnerAnnotation(ann grafanaAnnotation) OwnershipFields {
	f := OwnershipFields{}
	for _, tag := range ann.Tags {
		switch {
		case strings.HasPrefix(tag, "team:"):
			f.Team = strings.TrimPrefix(tag, "team:")
		case strings.HasPrefix(tag, "contact:"):
			f.Contact = strings.TrimPrefix(tag, "contact:")
		case strings.HasPrefix(tag, "repo:"):
			f.Repo = strings.TrimPrefix(tag, "repo:")
		case strings.HasPrefix(tag, "runbook:"):
			f.Runbook = strings.TrimPrefix(tag, "runbook:")
		}
	}
	for _, line := range strings.Split(ann.Text, "\n") {
		line = strings.TrimSpace(line)
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key, val = strings.TrimSpace(key), strings.TrimSpace(val)
		switch key {
		case "owner.team", "team":
			if f.Team == "" {
				f.Team = val
			}
		case "owner.contact", "contact":
			if f.Contact == "" {
				f.Contact = val
			}
		case "vcs.repository.url", "repo":
			if f.Repo == "" {
				f.Repo = val
			}
		case "owner.runbook.url", "runbook":
			if f.Runbook == "" {
				f.Runbook = val
			}
		}
	}
	return f
}

// handleOwnership is GET /services/{namespace}/{service}/ownership
func (a *App) handleOwnership(w http.ResponseWriter, req *http.Request) {
	if !requireGET(w, req) {
		return
	}
	ctx := a.requestContext(req)
	namespace, service := parseServiceRef(req)
	if !requireServiceParam(w, service) {
		return
	}
	own := a.resolveOwnership(ctx, req.Header, namespace, service, time.Now())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(own)
}

// handleNamespaceOwnership is GET /namespaces/{namespace}/ownership
func (a *App) handleNamespaceOwnership(w http.ResponseWriter, req *http.Request) {
	if !requireGET(w, req) {
		return
	}
	ctx := a.requestContext(req)
	namespace := queries.MustSanitizeLabel(req.PathValue("namespace"))
	if namespace == "" {
		http.Error(w, `{"error":"missing or invalid namespace"}`, http.StatusBadRequest)
		return
	}
	from, to := parseTimeRange(req)
	env := parseEnvironment(req)
	out := a.computeNamespaceOwnership(ctx, req, namespace, env, from, to)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (a *App) computeNamespaceOwnership(
	ctx context.Context,
	req *http.Request,
	namespace, env string,
	from, to time.Time,
) NamespaceOwnershipResponse {
	refs, truncated := a.namespaceServiceRefs(ctx, req, namespace, env, from, to, maxNamespaceScorecardServices)
	out := NamespaceOwnershipResponse{
		Services:  make(map[string]Ownership, len(refs)),
		Truncated: truncated,
		Scanned:   len(refs),
		Total:     len(refs),
	}
	// Match single-service ownership lookback (not the UI time picker window).
	telFrom := to.Add(-ownershipTelemetryLook)
	telBySvc, telOK := a.batchOwnershipTelemetry(ctx, req.Header, namespace, telFrom, to)
	now := time.Now()
	known := a.knownTeams()
	headers := req.Header
	for _, ref := range refs {
		tel := telBySvc[ref.Name]
		ann, annOK := a.fetchOwnershipAnnotation(ctx, headers, ref.Name)
		own := mergeOwnership(tel.fields, tel.lastMs, telOK, ann, annOK, known, now)
		out.Services[ref.Name] = own
		if own.hasTeam() {
			out.Owned++
		}
	}
	if out.Total > 0 {
		out.Coverage = float64(out.Owned) / float64(out.Total)
	}
	return out
}

type telOwnership struct {
	fields OwnershipFields
	lastMs int64
}

const maxFleetAnnotationFetches = 25

// batchOwnershipTelemetry returns per-service telemetry ownership.
// Keys are service names within a namespace scope, or "namespace/service" for fleet.
// ok is false when the ClickHouse query failed.
func (a *App) batchOwnershipTelemetry(
	ctx context.Context,
	headers http.Header,
	namespace string,
	from, to time.Time,
) (map[string]telOwnership, bool) {
	out := map[string]telOwnership{}
	if a.chClient == nil {
		return out, true
	}
	token := a.resolveServiceToken(ctx)
	ch := a.chClient.WithServiceToken(token).WithAuthHeaders(headers)
	fleet := namespace == "" || namespace == "_"
	sql := chquery.NamespaceOwnershipSQL(namespace, from, to)
	if fleet {
		sql = chquery.FleetOwnershipSQL(from, to)
	}
	res, err := ch.Query(ctx, sql, queries.ClickHouseQueryOptions{From: from, To: to})
	if err != nil {
		log.DefaultLogger.Debug("ownership batch query failed", "error", err)
		return out, false
	}
	for _, m := range res.Maps() {
		svc := chquery.ParseString(m["service"])
		if svc == "" {
			continue
		}
		key := svc
		if fleet {
			ns := chquery.ParseString(m["namespace"])
			if ns != "" {
				key = ns + "/" + svc
			}
		}
		out[key] = telOwnership{
			fields: OwnershipFields{
				Team:    chquery.ParseString(m["team"]),
				Contact: chquery.ParseString(m["contact"]),
				Repo:    chquery.ParseString(m["repo"]),
				Runbook: chquery.ParseString(m["runbook"]),
			},
			lastMs: int64(chquery.ParseFloat(m["last_seen_ms"])),
		}
	}
	return out, true
}

// handleFleetOwnership is GET /ownership/coverage — fleet hygiene for inventory.
func (a *App) handleFleetOwnership(w http.ResponseWriter, req *http.Request) {
	if !requireGET(w, req) {
		return
	}
	ctx := a.requestContext(req)
	from, to := parseTimeRange(req)
	out := a.computeFleetOwnership(ctx, req, from, to)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (a *App) computeFleetOwnership(ctx context.Context, req *http.Request, from, to time.Time) FleetOwnershipResponse {
	// Inventory services for denominator (UI range); ownership attrs use lookback.
	services := a.listInventoryServices(ctx, req, from, to)
	out := FleetOwnershipResponse{
		Services: make(map[string]Ownership),
		ByTeam:   make(map[string]int),
		Unowned:  []FleetUnownedService{},
	}
	if len(services) == 0 {
		return out
	}
	telFrom := to.Add(-ownershipTelemetryLook)
	telBySvc, telOK := a.batchOwnershipTelemetry(ctx, req.Header, "", telFrom, to)
	now := time.Now()
	known := a.knownTeams()
	headers := req.Header
	out.Total = len(services)
	annFetches := 0
	for _, s := range services {
		key := s.Name
		if s.Namespace != "" {
			key = s.Namespace + "/" + s.Name
		}
		tel := telBySvc[key]
		ann := OwnershipFields{}
		annOK := true
		// Bound annotation fan-out: only when telemetry lacks a team.
		if !tel.fields.hasTeam() && annFetches < maxFleetAnnotationFetches {
			ann, annOK = a.fetchOwnershipAnnotation(ctx, headers, s.Name)
			annFetches++
		}
		own := mergeOwnership(tel.fields, tel.lastMs, telOK, ann, annOK, known, now)
		out.Services[key] = own
		if own.hasTeam() {
			out.Owned++
			out.ByTeam[own.Team]++
		} else if !own.Unavailable {
			out.Unowned = append(out.Unowned, FleetUnownedService{Namespace: s.Namespace, Service: s.Name})
		}
	}
	if out.Total > 0 {
		out.Coverage = float64(out.Owned) / float64(out.Total)
	}
	return out
}

type inventoryRef struct {
	Namespace string
	Name      string
}

func (a *App) listInventoryServices(ctx context.Context, req *http.Request, from, to time.Time) []inventoryRef {
	step := 60 * time.Second
	var summaries []queries.ServiceSummary
	if a.clickStackMode() {
		summaries = a.fetchServiceSummariesClickHouse(ctx, a.chForRequest(req), from, to, step, false, "", "", "")
	} else {
		caps := a.cachedOrDetectCapabilities(ctx)
		if !caps.SpanMetrics.Detected {
			return nil
		}
		summaries = a.fetchServiceSummaries(ctx, caps, from, to, step, false, "", "", "", req.Header)
	}
	refs := make([]inventoryRef, 0, len(summaries))
	seen := map[string]struct{}{}
	for _, s := range summaries {
		key := s.Namespace + "/" + s.Name
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		refs = append(refs, inventoryRef{Namespace: s.Namespace, Name: s.Name})
	}
	return refs
}

func applyHasOwnerCheck(checks []ScorecardCheck, own Ownership) {
	c := scorecardCheckByKey(checks, "hasOwner")
	if c == nil {
		return
	}
	if own.Unavailable {
		c.OK = false
		c.NotApplicable = true
		c.Hint = "Ownership lookup unavailable — could not query telemetry or annotations."
		return
	}
	if own.hasTeam() {
		c.OK = true
		src := string(own.Source)
		if src == "" {
			src = "unknown"
		}
		c.Hint = fmt.Sprintf("Owner team %q from %s.", own.Team, src)
		return
	}
	c.OK = false
	c.Hint = "No owner.team — set OTEL_RESOURCE_ATTRIBUTES owner.team=… (see docs/ownership.md) or publish a clickhouse-apm:owner annotation."
}
