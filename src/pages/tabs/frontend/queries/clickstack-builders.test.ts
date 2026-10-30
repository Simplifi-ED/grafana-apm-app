import {
  browserBreakdownSql,
  browserEventsTopSql,
  browserEventsVolumeSql,
  browserFromUaExpr,
  clickStackFrontendWhere,
  cwvRatingExpr,
  interactivitySql,
  jsExceptionsSql,
  layoutStabilitySql,
  longtaskP75Sql,
  measurementsVolumeSql,
  pageLoadVitalsSql,
  perPageSql,
  ratingDistributionSql,
} from './clickstack-builders';

describe('clickstack frontend SQL builders', () => {
  it('scopes service and optional cluster', () => {
    expect(clickStackFrontendWhere('storefront')).toContain("ServiceName = 'storefront'");
    expect(clickStackFrontendWhere('storefront', 'prod')).toContain("ResourceAttributes['cluster'] = 'prod'");
  });

  it('rejects unsafe service names via sanitize', () => {
    expect(clickStackFrontendWhere("O'Reilly")).toBe("ServiceName = ''");
  });

  it('page load SQL uses webvitals and LCP/FCP/TTFB', () => {
    const sql = pageLoadVitalsSql('storefront', 'prod');
    expect(sql).toContain("SpanName = 'webvitals'");
    expect(sql).toContain("SpanAttributes['lcp']");
    expect(sql).toContain('$__timeFilter(Timestamp)');
    expect(sql).toContain('prod');
  });

  it('splits INP and CLS into dedicated queries', () => {
    expect(interactivitySql('svc')).toContain("SpanAttributes['inp']");
    expect(layoutStabilitySql('svc')).toContain("SpanAttributes['cls']");
  });

  it('measurements and exceptions series', () => {
    expect(measurementsVolumeSql('svc')).toContain("SpanName = 'webvitals'");
    const ex = jsExceptionsSql('svc');
    expect(ex).toContain('unhandledrejection');
    expect(ex).toContain("startsWith(SpanName, 'error.')");
  });

  it('browser events exclude webvitals/HTTP/errors and include longtask', () => {
    const top = browserEventsTopSql('storefront', 'prod');
    expect(top).toContain("NOT startsWith(SpanName, 'HTTP ')");
    expect(top).toContain("'webvitals'");
    expect(top).toContain('AS event');
    expect(browserEventsVolumeSql('svc')).toContain('AS events');
    expect(longtaskP75Sql('svc')).toContain("SpanName = 'longtask'");
    expect(longtaskP75Sql('svc')).toContain('Duration / 1e6');
  });

  it('rating distribution uses CWV thresholds', () => {
    expect(cwvRatingExpr('lcp', 'v')).toContain('2500');
    expect(cwvRatingExpr('lcp', 'v')).toContain('4000');
    expect(cwvRatingExpr('lcp', 'v')).toContain('needs-improvement');
    const sql = ratingDistributionSql('storefront');
    expect(sql).toContain('UNION ALL');
    expect(sql).toContain('good');
  });

  it('browser breakdown joins session UA', () => {
    const sql = browserBreakdownSql('storefront');
    expect(sql).toContain('ua_by_session');
    expect(sql).toContain('http.user_agent');
    expect(sql).toContain('rum.sessionId');
    expect(browserFromUaExpr('ua')).toContain('Chrome');
    expect(browserFromUaExpr('ua')).toContain('Edge');
  });

  it('per-page uses URL path()', () => {
    const sql = perPageSql('storefront');
    expect(sql).toContain("path(SpanAttributes['location.href'])");
    expect(sql).toContain('p75_lcp');
  });
});
