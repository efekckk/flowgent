import { useState } from 'react';
import { api } from '../api/client';
import type { RunResponse } from '../api/types';
import Icon from '../ui/Icon';

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
    <footer className="relative z-10 flex h-12 items-center justify-between border-t border-ink-500 bg-ink-700/95 px-5 backdrop-blur-sm">
      <div className="flex items-center gap-4 font-mono text-[10px] uppercase tracking-[0.32em] text-paper-400">
        <span className="text-cyan">last 10 runs</span>
        {recent.length === 0 && (
          <span className="text-paper-600">— no executions yet</span>
        )}
        {recent.length > 0 && (
          <div className="flex items-center gap-1.5">
            {recent.map((r) => (
              <span
                key={r.run_id}
                className={`inline-block h-2 w-2 rounded-sharp ${
                  r.status === 'succeeded' ? 'bg-moss shadow-[0_0_0_1px_rgba(134,239,172,0.35)]'
                  : r.status === 'failed' ? 'bg-rose shadow-[0_0_0_1px_rgba(251,113,133,0.35)]'
                  : 'bg-amber shadow-[0_0_0_1px_rgba(251,191,36,0.35)]'
                }`}
                title={`${r.status}${r.error ? ' — ' + r.error : ''}`}
              />
            ))}
          </div>
        )}
      </div>
      <button
        onClick={onRun}
        disabled={running}
        className="flex items-center gap-2 rounded-sharp border border-cyan/40 bg-cyan/15 px-4 py-1.5 font-mono text-[11px] uppercase tracking-[0.28em] text-cyan transition hover:bg-cyan/25 disabled:opacity-50"
      >
        {running ? <Icon name="spinner" size={12} className="animate-spin" /> : <Icon name="play" size={12} />}
        {running ? 'firing…' : 'fire run'}
      </button>
    </footer>
  );
}
