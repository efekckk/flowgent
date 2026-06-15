import { create } from 'zustand';
import { api } from '../api/client';
import type { WorkflowDTO } from '../api/types';

interface WorkflowsState {
  list: WorkflowDTO[];
  current: WorkflowDTO | null;
  loading: boolean;
  error: string | null;
  fetchList: () => Promise<void>;
  fetchOne: (id: string) => Promise<void>;
  createEmpty: (name: string) => Promise<WorkflowDTO>;
  setCurrent: (wf: WorkflowDTO | null) => void;
}

export const useWorkflowsStore = create<WorkflowsState>((set, get) => ({
  list: [],
  current: null,
  loading: false,
  error: null,

  fetchList: async () => {
    set({ loading: true, error: null });
    try {
      // Backend doesn't yet expose a list endpoint per workflow_handler.go;
      // for M5 we keep an in-memory list seeded by created workflows. M6
      // adds GET /v1/workflows that returns the workspace's list.
      set({ loading: false });
    } catch (err) {
      set({ error: String(err), loading: false });
    }
  },

  fetchOne: async (id: string) => {
    set({ loading: true, error: null });
    try {
      const wf = await api.getWorkflow(id);
      set({ current: wf, loading: false });
    } catch (err) {
      set({ error: String(err), loading: false });
    }
  },

  createEmpty: async (name: string) => {
    const wf = await api.createWorkflow(name, { nodes: [], edges: [] });
    set({ list: [...get().list, wf], current: wf });
    return wf;
  },

  setCurrent: (wf) => set({ current: wf }),
}));
