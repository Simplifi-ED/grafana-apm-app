package plugin

import (
	"regexp"
	"strings"
)

// classifyDependency determines the dependency type using service-graph
// connection_type, db_system from the service graph, and messaging_system labels.
func classifyDependency(name, connType string, dbSystemMap, messagingSystemMap map[string]string) string {
	switch connType {
	case "database":
		// Resolve specific DB type from service graph db_system label
		if dbSys, ok := dbSystemMap[name]; ok {
			return normalizeDBSystem(dbSys)
		}
		// Fallback: try hostname pattern matching
		return inferDBFromHostname(name)

	case "messaging_system":
		// If the server is a plain service name (not a broker hostname), this
		// represents a producer→consumer edge: the consumer is not a dependency
		// of the producer but rather a downstream subscriber. Classify these as
		// "service" so they are excluded from the dependencies list and appear
		// only in connected services.
		if !looksLikeHostname(name) && !isKnownBrokerName(name) {
			return "service"
		}
		if msgSys, ok := messagingSystemMap[name]; ok {
			return normalizeMessagingSystem(msgSys)
		}
		return "kafka"

	case "virtual_node":
		return "external"
	}

	// No connection_type: prefer db_system label, else exact enum / peer-alias fallback.
	if dbSys, ok := dbSystemMap[name]; ok {
		return normalizeDBSystem(dbSys)
	}
	return inferFromName(name)
}

// normalizeDBSystem maps OTel db.system / db.system.name values to display types.
// Unknown values become "database" (generic) — use knownDBSystemAlias when you need
// to distinguish "recognized enum" from "unknown string".
func normalizeDBSystem(dbSys string) string {
	if typ, ok := knownDBSystemAlias(dbSys); ok {
		return typ
	}
	return "database"
}

// knownDBSystemAlias reports whether name is an OTel db.system (or peer alias we
// emit as peer.service for a DB). Exact match only — no hostname substrings.
func knownDBSystemAlias(name string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "postgresql", "postgres":
		return "postgresql", true
	case "oracle":
		return "oracle", true
	case "mongodb", "mongo":
		return "mongodb", true
	case "redis", "valkey": // valkey is a peer.service alias, not a db.system enum
		return "redis", true
	case "mysql", "mariadb":
		return "mysql", true
	case "db2":
		return "db2", true
	case "opensearch", "elasticsearch":
		return "opensearch", true
	case "h2":
		return "h2", true
	case "other_sql":
		return "database", true
	default:
		return "", false
	}
}

// normalizeMessagingSystem maps OTel messaging.system values to display types.
func normalizeMessagingSystem(msgSys string) string {
	if typ, ok := knownMessagingSystemAlias(msgSys); ok {
		return typ
	}
	return "messaging"
}

// knownMessagingSystemAlias reports whether name is an OTel messaging.system enum
// (or a well-known broker peer.service). Exact match only.
func knownMessagingSystemAlias(name string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "kafka":
		return "kafka", true
	case "rabbitmq":
		return "rabbitmq", true
	case "nats", "pulsar", "activemq", "jms":
		return "messaging", true
	default:
		return "", false
	}
}

// inferDBFromHostname uses hostname patterns when connection_type is already
// "database" but the db_system metric label is missing. Fleet-specific heuristics
// (Nav RAC / a01db*) stay here — not in the no-connection_type path.
func inferDBFromHostname(name string) string {
	lower := strings.ToLower(name)
	if strings.HasPrefix(lower, "dmv") && strings.Contains(lower, "-scan") {
		return "oracle" // Oracle RAC scan listeners
	}
	if strings.HasPrefix(lower, "a01db") {
		return "postgresql" // Nav on-prem PostgreSQL hosts
	}
	if strings.Contains(lower, "redis") || strings.Contains(lower, "valkey") {
		return "redis"
	}
	if strings.Contains(lower, "opensearch") || strings.Contains(lower, "elastic") {
		return "opensearch"
	}
	if strings.Contains(lower, "mongo") {
		return "mongodb"
	}
	return "database"
}

// inferFromName is the last-resort classifier when connection_type is empty and
// no db_system / messaging_system label is available for this peer.
//
// Prefer Tempo/OTel labels (connection_type, db_system, messaging_system) over
// this path. Here we only accept exact OTel enum / peer aliases — never
// substrings or hostname guesses (those belong in inferDBFromHostname when
// connection_type is already "database").
func inferFromName(name string) string {
	lower := strings.ToLower(name)
	if typ, ok := knownDBSystemAlias(lower); ok {
		return typ
	}
	if typ, ok := knownMessagingSystemAlias(lower); ok {
		return typ
	}
	if isExternalHostname(lower) {
		return "external"
	}
	return "service"
}

// isExternalHostname returns true if the name looks like an external hostname
// rather than a Kubernetes internal service address (service.namespace).
//
// Heuristic:
//   - 1 part (no dots): internal K8s service name
//   - 2 parts: check if the second part is a known TLD (.no, .com, .io, etc.)
//     If yes → external (e.g., "auth.example.com"). If no → internal K8s (e.g.,
//     "ledger-person.finance" where "finance" is a namespace).
//   - 3+ parts: external (e.g., "graph.microsoft.com", "checkout-api.prod.example.com")
func isExternalHostname(name string) bool {
	// host:port with dots in the host part → check the host
	if strings.Contains(name, ":") {
		host, _, _ := strings.Cut(name, ":")
		return isExternalHostname(host)
	}
	// K8s FQDN patterns → internal
	if strings.Contains(name, ".svc.") || strings.HasSuffix(name, ".svc") {
		return false
	}
	parts := strings.Split(name, ".")
	switch len(parts) {
	case 1:
		return false // bare service name
	case 2:
		// 2-part: external only if the suffix is a known TLD
		return isKnownTLD(parts[1])
	default:
		return true // 3+ parts → external hostname
	}
}

// isKnownTLD returns true if the suffix matches a recognized top-level domain.
// This distinguishes "auth.example.com" (external) from "mysvc.org-namespace" (K8s).
func isKnownTLD(suffix string) bool {
	switch strings.ToLower(suffix) {
	case "com", "org", "net", "io", "no", "se", "dk", "fi", "uk", "de", "fr",
		"eu", "dev", "app", "cloud", "edu", "gov", "mil", "int", "biz", "info",
		"co", "us", "ca", "au", "nl", "be", "ch", "at", "pl", "it", "es", "pt":
		return true
	}
	return false
}

// looksLikeHostname returns true if the name contains dots or colons (host:port),
// indicating a hostname rather than a bare service name.
func looksLikeHostname(name string) bool {
	return strings.Contains(name, ".") || strings.Contains(name, ":")
}

// isKnownBrokerName reports whether name is a bare broker peer that should stay
// in the dependencies list under connection_type=messaging_system.
func isKnownBrokerName(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "kafka", "rabbitmq", "nats", "pulsar", "activemq", "redis", "valkey":
		return true
	default:
		return false
	}
}

// normalizeServiceName strips K8s FQDN suffixes to produce a short service name.
// Examples:
//
//	"access-api.checkout.svc.cluster.local" → "access-api"
//	"access-api.checkout.svc.cluster.local" → "access-api"
//	"access-api.checkout.svc" → "access-api"
//	"access-api.checkout" → "access-api" (namespace-qualified)
//	"vault.adeo.no" → "vault.adeo.no" (external, unchanged)
//	"access-api" → "access-api" (already short)
func normalizeServiceName(name string) string {
	// Strip port if present
	host := name
	if before, _, ok := strings.Cut(name, ":"); ok {
		host = before
	}

	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return name
	}

	// Pattern: {service}.{namespace}.svc[.anything]
	if len(parts) >= 3 && parts[2] == "svc" {
		return parts[0]
	}

	// Pattern: {service}.{namespace} — only if second part is NOT a known TLD
	if len(parts) == 2 && !isKnownTLD(parts[1]) {
		return parts[0]
	}

	return name
}

// hexSuffixRe matches a trailing dash followed by 12+ hex characters.
// Used to collapse preview/canary deployment names back to the base service:
//
//	"nav-dekoratoren-28e8c72f0abdc4109d600c" → "nav-dekoratoren"
//
// 12 chars minimum avoids false positives on short hex-like suffixes
// (e.g., "my-service-cafe" is only 4 hex chars).
var hexSuffixRe = regexp.MustCompile(`^(.+)-[0-9a-f]{12,}$`)

// stripHexSuffix removes long hex suffixes from service names, typically
// generated by preview/canary deployment pipelines. This is applied during
// response assembly (not during BFS queries, where raw metric names are needed).
func stripHexSuffix(name string) string {
	if m := hexSuffixRe.FindStringSubmatch(name); m != nil {
		return m[1]
	}
	return name
}

// extractK8sNamespace returns the namespace portion of a K8s service name, or "".
func extractK8sNamespace(name string) string { //nolint:unused
	host := name
	if before, _, ok := strings.Cut(name, ":"); ok {
		host = before
	}

	parts := strings.Split(host, ".")
	if len(parts) >= 3 && parts[2] == "svc" {
		return parts[1]
	}
	if len(parts) == 2 && !isKnownTLD(parts[1]) {
		return parts[1]
	}
	return ""
}
