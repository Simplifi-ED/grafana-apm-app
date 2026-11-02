/**
 * Sanitization utilities for user-controlled values interpolated into queries.
 * These prevent injection attacks via crafted URL params.
 */

/** Allowlist for label values: alphanumeric, dots, dashes, underscores, slashes, spaces, colons */
const SAFE_LABEL = /^[a-zA-Z0-9._\-/ :@]+$/;

/**
 * Validate a label value for safe use in PromQL/LogQL label matchers.
 * Returns the value if safe, or empty string if invalid.
 */
export function sanitizeLabelValue(value: string): string {
  if (!value || value.length > 256) {
    return '';
  }
  if (!SAFE_LABEL.test(value)) {
    return '';
  }
  return value;
}

/**
 * Escape a string for safe use inside double-quoted TraceQL/LogQL string literals.
 * Escapes backslashes and double quotes.
 */
export function escapeQueryString(value: string): string {
  return value.replace(/\\/g, '\\\\').replace(/"/g, '\\"');
}

/**
 * Quote a value as a ClickHouse single-quoted string literal (`'` → `''`).
 * Use for Explore/rawSql deep links — do not rely on sanitizeLabelValue alone
 * (it rejects apostrophes that are valid in escaped SQL literals).
 */
export function chStringLiteral(value: string): string {
  return `'${String(value).replace(/'/g, "''")}'`;
}
