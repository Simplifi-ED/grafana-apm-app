import { buildClickStackServiceScene } from './buildClickStackServiceScene';

describe('buildClickStackServiceScene', () => {
  const base = {
    service: 'nova-api',
    namespace: 'sanad',
    envFilter: 'prod-k3s-omnivya',
    percentile: '0.95',
    percentileLabel: 'P95',
    from: 'now-1h',
    to: 'now',
    clickHouseUid: 'clickhouse',
    hasServerSpans: true,
  };

  it('returns null without a ClickHouse UID', () => {
    expect(buildClickStackServiceScene({ ...base, clickHouseUid: '' })).toBeNull();
  });

  it('returns null without a service name', () => {
    expect(buildClickStackServiceScene({ ...base, service: '' })).toBeNull();
  });

  it('builds an EmbeddedScene for ClickStack RED panels', () => {
    const scene = buildClickStackServiceScene(base);
    expect(scene).not.toBeNull();
    expect(scene!.state.body).toBeDefined();
  });
});
