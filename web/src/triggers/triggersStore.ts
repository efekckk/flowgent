import { create } from 'zustand';
import { api } from '../api/client';
import type { Trigger } from '../api/types';

interface TriggersState {
  workflowId: string | null;
  items: Trigger[];
  loading: boolean;
  error: string | null;
  fetch: (workflowId: string) => Promise<void>;
  create: (
    workflowId: string,
    kind: 'cron' | 'webhook',
    config: Record<string, unknown>,
  ) => Promise<Trigger>;
  toggle: (id: string, enabled: boolean) => Promise<void>;
  remove: (id: string) => Promise<void>;
}

export const useTriggersStore = create<TriggersState>((set, get) => ({
  workflowId: null,
  items: [],
  loading: false,
  error: null,

  fetch: async (workflowId) => {
    set({ workflowId, loading: true, error: null });
    try {
      const r = await api.listTriggers(workflowId);
      set({ items: r.items ?? [], loading: false });
    } catch (err) {
      set({ error: String(err), loading: false });
    }
  },

  create: async (workflowId, kind, config) => {
    const created = await api.createTrigger(workflowId, kind, config);
    set((s) => ({ items: [...s.items, created] }));
    return created;
  },

  toggle: async (id, enabled) => {
    const cur = get().items.find((t) => t.id === id);
    if (!cur) return;
    await api.updateTrigger(id, { enabled, config: cur.config });
    set((s) => ({
      items: s.items.map((t) => (t.id === id ? { ...t, enabled } : t)),
    }));
  },

  remove: async (id) => {
    await api.deleteTrigger(id);
    set((s) => ({ items: s.items.filter((t) => t.id !== id) }));
  },
}));
