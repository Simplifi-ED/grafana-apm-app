import React from 'react';
import { render, screen } from '@testing-library/react';
import { OwnershipCard } from './OwnershipCard';
import { Ownership } from '../api/ownership';

function owned(partial: Partial<Ownership> = {}): Ownership {
  return {
    team: 'platform',
    contact: '#platform',
    repo: 'https://github.com/o/r',
    source: 'telemetry',
    stale: false,
    teamKnown: true,
    ...partial,
  };
}

describe('OwnershipCard', () => {
  it('renders owned strip', () => {
    render(<OwnershipCard data={owned()} compact />);
    expect(screen.getByTestId('ownership-owned')).toBeInTheDocument();
    expect(screen.getByText('platform')).toBeInTheDocument();
    expect(screen.getByText('telemetry')).toBeInTheDocument();
  });

  it('renders unowned empty state', () => {
    render(
      <OwnershipCard
        data={{
          stale: false,
          teamKnown: true,
          emptyHint: {
            attrs: ['owner.team'],
            snippet: 'export OTEL_RESOURCE_ATTRIBUTES="owner.team=platform"',
            docPath: 'docs/ownership.md',
          },
        }}
      />
    );
    expect(screen.getByTestId('ownership-unowned')).toBeInTheDocument();
    expect(screen.getByText(/Add/)).toBeInTheDocument();
  });

  it('shows stale and conflict', () => {
    render(
      <OwnershipCard
        data={owned({
          stale: true,
          conflict: {
            winner: 'telemetry',
            losers: [{ source: 'annotation', fields: { team: 'other' } }],
          },
        })}
      />
    );
    expect(screen.getByText('stale')).toBeInTheDocument();
    expect(screen.getByText(/Other source disagreed/)).toBeInTheDocument();
  });
});
