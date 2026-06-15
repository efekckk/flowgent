import { Handle, Position } from 'reactflow';

interface BaseNodeProps {
  data: { id: string; tool: string; params: Record<string, unknown> };
  colorClass: string;
  icon: string;
  showFalseHandle?: boolean;
}

export default function BaseNode({ data, colorClass, icon, showFalseHandle }: BaseNodeProps) {
  return (
    <div className="rounded-md border bg-white shadow-sm" style={{ width: 200 }}>
      <div className={`flex items-center gap-2 rounded-t-md px-3 py-2 ${colorClass}`}>
        <span aria-hidden="true">{icon}</span>
        <span className="truncate font-medium text-white">{data.tool}</span>
      </div>
      <div className="truncate px-3 py-2 text-xs text-slate-500">id: {data.id}</div>
      <Handle type="target" position={Position.Top} id="main" />
      <Handle type="source" position={Position.Bottom} id={showFalseHandle ? 'true' : 'main'} />
      {showFalseHandle && (
        <Handle type="source" position={Position.Bottom} id="false" style={{ left: '75%' }} />
      )}
    </div>
  );
}
