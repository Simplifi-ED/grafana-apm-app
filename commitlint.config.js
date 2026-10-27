// Conventional Commits rules for clickhouse-apm-app. This underpins release-please's
// version inference and CHANGELOG generation. package.json has no "type": "module",
// so this file is CommonJS (module.exports) — unlike the browser SDK SDK's ESM variant.
module.exports = {
  extends: ['@commitlint/config-conventional'],
};
