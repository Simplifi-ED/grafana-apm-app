import { applyScorecardOverrides, getScorecardNaOverrides, setScorecardNaOverride } from './scorecardOverrides';
import { ScorecardResponse } from '../api/scorecard';

const base: ScorecardResponse = {
  readiness: {
    score: 1,
    total: 2,
    checks: [
      { key: 'faro', label: 'Browser RUM', ok: false, hint: 'Add Faro' },
      { key: 'alertRules', label: 'Alert rules', ok: true, hint: '' },
    ],
  },
};

describe('scorecardOverrides', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('persists and reads N/A overrides per service', () => {
    setScorecardNaOverride('ns', 'svc', 'faro', true, 'prod');
    expect(Array.from(getScorecardNaOverrides('ns', 'svc', 'prod'))).toEqual(['faro']);
    expect(getScorecardNaOverrides('ns', 'svc', 'dev').size).toBe(0);

    setScorecardNaOverride('ns', 'svc', 'faro', false, 'prod');
    expect(getScorecardNaOverrides('ns', 'svc', 'prod').size).toBe(0);
  });

  it('applies overrides and recomputes score/total', () => {
    const next = applyScorecardOverrides(base, new Set(['faro']));
    expect(next.readiness.score).toBe(1);
    expect(next.readiness.total).toBe(1);
    expect(next.readiness.checks[0].notApplicable).toBe(true);
    expect(next.readiness.checks[0].ok).toBe(false);
  });

  it('leaves response unchanged when no overrides', () => {
    expect(applyScorecardOverrides(base, new Set())).toBe(base);
  });
});
