import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import CredentialsPage from './CredentialsPage';
import { useCredentialsStore } from './credentialsStore';
import { AuthContext, type AuthState } from '../auth/AuthProvider';

function renderPage() {
  const value: AuthState = {
    user: { id: 'usr_1', email: 'x@y.com' }, loading: false,
    signup: vi.fn(), login: vi.fn(), logout: vi.fn(),
  };
  return render(
    <AuthContext.Provider value={value}>
      <MemoryRouter><CredentialsPage /></MemoryRouter>
    </AuthContext.Provider>,
  );
}

describe('CredentialsPage', () => {
  beforeEach(() => {
    useCredentialsStore.setState({ items: [], loading: false, error: null });
  });

  it('renders the add form', () => {
    renderPage();
    expect(screen.getByText(/Add a credential/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Name/)).toBeInTheDocument();
    expect(screen.getByLabelText(/API key/)).toBeInTheDocument();
  });

  it('shows the empty-state message when no credentials', () => {
    renderPage();
    expect(screen.getByText(/No credentials yet/)).toBeInTheDocument();
  });

  it('renders saved credentials in the table', () => {
    useCredentialsStore.setState({
      items: [
        { id: 'cred_1', name: 'openai_default', type: 'openai', created_at: new Date().toISOString() },
      ],
    });
    renderPage();
    expect(screen.getByText('openai_default')).toBeInTheDocument();
    expect(screen.getByText('openai')).toBeInTheDocument();
  });
});
