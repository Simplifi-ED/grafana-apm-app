import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { IssuesAttentionStrip } from './IssuesAttentionStrip';

describe('IssuesAttentionStrip', () => {
  it('hides when counts are zero', () => {
    const { container } = render(
      <IssuesAttentionStrip attention={{ new: 0, returnedAfterResolve: 0 }} onViewIssues={jest.fn()} />
    );
    expect(container).toBeEmptyDOMElement();
  });

  it('shows new and returned counts with CTAs', () => {
    const onViewIssues = jest.fn();
    const onViewAlerts = jest.fn();
    render(
      <IssuesAttentionStrip
        attention={{ new: 2, returnedAfterResolve: 1 }}
        onViewIssues={onViewIssues}
        onViewAlerts={onViewAlerts}
      />
    );
    expect(screen.getByText(/2 new/)).toBeInTheDocument();
    expect(screen.getByText(/1 returned after resolve/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'View Issues' }));
    expect(onViewIssues).toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Alert templates' }));
    expect(onViewAlerts).toHaveBeenCalled();
  });
});
