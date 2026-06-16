export interface User {
  id: string;
  email: string;
}

export interface Workspace {
  id: string;
  name: string;
}

export interface SignupResponse {
  user: User;
  workspace: Workspace;
}

export interface LoginResponse {
  user: User;
}

export interface MeResponse {
  user: User;
}

export interface WorkflowNode {
  id: string;
  tool: string;
  params: Record<string, unknown>;
  position?: [number, number];
  credential?: string;
}

export interface WorkflowEdge {
  from: string;
  from_port: string;
  to: string;
  to_port: string;
}

export interface WorkflowDefinition {
  id?: string;
  name?: string;
  trigger?: { type: string; payload?: Record<string, unknown> };
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
}

export interface WorkflowDTO {
  id: string;
  name: string;
  status: 'draft' | 'active' | 'paused' | 'archived';
  version: number;
  definition?: WorkflowDefinition;
}

export interface RunResponse {
  run_id: string;
  status: 'queued' | 'running' | 'succeeded' | 'failed' | 'cancelled';
  error?: string;
}

export interface ErrorEnvelope {
  error: {
    code: string;
    message: string;
  };
}

export interface ChatProposalPayload {
  name: string;
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
}

export interface ChatPatchPayload {
  patches: Array<{
    op: 'add' | 'remove' | 'replace' | 'move' | 'copy' | 'test';
    path: string;
    value?: unknown;
    from?: string;
  }>;
}

export type SSEEvent =
  | { type: 'text'; content: string }
  | { type: 'proposal'; tool: string; payload: ChatProposalPayload }
  | { type: 'patch'; tool: string; payload: ChatPatchPayload }
  | { type: 'error'; error: string }
  | { type: 'done' };

export interface CredentialDTO {
  id: string;
  name: string;
  type: string;
  created_at: string;
}

export interface CredentialList {
  items: CredentialDTO[];
}

export interface Trigger {
  id: string;
  workflow_id: string;
  kind: 'cron' | 'webhook';
  config: Record<string, unknown>;
  enabled: boolean;
  webhook_url?: string;
}

export interface TriggerList {
  items: Trigger[];
}

export interface WorkflowRun {
  id: string;
  workflow_id: string;
  status: string;
  trigger_id?: string | null;
  trigger_kind?: string | null;
  parent_run_id?: string | null;
  started_at?: string | null;
  finished_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface ListRunsResponse {
  items: WorkflowRun[];
  next_cursor: string;
}
