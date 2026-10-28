import React from 'react';
import { render, screen } from '@testing-library/react';
import { CardGridSkeleton, PanelSkeleton, TableSkeleton } from './Skeleton';

describe('skeletons', () => {
  it('TableSkeleton exposes a busy loading status', () => {
    render(<TableSkeleton rows={3} />);
    expect(screen.getByRole('status', { name: /loading/i })).toHaveAttribute('aria-busy', 'true');
  });

  it('CardGridSkeleton and PanelSkeleton expose busy status', () => {
    const { rerender } = render(<CardGridSkeleton cards={2} />);
    expect(screen.getByRole('status', { name: /loading/i })).toHaveAttribute('aria-busy', 'true');
    rerender(<PanelSkeleton height={120} />);
    expect(screen.getByRole('status', { name: /loading/i })).toHaveAttribute('aria-busy', 'true');
  });
});
