import { useEffect, useState, type FormEvent } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useTriggersStore } from './triggersStore';
import { useWorkflowsStore } from '../workflows/workflowsStore';
import { useAuth } from '../auth/useAuth';
import LogSearchBar from '../search/LogSearchBar';

const CRON_PRESETS: Array<{ value: string; label: string }> = [
  { value: '', label: 'Select a preset…' },
  { value: '@hourly', label: 'Every hour (@hourly)' },
  { value: '@daily', label: 'Every day (@daily)' },
  { value: '@weekly', label: 'Every week (@weekly)' },
  { value: '@monthly', label: 'Every month (@monthly)' },
  { value: '@every 5m', label: 'Every 5 minutes' },
  { value: '@every 1h', label: 'Every 1 hour' },
];

export default function TriggersPage() {
  const { id: workflowId } = useParams<{ id: string }>();
  const { items, fetch, create, toggle, remove, error } = useTriggersStore();
  const { current, fetchOne } = useWorkflowsStore();
  const { logout, workspace } = useAuth();

  const [cron, setCron] = useState('');
  const [preset, setPreset] = useState('');
  const [cronSubmitting, setCronSubmitting] = useState(false);
  const [cronError, setCronError] = useState<string | null>(null);

  const [webhookSecret, setWebhookSecret] = useState('');
  const [webhookSubmitting, setWebhookSubmitting] = useState(false);
  const [webhookError, setWebhookError] = useState<string | null>(null);

  const [copiedId, setCopiedId] = useState<string | null>(null);

  useEffect(() => {
    if (workflowId) {
      fetch(workflowId);
      fetchOne(workflowId);
    }
  }, [workflowId, fetch, fetchOne]);

  async function onSubmitCron(e: FormEvent) {
    e.preventDefault();
    if (!workflowId) return;
    const expr = cron.trim();
    if (!expr) {
      setCronError('Cron expression is required');
      return;
    }
    setCronSubmitting(true);
    setCronError(null);
    try {
      await create(workflowId, 'cron', { cron: expr });
      setCron('');
      setPreset('');
    } catch (err) {
      setCronError(String(err));
    } finally {
      setCronSubmitting(false);
    }
  }

  async function onSubmitWebhook(e: FormEvent) {
    e.preventDefault();
    if (!workflowId) return;
    setWebhookSubmitting(true);
    setWebhookError(null);
    try {
      const cfg: Record<string, unknown> = {};
      const s = webhookSecret.trim();
      if (s) cfg.secret = s;
      await create(workflowId, 'webhook', cfg);
      setWebhookSecret('');
    } catch (err) {
      setWebhookError(String(err));
    } finally {
      setWebhookSubmitting(false);
    }
  }

  async function copyUrl(id: string, url: string) {
    try {
      await navigator.clipboard.writeText(url);
      setCopiedId(id);
      setTimeout(() => setCopiedId((cur) => (cur === id ? null : cur)), 1500);
    } catch {
      // clipboard unavailable; no-op
    }
  }

  const workflowLabel = current && current.id === workflowId ? current.name : (workflowId || '');

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
        <Link
          to={workflowId ? `/workflows/${workflowId}/triggers` : '#'}
          className="rounded-md bg-indigo-50 px-3 py-2 text-sm font-medium text-indigo-700"
        >
          Triggers
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
        <h1 className="text-2xl font-semibold text-slate-800">
          Triggers for <span className="font-mono text-indigo-700">{workflowLabel}</span>
        </h1>
        <p className="mt-1 text-sm text-slate-500">
          Schedule runs on a cron expression, or expose a webhook URL that runs this workflow when called.
        </p>

        <div className="mt-6 grid max-w-4xl gap-4 md:grid-cols-2">
          <form onSubmit={onSubmitCron} className="space-y-3 rounded-md border border-slate-200 bg-white p-4">
            <h2 className="text-sm font-semibold text-slate-700">Add cron trigger</h2>
            <label className="block">
              <span className="text-sm font-medium text-slate-700">Preset</span>
              <select
                value={preset}
                onChange={(e) => {
                  const v = e.target.value;
                  setPreset(v);
                  if (v) setCron(v);
                }}
                className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm"
              >
                {CRON_PRESETS.map((p) => (
                  <option key={p.value || 'none'} value={p.value}>{p.label}</option>
                ))}
              </select>
            </label>
            <label className="block">
              <span className="text-sm font-medium text-slate-700">Cron expression</span>
              <input
                type="text"
                value={cron}
                onChange={(e) => setCron(e.target.value)}
                placeholder="@daily, 0 9 * * *, @every 5m"
                className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 font-mono text-sm"
              />
            </label>
            {cronError && <p className="text-sm text-red-600">{cronError}</p>}
            <button
              type="submit"
              disabled={cronSubmitting}
              className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {cronSubmitting ? 'Adding…' : 'Add cron trigger'}
            </button>
          </form>

          <form onSubmit={onSubmitWebhook} className="space-y-3 rounded-md border border-slate-200 bg-white p-4">
            <h2 className="text-sm font-semibold text-slate-700">Add webhook trigger</h2>
            <label className="block">
              <span className="text-sm font-medium text-slate-700">HMAC secret (optional)</span>
              <input
                type="password"
                value={webhookSecret}
                onChange={(e) => setWebhookSecret(e.target.value)}
                placeholder="leave blank for no signature check"
                autoComplete="new-password"
                className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 font-mono text-sm"
              />
            </label>
            <p className="text-xs text-slate-500">
              When set, incoming requests must include an <code>X-Flowgent-Signature</code> HMAC-SHA256 header.
            </p>
            {webhookError && <p className="text-sm text-red-600">{webhookError}</p>}
            <button
              type="submit"
              disabled={webhookSubmitting}
              className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {webhookSubmitting ? 'Adding…' : 'Add webhook trigger'}
            </button>
          </form>
        </div>

        <section className="mt-8 max-w-4xl">
          <h2 className="text-sm font-semibold text-slate-700">Existing triggers</h2>
          {error && <p className="mt-2 text-sm text-red-600">{error}</p>}
          {items.length === 0 ? (
            <p className="mt-2 text-sm text-slate-400">No triggers yet.</p>
          ) : (
            <ul className="mt-3 space-y-3">
              {items.map((t) => {
                const hasSecret = Boolean((t.config as Record<string, unknown>)?.secret);
                const cronExpr = (t.config as Record<string, unknown>)?.cron as string | undefined;
                return (
                  <li
                    key={t.id}
                    className="rounded-md border border-slate-200 bg-white p-4"
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <span
                        className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                          t.enabled
                            ? 'bg-emerald-100 text-emerald-700'
                            : 'bg-slate-100 text-slate-500'
                        }`}
                      >
                        {t.enabled ? 'Enabled' : 'Disabled'}
                      </span>
                      <span className="rounded-md bg-indigo-50 px-2 py-0.5 text-xs font-semibold uppercase tracking-wide text-indigo-700">
                        {t.kind}
                      </span>
                      <span className="font-mono text-xs text-slate-500">{t.id}</span>
                    </div>

                    {t.kind === 'cron' && (
                      <div className="mt-3">
                        <div className="text-xs text-slate-500">Cron expression</div>
                        <div className="font-mono text-sm text-slate-800">{cronExpr || '—'}</div>
                      </div>
                    )}

                    {t.kind === 'webhook' && (
                      <div className="mt-3 space-y-2">
                        <div className="text-xs text-slate-500">Webhook URL</div>
                        <div className="flex flex-wrap items-center gap-2">
                          <code className="break-all rounded bg-slate-50 px-2 py-1 font-mono text-xs text-slate-700">
                            {t.webhook_url || '—'}
                          </code>
                          {t.webhook_url && (
                            <button
                              type="button"
                              onClick={() => copyUrl(t.id, t.webhook_url!)}
                              className="rounded-md border border-slate-300 px-2 py-1 text-xs text-slate-700 hover:bg-slate-50"
                            >
                              {copiedId === t.id ? 'Copied!' : 'Copy'}
                            </button>
                          )}
                        </div>
                        <div className="text-xs text-slate-500">
                          Secret: {hasSecret ? 'configured' : 'no secret'}
                        </div>
                      </div>
                    )}

                    <div className="mt-4 flex gap-2">
                      <button
                        type="button"
                        onClick={() => toggle(t.id, !t.enabled)}
                        className="rounded-md border border-slate-300 px-3 py-1.5 text-sm text-slate-700 hover:bg-slate-50"
                      >
                        {t.enabled ? 'Disable' : 'Enable'}
                      </button>
                      <button
                        type="button"
                        onClick={() => {
                          if (confirm(`Delete this ${t.kind} trigger?`)) remove(t.id);
                        }}
                        className="rounded-md border border-red-200 px-3 py-1.5 text-sm text-red-600 hover:bg-red-50"
                      >
                        Delete
                      </button>
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </section>
      </main>
    </div>
  );
}
