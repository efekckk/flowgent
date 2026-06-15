import { createContext, useEffect, useState, type ReactNode } from 'react';
import { api, APIError } from '../api/client';
import type { User } from '../api/types';

export interface AuthState {
  user: User | null;
  loading: boolean;
  signup: (email: string, password: string) => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

export const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.me()
      .then((r) => setUser(r.user))
      .catch((err: unknown) => {
        if (err instanceof APIError && err.status === 401) {
          // not logged in; that's fine
        }
      })
      .finally(() => setLoading(false));
  }, []);

  const value: AuthState = {
    user, loading,
    signup: async (email, password) => {
      const r = await api.signup(email, password);
      setUser(r.user);
    },
    login: async (email, password) => {
      const r = await api.login(email, password);
      setUser(r.user);
    },
    logout: async () => {
      await api.logout();
      setUser(null);
    },
  };
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
