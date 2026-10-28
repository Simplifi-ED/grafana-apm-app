import React from 'react';
import { Badge, type BadgeColor } from '@grafana/ui';

/**
 * Framework → Grafana Badge color (named viz colors from @grafana/ui Badge).
 * @see https://developers.grafana.com/ui/latest/index.html?path=/docs/information-badge--docs
 */
export const FRAMEWORK_BADGES: Record<string, { label: string; color: BadgeColor }> = {
  Ktor: { label: 'Ktor', color: 'purple' },
  'Spring Boot': { label: 'Spring', color: 'green' },
  'Node.js': { label: 'Node.js', color: 'green' },
  Go: { label: 'Go', color: 'blue' },
  Java: { label: 'Java', color: 'blue' },
  Python: { label: 'Python', color: 'blue' },
  '.NET': { label: '.NET', color: 'purple' },
  Rust: { label: 'Rust', color: 'orange' },
  'C++': { label: 'C++', color: 'purple' },
  cpp: { label: 'C++', color: 'purple' },
  C: { label: 'C', color: 'darkgrey' },
  Browser: { label: 'Browser', color: 'blue' },
  webjs: { label: 'Browser', color: 'blue' },
  TypeScript: { label: 'TypeScript', color: 'blue' },
  PHP: { label: 'PHP', color: 'purple' },
  Ruby: { label: 'Ruby', color: 'red' },
  Swift: { label: 'Swift', color: 'orange' },
  Kotlin: { label: 'Kotlin', color: 'purple' },
};

/** Compact framework/SDK badge using Grafana UI Badge. */
export function FrameworkBadge({ framework, className }: { framework?: string; className?: string }) {
  if (!framework) {
    return null;
  }
  const key = framework.trim();
  const info = FRAMEWORK_BADGES[framework] ?? FRAMEWORK_BADGES[key];
  if (!info) {
    return <Badge className={className} text={framework} color="darkgrey" />;
  }
  return (
    <Badge
      className={className}
      text={info.label}
      color={info.color}
      icon={info.label === 'Browser' ? 'monitor' : undefined}
    />
  );
}
