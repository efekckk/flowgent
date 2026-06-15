import { useState } from 'react';
import { api } from '../api/client';
import type { RunResponse } from '../api/types';

interface Props {
  workflowId: string;
}

export default function RunBar({ workflowId }: Props) {
  const [running, setRunning] = useState(false);
  const [recent, setRecent] = useState<RunResponse[]>([]);

  async function onRun() {
    setRunning(true);
    try {
      const r = await api.runWorkflow(workflowId);
      setRecent((prev) => [r, ...prev].slice(0, 10));
    } catch (err) {
      setRecent((prev) => [
        { run_id: 'err-' + Date.now(), status: 'failed' as const, error: String(err) },
        ...prev,
      ].slice(0, 10));
    } finally {
      setRunning(false);
    }
  }

  return (
    <footer className="flex h-12 items-center justify-between border-t border-slate-200 bg-white px-4">
      <div className="flex items-center gap-2 text-xs text-slate-500">
        <span className="font-semibold uppercase tracking-wide">Run history:</span>
        {recent.length === 0 && <span className="text-slate-400">(no runs yet)</span>}
        {recent.map((r) => (
          <span
            key={r.run_id}
            className={`inline-block h-2 w-2 rounded-full ${
              r.status === 'succeeded' ? 'bg-emerald-500'
              : r.status === 'failed' ? 'bg-red-500'
              : 'bg-yellow-500'
            }`}
            title={`${r.status}${r.error ? ' — ' + r.error : ''}`}
          />
        ))}
      </div>
      <button
        onClick={onRun}
        disabled={running}
        className="rounded-md bg-emerald-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-emerald-700 disabled:opacity-50"
      >
        {running ? 'Running…' : '▶ Run now'}
      </button>
    </footer>
  );
}
