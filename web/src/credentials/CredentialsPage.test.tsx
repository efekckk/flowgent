import { render, screen, fireEvent, waitFor } from '@testing-library/react';
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

describe('CredentialsPage type-switching', () => {
  beforeEach(() => {
    useCredentialsStore.setState({ items: [], loading: false, error: null });
  });

  it('renders SMTP fields when smtp type is selected', () => {
    renderPage();
    const select = screen.getByLabelText(/Type/) as HTMLSelectElement;
    fireEvent.change(select, { target: { value: 'smtp' } });
    expect(screen.getByLabelText(/Host/)).toBeInTheDocument();
    expect(screen.getByLabelText(/Port/)).toBeInTheDocument();
    expect(screen.getByLabelText(/Username/)).toBeInTheDocument();
    expect(screen.getByLabelText(/^Password$/)).toBeInTheDocument();
    expect(screen.getByLabelText(/From address/)).toBeInTheDocument();
  });

  it('renders DSN field when postgres type is selected', () => {
    renderPage();
    const select = screen.getByLabelText(/Type/) as HTMLSelectElement;
    fireEvent.change(select, { target: { value: 'postgres' } });
    expect(screen.getByLabelText(/DSN/)).toBeInTheDocument();
  });

  it('renders the Slack webhook URL field', () => {
    renderPage();
    const select = screen.getByLabelText(/Type/) as HTMLSelectElement;
    fireEvent.change(select, { target: { value: 'slack_webhook' } });
    expect(screen.getByLabelText(/Webhook URL/)).toBeInTheDocument();
  });

  it('renders the Telegram bot token field', () => {
    renderPage();
    const select = screen.getByLabelText(/Type/) as HTMLSelectElement;
    fireEvent.change(select, { target: { value: 'telegram_bot' } });
    expect(screen.getByLabelText(/Bot token/)).toBeInTheDocument();
  });

  it('submits an SMTP secret with all fields', async () => {
    const createSpy = vi.fn().mockResolvedValue(undefined);
    useCredentialsStore.setState({ create: createSpy });

    renderPage();

    fireEvent.change(screen.getByLabelText(/Type/), { target: { value: 'smtp' } });
    fireEvent.change(screen.getByLabelText(/Name/), { target: { value: 'smtp_prod' } });
    fireEvent.change(screen.getByLabelText(/Host/), { target: { value: 'smtp.example.com' } });
    fireEvent.change(screen.getByLabelText(/Port/), { target: { value: '587' } });
    fireEvent.change(screen.getByLabelText(/Username/), { target: { value: 'apikey' } });
    fireEvent.change(screen.getByLabelText(/^Password$/), { target: { value: 'sg-xyz' } });
    fireEvent.change(screen.getByLabelText(/From address/), { target: { value: 'bot@example.com' } });

    fireEvent.click(screen.getByRole('button', { name: /Save credential/i }));

    await waitFor(() => expect(createSpy).toHaveBeenCalledTimes(1));
    expect(createSpy).toHaveBeenCalledWith('smtp_prod', 'smtp', {
      host: 'smtp.example.com',
      port: '587',
      username: 'apikey',
      password: 'sg-xyz',
      from: 'bot@example.com',
    });
  });
});
