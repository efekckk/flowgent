import { useEffect } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useRunsStore } from './runsStore';
import { useWorkflowsStore } from '../workflows/workflowsStore';
import type { WorkflowRun } from '../api/types';
import { SectionSidebar, PageHeader, StatusPill } from '../ui/SectionShell';
import Icon from '../ui/Icon';

const STATUS_OPTIONS: Array<{ value: string; label: string }> = [
  { value: '', label: 'All statuses' },
  { value: 'running', label: 'Running' },
  { value: 'succeeded', label: 'Succeeded' },
  { value: 'failed', label: 'Failed' },
];

function formatStarted(iso?: string | null): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString();
}

function formatDuration(started?: string | null, finished?: string | null): string {
  if (!started || !finished) return '—';
  const a = new Date(started).getTime();
  const b = new Date(finished).getTime();
  if (Number.isNaN(a) || Number.isNaN(b) || b < a) return '—';
  const ms = b - a;
  if (ms < 1000) return `${ms}ms`;
  const s = ms / 1000;
  if (s < 60) return `${s.toFixed(s < 10 ? 2 : 1)}s`;
  const m = Math.floor(s / 60);
  const rs = Math.round(s - m * 60);
  return `${m}m ${rs}s`;
}

function localToRFC3339(v: string): string {
  if (!v) return '';
  const d = new Date(v);
  if (Number.isNaN(d.getTime())) return '';
  return d.toISOString();
}

function rfc3339ToLocal(v: string): string {
  if (!v) return '';
  const d = new Date(v);
  if (Number.isNaN(d.getTime())) return '';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export default function RunListPage() {
  const { id: workflowId } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const {
    items, nextCursor, status, from, to, loading, error,
    setFilter, fetch, loadMore,
  } = useRunsStore();
  const { current, fetchOne } = useWorkflowsStore();

  useEffect(() => {
    if (workflowId) {
      fetch(workflowId);
      fetchOne(workflowId);
    }
  }, [workflowId, fetch, fetchOne]);

  const workflowName = current && current.id === workflowId ? current.name : null;

  function onChangeStatus(value: string) {
    setFilter({ status: value });
    if (workflowId) fetch(workflowId);
  }
  function onChangeFrom(value: string) {
    setFilter({ from: localToRFC3339(value) });
    if (workflowId) fetch(workflowId);
  }
  function onChangeTo(value: string) {
    setFilter({ to: localToRFC3339(value) });
    if (workflowId) fetch(workflowId);
  }
  function rowKey(r: WorkflowRun): string { return r.id; }

  return (
    <div className="flex h-full bg-ink-900">
      <SectionSidebar workflowId={workflowId} workflowName={workflowName} active="runs" />

      <main className="flex-1 overflow-y-auto">
        <PageHeader
          eyebrow="ledger · execution history"
          title="Runs"
          description="Past executions of this sheet. Filter by status or by drafted window."
          meta={workflowId && (
            <span className="pill pill-cyan">
              <Icon name="dot" size={6} />
              {workflowName || workflowId}
            </span>
          )}
          actions={workflowId && (
            <Link
              to={`/workflows/${workflowId}/triggers`}
              className="flex items-center gap-2 rounded-sharp border border-ink-500 bg-ink-700 px-3 py-1.5 font-mono text-[11px] uppercase tracking-[0.24em] text-paper-200 transition hover:border-cyan/40 hover:text-cyan"
            >
              <Icon name="clock" size={12} />
              triggers
            </Link>
          )}
        />

        <div className="px-8 py-6">
          <div className="corners relative bg-ink-700/60 p-5">
            <span className="corner-bl" />
            <span className="corner-br" />
            <div className="mb-4 flex items-center gap-2 font-mono text-[10px] uppercase tracking-[0.32em] text-paper-400">
              <Icon name="search" size={12} className="text-cyan" />
              filters
            </div>
            <div className="flex flex-wrap items-end gap-4">
              <label className="block">
                <span className="block font-mono text-[10px] uppercase tracking-[0.28em] text-paper-400">status</span>
                <select
                  aria-label="Status"
                  value={status}
                  onChange={(e) => onChangeStatus(e.target.value)}
                  className="mt-1.5 block rounded-sharp border border-ink-500 bg-ink-800 px-3 py-2 font-mono text-sm text-paper-50 focus:border-cyan focus:outline-none"
                >
                  {STATUS_OPTIONS.map((o) => (
                    <option key={o.value || 'all'} value={o.value}>{o.label}</option>
                  ))}
                </select>
              </label>
              <label className="block">
                <span className="block font-mono text-[10px] uppercase tracking-[0.28em] text-paper-400">from</span>
                <input
                  type="datetime-local"
                  aria-label="From"
                  value={rfc3339ToLocal(from)}
                  onChange={(e) => onChangeFrom(e.target.value)}
                  className="mt-1.5 block rounded-sharp border border-ink-500 bg-ink-800 px-3 py-2 font-mono text-sm text-paper-50 focus:border-cyan focus:outline-none"
                />
              </label>
              <label className="block">
                <span className="block font-mono text-[10px] uppercase tracking-[0.28em] text-paper-400">to</span>
                <input
                  type="datetime-local"
                  aria-label="To"
                  value={rfc3339ToLocal(to)}
                  onChange={(e) => onChangeTo(e.target.value)}
                  className="mt-1.5 block rounded-sharp border border-ink-500 bg-ink-800 px-3 py-2 font-mono text-sm text-paper-50 focus:border-cyan focus:outline-none"
                />
              </label>
            </div>
          </div>

          {error && (
            <div className="mt-4 flex items-center gap-2 border border-rose/40 bg-rose/5 px-3 py-2 font-mono text-xs text-rose">
              <Icon name="x" size={12} />
              {error}
            </div>
          )}

          <section className="mt-6">
            {items.length === 0 && !loading ? (
              <p className="border border-dashed border-ink-500 bg-ink-700/30 px-5 py-10 text-center font-mono text-xs uppercase tracking-[0.28em] text-paper-400">
                no runs match these filters.
              </p>
            ) : (
              <div className="corners relative overflow-hidden border border-ink-500 bg-ink-700/40">
                <span className="corner-bl" />
                <span className="corner-br" />
                <table className="min-w-full">
                  <thead className="border-b border-ink-500 bg-ink-700/80">
                    <tr>
                      {['status', 'started', 'duration', 'trigger', 'id'].map((h) => (
                        <th key={h} scope="col" className="px-4 py-2.5 text-left font-mono text-[10px] uppercase tracking-[0.28em] text-paper-400">
                          {h}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {items.map((r) => (
                      <tr
                        key={rowKey(r)}
                        onClick={() => navigate(`/runs/${r.id}`)}
                        className="cursor-pointer border-b border-ink-500/60 transition hover:bg-cyan/5"
                      >
                        <td className="px-4 py-2.5"><StatusPill status={r.status} /></td>
                        <td className="px-4 py-2.5 font-mono text-xs text-paper-200 tabular-nums">{formatStarted(r.started_at)}</td>
                        <td className="px-4 py-2.5 font-mono text-xs text-paper-200 tabular-nums">{formatDuration(r.started_at, r.finished_at)}</td>
                        <td className="px-4 py-2.5 font-mono text-xs text-paper-400">
                          <span>{r.trigger_kind || 'manual'}</span>
                          {r.parent_run_id && (
                            <span className="ml-2 inline-flex items-center gap-1 border border-cyan/40 bg-cyan/5 px-1.5 py-0.5 text-[9px] uppercase tracking-[0.24em] text-cyan">
                              replay
                            </span>
                          )}
                        </td>
                        <td className="px-4 py-2.5 font-mono text-[11px] text-paper-600 tabular-nums">{r.id}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {loading && (
              <div className="mt-3 flex items-center gap-2 font-mono text-xs uppercase tracking-[0.28em] text-paper-400">
                <Icon name="spinner" size={12} className="animate-spin text-cyan" />
                loading…
              </div>
            )}

            {nextCursor && (
              <div className="mt-4">
                <button
                  type="button"
                  onClick={() => loadMore()}
                  disabled={loading}
                  className="rounded-sharp border border-ink-500 bg-ink-700 px-3 py-1.5 font-mono text-[11px] uppercase tracking-[0.24em] text-paper-200 transition hover:border-cyan/40 hover:text-cyan disabled:opacity-50"
                >
                  {loading ? 'Loading…' : 'Load more'}
                </button>
              </div>
            )}
          </section>
        </div>
      </main>
    </div>
  );
}
