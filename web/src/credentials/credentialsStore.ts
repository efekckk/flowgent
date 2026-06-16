import { create } from 'zustand';
import { api } from '../api/client';
import type { CredentialDTO } from '../api/types';

interface CredentialsState {
  items: CredentialDTO[];
  loading: boolean;
  error: string | null;
  fetch: () => Promise<void>;
  create: (name: string, type: string, secret: Record<string, unknown>) => Promise<void>;
  remove: (id: string) => Promise<void>;
}

export const useCredentialsStore = create<CredentialsState>((set, get) => ({
  items: [],
  loading: false,
  error: null,
  fetch: async () => {
    set({ loading: true, error: null });
    try {
      const r = await api.listCredentials();
      set({ items: r.items, loading: false });
    } catch (err) {
      set({ error: String(err), loading: false });
    }
  },
  create: async (name, type, secret) => {
    const r = await api.createCredential(name, type, secret);
    set({ items: [r, ...get().items] });
  },
  remove: async (id) => {
    await api.deleteCredential(id);
    set({ items: get().items.filter((c) => c.id !== id) });
  },
}));
