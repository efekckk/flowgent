import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import ReactFlow, {
  Background, BackgroundVariant, Controls, MiniMap,
  Handle, Position,
  type Node, type Edge, type NodeMouseHandler, type NodeProps,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { useRunDetailStore } from './runDetailStore';
import { useRunLogStream } from './useRunLogStream';
import { useWorkflowsStore } from '../workflows/workflowsStore';
import { workflowToFlow } from '../canvas/workflowToFlow';
import type { NodeRun, RunLogEvent } from '../api/types';
import { SectionSidebar, StatusPill } from '../ui/SectionShell';
import Icon from '../ui/Icon';

type NodeStatus = 'pending' | 'running' | 'succeeded' | 'failed' | string;

function nodeAccent(status: NodeStatus): { ring: string; border: string; label: string } {
  switch (status) {
    case 'succeeded':
      return { ring: 'shadow-[0_0_0_1px_rgba(134,239,172,0.55)]', border: 'border-moss/60', label: 'text-moss' };
    case 'failed':
      return { ring: 'shadow-[0_0_0_1px_rgba(251,113,133,0.6)]', border: 'border-rose/60', label: 'text-rose' };
    case 'running':
      return { ring: 'shadow-[0_0_0_1px_rgba(251,191,36,0.6)] animate-pulse', border: 'border-amber/60', label: 'text-amber' };
    case 'pending':
    default:
      return { ring: 'shadow-[0_0_0_1px_rgba(125,211,252,0.18)]', border: 'border-ink-500', label: 'text-paper-400' };
  }
}

interface StatusNodeData {
  id: string;
  tool: string;
  status: NodeStatus;
}

function StatusNode({ data }: NodeProps<StatusNodeData>) {
  const accent = nodeAccent(data.status);
  return (
    <div
      data-testid={`run-node-${data.id}`}
      className={`relative rounded-sharp border bg-ink-700/95 font-mono backdrop-blur-sm ${accent.border} ${accent.ring}`}
      style={{ width: 208 }}
    >
      <div className="flex items-center justify-between border-b border-ink-500/80 px-3 py-1.5">
        <span className="truncate text-[10px] uppercase tracking-[0.28em] text-paper-200">{data.tool}</span>
        <span className={`text-[10px] uppercase tracking-[0.24em] ${accent.label}`}>
          {data.status || 'pending'}
        </span>
      </div>
      <div className="truncate px-3 py-2 text-[10px] uppercase tracking-[0.22em] text-paper-600">
        id · {data.id}
      </div>
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

function levelClass(level: string): string {
  switch (level) {
    case 'error': return 'text-rose';
    case 'warn':  return 'text-amber';
    case 'debug': return 'text-paper-600';
    default:      return 'text-paper-200';
  }
}

function prettyJSON(v: unknown): string {
  if (v === undefined || v === null) return '';
  if (typeof v === 'string') {
    try { return JSON.stringify(JSON.parse(v), null, 2); }
    catch { return v; }
  }
  try { return JSON.stringify(v, null, 2); }
  catch { return String(v); }
}

type RightTab = 'logs' | 'io';

function LogsPanel({ logs }: { logs: RunLogEvent[] }) {
  const scrollRef = useRef<HTMLDivElement | null>(null);
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
    return (
      <div className="p-6 font-mono text-xs uppercase tracking-[0.28em] text-paper-400">
        No log lines yet.
        <span className="caret" />
      </div>
    );
  }

  return (
    <div
      ref={scrollRef}
      onScroll={onScroll}
      data-testid="logs-panel"
      className="h-full overflow-y-auto bg-ink-800/40 p-3 font-mono text-[11px] leading-5"
    >
      {logs.map((l, i) => (
        <div key={l.id ?? i} className="grid grid-cols-[80px_56px_minmax(80px,140px)_1fr] gap-2 border-b border-ink-700/40 py-0.5">
          <span className="shrink-0 text-paper-600 tabular-nums">{formatTime(l.at)}</span>
          <span className={`shrink-0 uppercase tracking-[0.18em] ${levelClass(l.level)}`}>{l.level}</span>
          <span className="truncate text-cyan">{l.node_id || '—'}</span>
          <span className="break-all text-paper-200">{l.message}</span>
        </div>
      ))}
    </div>
  );
}

function NodeIOPanel({ node }: { node: NodeRun | null }) {
  if (!node) {
    return (
      <div className="p-6 font-mono text-xs uppercase tracking-[0.28em] text-paper-400">
        Select a node to inspect its input and output.
      </div>
    );
  }
  return (
    <div className="h-full overflow-y-auto p-5 text-xs">
      <div className="flex items-center justify-between border-b border-ink-500 pb-3">
        <div>
          <div className="font-mono text-[10px] uppercase tracking-[0.32em] text-paper-400">node</div>
          <div className="font-mono text-sm text-paper-50">{node.node_id}</div>
        </div>
        <StatusPill status={node.status} />
      </div>
      {node.error !== undefined && node.error !== null && (
        <section className="mt-4">
          <h4 className="font-mono text-[10px] uppercase tracking-[0.32em] text-rose">error</h4>
          <pre className="mt-2 whitespace-pre-wrap border border-rose/30 bg-rose/5 p-3 font-mono text-[11px] text-rose/90">{prettyJSON(node.error)}</pre>
        </section>
      )}
      <section className="mt-4">
        <h4 className="font-mono text-[10px] uppercase tracking-[0.32em] text-paper-400">input</h4>
        <pre className="mt-2 whitespace-pre-wrap border border-ink-500 bg-ink-800 p-3 font-mono text-[11px] text-paper-200">{prettyJSON(node.input) || '—'}</pre>
      </section>
      <section className="mt-4">
        <h4 className="font-mono text-[10px] uppercase tracking-[0.32em] text-paper-400">output</h4>
        <pre className="mt-2 whitespace-pre-wrap border border-ink-500 bg-ink-800 p-3 font-mono text-[11px] text-paper-200">{prettyJSON(node.output) || '—'}</pre>
      </section>
    </div>
  );
}

export default function RunDetailPage() {
  const { id: runId } = useParams<{ id: string }>();
  const navigate = useNavigate();
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

  const workflowName = workflow?.id === run?.workflow_id ? workflow?.name : null;

  return (
    <div className="flex h-full bg-ink-900">
      <SectionSidebar
        workflowId={run?.workflow_id || null}
        workflowName={workflowName || null}
        active="runs"
      />

      <main className="flex min-w-0 flex-1 flex-col">
        <header className="flex items-center justify-between border-b border-ink-500 bg-ink-700/80 px-6 py-3 backdrop-blur-sm">
          <div className="flex items-center gap-4">
            <div>
              <div className="font-mono text-[10px] uppercase tracking-[0.36em] text-cyan">run · ledger</div>
              <div className="mt-0.5 flex items-center gap-3">
                <span className="font-mono text-sm text-paper-200">{runId}</span>
                {run && <StatusPill status={run.status} />}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-3">
            {replayError && <span className="font-mono text-[11px] text-rose">{replayError}</span>}
            <button
              type="button"
              onClick={onReplay}
              disabled={!runId || replaying}
              className="flex items-center gap-2 rounded-sharp border border-cyan/40 bg-cyan/10 px-3 py-1.5 font-mono text-[11px] uppercase tracking-[0.24em] text-cyan transition hover:bg-cyan/20 disabled:opacity-50"
            >
              {replaying ? <Icon name="spinner" size={12} className="animate-spin" /> : <Icon name="play" size={12} />}
              {replaying ? 'Replaying…' : 'Replay'}
            </button>
          </div>
        </header>

        {error && (
          <div className="border-b border-rose/30 bg-rose/10 px-6 py-2 font-mono text-xs text-rose">
            {error}
          </div>
        )}

        <div className="flex min-h-0 flex-1">
          <section className="relative flex-[3] border-r border-ink-500 bg-ink-800">
            {loading && !workflow ? (
              <div className="flex h-full items-center justify-center gap-2 font-mono text-xs uppercase tracking-[0.28em] text-paper-400">
                <Icon name="spinner" size={12} className="animate-spin text-cyan" />
                loading…
              </div>
            ) : flowNodes.length === 0 ? (
              <p className="p-6 font-mono text-xs uppercase tracking-[0.28em] text-paper-400">
                No workflow definition available for this run.
              </p>
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
                  proOptions={{ hideAttribution: true }}
                  defaultEdgeOptions={{
                    type: 'smoothstep',
                    style: { stroke: 'rgba(125, 211, 252, 0.55)', strokeWidth: 1.25 },
                  }}
                >
                  <Background variant={BackgroundVariant.Dots} gap={24} size={1} color="rgba(125, 211, 252, 0.18)" />
                  <Controls showInteractive={false} position="bottom-right" />
                  <MiniMap pannable zoomable
                    nodeColor="#7DD3FC" nodeStrokeColor="#0EA5E9" nodeBorderRadius={0}
                    maskColor="rgba(11, 18, 32, 0.6)"
                  />
                </ReactFlow>
                <div className="pointer-events-none absolute inset-0" style={{ background: 'radial-gradient(ellipse at center, transparent 55%, rgba(0,0,0,0.45) 100%)' }} />
              </div>
            )}
          </section>

          <section className="flex flex-[2] flex-col bg-ink-700/50">
            <div className="flex border-b border-ink-500 bg-ink-700">
              <button
                type="button"
                onClick={() => setTab('logs')}
                className={`flex items-center gap-2 px-5 py-2.5 font-mono text-[11px] uppercase tracking-[0.28em] transition ${
                  tab === 'logs'
                    ? 'border-b-2 border-cyan text-cyan'
                    : 'border-b-2 border-transparent text-paper-400 hover:text-paper-200'
                }`}
              >
                <Icon name="scroll" size={12} /> Logs
              </button>
              <button
                type="button"
                onClick={() => selected && setTab('io')}
                disabled={!selected}
                className={`flex items-center gap-2 px-5 py-2.5 font-mono text-[11px] uppercase tracking-[0.28em] transition ${
                  tab === 'io'
                    ? 'border-b-2 border-cyan text-cyan'
                    : 'border-b-2 border-transparent text-paper-400 hover:text-paper-200'
                } disabled:opacity-40 disabled:hover:text-paper-400`}
              >
                <Icon name="braces" size={12} /> Node IO
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
