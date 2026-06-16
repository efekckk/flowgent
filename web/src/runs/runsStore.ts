import { create } from 'zustand';
import { api } from '../api/client';
import type { WorkflowRun } from '../api/types';

interface RunsState {
  workflowId: string | null;
  items: WorkflowRun[];
  nextCursor: string;
  status: string;        // '' = all
  from: string;          // RFC3339 or ''
  to: string;            // RFC3339 or ''
  loading: boolean;
  error: string | null;

  setFilter: (patch: Partial<{ status: string; from: string; to: string }>) => void;
  fetch: (workflowId: string, reset?: boolean) => Promise<void>;
  loadMore: () => Promise<void>;
}

export const useRunsStore = create<RunsState>((set, get) => ({
  workflowId: null,
  items: [],
  nextCursor: '',
  status: '',
  from: '',
  to: '',
  loading: false,
  error: null,

  setFilter: (patch) => set((s) => ({ ...s, ...patch })),

  fetch: async (workflowId, reset = true) => {
    set({ workflowId, loading: true, error: null });
    if (reset) set({ items: [], nextCursor: '' });
    try {
      const { status, from, to } = get();
      const out = await api.listRuns(workflowId, { status, from, to, limit: 50 });
      set({ items: out.items ?? [], nextCursor: out.next_cursor ?? '', loading: false });
    } catch (err) {
      set({ error: String(err), loading: false });
    }
  },

  loadMore: async () => {
    const { workflowId, items, nextCursor, status, from, to } = get();
    if (!workflowId || !nextCursor) return;
    set({ loading: true });
    try {
      const out = await api.listRuns(workflowId, { status, from, to, limit: 50, cursor: nextCursor });
      set({ items: [...items, ...(out.items ?? [])], nextCursor: out.next_cursor ?? '', loading: false });
    } catch (err) {
      set({ error: String(err), loading: false });
    }
  },
}));
