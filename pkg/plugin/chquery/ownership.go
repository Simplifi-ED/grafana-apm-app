package chquery

import (
	"fmt"
	"time"

	"github.com/nais/grafana-otel-plugin/pkg/plugin/queries"
)

// Ownership resource attribute keys (docs/ownership.md).
const (
	ResOwnerTeam    = "owner.team"
	ResOwnerContact = "owner.contact"
	ResVCSRepoURL   = "vcs.repository.url"
	ResOwnerRunbook = "owner.runbook.url"
)

func ownershipAttrPresent() string {
	return fmt.Sprintf(
		`(ResourceAttributes['%s'] != '' OR ResourceAttributes['%s'] != '' OR ResourceAttributes['%s'] != '' OR ResourceAttributes['%s'] != '')`,
		ResOwnerTeam, ResOwnerContact, ResVCSRepoURL, ResOwnerRunbook,
	)
}

// OwnershipAttrsSQL returns latest non-empty ownership ResourceAttributes for a
// service (service-level; environment ignored). last_seen_ms is epoch ms of the
// newest row that carried any ownership attr.
func OwnershipAttrsSQL(service, namespace string, from, to time.Time) string {
	where := fmt.Sprintf(`%s AND ServiceName = %s`, TimestampFilter(from, to), q(service))
	if namespace != "" && namespace != "_" {
		where += fmt.Sprintf(
			" AND (ResourceAttributes['%s'] = %s OR ResourceAttributes['%s'] = %s)",
			ResK8sNamespace, q(namespace), ResServiceNamespace, q(namespace),
		)
	}
	hasOwner := ownershipAttrPresent()
	return fmt.Sprintf(`
SELECT
  argMaxIf(ResourceAttributes['%s'], %s, ResourceAttributes['%s'] != '') AS team,
  argMaxIf(ResourceAttributes['%s'], %s, ResourceAttributes['%s'] != '') AS contact,
  argMaxIf(ResourceAttributes['%s'], %s, ResourceAttributes['%s'] != '') AS repo,
  argMaxIf(ResourceAttributes['%s'], %s, ResourceAttributes['%s'] != '') AS runbook,
  toUnixTimestamp64Milli(maxIf(%s, %s)) AS last_seen_ms
FROM %s
WHERE %s AND %s
`,
		ResOwnerTeam, ColTimestamp, ResOwnerTeam,
		ResOwnerContact, ColTimestamp, ResOwnerContact,
		ResVCSRepoURL, ColTimestamp, ResVCSRepoURL,
		ResOwnerRunbook, ColTimestamp, ResOwnerRunbook,
		ColTimestamp, hasOwner,
		table(queries.CHTableTraces),
		where, hasOwner,
	)
}

// NamespaceOwnershipSQL batches ownership attrs for services in one namespace.
func NamespaceOwnershipSQL(namespace string, from, to time.Time) string {
	where := TimestampFilter(from, to)
	if namespace != "" && namespace != "_" {
		where += fmt.Sprintf(
			" AND (ResourceAttributes['%s'] = %s OR ResourceAttributes['%s'] = %s)",
			ResK8sNamespace, q(namespace), ResServiceNamespace, q(namespace),
		)
	}
	hasOwner := ownershipAttrPresent()
	return fmt.Sprintf(`
SELECT
  ServiceName AS service,
  argMaxIf(ResourceAttributes['%s'], %s, ResourceAttributes['%s'] != '') AS team,
  argMaxIf(ResourceAttributes['%s'], %s, ResourceAttributes['%s'] != '') AS contact,
  argMaxIf(ResourceAttributes['%s'], %s, ResourceAttributes['%s'] != '') AS repo,
  argMaxIf(ResourceAttributes['%s'], %s, ResourceAttributes['%s'] != '') AS runbook,
  toUnixTimestamp64Milli(maxIf(%s, %s)) AS last_seen_ms
FROM %s
WHERE %s AND %s
GROUP BY ServiceName
`,
		ResOwnerTeam, ColTimestamp, ResOwnerTeam,
		ResOwnerContact, ColTimestamp, ResOwnerContact,
		ResVCSRepoURL, ColTimestamp, ResVCSRepoURL,
		ResOwnerRunbook, ColTimestamp, ResOwnerRunbook,
		ColTimestamp, hasOwner,
		table(queries.CHTableTraces),
		where, hasOwner,
	)
}

// FleetOwnershipSQL batches ownership for the whole fleet (inventory hygiene).
// Grouped by namespace + ServiceName so same-named services in different
// namespaces stay distinct.
func FleetOwnershipSQL(from, to time.Time) string {
	where := TimestampFilter(from, to)
	hasOwner := ownershipAttrPresent()
	nsExpr := fmt.Sprintf(
		"coalesce(nullIf(ResourceAttributes['%s'], ''), nullIf(ResourceAttributes['%s'], ''), '')",
		ResK8sNamespace, ResServiceNamespace,
	)
	return fmt.Sprintf(`
SELECT
  ServiceName AS service,
  %s AS namespace,
  argMaxIf(ResourceAttributes['%s'], %s, ResourceAttributes['%s'] != '') AS team,
  argMaxIf(ResourceAttributes['%s'], %s, ResourceAttributes['%s'] != '') AS contact,
  argMaxIf(ResourceAttributes['%s'], %s, ResourceAttributes['%s'] != '') AS repo,
  argMaxIf(ResourceAttributes['%s'], %s, ResourceAttributes['%s'] != '') AS runbook,
  toUnixTimestamp64Milli(maxIf(%s, %s)) AS last_seen_ms
FROM %s
WHERE %s AND %s
GROUP BY namespace, ServiceName
`,
		nsExpr,
		ResOwnerTeam, ColTimestamp, ResOwnerTeam,
		ResOwnerContact, ColTimestamp, ResOwnerContact,
		ResVCSRepoURL, ColTimestamp, ResVCSRepoURL,
		ResOwnerRunbook, ColTimestamp, ResOwnerRunbook,
		ColTimestamp, hasOwner,
		table(queries.CHTableTraces),
		where, hasOwner,
	)
}
