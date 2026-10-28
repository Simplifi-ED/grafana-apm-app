import type { GrafanaTheme2, ThemeTypographyVariant } from '@grafana/data';

/**
 * Grafana 13.3 stable prefers typography.{sm,base,md,lg,xl,xxl};
 * Grafana nightly (13.3.0-*) currently only ships the older
 * {bodySmall,body,h5,h4,h2,h1} names. Resolve either shape.
 */
type ModernVariant = 'sm' | 'base' | 'md' | 'lg' | 'xl' | 'xxl';

const LEGACY: Record<ModernVariant, 'bodySmall' | 'body' | 'h5' | 'h4' | 'h2' | 'h1'> = {
  sm: 'bodySmall',
  base: 'body',
  md: 'h5',
  lg: 'h4',
  xl: 'h2',
  xxl: 'h1',
};

function isVariant(v: unknown): v is ThemeTypographyVariant {
  return typeof v === 'object' && v !== null && 'fontSize' in v && (v as ThemeTypographyVariant).fontSize != null;
}

export function typographyVariant(theme: GrafanaTheme2, variant: ModernVariant): ThemeTypographyVariant {
  const t = theme.typography as GrafanaTheme2['typography'] & Record<string, unknown>;
  const modern = t[variant];
  if (isVariant(modern)) {
    return modern;
  }
  const legacy = t[LEGACY[variant]];
  if (isVariant(legacy)) {
    return legacy;
  }
  return {
    fontSize: `${theme.typography.fontSize}px`,
    fontWeight: theme.typography.fontWeightRegular,
    lineHeight: 1.5,
    fontFamily: theme.typography.fontFamily,
  };
}

export function typographyFontSize(theme: GrafanaTheme2, variant: ModernVariant): string {
  return typographyVariant(theme, variant).fontSize;
}
