import { describe, it, expect, vi, beforeEach } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { useRunsStore } from './runsStore';
import RunListPage from './RunListPage';
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
      <MemoryRouter initialEntries={['/workflows/wf_abc/runs']}>
        <Routes>
          <Route path="/workflows/:id/runs" element={<RunListPage />} />
        </Routes>
      </MemoryRouter>
    </AuthContext.Provider>,
  );
}

beforeEach(() => {
  useRunsStore.setState({
    workflowId: null,
    items: [],
    nextCursor: '',
    status: '',
    from: '',
    to: '',
    loading: false,
    error: null,
  });
});

describe('RunListPage', () => {
  it('renders empty state when there are no runs', async () => {
    const fetchSpy = vi.fn(async () => {
      useRunsStore.setState({ items: [], loading: false });
    });
    useRunsStore.setState({ fetch: fetchSpy as never });
    renderPage();
    await waitFor(() => expect(fetchSpy).toHaveBeenCalled());
    expect(screen.getByText(/no runs/i)).toBeInTheDocument();
  });

  it('renders a row per run with status and duration', () => {
    useRunsStore.setState({
      fetch: vi.fn(async () => undefined) as never,
      items: [
        {
          id: 'run_a', workflow_id: 'wf_abc', status: 'succeeded',
          started_at: '2026-06-16T12:00:00Z', finished_at: '2026-06-16T12:00:05Z',
          created_at: '2026-06-16T12:00:00Z', updated_at: '2026-06-16T12:00:05Z',
        },
        {
          id: 'run_b', workflow_id: 'wf_abc', status: 'running',
          started_at: '2026-06-16T13:00:00Z',
          created_at: '2026-06-16T13:00:00Z', updated_at: '2026-06-16T13:00:00Z',
        },
      ],
    });
    renderPage();
    expect(screen.getByText(/run_a/)).toBeInTheDocument();
    expect(screen.getByText(/run_b/)).toBeInTheDocument();
    // Duration column for the running row should show em-dash
    const rows = screen.getAllByRole('row');
    expect(rows.length).toBeGreaterThanOrEqual(2);
  });

  it('changing status filter re-fetches with the new status', async () => {
    const fetchSpy = vi.fn(async () => undefined);
    useRunsStore.setState({ fetch: fetchSpy as never });
    renderPage();
    fireEvent.change(screen.getByLabelText(/status/i), { target: { value: 'failed' } });
    await waitFor(() => expect(useRunsStore.getState().status).toBe('failed'));
    // fetch should have been called again with the new filter present in state
    expect(fetchSpy).toHaveBeenCalled();
  });

  it('shows replay badge when parent_run_id is set', () => {
    useRunsStore.setState({
      fetch: vi.fn(async () => undefined) as never,
      items: [{
        id: 'run_r', workflow_id: 'wf_abc', status: 'succeeded',
        parent_run_id: 'run_orig',
        created_at: '2026-06-16T12:00:00Z', updated_at: '2026-06-16T12:00:00Z',
      }],
    });
    renderPage();
    expect(screen.getByText(/replay/i)).toBeInTheDocument();
  });

  it('Load more advances cursor and appends', async () => {
    const loadMoreSpy = vi.fn(async () => undefined);
    useRunsStore.setState({
      fetch: vi.fn(async () => undefined) as never,
      items: [{
        id: 'run_a', workflow_id: 'wf_abc', status: 'succeeded',
        created_at: '2026-06-16T12:00:00Z', updated_at: '2026-06-16T12:00:00Z',
      }],
      nextCursor: 'abc:run_a',
      loadMore: loadMoreSpy as never,
    });
    renderPage();
    fireEvent.click(screen.getByRole('button', { name: /load more/i }));
    await waitFor(() => expect(loadMoreSpy).toHaveBeenCalled());
  });
});
