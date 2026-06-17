import type { ReactNode } from 'react';
import { Handle, Position } from 'reactflow';

interface BaseNodeProps {
  data: { id: string; tool: string; params: Record<string, unknown> };
  /** Tailwind text color class for accent line / glyph (e.g. "text-nodeTrigger") */
  accentClass: string;
  /** Short category label printed in the title strip (e.g. "TRIGGER", "LLM") */
  category: string;
  icon: ReactNode;
  showFalseHandle?: boolean;
}

export default function BaseNode({ data, accentClass, category, icon, showFalseHandle }: BaseNodeProps) {
  return (
    <div
      className={`relative font-mono text-paper-50 ${accentClass}`}
      style={{ width: 224 }}
    >
      {/* corner brackets */}
      <span className="absolute -left-px -top-px h-2 w-2 border-l border-t border-current opacity-80" />
      <span className="absolute -right-px -top-px h-2 w-2 border-r border-t border-current opacity-80" />
      <span className="absolute -bottom-px -left-px h-2 w-2 border-b border-l border-current opacity-80" />
      <span className="absolute -bottom-px -right-px h-2 w-2 border-b border-r border-current opacity-80" />

      <div className="rounded-sharp border border-ink-500 bg-ink-700/95 shadow-callout backdrop-blur-sm">
        <div className="flex items-center gap-2 border-b border-ink-500 bg-ink-600/80 px-3 py-1.5">
          <span className="flex h-5 w-5 items-center justify-center">{icon}</span>
          <span className="text-[10px] uppercase tracking-[0.28em] text-current">
            {category}
          </span>
          <span className="ml-auto text-[10px] tabular-nums text-paper-600">
            #{data.id.slice(-4)}
          </span>
        </div>
        <div className="px-3 py-2.5">
          <div className="truncate text-sm font-medium text-paper-50">{data.tool}</div>
          <div className="mt-0.5 truncate text-[10px] uppercase tracking-[0.22em] text-paper-600">
            id · {data.id}
          </div>
        </div>
        {/* accent baseline */}
        <div className="h-px w-full" style={{ background: 'currentColor', opacity: 0.5 }} />
      </div>

      <Handle type="target" position={Position.Top} id="main" />
      <Handle type="source" position={Position.Bottom} id={showFalseHandle ? 'true' : 'main'} />
      {showFalseHandle && (
        <Handle type="source" position={Position.Bottom} id="false" style={{ left: '75%' }} />
      )}
    </div>
  );
}
