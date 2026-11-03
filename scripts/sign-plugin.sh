#!/usr/bin/env bash
# Sign dist/ for Grafana.
#
# - GRAFANA_PLUGIN_ROOT_URLS set  → private signature (allowed before catalog approval)
# - GRAFANA_PLUGIN_ROOT_URLS empty → public signature at whatever level Grafana.com
#   has granted this plugin (community/commercial). Do NOT force --signatureType:
#   the API rejects community until review assigns that level (409 InvalidArgument).
#
# Flip private → community after verification: clear the GRAFANA_PLUGIN_ROOT_URLS
# repo secret. No code change required.
set -euo pipefail

: "${GRAFANA_ACCESS_POLICY_TOKEN:?GRAFANA_ACCESS_POLICY_TOKEN is required}"

if [[ -n "${GRAFANA_PLUGIN_ROOT_URLS:-}" ]]; then
  echo "Signing plugin (private) with rootUrls"
  exec npx --yes @grafana/sign-plugin@latest --rootUrls "$GRAFANA_PLUGIN_ROOT_URLS"
fi

echo "Signing plugin (public — Grafana-granted signature level)"
exec npx --yes @grafana/sign-plugin@latest
