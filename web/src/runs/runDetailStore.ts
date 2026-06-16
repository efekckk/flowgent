import { create } from 'zustand';
import { api } from '../api/client';
import type { GetRunResponse, NodeRun, RunLogEvent, WorkflowRun } from '../api/types';

interface RunDetailState {
  runId: string | null;
  run: WorkflowRun | null;
  nodes: NodeRun[];
  logs: RunLogEvent[];
  loading: boolean;
  error: string | null;
  fetch: (runId: string) => Promise<void>;
  appendLog: (event: RunLogEvent) => void;
  refreshNodes: () => Promise<void>;
  replay: (runId: string) => Promise<string>;
}

export const useRunDetailStore = create<RunDetailState>((set, get) => ({
  runId: null,
  run: null,
  nodes: [],
  logs: [],
  loading: false,
  error: null,

  fetch: async (runId) => {
    set({ runId, loading: true, error: null, logs: [] });
    try {
      const data: GetRunResponse = await api.getRun(runId);
      set({ run: data.run, nodes: data.nodes ?? [], loading: false });
    } catch (err) {
      set({ error: String(err), loading: false });
    }
  },

  appendLog: (event) => set((s) => ({ logs: [...s.logs, event] })),

  refreshNodes: async () => {
    const id = get().runId;
    if (!id) return;
    try {
      const data = await api.getRun(id);
      set({ run: data.run, nodes: data.nodes ?? [] });
    } catch (err) {
      set({ error: String(err) });
    }
  },

  // Wraps the API call so tests can stub it via setState without mocking
  // module imports. Returns the new run id on success.
  replay: async (runId) => {
    const out = await api.replayRun(runId);
    return out.run_id;
  },
}));
