import type { WorkflowNode } from '../api/types';
import Icon from '../ui/Icon';

interface Props {
  node: WorkflowNode | null;
  onClose: () => void;
}

export default function NodeInspector({ node, onClose }: Props) {
  if (!node) return null;
  return (
    <div className="corners absolute right-4 top-4 z-10 w-80 animate-draft-in bg-ink-700/95 shadow-callout backdrop-blur-sm">
      <span className="corner-bl" />
      <span className="corner-br" />
      <div className="flex items-center justify-between border-b border-ink-500 px-4 py-2.5">
        <div className="flex flex-col">
          <span className="font-mono text-[10px] uppercase tracking-[0.32em] text-cyan">node · callout</span>
          <span className="truncate font-mono text-sm text-paper-50">{node.id}</span>
        </div>
        <button
          onClick={onClose}
          className="rounded-sharp border border-ink-500 p-1 text-paper-400 transition hover:border-cyan/40 hover:text-cyan"
          aria-label="Close"
        >
          <Icon name="x" size={12} />
        </button>
      </div>
      <div className="space-y-4 px-4 py-4">
        <div>
          <div className="font-mono text-[10px] uppercase tracking-[0.32em] text-paper-400">tool</div>
          <code className="mt-1 block border border-ink-500 bg-ink-800 px-2 py-1 font-mono text-xs text-cyan">{node.tool}</code>
        </div>
        <div>
          <div className="font-mono text-[10px] uppercase tracking-[0.32em] text-paper-400">params</div>
          <pre className="mt-1 max-h-64 overflow-auto border border-ink-500 bg-ink-800 p-2 font-mono text-[11px] leading-5 text-paper-200">
{JSON.stringify(node.params, null, 2)}
          </pre>
        </div>
      </div>
    </div>
  );
}
