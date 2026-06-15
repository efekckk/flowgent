import type { WorkflowNode } from '../api/types';

interface Props {
  node: WorkflowNode | null;
  onClose: () => void;
}

export default function NodeInspector({ node, onClose }: Props) {
  if (!node) return null;
  return (
    <div className="absolute right-4 top-4 z-10 w-80 rounded-md border border-slate-200 bg-white shadow-lg">
      <div className="flex items-center justify-between border-b border-slate-200 px-3 py-2">
        <div className="flex flex-col">
          <span className="text-xs text-slate-500">Node</span>
          <span className="truncate font-medium text-slate-800">{node.id}</span>
        </div>
        <button onClick={onClose} className="text-slate-400 hover:text-slate-600" aria-label="Close">×</button>
      </div>
      <div className="space-y-3 px-3 py-3 text-sm">
        <div>
          <div className="text-xs uppercase tracking-wide text-slate-500">Tool</div>
          <code className="block rounded bg-slate-100 px-2 py-1 font-mono text-xs text-slate-700">{node.tool}</code>
        </div>
        <div>
          <div className="text-xs uppercase tracking-wide text-slate-500">Params</div>
          <pre className="max-h-64 overflow-auto rounded bg-slate-100 px-2 py-1 font-mono text-xs text-slate-700">
{JSON.stringify(node.params, null, 2)}
          </pre>
        </div>
      </div>
    </div>
  );
}
