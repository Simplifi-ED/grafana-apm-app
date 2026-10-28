import { defineConfig } from 'eslint/config';
import baseConfig from './.config/eslint.config.mjs';

export default defineConfig([
  {
    ignores: [
      'nais-doc/**',
      'helm-charts/**',
      'apm-client/**',
      '**/logs',
      '**/*.log',
      '**/npm-debug.log*',
      '**/yarn-debug.log*',
      '**/yarn-error.log*',
      '**/.pnpm-debug.log*',
      '**/node_modules/',
      '.yarn/cache',
      '.yarn/unplugged',
      '.yarn/build-state.yml',
      '.yarn/install-state.gz',
      '**/.pnp.*',
      '**/pids',
      '**/*.pid',
      '**/*.seed',
      '**/*.pid.lock',
      '**/lib-cov',
      '**/coverage',
      '**/dist/',
      '**/artifacts/',
      '**/work/',
      '**/ci/',
      'test-results/',
      'playwright-report/',
      'blob-report/',
      'playwright/.cache/',
      'playwright/.auth/',
      // Local Playwright browser downloads (gitignored): vendored helper scripts
      // in Chromium bundles trip no-eval and fail `mise run frontend:check`
      // on machines where `playwright install` has run.
      '.playwright-browsers/',
      '**/.idea',
      '**/.eslintcache',
    ],
  },
  ...baseConfig,
  // React Compiler lint (eslint-plugin-react-hooks ≥7.1) flags common
  // fetch-in-effect loading patterns used across this plugin. Keep the
  // classic hooks rules; defer Compiler-strict setState-in-effect until a
  // dedicated refactor.
  {
    rules: {
      'react-hooks/set-state-in-effect': 'off',
    },
  },
]);
