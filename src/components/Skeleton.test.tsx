import React from 'react';
import { render, screen } from '@testing-library/react';
import { TableSkeleton } from './Skeleton';

describe('TableSkeleton', () => {
  it('exposes a busy loading status', () => {
    render(<TableSkeleton rows={3} />);
    expect(screen.getByRole('status', { name: /loading/i })).toHaveAttribute('aria-busy', 'true');
  });
});
