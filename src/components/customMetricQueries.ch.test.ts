import { CustomMetric } from '../api/client';
import { customMetricExploreSQL, customMetricPanelsCH, customMetricScrapeFilter } from './customMetricQueries.ch';

function metric(over: Partial<CustomMetric>): CustomMetric {
  return { name: 'x', type: 'gauge', help: '', unit: '', series: 1, highCardinality: false, chart: 'gauge', ...over };
}

describe('customMetricScrapeFilter', () => {
  it('scopes by scrape ServiceName and optional namespace', () => {
    expect(customMetricScrapeFilter('checkout', 'prod-shop')).toContain("ServiceName = 'checkout'");
    expect(customMetricScrapeFilter('checkout', 'prod-shop')).toContain("Attributes['namespace'] = 'prod-shop'");
    expect(customMetricScrapeFilter('checkout', '')).not.toContain('namespace');
  });
});

describe('customMetricPanelsCH', () => {
  it('charts a counter as sum(Value)/interval on otel_metrics_sum', () => {
    const panels = customMetricPanelsCH(
      metric({ name: 'orders_total', type: 'counter', chart: 'rate' }),
      'checkout',
      'ns'
    );
    expect(panels).toHaveLength(1);
    expect(panels[0].rawSql).toContain('otel_metrics_sum');
    expect(panels[0].rawSql).toContain("MetricName = 'orders_total'");
    expect(panels[0].rawSql).toContain('sum(Value) / $__interval_s');
  });

  it('charts a histogram as quantilePrometheusHistogram p95', () => {
    const panels = customMetricPanelsCH(
      metric({ name: 'batch_duration_seconds', type: 'histogram', unit: 'seconds', chart: 'p95' }),
      'checkout',
      ''
    );
    expect(panels).toHaveLength(1);
    expect(panels[0].unit).toBe('s');
    expect(panels[0].rawSql).toContain('otel_metrics_histogram');
    expect(panels[0].rawSql).toContain('quantilePrometheusHistogram(0.95)');
  });

  it('charts a summary as throughput + avg on otel_metrics_summary', () => {
    const panels = customMetricPanelsCH(
      metric({ name: 'http_server_requests_seconds', type: 'summary', unit: 'seconds', chart: 'summary' }),
      'checkout',
      'ns'
    );
    expect(panels).toHaveLength(2);
    expect(panels[0].rawSql).toContain('sum(Count) / $__interval_s');
    expect(panels[1].rawSql).toContain('sum(Sum) / nullIf(sum(Count), 0)');
    expect(panels[1].unit).toBe('s');
  });

  it('charts a gauge as avg(Value) on otel_metrics_gauge', () => {
    const panels = customMetricPanelsCH(metric({ name: 'queue_depth', chart: 'gauge' }), 'checkout', '');
    expect(panels).toHaveLength(1);
    expect(panels[0].rawSql).toContain('otel_metrics_gauge');
    expect(panels[0].rawSql).toContain('avg(Value)');
  });
});

describe('customMetricExploreSQL', () => {
  it('returns the primary panel SQL', () => {
    const sql = customMetricExploreSQL(metric({ name: 'q', chart: 'gauge' }), 'svc', '');
    expect(sql).toContain("MetricName = 'q'");
    expect(sql).toBe(customMetricPanelsCH(metric({ name: 'q', chart: 'gauge' }), 'svc', '')[0].rawSql);
  });
});
