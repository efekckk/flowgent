import { useEffect } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useRunsStore } from './runsStore';
import { useWorkflowsStore } from '../workflows/workflowsStore';
import { useAuth } from '../auth/useAuth';
import type { WorkflowRun } from '../api/types';
import LogSearchBar from '../search/LogSearchBar';

const STATUS_OPTIONS: Array<{ value: string; label: string }> = [
  { value: '', label: 'All statuses' },
  { value: 'running', label: 'Running' },
  { value: 'succeeded', label: 'Succeeded' },
  { value: 'failed', label: 'Failed' },
];

function statusClasses(status: string): string {
  switch (status) {
    case 'succeeded':
      return 'bg-emerald-100 text-emerald-700';
    case 'failed':
      return 'bg-red-100 text-red-700';
    case 'running':
      return 'bg-yellow-100 text-yellow-700';
    case 'cancelled':
      return 'bg-slate-100 text-slate-500';
    default:
      return 'bg-slate-100 text-slate-600';
  }
}

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

// Convert a value from <input type="datetime-local"> to RFC3339 (UTC) or '' if blank.
function localToRFC3339(v: string): string {
  if (!v) return '';
  const d = new Date(v);
  if (Number.isNaN(d.getTime())) return '';
  return d.toISOString();
}

// Convert RFC3339 stored value back to value usable by datetime-local input.
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
  const { logout, workspace } = useAuth();

  useEffect(() => {
    if (workflowId) {
      fetch(workflowId);
      fetchOne(workflowId);
    }
  }, [workflowId, fetch, fetchOne]);

  const workflowLabel = current && current.id === workflowId ? current.name : (workflowId || '');

  function onChangeStatus(value: string) {
    setFilter({ status: value });
    if (workflowId) fetch(workflowId);
  }

  function onChangeFrom(value: string) {
    const rfc = localToRFC3339(value);
    setFilter({ from: rfc });
    if (workflowId) fetch(workflowId);
  }

  function onChangeTo(value: string) {
    const rfc = localToRFC3339(value);
    setFilter({ to: rfc });
    if (workflowId) fetch(workflowId);
  }

  function rowKey(r: WorkflowRun): string {
    return r.id;
  }

  return (
    <div className="flex h-full bg-slate-50">
      <aside className="flex h-full w-64 flex-col border-r border-slate-200 bg-white p-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">Workflow</h2>
        <Link to="/workflows" className="mt-3 rounded-md px-3 py-2 text-sm text-slate-700 hover:bg-slate-50">← Back to workflows</Link>
        {workflowId && (
          <Link
            to={`/workflows/${workflowId}`}
            className="rounded-md px-3 py-2 text-sm text-slate-700 hover:bg-slate-50"
          >
            Editor
          </Link>
        )}
        {workflowId && (
          <Link
            to={`/workflows/${workflowId}/triggers`}
            className="rounded-md px-3 py-2 text-sm text-slate-700 hover:bg-slate-50"
          >
            Triggers
          </Link>
        )}
        <Link
          to={workflowId ? `/workflows/${workflowId}/runs` : '#'}
          className="rounded-md bg-indigo-50 px-3 py-2 text-sm font-medium text-indigo-700"
        >
          Runs
        </Link>
        <Link to="/credentials" className="rounded-md px-3 py-2 text-sm text-slate-700 hover:bg-slate-50">Credentials</Link>
        <div className="mt-auto space-y-2">
          <LogSearchBar workspaceId={workspace?.id ?? null} />
          <button onClick={logout} className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm text-slate-600 hover:bg-slate-50">
            Sign out
          </button>
        </div>
      </aside>

      <main className="flex-1 overflow-y-auto p-8">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h1 className="text-2xl font-semibold text-slate-800">
            Runs for <span className="font-mono text-indigo-700">{workflowLabel}</span>
          </h1>
          {workflowId && (
            <Link
              to={`/workflows/${workflowId}/triggers`}
              className="rounded-md border border-slate-300 px-3 py-1.5 text-sm text-slate-700 hover:bg-slate-50"
            >
              🔔 Triggers
            </Link>
          )}
        </div>
        <p className="mt-1 text-sm text-slate-500">
          Past executions of this workflow. Filter by status or time window.
        </p>

        <div className="mt-6 flex flex-wrap items-end gap-4 rounded-md border border-slate-200 bg-white p-4">
          <label className="block">
            <span className="block text-xs font-medium uppercase tracking-wide text-slate-500">Status</span>
            <select
              aria-label="Status"
              value={status}
              onChange={(e) => onChangeStatus(e.target.value)}
              className="mt-1 block rounded-md border border-slate-300 px-3 py-2 text-sm"
            >
              {STATUS_OPTIONS.map((o) => (
                <option key={o.value || 'all'} value={o.value}>{o.label}</option>
              ))}
            </select>
          </label>
          <label className="block">
            <span className="block text-xs font-medium uppercase tracking-wide text-slate-500">From</span>
            <input
              type="datetime-local"
              aria-label="From"
              value={rfc3339ToLocal(from)}
              onChange={(e) => onChangeFrom(e.target.value)}
              className="mt-1 block rounded-md border border-slate-300 px-3 py-2 text-sm"
            />
          </label>
          <label className="block">
            <span className="block text-xs font-medium uppercase tracking-wide text-slate-500">To</span>
            <input
              type="datetime-local"
              aria-label="To"
              value={rfc3339ToLocal(to)}
              onChange={(e) => onChangeTo(e.target.value)}
              className="mt-1 block rounded-md border border-slate-300 px-3 py-2 text-sm"
            />
          </label>
        </div>

        {error && <p className="mt-4 text-sm text-red-600">{error}</p>}

        <section className="mt-6">
          {items.length === 0 && !loading ? (
            <p className="text-sm text-slate-400">No runs match these filters.</p>
          ) : (
            <div className="overflow-hidden rounded-md border border-slate-200 bg-white">
              <table className="min-w-full divide-y divide-slate-200">
                <thead className="bg-slate-50">
                  <tr>
                    <th scope="col" className="px-4 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">Status</th>
                    <th scope="col" className="px-4 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">Started at</th>
                    <th scope="col" className="px-4 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">Duration</th>
                    <th scope="col" className="px-4 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">Trigger</th>
                    <th scope="col" className="px-4 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">ID</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {items.map((r) => (
                    <tr
                      key={rowKey(r)}
                      onClick={() => navigate(`/runs/${r.id}`)}
                      className="cursor-pointer hover:bg-slate-50"
                    >
                      <td className="px-4 py-2">
                        <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${statusClasses(r.status)}`}>
                          {r.status}
                        </span>
                      </td>
                      <td className="px-4 py-2 text-sm text-slate-700">{formatStarted(r.started_at)}</td>
                      <td className="px-4 py-2 text-sm text-slate-700">{formatDuration(r.started_at, r.finished_at)}</td>
                      <td className="px-4 py-2 text-sm text-slate-700">
                        <span className="font-mono text-xs text-slate-600">{r.trigger_kind || 'manual'}</span>
                        {r.parent_run_id && (
                          <span className="ml-2 rounded-full bg-indigo-50 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-indigo-700">
                            replay
                          </span>
                        )}
                      </td>
                      <td className="px-4 py-2 font-mono text-xs text-slate-500">{r.id}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {loading && <p className="mt-3 text-sm text-slate-400">Loading…</p>}

          {nextCursor && (
            <div className="mt-4">
              <button
                type="button"
                onClick={() => loadMore()}
                disabled={loading}
                className="rounded-md border border-slate-300 bg-white px-3 py-1.5 text-sm text-slate-700 hover:bg-slate-50 disabled:opacity-50"
              >
                {loading ? 'Loading…' : 'Load more'}
              </button>
            </div>
          )}
        </section>
      </main>
    </div>
  );
}
