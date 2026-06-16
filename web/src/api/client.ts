import type {
  SignupResponse, LoginResponse, MeResponse,
  WorkflowDTO, RunResponse, ErrorEnvelope, WorkflowDefinition,
  CredentialDTO, Trigger, TriggerList, ListRunsResponse,
  GetRunResponse, SearchResponse,
} from './types';

const BASE = ''; // same-origin; dev uses Vite proxy

class APIError extends Error {
  constructor(public status: number, public code: string, message: string) {
    super(message);
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(BASE + path, {
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...(init.headers || {}),
    },
    ...init,
  });
  if (!res.ok) {
    let env: ErrorEnvelope | null = null;
    try { env = await res.json(); } catch { /* non-JSON body */ }
    throw new APIError(
      res.status,
      env?.error?.code || 'unknown',
      env?.error?.message || res.statusText,
    );
  }
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

export const api = {
  // Auth
  signup: (email: string, password: string) =>
    request<SignupResponse>('/v1/auth/signup', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    }),
  login: (email: string, password: string) =>
    request<LoginResponse>('/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    }),
  logout: () => request<void>('/v1/auth/logout', { method: 'POST' }),
  me: () => request<MeResponse>('/v1/me'),

  // Workflows
  createWorkflow: (name: string, definition: WorkflowDefinition) =>
    request<WorkflowDTO>('/v1/workflows', {
      method: 'POST',
      body: JSON.stringify({ name, definition }),
    }),
  getWorkflow: (id: string) =>
    request<WorkflowDTO>(`/v1/workflows/${id}`),
  runWorkflow: (id: string, triggerPayload?: Record<string, unknown>) =>
    request<RunResponse>(`/v1/workflows/${id}/run`, {
      method: 'POST',
      body: JSON.stringify(triggerPayload || {}),
    }),

  // Chat is SSE — see chat/useChat.ts in a later task

  // Credentials
  listCredentials: () =>
    request<{ items: CredentialDTO[] }>('/v1/credentials'),
  createCredential: (name: string, type: string, secret: Record<string, unknown>) =>
    request<CredentialDTO>('/v1/credentials', {
      method: 'POST',
      body: JSON.stringify({ name, type, secret }),
    }),
  deleteCredential: (id: string) =>
    request<void>(`/v1/credentials/${id}`, { method: 'DELETE' }),

  // Triggers
  listTriggers: (workflowId: string) =>
    request<TriggerList>(`/v1/workflows/${workflowId}/triggers`),
  createTrigger: (
    workflowId: string,
    kind: 'cron' | 'webhook',
    config: Record<string, unknown>,
  ) =>
    request<Trigger>(`/v1/workflows/${workflowId}/triggers`, {
      method: 'POST',
      body: JSON.stringify({ kind, config }),
    }),
  updateTrigger: (
    id: string,
    patch: { enabled?: boolean; config?: Record<string, unknown> },
  ) =>
    request<void>(`/v1/triggers/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(patch),
    }),
  deleteTrigger: (id: string) =>
    request<void>(`/v1/triggers/${id}`, { method: 'DELETE' }),

  // Runs
  listRuns: (
    workflowId: string,
    opts: { status?: string; from?: string; to?: string; limit?: number; cursor?: string } = {},
  ) => {
    const params = new URLSearchParams();
    if (opts.status) params.set('status', opts.status);
    if (opts.from) params.set('from', opts.from);
    if (opts.to) params.set('to', opts.to);
    if (opts.limit) params.set('limit', String(opts.limit));
    if (opts.cursor) params.set('cursor', opts.cursor);
    const qs = params.toString();
    return request<ListRunsResponse>(`/v1/workflows/${workflowId}/runs${qs ? '?' + qs : ''}`);
  },
  getRun: (id: string) =>
    request<GetRunResponse>(`/v1/runs/${id}`),
  replayRun: (id: string) =>
    request<{ run_id: string }>(`/v1/runs/${id}/replay`, { method: 'POST' }),

  // Search
  searchRunLogs: (wsID: string, q: string, limit = 20) =>
    request<SearchResponse>(
      `/v1/workspaces/${wsID}/runs/search?q=${encodeURIComponent(q)}&limit=${limit}`,
    ),
};

export { APIError };
