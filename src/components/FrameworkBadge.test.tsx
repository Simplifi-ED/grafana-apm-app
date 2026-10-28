import React from 'react';
import { render, screen } from '@testing-library/react';
import { FrameworkBadge } from './FrameworkBadge';

describe('FrameworkBadge', () => {
  it('renders nothing without a framework', () => {
    const { container } = render(<FrameworkBadge />);
    expect(container).toBeEmptyDOMElement();
  });

  it('renders a Grafana Badge for a known framework', () => {
    render(<FrameworkBadge framework="Go" />);
    expect(screen.getByText('Go')).toBeInTheDocument();
  });

  it('maps alias keys (cpp / webjs)', () => {
    const { rerender } = render(<FrameworkBadge framework="cpp" />);
    expect(screen.getByText('C++')).toBeInTheDocument();
    rerender(<FrameworkBadge framework="webjs" />);
    expect(screen.getByText('Browser')).toBeInTheDocument();
  });

  it('falls back for unknown frameworks', () => {
    render(<FrameworkBadge framework="Fortran" />);
    expect(screen.getByText('Fortran')).toBeInTheDocument();
  });
});
