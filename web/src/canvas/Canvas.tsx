import type { WorkflowDefinition } from '../api/types';

export default function Canvas({ definition }: { definition: WorkflowDefinition }) {
  return (
    <div className="flex h-full items-center justify-center bg-slate-100 text-slate-400">
      Canvas: {definition.nodes.length} nodes, {definition.edges.length} edges
    </div>
  );
}
