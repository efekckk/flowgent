import { describe, it, expect, vi, beforeEach } from 'vitest';
import { fireEvent, render, screen, waitFor, act } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import LogSearchBar from './LogSearchBar';

let searchSpy: ReturnType<typeof vi.fn>;

vi.mock('../api/client', async () => {
  const actual = await vi.importActual<typeof import('../api/client')>('../api/client');
  return {
    ...actual,
    api: {
      ...actual.api,
      searchRunLogs: (...args: unknown[]) => searchSpy(...args),
    },
  };
});

beforeEach(() => {
  searchSpy = vi.fn(async () => ({ hits: [] }));
});

describe('LogSearchBar', () => {
  it('does not call the API when query is under 3 chars', async () => {
    vi.useFakeTimers();
    render(
      <MemoryRouter>
        <LogSearchBar workspaceId="ws_a" />
      </MemoryRouter>,
    );
    fireEvent.change(screen.getByPlaceholderText(/search logs/i), { target: { value: 'ab' } });
    act(() => { vi.advanceTimersByTime(600); });
    expect(searchSpy).not.toHaveBeenCalled();
    vi.useRealTimers();
  });

  it('calls the API after 500ms debounce when query is >= 3 chars', async () => {
    vi.useFakeTimers();
    render(
      <MemoryRouter>
        <LogSearchBar workspaceId="ws_a" />
      </MemoryRouter>,
    );
    fireEvent.change(screen.getByPlaceholderText(/search logs/i), { target: { value: 'slack' } });
    act(() => { vi.advanceTimersByTime(600); });
    vi.useRealTimers();
    await waitFor(() => expect(searchSpy).toHaveBeenCalledWith('ws_a', 'slack', 20));
  });

  it('renders highlighted snippet with <mark> for the marker', async () => {
    searchSpy = vi.fn(async () => ({
      hits: [
        {
          run_id: 'run_x', workflow_id: 'wf_a', node_id: 'fetch',
          message: 'slack notify ok',
          snippet: 'slack «notify» ok',
          at: '2026-06-16T12:00:00Z',
        },
      ],
    }));
    vi.useFakeTimers();
    render(
      <MemoryRouter>
        <LogSearchBar workspaceId="ws_a" />
      </MemoryRouter>,
    );
    fireEvent.change(screen.getByPlaceholderText(/search logs/i), { target: { value: 'notify' } });
    act(() => { vi.advanceTimersByTime(600); });
    vi.useRealTimers();
    await waitFor(() => expect(screen.getByText('notify').tagName.toLowerCase()).toBe('mark'));
  });

  it('clicking a hit clears the query and closes the dropdown', async () => {
    searchSpy = vi.fn(async () => ({
      hits: [
        {
          run_id: 'run_abc', workflow_id: 'wf_a', node_id: '',
          message: 'm', snippet: 'm', at: '2026-06-16T12:00:00Z',
        },
      ],
    }));
    vi.useFakeTimers();
    render(
      <MemoryRouter>
        <LogSearchBar workspaceId="ws_a" />
      </MemoryRouter>,
    );
    fireEvent.change(screen.getByPlaceholderText(/search logs/i), { target: { value: 'abc' } });
    act(() => { vi.advanceTimersByTime(600); });
    vi.useRealTimers();
    const hitButton = await screen.findByRole('button');
    fireEvent.click(hitButton);
    await waitFor(() => {
      const input = screen.getByPlaceholderText(/search logs/i) as HTMLInputElement;
      expect(input.value).toBe('');
    });
  });

  it('escapes HTML in snippets', async () => {
    searchSpy = vi.fn(async () => ({
      hits: [
        {
          run_id: 'run_x', workflow_id: 'wf_a', node_id: '',
          message: '<script>x</script>',
          snippet: '<script>x</script>',
          at: '2026-06-16T12:00:00Z',
        },
      ],
    }));
    vi.useFakeTimers();
    render(
      <MemoryRouter>
        <LogSearchBar workspaceId="ws_a" />
      </MemoryRouter>,
    );
    fireEvent.change(screen.getByPlaceholderText(/search logs/i), { target: { value: 'xxx' } });
    act(() => { vi.advanceTimersByTime(600); });
    vi.useRealTimers();
    await waitFor(() => {
      const buttons = screen.getAllByRole('button');
      const html = buttons[0].innerHTML;
      expect(html).toContain('&lt;script&gt;');
      expect(html).not.toContain('<script>');
    });
  });

  it('renders nothing when workspaceId is null', () => {
    const { container } = render(
      <MemoryRouter>
        <LogSearchBar workspaceId={null} />
      </MemoryRouter>,
    );
    expect(container.firstChild).toBeNull();
  });
});
