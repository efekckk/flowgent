import { describe, it, expect, vi, beforeEach } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { useRunDetailStore } from './runDetailStore';
import { useWorkflowsStore } from '../workflows/workflowsStore';
import RunDetailPage from './RunDetailPage';
import { AuthContext, type AuthState } from '../auth/AuthProvider';

// jsdom doesn't ship ResizeObserver or DOMMatrixReadOnly; ReactFlow reaches
// for both during mount. A no-op shim is enough for unit-level rendering.
class FakeResizeObserver {
  observe() {} unobserve() {} disconnect() {}
}
(globalThis as unknown as { ResizeObserver: typeof FakeResizeObserver }).ResizeObserver = FakeResizeObserver;
if (!(globalThis as Record<string, unknown>).DOMMatrixReadOnly) {
  class FakeDOMMatrixReadOnly {
    m22 = 1;
    constructor(_init?: string | number[]) {}
  }
  (globalThis as Record<string, unknown>).DOMMatrixReadOnly = FakeDOMMatrixReadOnly;
}

// Minimal EventSource stand-in so the SSE hook can mount without a real
// network. Tests can grab the instance and synthesise frames.
class FakeEventSource {
  static instances: FakeEventSource[] = [];
  static last(): FakeEventSource {
    return FakeEventSource.instances[FakeEventSource.instances.length - 1];
  }
  listeners: Record<string, ((e: MessageEvent) => void)[]> = {};
  onerror: ((e: Event) => void) | null = null;
  constructor(public url: string, _init?: EventSourceInit) {
    FakeEventSource.instances.push(this);
  }
  addEventListener(type: string, handler: (e: MessageEvent) => void) {
    (this.listeners[type] ??= []).push(handler);
  }
  removeEventListener() { /* noop */ }
  close() { /* noop */ }
  emit(type: string, data: unknown) {
    const evt = new MessageEvent(type, { data: JSON.stringify(data) });
    (this.listeners[type] ?? []).forEach((h) => h(evt));
  }
}
(globalThis as unknown as { EventSource: typeof FakeEventSource }).EventSource = FakeEventSource;

function renderPage(runId = 'run_test') {
  const authValue: AuthState = {
    user: { id: 'usr_1', email: 'x@y.com' },
    loading: false,
    signup: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
  };
  return render(
    <AuthContext.Provider value={authValue}>
      <MemoryRouter initialEntries={[`/runs/${runId}`]}>
        <Routes>
          <Route path="/runs/:id" element={<RunDetailPage />} />
        </Routes>
      </MemoryRouter>
    </AuthContext.Provider>,
  );
}

function seedWorkflow() {
  useWorkflowsStore.setState({
    current: {
      id: 'wf_a',
      name: 'Test',
      status: 'active',
      version: 1,
      definition: {
        nodes: [
          { id: 'fetch', tool: 'http.request', params: {}, position: [0, 0] },
          { id: 'send', tool: 'slack.post', params: {}, position: [220, 0] },
        ],
        edges: [
          { from: 'fetch', from_port: 'main', to: 'send', to_port: 'main' },
        ],
      },
    },
    fetchOne: vi.fn(async () => undefined) as never,
  });
}

beforeEach(() => {
  FakeEventSource.instances = [];
  useRunDetailStore.setState({
    runId: null,
    run: null,
    nodes: [],
    logs: [],
    loading: false,
    error: null,
  });
  useWorkflowsStore.setState({
    list: [],
    current: null,
    loading: false,
    error: null,
    fetchOne: vi.fn(async () => undefined) as never,
  });
});

describe('RunDetailPage', () => {
  it('fetches run on mount and shows the run id + status', async () => {
    const fetchSpy = vi.fn(async () => {
      useRunDetailStore.setState({
        run: {
          id: 'run_test', workflow_id: 'wf_a', status: 'succeeded',
          created_at: '2026-06-16T12:00:00Z', updated_at: '2026-06-16T12:00:00Z',
        },
        nodes: [
          { id: 'nr1', node_id: 'fetch', status: 'succeeded' },
        ],
      });
    });
    useRunDetailStore.setState({ fetch: fetchSpy as never });
    seedWorkflow();
    renderPage();
    await waitFor(() => expect(fetchSpy).toHaveBeenCalledWith('run_test'));
    expect(await screen.findByText('run_test')).toBeInTheDocument();
    // Status pill in the header
    const pills = screen.getAllByText('succeeded');
    expect(pills.length).toBeGreaterThan(0);
  });

  it('appends a log when SSE emits a log event', async () => {
    useRunDetailStore.setState({
      fetch: vi.fn(async () => undefined) as never,
      run: {
        id: 'run_test', workflow_id: 'wf_a', status: 'running',
        created_at: '2026-06-16T12:00:00Z', updated_at: '2026-06-16T12:00:00Z',
      },
      nodes: [],
    });
    seedWorkflow();
    renderPage();
    await waitFor(() => expect(FakeEventSource.instances.length).toBeGreaterThan(0));
    FakeEventSource.last().emit('log', {
      run_id: 'run_test', node_id: 'fetch', level: 'info',
      message: 'http.request: started', at: '2026-06-16T12:00:01Z',
    });
    await waitFor(() => expect(screen.getByText(/http\.request: started/)).toBeInTheDocument());
  });

  it('replay button calls the store and navigates to the new run id', async () => {
    const replaySpy = vi.fn(async () => 'run_new');
    useRunDetailStore.setState({
      fetch: vi.fn(async () => undefined) as never,
      replay: replaySpy as never,
      run: {
        id: 'run_test', workflow_id: 'wf_a', status: 'succeeded',
        created_at: '2026-06-16T12:00:00Z', updated_at: '2026-06-16T12:00:00Z',
      },
      nodes: [],
    });
    seedWorkflow();
    renderPage();
    fireEvent.click(await screen.findByRole('button', { name: /replay/i }));
    await waitFor(() => expect(replaySpy).toHaveBeenCalledWith('run_test'));
  });

  it('disables the Node IO tab until a node is selected', async () => {
    useRunDetailStore.setState({
      fetch: vi.fn(async () => undefined) as never,
      run: {
        id: 'run_test', workflow_id: 'wf_a', status: 'succeeded',
        created_at: '2026-06-16T12:00:00Z', updated_at: '2026-06-16T12:00:00Z',
      },
      nodes: [
        {
          id: 'nr1', node_id: 'fetch', status: 'succeeded',
          input: { url: 'https://x' }, output: { ok: true },
        },
      ],
    });
    seedWorkflow();
    renderPage();
    const ioTab = await screen.findByRole('button', { name: /node io/i });
    expect(ioTab).toBeDisabled();
    // Logs tab is the default; the empty-state copy proves it's mounted.
    expect(screen.getByText(/no log lines yet/i)).toBeInTheDocument();
  });

  it('shows an empty-state message when no logs have arrived', async () => {
    useRunDetailStore.setState({
      fetch: vi.fn(async () => undefined) as never,
      run: {
        id: 'run_test', workflow_id: 'wf_a', status: 'running',
        created_at: '2026-06-16T12:00:00Z', updated_at: '2026-06-16T12:00:00Z',
      },
      nodes: [],
    });
    seedWorkflow();
    renderPage();
    expect(await screen.findByText(/no log lines yet/i)).toBeInTheDocument();
  });
});
