import { describe, it, expect, vi, beforeEach } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { useTriggersStore } from './triggersStore';
import TriggersPage from './TriggersPage';
import { AuthContext, type AuthState } from '../auth/AuthProvider';

function renderPage() {
  const value: AuthState = {
    user: { id: 'usr_1', email: 'x@y.com' },
    loading: false,
    signup: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
  };
  return render(
    <AuthContext.Provider value={value}>
      <MemoryRouter initialEntries={['/workflows/wf_abc/triggers']}>
        <Routes>
          <Route path="/workflows/:id/triggers" element={<TriggersPage />} />
        </Routes>
      </MemoryRouter>
    </AuthContext.Provider>,
  );
}

beforeEach(() => {
  useTriggersStore.setState({
    items: [],
    loading: false,
    error: null,
    workflowId: null,
  });
});

describe('TriggersPage', () => {
  it('renders empty state when there are no triggers', async () => {
    const fetchSpy = vi.fn(async () => {
      useTriggersStore.setState({ items: [], loading: false });
    });
    useTriggersStore.setState({ fetch: fetchSpy as never });
    renderPage();
    await waitFor(() => expect(fetchSpy).toHaveBeenCalled());
    expect(screen.getByText(/no triggers/i)).toBeInTheDocument();
  });

  it('renders the create cron form with a preset dropdown', () => {
    renderPage();
    const presetSelect = screen.getByLabelText(/preset/i) as HTMLSelectElement;
    expect(presetSelect).toBeInTheDocument();
    expect(screen.getByLabelText(/cron expression/i)).toBeInTheDocument();
  });

  it('preset fills the cron expression input', () => {
    renderPage();
    const presetSelect = screen.getByLabelText(/preset/i) as HTMLSelectElement;
    const cronInput = screen.getByLabelText(/cron expression/i) as HTMLInputElement;
    fireEvent.change(presetSelect, { target: { value: '@daily' } });
    expect(cronInput.value).toBe('@daily');
  });

  it('submits cron trigger via store.create', async () => {
    const createSpy = vi.fn(async () => ({
      id: 'trg_x',
      workflow_id: 'wf_abc',
      kind: 'cron' as const,
      config: { cron: '@hourly' },
      enabled: true,
    }));
    useTriggersStore.setState({ create: createSpy as never });

    renderPage();
    const cronInput = screen.getByLabelText(/cron expression/i) as HTMLInputElement;
    fireEvent.change(cronInput, { target: { value: '@hourly' } });
    fireEvent.click(screen.getByRole('button', { name: /add cron trigger/i }));
    await waitFor(() =>
      expect(createSpy).toHaveBeenCalledWith('wf_abc', 'cron', { cron: '@hourly' }),
    );
  });

  it('renders existing webhook trigger with a copy button', () => {
    useTriggersStore.setState({
      fetch: vi.fn(async () => undefined) as never,
      items: [
        {
          id: 'trg_w',
          workflow_id: 'wf_abc',
          kind: 'webhook',
          config: { token: 'tok_xyz' },
          enabled: true,
          webhook_url: 'http://localhost:8080/webhooks/trg_w/tok_xyz',
        },
      ],
    });
    renderPage();
    expect(screen.getAllByText(/trg_w/).length).toBeGreaterThan(0);
    expect(screen.getByRole('button', { name: /copy/i })).toBeInTheDocument();
  });

  it('toggle enabled fires store.toggle', async () => {
    const toggleSpy = vi.fn(async () => undefined);
    useTriggersStore.setState({
      fetch: vi.fn(async () => undefined) as never,
      items: [
        {
          id: 'trg_c',
          workflow_id: 'wf_abc',
          kind: 'cron',
          config: { cron: '@daily' },
          enabled: true,
        },
      ],
      toggle: toggleSpy as never,
    });
    renderPage();
    fireEvent.click(screen.getByRole('button', { name: /disable/i }));
    await waitFor(() => expect(toggleSpy).toHaveBeenCalledWith('trg_c', false));
  });
});
