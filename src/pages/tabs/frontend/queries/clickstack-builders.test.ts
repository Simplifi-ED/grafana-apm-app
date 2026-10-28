import {
  browserBreakdownSql,
  browserFromUaExpr,
  clickStackFrontendWhere,
  cwvRatingExpr,
  interactivitySql,
  jsExceptionsSql,
  layoutStabilitySql,
  measurementsVolumeSql,
  pageLoadVitalsSql,
  perPageSql,
  ratingDistributionSql,
} from './clickstack-builders';

describe('clickstack frontend SQL builders', () => {
  it('scopes service and optional cluster', () => {
    expect(clickStackFrontendWhere('nova-frontend')).toContain("ServiceName = 'nova-frontend'");
    expect(clickStackFrontendWhere('nova-frontend', "prod-k3s-omnivya")).toContain(
      "ResourceAttributes['cluster'] = 'prod-k3s-omnivya'"
    );
  });

  it('rejects unsafe service names via sanitize', () => {
    expect(clickStackFrontendWhere("O'Reilly")).toBe("ServiceName = ''");
  });

  it('page load SQL uses webvitals and LCP/FCP/TTFB', () => {
    const sql = pageLoadVitalsSql('nova-frontend', 'prod');
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

  it('rating distribution uses CWV thresholds', () => {
    expect(cwvRatingExpr('lcp', 'v')).toContain('2500');
    expect(cwvRatingExpr('lcp', 'v')).toContain('4000');
    expect(cwvRatingExpr('lcp', 'v')).toContain('needs-improvement');
    const sql = ratingDistributionSql('nova-frontend');
    expect(sql).toContain('UNION ALL');
    expect(sql).toContain('good');
  });

  it('browser breakdown joins session UA', () => {
    const sql = browserBreakdownSql('nova-frontend');
    expect(sql).toContain('ua_by_session');
    expect(sql).toContain("http.user_agent");
    expect(sql).toContain('rum.sessionId');
    expect(browserFromUaExpr('ua')).toContain('Chrome');
    expect(browserFromUaExpr('ua')).toContain('Edge');
  });

  it('per-page uses URL path()', () => {
    const sql = perPageSql('nova-frontend');
    expect(sql).toContain("path(SpanAttributes['location.href'])");
    expect(sql).toContain('p75_lcp');
  });
});
