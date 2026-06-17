import { Link, useParams } from 'react-router-dom';
import { useEffect, useMemo, useState } from 'react';
import WorkflowList from './WorkflowList';
import Canvas from '../canvas/Canvas';
import ChatPanel from '../chat/ChatPanel';
import NodeInspector from '../inspector/NodeInspector';
import RunBar from '../runs/RunBar';
import Icon from '../ui/Icon';
import { useWorkflowsStore } from './workflowsStore';
import { useChat } from '../chat/useChat';

export default function WorkflowsPage() {
  const { id } = useParams<{ id: string }>();
  const { current, fetchOne, setCurrent } = useWorkflowsStore();
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

  useEffect(() => {
    if (id) fetchOne(id);
    setSelectedNodeId(null);
  }, [id, fetchOne]);

  const selectedNode = useMemo(() => {
    if (!current?.definition || !selectedNodeId) return null;
    return current.definition.nodes.find((n) => n.id === selectedNodeId) || null;
  }, [current, selectedNodeId]);

  const { send } = useChat(id || '', (def) => {
    if (!current) return;
    setCurrent({ ...current, definition: def });
  });

  return (
    <div className="flex h-full bg-ink-900">
      <WorkflowList />
      <main className="flex flex-1 overflow-hidden">
        {!id && (
          <div className="drafting-table flex flex-1 flex-col items-center justify-center px-8 text-center">
            <div className="font-mono text-[10px] uppercase tracking-[0.36em] text-paper-400">
              no sheet open
            </div>
            <h2 className="h-display mt-3 text-3xl text-paper-50">The drafting table is clear.</h2>
            <p className="mt-2 max-w-md font-mono text-xs text-paper-400">
              Open a sheet from the left or draft a new workflow to begin.
            </p>
          </div>
        )}
        {id && !current && (
          <div className="drafting-table flex flex-1 items-center justify-center">
            <div className="flex items-center gap-3 font-mono text-xs uppercase tracking-[0.32em] text-paper-400">
              <Icon name="spinner" size={14} className="animate-spin text-cyan" />
              opening sheet…
            </div>
          </div>
        )}
        {id && current && (
          <>
            <div className="relative flex flex-1 flex-col bg-ink-800">
              <header className="relative z-10 flex items-center justify-between border-b border-ink-500 bg-ink-700/95 px-6 py-3 backdrop-blur-sm">
                <div className="flex items-center gap-6">
                  <div>
                    <div className="font-mono text-[10px] uppercase tracking-[0.32em] text-paper-400">
                      sheet · {current.id.slice(0, 12)}
                    </div>
                    <h1 className="h-display mt-0.5 text-xl text-paper-50">{current.name}</h1>
                  </div>
                  <div className="hidden h-10 w-px bg-ink-500 lg:block" />
                  <div className="hidden flex-col gap-1 font-mono text-[10px] uppercase tracking-[0.28em] text-paper-400 lg:flex">
                    <span>rev · v{current.version}</span>
                    <span>state · {current.status}</span>
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  <Link
                    to={`/workflows/${current.id}/triggers`}
                    className="flex items-center gap-2 rounded-sharp border border-ink-500 bg-ink-800 px-3 py-1.5 font-mono text-[11px] uppercase tracking-[0.24em] text-paper-200 transition hover:border-cyan/40 hover:text-cyan"
                  >
                    <Icon name="clock" size={12} />
                    triggers
                  </Link>
                  <Link
                    to={`/workflows/${current.id}/runs`}
                    className="flex items-center gap-2 rounded-sharp border border-ink-500 bg-ink-800 px-3 py-1.5 font-mono text-[11px] uppercase tracking-[0.24em] text-paper-200 transition hover:border-cyan/40 hover:text-cyan"
                  >
                    <Icon name="scroll" size={12} />
                    runs
                  </Link>
                </div>
              </header>

              <div className="relative flex-1 overflow-hidden">
                {/* Drafting marks in corners */}
                <div className="pointer-events-none absolute left-3 top-3 z-10 font-mono text-[10px] uppercase tracking-[0.32em] text-paper-600">
                  N ↑
                </div>
                <div className="pointer-events-none absolute right-3 top-3 z-10 font-mono text-[10px] uppercase tracking-[0.32em] text-paper-600">
                  scale · auto
                </div>
                <div className="pointer-events-none absolute bottom-16 left-3 z-10 font-mono text-[10px] uppercase tracking-[0.32em] text-paper-600">
                  origin · 0,0
                </div>
                {current.definition && (
                  <Canvas
                    definition={current.definition}
                    onSelectNode={setSelectedNodeId}
                  />
                )}
              </div>

              <RunBar workflowId={current.id} />
              <NodeInspector node={selectedNode} onClose={() => setSelectedNodeId(null)} />
            </div>
            <ChatPanel workflowId={current.id} onSend={send} />
          </>
        )}
      </main>
    </div>
  );
}
