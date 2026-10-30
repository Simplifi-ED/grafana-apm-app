import { buildClickStackFrontendScene } from './buildClickStackFrontendScene';

describe('buildClickStackFrontendScene', () => {
  const base = {
    service: 'storefront',
    environment: 'prod',
    from: 'now-1h',
    to: 'now',
    clickHouseUid: 'clickhouse',
  };

  it('returns null without uid or service', () => {
    expect(buildClickStackFrontendScene({ ...base, clickHouseUid: '' })).toBeNull();
    expect(buildClickStackFrontendScene({ ...base, service: '' })).toBeNull();
  });

  it('builds an EmbeddedScene with CH-backed panels', () => {
    const scene = buildClickStackFrontendScene(base);
    expect(scene).not.toBeNull();
    expect(scene!.state.body).toBeDefined();
  });
});
