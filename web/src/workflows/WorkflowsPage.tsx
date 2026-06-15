import { useParams } from 'react-router-dom';
import { useEffect, useMemo, useState } from 'react';
import WorkflowList from './WorkflowList';
import Canvas from '../canvas/Canvas';
import ChatPanel from '../chat/ChatPanel';
import NodeInspector from '../inspector/NodeInspector';
import { useWorkflowsStore } from './workflowsStore';

export default function WorkflowsPage() {
  const { id } = useParams<{ id: string }>();
  const { current, fetchOne } = useWorkflowsStore();
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

  useEffect(() => {
    if (id) fetchOne(id);
    setSelectedNodeId(null);
  }, [id, fetchOne]);

  const selectedNode = useMemo(() => {
    if (!current?.definition || !selectedNodeId) return null;
    return current.definition.nodes.find((n) => n.id === selectedNodeId) || null;
  }, [current, selectedNodeId]);

  return (
    <div className="flex h-full bg-slate-50">
      <WorkflowList />
      <main className="flex flex-1 overflow-hidden">
        {!id && (
          <div className="flex flex-1 items-center justify-center text-slate-400">
            Select or create a workflow to begin.
          </div>
        )}
        {id && !current && (
          <div className="flex flex-1 items-center justify-center text-slate-400">
            Loading workflow…
          </div>
        )}
        {id && current && (
          <>
            <div className="relative flex flex-1 flex-col">
              <header className="border-b border-slate-200 bg-white px-4 py-3">
                <h1 className="text-lg font-semibold text-slate-800">{current.name}</h1>
                <p className="text-xs text-slate-500">Status: {current.status} · v{current.version}</p>
              </header>
              <div className="flex-1 overflow-hidden">
                {current.definition && (
                  <Canvas
                    definition={current.definition}
                    onSelectNode={setSelectedNodeId}
                  />
                )}
              </div>
              <NodeInspector node={selectedNode} onClose={() => setSelectedNodeId(null)} />
            </div>
            <ChatPanel workflowId={current.id} />
          </>
        )}
      </main>
    </div>
  );
}
