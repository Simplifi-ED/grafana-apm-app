/**
 * ClickHouse SQL for HyperDX / @hyperdx/otel-web Frontend tab panels.
 * Spans live in otel_traces (SpanName=webvitals); browser UA is joined via rum.sessionId.
 */

import { sanitizeLabelValue } from '../../../../utils/sanitize';

function sqlStr(value: string): string {
  return `'${sanitizeLabelValue(value).replace(/'/g, "''")}'`;
}

/** Service + optional cluster filter for browser spans. */
export function clickStackFrontendWhere(service: string, environment = ''): string {
  let where = `ServiceName = ${sqlStr(service)}`;
  if (environment) {
    where += ` AND ResourceAttributes['cluster'] = ${sqlStr(environment)}`;
  }
  return where;
}

/** Parse Chrome/Edge/Firefox/Safari from a UA string column expression. */
export function browserFromUaExpr(uaCol: string): string {
  return `multiIf(
  positionCaseInsensitive(${uaCol}, 'Edg/') > 0, 'Edge',
  positionCaseInsensitive(${uaCol}, 'Chrome/') > 0, 'Chrome',
  positionCaseInsensitive(${uaCol}, 'Firefox/') > 0, 'Firefox',
  positionCaseInsensitive(${uaCol}, 'Safari/') > 0, 'Safari',
  'Other'
)`;
}

/** CWV rating from a numeric value expression (web.dev thresholds). */
export function cwvRatingExpr(vital: 'lcp' | 'fcp' | 'cls' | 'inp' | 'ttfb', valueExpr: string): string {
  const bounds: Record<string, [number, number]> = {
    lcp: [2500, 4000],
    fcp: [1800, 3000],
    cls: [0.1, 0.25],
    inp: [200, 500],
    ttfb: [800, 1800],
  };
  const [good, poor] = bounds[vital];
  return `multiIf(${valueExpr} <= ${good}, 'good', ${valueExpr} <= ${poor}, 'needs-improvement', 'poor')`;
}

export function pageLoadVitalsSql(service: string, environment = ''): string {
  const where = clickStackFrontendWhere(service, environment);
  return `
SELECT
  $__timeInterval(Timestamp) AS time,
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['lcp']), SpanAttributes['lcp'] != '') AS lcp,
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['fcp']), SpanAttributes['fcp'] != '') AS fcp,
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['ttfb']), SpanAttributes['ttfb'] != '') AS ttfb
FROM \`default\`.\`otel_traces\`
WHERE ${where}
  AND SpanName = 'webvitals'
  AND $__timeFilter(Timestamp)
GROUP BY time
ORDER BY time`.trim();
}

export function interactivitySql(service: string, environment = ''): string {
  const where = clickStackFrontendWhere(service, environment);
  return `
SELECT
  $__timeInterval(Timestamp) AS time,
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['inp']), SpanAttributes['inp'] != '') AS inp
FROM \`default\`.\`otel_traces\`
WHERE ${where}
  AND SpanName = 'webvitals'
  AND $__timeFilter(Timestamp)
GROUP BY time
ORDER BY time`.trim();
}

export function layoutStabilitySql(service: string, environment = ''): string {
  const where = clickStackFrontendWhere(service, environment);
  return `
SELECT
  $__timeInterval(Timestamp) AS time,
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['cls']), SpanAttributes['cls'] != '') AS cls
FROM \`default\`.\`otel_traces\`
WHERE ${where}
  AND SpanName = 'webvitals'
  AND $__timeFilter(Timestamp)
GROUP BY time
ORDER BY time`.trim();
}

export function measurementsVolumeSql(service: string, environment = ''): string {
  const where = clickStackFrontendWhere(service, environment);
  return `
SELECT
  $__timeInterval(Timestamp) AS time,
  count() AS measurements
FROM \`default\`.\`otel_traces\`
WHERE ${where}
  AND SpanName = 'webvitals'
  AND $__timeFilter(Timestamp)
GROUP BY time
ORDER BY time`.trim();
}

export function jsExceptionsSql(service: string, environment = ''): string {
  const where = clickStackFrontendWhere(service, environment);
  return `
SELECT
  $__timeInterval(Timestamp) AS time,
  count() AS exceptions
FROM \`default\`.\`otel_traces\`
WHERE ${where}
  AND (SpanName IN ('unhandledrejection', 'error', 'exception')
       OR startsWith(SpanName, 'error.'))
  AND $__timeFilter(Timestamp)
GROUP BY time
ORDER BY time`.trim();
}

/** good / needs-improvement / poor counts across all measured vitals. */
export function ratingDistributionSql(service: string, environment = ''): string {
  const where = clickStackFrontendWhere(service, environment);
  return `
SELECT rating, count() AS value FROM (
  SELECT ${cwvRatingExpr('lcp', "toFloat64OrZero(SpanAttributes['lcp'])")} AS rating
  FROM \`default\`.\`otel_traces\`
  WHERE ${where} AND SpanName = 'webvitals' AND SpanAttributes['lcp'] != '' AND $__timeFilter(Timestamp)
  UNION ALL
  SELECT ${cwvRatingExpr('fcp', "toFloat64OrZero(SpanAttributes['fcp'])")} AS rating
  FROM \`default\`.\`otel_traces\`
  WHERE ${where} AND SpanName = 'webvitals' AND SpanAttributes['fcp'] != '' AND $__timeFilter(Timestamp)
  UNION ALL
  SELECT ${cwvRatingExpr('cls', "toFloat64OrZero(SpanAttributes['cls'])")} AS rating
  FROM \`default\`.\`otel_traces\`
  WHERE ${where} AND SpanName = 'webvitals' AND SpanAttributes['cls'] != '' AND $__timeFilter(Timestamp)
  UNION ALL
  SELECT ${cwvRatingExpr('inp', "toFloat64OrZero(SpanAttributes['inp'])")} AS rating
  FROM \`default\`.\`otel_traces\`
  WHERE ${where} AND SpanName = 'webvitals' AND SpanAttributes['inp'] != '' AND $__timeFilter(Timestamp)
  UNION ALL
  SELECT ${cwvRatingExpr('ttfb', "toFloat64OrZero(SpanAttributes['ttfb'])")} AS rating
  FROM \`default\`.\`otel_traces\`
  WHERE ${where} AND SpanName = 'webvitals' AND SpanAttributes['ttfb'] != '' AND $__timeFilter(Timestamp)
)
GROUP BY rating
ORDER BY rating`.trim();
}

/** Avg vitals by browser — UA from sibling spans on the same rum.sessionId. */
export function browserBreakdownSql(service: string, environment = ''): string {
  const where = clickStackFrontendWhere(service, environment);
  const browser = browserFromUaExpr('u.ua');
  return `
WITH ua_by_session AS (
  SELECT
    ResourceAttributes['rum.sessionId'] AS sid,
    any(SpanAttributes['http.user_agent']) AS ua
  FROM \`default\`.\`otel_traces\`
  WHERE ${where}
    AND ResourceAttributes['rum.sessionId'] != ''
    AND SpanAttributes['http.user_agent'] != ''
    AND $__timeFilter(Timestamp)
  GROUP BY sid
),
vitals AS (
  SELECT
    ResourceAttributes['rum.sessionId'] AS sid,
    toFloat64OrZero(SpanAttributes['lcp']) AS lcp,
    toFloat64OrZero(SpanAttributes['fcp']) AS fcp,
    toFloat64OrZero(SpanAttributes['ttfb']) AS ttfb,
    toFloat64OrZero(SpanAttributes['inp']) AS inp,
    toFloat64OrZero(SpanAttributes['cls']) AS cls
  FROM \`default\`.\`otel_traces\`
  WHERE ${where}
    AND SpanName = 'webvitals'
    AND ResourceAttributes['rum.sessionId'] != ''
    AND $__timeFilter(Timestamp)
)
SELECT
  ${browser} AS browser,
  avgIf(v.lcp, v.lcp > 0) AS avg_lcp,
  avgIf(v.fcp, v.fcp > 0) AS avg_fcp,
  avgIf(v.ttfb, v.ttfb > 0) AS avg_ttfb,
  avgIf(v.inp, v.inp > 0) AS avg_inp,
  avgIf(v.cls, v.cls > 0) AS avg_cls,
  count() AS samples
FROM vitals v
INNER JOIN ua_by_session u ON v.sid = u.sid
GROUP BY browser
ORDER BY samples DESC
LIMIT 20`.trim();
}

/** P75 vitals by URL path. */
export function perPageSql(service: string, environment = ''): string {
  const where = clickStackFrontendWhere(service, environment);
  return `
SELECT
  path(SpanAttributes['location.href']) AS page,
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['lcp']), SpanAttributes['lcp'] != '') AS p75_lcp,
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['fcp']), SpanAttributes['fcp'] != '') AS p75_fcp,
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['ttfb']), SpanAttributes['ttfb'] != '') AS p75_ttfb,
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['inp']), SpanAttributes['inp'] != '') AS p75_inp,
  quantileIf(0.75)(toFloat64OrZero(SpanAttributes['cls']), SpanAttributes['cls'] != '') AS p75_cls,
  count() AS samples
FROM \`default\`.\`otel_traces\`
WHERE ${where}
  AND SpanName = 'webvitals'
  AND SpanAttributes['location.href'] != ''
  AND $__timeFilter(Timestamp)
GROUP BY page
ORDER BY samples DESC
LIMIT 50`.trim();
}
