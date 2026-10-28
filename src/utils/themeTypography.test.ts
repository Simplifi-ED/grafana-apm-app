import { createTheme } from '@grafana/data';
import { typographyFontSize, typographyVariant } from './themeTypography';

describe('themeTypography', () => {
  it('reads modern sm/base variants when present', () => {
    const theme = createTheme();
    // Our @grafana/data build still provides modern names; assert the happy path.
    if (theme.typography.sm) {
      expect(typographyFontSize(theme, 'sm')).toBe(theme.typography.sm.fontSize);
    }
    if (theme.typography.base) {
      expect(typographyFontSize(theme, 'base')).toBe(theme.typography.base.fontSize);
    }
  });

  it('falls back to bodySmall/body when modern names are stripped (Grafana nightly)', () => {
    const theme = createTheme();
    const stripped = {
      ...theme,
      typography: {
        ...theme.typography,
        sm: undefined,
        base: undefined,
        md: undefined,
        lg: undefined,
        xl: undefined,
        xxl: undefined,
      },
    };

    expect(typographyFontSize(stripped as typeof theme, 'sm')).toBe(theme.typography.bodySmall.fontSize);
    expect(typographyFontSize(stripped as typeof theme, 'base')).toBe(theme.typography.body.fontSize);
    expect(typographyFontSize(stripped as typeof theme, 'md')).toBe(theme.typography.h5.fontSize);
    expect(typographyVariant(stripped as typeof theme, 'lg').fontSize).toBe(theme.typography.h4.fontSize);
  });
});
