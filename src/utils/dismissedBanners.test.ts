import { config } from '@grafana/runtime';
import { dismissBanner, isBannerDismissed, BANNER_OWNERSHIP_COVERAGE } from './dismissedBanners';

jest.mock('@grafana/runtime', () => ({
  config: { bootData: { user: { orgId: 1 } } },
}));

function setOrgId(orgId: number) {
  (config as { bootData: { user: { orgId: number } } }).bootData.user.orgId = orgId;
}

describe('dismissedBanners', () => {
  beforeEach(() => {
    localStorage.clear();
    setOrgId(1);
  });

  it('starts undismissed', () => {
    expect(isBannerDismissed(BANNER_OWNERSHIP_COVERAGE)).toBe(false);
  });

  it('persists dismiss across reads', () => {
    dismissBanner(BANNER_OWNERSHIP_COVERAGE);
    expect(isBannerDismissed(BANNER_OWNERSHIP_COVERAGE)).toBe(true);
  });

  it('scopes dismissals by Grafana organization', () => {
    dismissBanner(BANNER_OWNERSHIP_COVERAGE);
    setOrgId(2);
    expect(isBannerDismissed(BANNER_OWNERSHIP_COVERAGE)).toBe(false);
    setOrgId(1);
    expect(isBannerDismissed(BANNER_OWNERSHIP_COVERAGE)).toBe(true);
  });

  it('ignores corrupt localStorage payloads', () => {
    localStorage.setItem('omnivya-apm:dismissed-banners', 'not-json');
    expect(isBannerDismissed(BANNER_OWNERSHIP_COVERAGE)).toBe(false);

    localStorage.setItem('omnivya-apm:dismissed-banners', JSON.stringify({ version: 1, ids: [123] }));
    expect(isBannerDismissed(BANNER_OWNERSHIP_COVERAGE)).toBe(false);
  });

  it('swallows localStorage write failures', () => {
    const spy = jest.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('quota');
    });
    expect(() => dismissBanner(BANNER_OWNERSHIP_COVERAGE)).not.toThrow();
    spy.mockRestore();
  });
});
