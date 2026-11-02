import { config } from '@grafana/runtime';
import { canEditPlugin } from './orgRole';

jest.mock('@grafana/runtime', () => ({
  config: { bootData: { user: { orgRole: 'Viewer' } } },
}));

function setRole(role: string) {
  (config as { bootData: { user: { orgRole: string } } }).bootData.user.orgRole = role;
}

describe('canEditPlugin', () => {
  afterEach(() => setRole('Viewer'));

  it('allows Admin and Editor', () => {
    setRole('Admin');
    expect(canEditPlugin()).toBe(true);
    setRole('Editor');
    expect(canEditPlugin()).toBe(true);
    setRole('editor');
    expect(canEditPlugin()).toBe(true);
  });

  it('blocks Viewer and missing role', () => {
    setRole('Viewer');
    expect(canEditPlugin()).toBe(false);
    setRole('');
    expect(canEditPlugin()).toBe(false);
  });
});
