import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { FeedbackPanel } from './FeedbackPanel';
import * as client from '../../../../api/client';

jest.mock('../../../../api/client', () => ({
  ...jest.requireActual('../../../../api/client'),
  getFeedback: jest.fn(),
}));

function renderPanel(clickStack = false) {
  return render(
    <MemoryRouter>
      <FeedbackPanel namespace="ns" service="svc" clickStack={clickStack} />
    </MemoryRouter>
  );
}

describe('FeedbackPanel', () => {
  const getFeedback = client.getFeedback as jest.Mock;

  beforeEach(() => {
    getFeedback.mockReset();
  });

  it('renders feedback entries', async () => {
    getFeedback.mockResolvedValue({
      feedback: [
        {
          timeMs: Date.now() - 120_000,
          message: 'Submit button stuck',
          category: 'bug',
          appVersion: '1.72.1',
          sessionId: 'sess-abcdef',
        },
      ],
    });
    renderPanel(true);

    await waitFor(() => expect(screen.getByText('Submit button stuck')).toBeInTheDocument());
    expect(screen.getByText('bug')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Feedback' })).toBeInTheDocument();
    expect(screen.getByRole('list', { name: 'Feedback' })).toBeInTheDocument();
  });

  it('shows soft empty when there is no feedback', async () => {
    getFeedback.mockResolvedValue({ feedback: [] });
    renderPanel();

    await waitFor(() => expect(screen.getByText('No feedback')).toBeInTheDocument());
    expect(screen.getByText(/selected time range/)).toBeInTheDocument();
  });

  it('surfaces ClickStack alias-miss hint', async () => {
    getFeedback.mockResolvedValue({
      feedback: [],
      hint: 'No serviceAliases entry for "nova-frontend". Add jsonData.serviceAliases…',
    });
    renderPanel(true);

    await waitFor(() => expect(screen.getByText(/serviceAliases/)).toBeInTheDocument());
    expect(screen.getByText('Configure service alias')).toBeInTheDocument();
  });

  it('surfaces enablement hint when faro.feedback is absent', async () => {
    getFeedback.mockResolvedValue({
      feedback: [],
      hint: 'No faro.feedback events in ClickHouse… see plan ticket 21.',
    });
    renderPanel(true);

    await waitFor(() => expect(screen.getByText('Feedback events not detected')).toBeInTheDocument());
    expect(screen.getByText(/ticket 21/)).toBeInTheDocument();
  });

  it('surfaces Loki unavailability on Faro path', async () => {
    getFeedback.mockResolvedValue({ feedback: [], unavailable: true });
    renderPanel(false);

    await waitFor(() => expect(screen.getByText('Loki is not available')).toBeInTheDocument());
  });
});
