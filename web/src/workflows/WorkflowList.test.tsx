import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, it, expect, beforeEach } from 'vitest';
import WorkflowList from './WorkflowList';
import { useWorkflowsStore } from './workflowsStore';
import { AuthContext, type AuthState } from '../auth/AuthProvider';
import type { User } from '../api/types';

function renderList(user: User | null) {
  const value: AuthState = {
    user, loading: false,
    signup: async () => {}, login: async () => {}, logout: async () => {},
  };
  return render(
    <AuthContext.Provider value={value}>
      <MemoryRouter><WorkflowList /></MemoryRouter>
    </AuthContext.Provider>,
  );
}

describe('WorkflowList', () => {
  beforeEach(() => {
    useWorkflowsStore.setState({ list: [], current: null });
  });

  it('shows the signed-in email', () => {
    renderList({ id: 'usr_1', email: 'foo@bar.com' });
    expect(screen.getByText('foo@bar.com')).toBeInTheDocument();
  });

  it('renders an empty-state message when there are no workflows', () => {
    renderList({ id: 'usr_1', email: 'foo@bar.com' });
    expect(screen.getByText(/no workflows yet/i)).toBeInTheDocument();
  });

  it('renders workflow rows from the store', () => {
    useWorkflowsStore.setState({
      list: [
        { id: 'wf_1', name: 'Daily summary', status: 'draft', version: 1 },
        { id: 'wf_2', name: 'Order notifier', status: 'active', version: 2 },
      ],
    });
    renderList({ id: 'usr_1', email: 'foo@bar.com' });
    expect(screen.getByText('Daily summary')).toBeInTheDocument();
    expect(screen.getByText('Order notifier')).toBeInTheDocument();
  });
});
