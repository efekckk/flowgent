import { createContext, useEffect, useState, type ReactNode } from 'react';
import { api, APIError } from '../api/client';
import type { User, Workspace } from '../api/types';

export interface AuthState {
  user: User | null;
  workspace?: Workspace | null;
  loading: boolean;
  signup: (email: string, password: string) => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

export const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.me()
      .then((r) => {
        setUser(r.user);
        setWorkspace(r.workspace ?? null);
      })
      .catch((err: unknown) => {
        if (err instanceof APIError && err.status === 401) {
          // not logged in; that's fine
        }
      })
      .finally(() => setLoading(false));
  }, []);

  // After login we don't yet have the workspace; signup gives us one. Re-fetch
  // /v1/me lazily so the search bar can light up without a refresh.
  async function refreshWorkspace() {
    try {
      const r = await api.me();
      setWorkspace(r.workspace ?? null);
    } catch {
      /* ignore */
    }
  }

  const value: AuthState = {
    user, workspace, loading,
    signup: async (email, password) => {
      const r = await api.signup(email, password);
      setUser(r.user);
      setWorkspace(r.workspace ?? null);
    },
    login: async (email, password) => {
      const r = await api.login(email, password);
      setUser(r.user);
      await refreshWorkspace();
    },
    logout: async () => {
      await api.logout();
      setUser(null);
      setWorkspace(null);
    },
  };
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
