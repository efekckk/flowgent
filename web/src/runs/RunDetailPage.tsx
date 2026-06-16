import { useEffect, useMemo, useRef, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import ReactFlow, {
  Background, Controls, MiniMap,
  Handle, Position,
  type Node, type Edge, type NodeMouseHandler, type NodeProps,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { useRunDetailStore } from './runDetailStore';
import { useRunLogStream } from './useRunLogStream';
import { useWorkflowsStore } from '../workflows/workflowsStore';
import { useAuth } from '../auth/useAuth';
import { workflowToFlow } from '../canvas/workflowToFlow';
import type { NodeRun, RunLogEvent } from '../api/types';

type NodeStatus = 'pending' | 'running' | 'succeeded' | 'failed' | string;

function statusPill(status: string): string {
  switch (status) {
    case 'succeeded':
      return 'bg-emerald-100 text-emerald-700';
    case 'failed':
      return 'bg-red-100 text-red-700';
    case 'running':
      return 'bg-yellow-100 text-yellow-700';
    case 'cancelled':
      return 'bg-slate-100 text-slate-500';
    default:
      return 'bg-slate-100 text-slate-600';
  }
}

// Fill + ring classes per node-run status. "pending" is the implicit default
// for nodes that haven't been picked up by the executor yet.
function nodeStatusClasses(status: NodeStatus): string {
  switch (status) {
    case 'succeeded':
      return 'bg-emerald-500 ring-emerald-300';
    case 'failed':
      return 'bg-red-500 ring-red-300';
    case 'running':
      return 'bg-yellow-400 ring-yellow-200 animate-pulse';
    case 'pending':
    default:
      return 'bg-slate-300 ring-slate-200';
  }
}

interface StatusNodeData {
  id: string;
  tool: string;
  status: NodeStatus;
}

function StatusNode({ data }: NodeProps<StatusNodeData>) {
  const tone = nodeStatusClasses(data.status);
  return (
    <div
      data-testid={`run-node-${data.id}`}
      className={`rounded-md ring-4 shadow-sm ${tone}`}
      style={{ width: 200 }}
    >
      <div className="flex items-center justify-between rounded-t-md bg-black/10 px-3 py-2">
        <span className="truncate font-medium text-white">{data.tool}</span>
        <span className="rounded-full bg-white/90 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-slate-700">
          {data.status || 'pending'}
        </span>
      </div>
      <div className="truncate px-3 py-2 text-xs text-white/90">id: {data.id}</div>
      <Handle type="target" position={Position.Top} id="main" />
      <Handle type="source" position={Position.Bottom} id="main" />
    </div>
  );
}

const nodeTypes = { runNode: StatusNode };

function formatTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleTimeString();
}

function levelClasses(level: string): string {
  switch (level) {
    case 'error':
      return 'text-red-600';
    case 'warn':
      return 'text-yellow-600';
    case 'debug':
      return 'text-slate-400';
    default:
      return 'text-slate-700';
  }
}

function prettyJSON(v: unknown): string {
  if (v === undefined || v === null) return '';
  if (typeof v === 'string') {
    // backend uses json.RawMessage so values arrive already-parsed for JSON
    // objects, but a raw string may also slip through (e.g. error fields).
    try { return JSON.stringify(JSON.parse(v), null, 2); }
    catch { return v; }
  }
  try { return JSON.stringify(v, null, 2); }
  catch { return String(v); }
}

type RightTab = 'logs' | 'io';

function LogsPanel({ logs }: { logs: RunLogEvent[] }) {
  const scrollRef = useRef<HTMLDivElement | null>(null);
  // Track whether the user has scrolled away from the bottom; if so we
  // stop pinning so they can read the history without being yanked back.
  const stickRef = useRef(true);

  function onScroll() {
    const el = scrollRef.current;
    if (!el) return;
    const distance = el.scrollHeight - (el.scrollTop + el.clientHeight);
    stickRef.current = distance < 20;
  }

  useEffect(() => {
    const el = scrollRef.current;
    if (!el || !stickRef.current) return;
    el.scrollTop = el.scrollHeight;
  }, [logs.length]);

  if (logs.length === 0) {
    return <p className="p-4 text-sm text-slate-400">No log lines yet.</p>;
  }

  return (
    <div
      ref={scrollRef}
      onScroll={onScroll}
      data-testid="logs-panel"
      className="h-full overflow-y-auto p-3 font-mono text-xs leading-5"
    >
      {logs.map((l, i) => (
        <div key={l.id ?? i} className="flex gap-2">
          <span className="shrink-0 text-slate-400">{formatTime(l.at)}</span>
          <span className={`shrink-0 w-12 uppercase ${levelClasses(l.level)}`}>{l.level}</span>
          {l.node_id && <span className="shrink-0 text-indigo-500">{l.node_id}</span>}
          <span className="break-all text-slate-700">{l.message}</span>
        </div>
      ))}
    </div>
  );
}

function NodeIOPanel({ node }: { node: NodeRun | null }) {
  if (!node) {
    return <p className="p-4 text-sm text-slate-400">Select a node to inspect its input and output.</p>;
  }
  return (
    <div className="h-full overflow-y-auto p-4 text-xs">
      <h3 className="text-sm font-semibold text-slate-700">
        {node.node_id} <span className={`ml-2 rounded-full px-2 py-0.5 ${statusPill(node.status)}`}>{node.status}</span>
      </h3>
      {node.error !== undefined && node.error !== null && (
        <section className="mt-3">
          <h4 className="text-xs font-semibold uppercase tracking-wide text-red-600">Error</h4>
          <pre className="mt-1 whitespace-pre-wrap rounded-md bg-red-50 p-2 text-red-700">{prettyJSON(node.error)}</pre>
        </section>
      )}
      <section className="mt-3">
        <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500">Input</h4>
        <pre className="mt-1 whitespace-pre-wrap rounded-md bg-slate-50 p-2 text-slate-800">{prettyJSON(node.input) || '—'}</pre>
      </section>
      <section className="mt-3">
        <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500">Output</h4>
        <pre className="mt-1 whitespace-pre-wrap rounded-md bg-slate-50 p-2 text-slate-800">{prettyJSON(node.output) || '—'}</pre>
      </section>
    </div>
  );
}

export default function RunDetailPage() {
  const { id: runId } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { logout } = useAuth();
  const { run, nodes, logs, loading, error, fetch, replay } = useRunDetailStore();
  const { current: workflow, fetchOne } = useWorkflowsStore();

  const [selected, setSelected] = useState<string | null>(null);
  const [tab, setTab] = useState<RightTab>('logs');
  const [replaying, setReplaying] = useState(false);
  const [replayError, setReplayError] = useState<string | null>(null);

  useEffect(() => {
    if (runId) fetch(runId);
  }, [runId, fetch]);

  useEffect(() => {
    if (run?.workflow_id) fetchOne(run.workflow_id);
  }, [run?.workflow_id, fetchOne]);

  useRunLogStream(runId ?? null);

  // Build a lookup from node_id → latest NodeRun (the executor may write
  // multiple iterations for loops; prefer the most recent).
  const nodeRunByNodeId = useMemo(() => {
    const m = new Map<string, NodeRun>();
    for (const nr of nodes) {
      const existing = m.get(nr.node_id);
      if (!existing) { m.set(nr.node_id, nr); continue; }
      const a = existing.finished_at || existing.started_at || '';
      const b = nr.finished_at || nr.started_at || '';
      if (b >= a) m.set(nr.node_id, nr);
    }
    return m;
  }, [nodes]);

  // Compose ReactFlow nodes from the workflow definition, overlaying live
  // status from node_runs. Memo keyed on both inputs so colours update as
  // SSE events trigger refreshNodes.
  const { flowNodes, flowEdges } = useMemo(() => {
    const def = workflow?.definition;
    if (!def) return { flowNodes: [] as Node[], flowEdges: [] as Edge[] };
    const base = workflowToFlow(def);
    const flowNodes: Node[] = base.nodes.map((n) => {
      const status = nodeRunByNodeId.get(n.id)?.status || 'pending';
      const tool = (n.data as { tool: string }).tool;
      return {
        ...n,
        type: 'runNode',
        data: { id: n.id, tool, status } satisfies StatusNodeData,
      };
    });
    return { flowNodes, flowEdges: base.edges };
  }, [workflow, nodeRunByNodeId]);

  const onNodeClick: NodeMouseHandler = (_, node) => {
    setSelected(node.id);
    setTab('io');
  };

  const selectedNodeRun = selected ? nodeRunByNodeId.get(selected) ?? null : null;

  async function onReplay() {
    if (!runId) return;
    setReplaying(true);
    setReplayError(null);
    try {
      const newId = await replay(runId);
      navigate(`/runs/${newId}`);
    } catch (err) {
      setReplayError(String(err));
    } finally {
      setReplaying(false);
    }
  }

  return (
    <div className="flex h-screen bg-slate-50">
      <aside className="flex h-full w-56 flex-col border-r border-slate-200 bg-white p-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">Run</h2>
        <Link to="/workflows" className="mt-3 rounded-md px-3 py-2 text-sm text-slate-700 hover:bg-slate-50">← Workflows</Link>
        {run?.workflow_id && (
          <Link
            to={`/workflows/${run.workflow_id}/runs`}
            className="rounded-md px-3 py-2 text-sm text-slate-700 hover:bg-slate-50"
          >
            All runs
          </Link>
        )}
        <button onClick={logout} className="mt-auto rounded-md border border-slate-300 px-3 py-2 text-sm text-slate-600 hover:bg-slate-50">
          Sign out
        </button>
      </aside>

      <main className="flex flex-1 flex-col">
        <header className="flex items-center justify-between border-b border-slate-200 bg-white px-6 py-3">
          <div className="flex items-center gap-3">
            <h1 className="text-lg font-semibold text-slate-800">Run</h1>
            <span className="font-mono text-sm text-slate-500">{runId}</span>
            {run && (
              <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${statusPill(run.status)}`}>
                {run.status}
              </span>
            )}
          </div>
          <div className="flex items-center gap-2">
            {replayError && <span className="text-xs text-red-600">{replayError}</span>}
            <button
              type="button"
              onClick={onReplay}
              disabled={!runId || replaying}
              className="rounded-md border border-indigo-300 bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {replaying ? 'Replaying…' : 'Replay'}
            </button>
          </div>
        </header>

        {error && <p className="bg-red-50 px-6 py-2 text-sm text-red-700">{error}</p>}

        <div className="flex min-h-0 flex-1">
          <section className="flex-[3] border-r border-slate-200">
            {loading && !workflow ? (
              <p className="p-6 text-sm text-slate-400">Loading…</p>
            ) : flowNodes.length === 0 ? (
              <p className="p-6 text-sm text-slate-400">No workflow definition available for this run.</p>
            ) : (
              <div className="h-full w-full">
                <ReactFlow
                  nodes={flowNodes}
                  edges={flowEdges}
                  nodeTypes={nodeTypes}
                  onNodeClick={onNodeClick}
                  onPaneClick={() => setSelected(null)}
                  fitView
                  panOnDrag
                  panOnScroll={false}
                  nodesDraggable={false}
                  nodesConnectable={false}
                  edgesFocusable={false}
                  elementsSelectable
                >
                  <Background />
                  <Controls showInteractive={false} />
                  <MiniMap pannable zoomable />
                </ReactFlow>
              </div>
            )}
          </section>

          <section className="flex flex-[2] flex-col bg-white">
            <div className="flex border-b border-slate-200">
              <button
                type="button"
                onClick={() => setTab('logs')}
                className={`px-4 py-2 text-sm font-medium ${tab === 'logs' ? 'border-b-2 border-indigo-500 text-indigo-700' : 'text-slate-500 hover:text-slate-700'}`}
              >
                Logs
              </button>
              <button
                type="button"
                onClick={() => selected && setTab('io')}
                disabled={!selected}
                className={`px-4 py-2 text-sm font-medium ${tab === 'io' ? 'border-b-2 border-indigo-500 text-indigo-700' : 'text-slate-500 hover:text-slate-700'} disabled:opacity-40 disabled:hover:text-slate-500`}
              >
                Node IO
              </button>
            </div>
            <div className="min-h-0 flex-1">
              {tab === 'logs' ? <LogsPanel logs={logs} /> : <NodeIOPanel node={selectedNodeRun} />}
            </div>
          </section>
        </div>
      </main>
    </div>
  );
}
