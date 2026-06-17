import { useEffect, useState, type FormEvent } from 'react';
import { useParams } from 'react-router-dom';
import { useTriggersStore } from './triggersStore';
import { useWorkflowsStore } from '../workflows/workflowsStore';
import { SectionSidebar, PageHeader } from '../ui/SectionShell';
import Icon from '../ui/Icon';

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

  const workflowName = current && current.id === workflowId ? current.name : null;

  return (
    <div className="flex h-full bg-ink-900">
      <SectionSidebar workflowId={workflowId} workflowName={workflowName} active="triggers" />

      <main className="flex-1 overflow-y-auto">
        <PageHeader
          eyebrow="firing pins · trigger registry"
          title="Triggers"
          description="Drive this sheet on a schedule, or expose a webhook URL that fires it when called."
          meta={workflowId && (
            <span className="pill pill-cyan">
              <Icon name="dot" size={6} />
              {workflowName || workflowId}
            </span>
          )}
        />

        <div className="px-8 py-6">
          <div className="grid max-w-5xl gap-5 md:grid-cols-2">
            <form onSubmit={onSubmitCron} className="corners relative space-y-4 bg-ink-700/60 p-5">
              <span className="corner-bl" />
              <span className="corner-br" />
              <div className="flex items-center gap-2 border-b border-ink-500 pb-3 font-mono text-[10px] uppercase tracking-[0.32em] text-cyan">
                <Icon name="clock" size={12} />
                add cron trigger
              </div>
              <label className="block">
                <span className="font-mono text-[10px] uppercase tracking-[0.28em] text-paper-400">preset</span>
                <select
                  value={preset}
                  onChange={(e) => {
                    const v = e.target.value;
                    setPreset(v);
                    if (v) setCron(v);
                  }}
                  className="mt-1.5 block w-full rounded-sharp border border-ink-500 bg-ink-800 px-3 py-2 font-mono text-sm text-paper-50 focus:border-cyan focus:outline-none"
                >
                  {CRON_PRESETS.map((p) => (
                    <option key={p.value || 'none'} value={p.value}>{p.label}</option>
                  ))}
                </select>
              </label>
              <label className="block">
                <span className="font-mono text-[10px] uppercase tracking-[0.28em] text-paper-400">cron expression</span>
                <input
                  type="text"
                  value={cron}
                  onChange={(e) => setCron(e.target.value)}
                  placeholder="@daily, 0 9 * * *, @every 5m"
                  className="mt-1.5 block w-full rounded-sharp border border-ink-500 bg-ink-800 px-3 py-2 font-mono text-sm text-paper-50 placeholder:text-paper-600 focus:border-cyan focus:outline-none"
                />
              </label>
              {cronError && (
                <p className="flex items-center gap-2 font-mono text-xs text-rose">
                  <Icon name="x" size={12} /> {cronError}
                </p>
              )}
              <button
                type="submit"
                disabled={cronSubmitting}
                className="flex items-center gap-2 rounded-sharp border border-cyan/40 bg-cyan/10 px-4 py-2 font-mono text-[11px] uppercase tracking-[0.24em] text-cyan transition hover:bg-cyan/20 disabled:opacity-50"
              >
                {cronSubmitting ? <Icon name="spinner" size={12} className="animate-spin" /> : <Icon name="plus" size={12} />}
                {cronSubmitting ? 'Adding…' : 'Add cron trigger'}
              </button>
            </form>

            <form onSubmit={onSubmitWebhook} className="corners relative space-y-4 bg-ink-700/60 p-5">
              <span className="corner-bl" />
              <span className="corner-br" />
              <div className="flex items-center gap-2 border-b border-ink-500 pb-3 font-mono text-[10px] uppercase tracking-[0.32em] text-cyan">
                <Icon name="webhook" size={12} />
                add webhook trigger
              </div>
              <label className="block">
                <span className="font-mono text-[10px] uppercase tracking-[0.28em] text-paper-400">hmac secret · optional</span>
                <input
                  type="password"
                  value={webhookSecret}
                  onChange={(e) => setWebhookSecret(e.target.value)}
                  placeholder="leave blank for no signature check"
                  autoComplete="new-password"
                  className="mt-1.5 block w-full rounded-sharp border border-ink-500 bg-ink-800 px-3 py-2 font-mono text-sm text-paper-50 placeholder:text-paper-600 focus:border-cyan focus:outline-none"
                />
              </label>
              <p className="font-mono text-[11px] text-paper-400">
                When set, incoming requests must include an{' '}
                <code className="border border-ink-500 bg-ink-800 px-1.5 py-0.5 text-cyan">X-Flowgent-Signature</code>{' '}
                HMAC-SHA256 header.
              </p>
              {webhookError && (
                <p className="flex items-center gap-2 font-mono text-xs text-rose">
                  <Icon name="x" size={12} /> {webhookError}
                </p>
              )}
              <button
                type="submit"
                disabled={webhookSubmitting}
                className="flex items-center gap-2 rounded-sharp border border-cyan/40 bg-cyan/10 px-4 py-2 font-mono text-[11px] uppercase tracking-[0.24em] text-cyan transition hover:bg-cyan/20 disabled:opacity-50"
              >
                {webhookSubmitting ? <Icon name="spinner" size={12} className="animate-spin" /> : <Icon name="plus" size={12} />}
                {webhookSubmitting ? 'Adding…' : 'Add webhook trigger'}
              </button>
            </form>
          </div>

          <section className="mt-10 max-w-5xl">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="font-mono text-[10px] uppercase tracking-[0.32em] text-paper-400">existing triggers</h2>
              <span className="font-mono text-[10px] text-paper-600">{items.length.toString().padStart(2, '0')} registered</span>
            </div>
            {error && (
              <p className="mt-2 flex items-center gap-2 font-mono text-xs text-rose">
                <Icon name="x" size={12} /> {error}
              </p>
            )}
            {items.length === 0 ? (
              <p className="border border-dashed border-ink-500 bg-ink-700/30 px-5 py-10 text-center font-mono text-xs uppercase tracking-[0.28em] text-paper-400">
                no triggers yet.
              </p>
            ) : (
              <ul className="space-y-3">
                {items.map((t) => {
                  const hasSecret = Boolean((t.config as Record<string, unknown>)?.secret);
                  const cronExpr = (t.config as Record<string, unknown>)?.cron as string | undefined;
                  return (
                    <li
                      key={t.id}
                      className="corners relative bg-ink-700/60 p-4"
                    >
                      <span className="corner-bl" />
                      <span className="corner-br" />
                      <div className="flex flex-wrap items-center gap-2">
                        <span className={`pill ${t.enabled ? 'pill-moss' : ''}`}>
                          <Icon name="dot" size={6} />
                          {t.enabled ? 'enabled' : 'disabled'}
                        </span>
                        <span className="pill pill-cyan">
                          {t.kind === 'cron' ? <Icon name="clock" size={10} /> : <Icon name="webhook" size={10} />}
                          {t.kind}
                        </span>
                        <span className="ml-auto font-mono text-[11px] text-paper-600">{t.id}</span>
                      </div>

                      {t.kind === 'cron' && (
                        <div className="mt-3">
                          <div className="font-mono text-[10px] uppercase tracking-[0.28em] text-paper-400">cron expression</div>
                          <div className="mt-1 font-mono text-sm text-paper-50">{cronExpr || '—'}</div>
                        </div>
                      )}

                      {t.kind === 'webhook' && (
                        <div className="mt-3 space-y-2">
                          <div className="font-mono text-[10px] uppercase tracking-[0.28em] text-paper-400">webhook url</div>
                          <div className="flex flex-wrap items-center gap-2">
                            <code className="break-all border border-ink-500 bg-ink-800 px-2 py-1 font-mono text-xs text-cyan">
                              {t.webhook_url || '—'}
                            </code>
                            {t.webhook_url && (
                              <button
                                type="button"
                                onClick={() => copyUrl(t.id, t.webhook_url!)}
                                className="flex items-center gap-1.5 rounded-sharp border border-ink-500 bg-ink-700 px-2 py-1 font-mono text-[11px] uppercase tracking-[0.24em] text-paper-200 transition hover:border-cyan/40 hover:text-cyan"
                              >
                                <Icon name="copy" size={12} />
                                {copiedId === t.id ? 'Copied!' : 'Copy'}
                              </button>
                            )}
                          </div>
                          <div className="font-mono text-[11px] text-paper-400">
                            secret · <span className={hasSecret ? 'text-moss' : 'text-paper-600'}>{hasSecret ? 'configured' : 'none'}</span>
                          </div>
                        </div>
                      )}

                      <div className="mt-4 flex gap-2 border-t border-ink-500 pt-3">
                        <button
                          type="button"
                          onClick={() => toggle(t.id, !t.enabled)}
                          className="rounded-sharp border border-ink-500 bg-ink-700 px-3 py-1.5 font-mono text-[11px] uppercase tracking-[0.24em] text-paper-200 transition hover:border-cyan/40 hover:text-cyan"
                        >
                          {t.enabled ? 'Disable' : 'Enable'}
                        </button>
                        <button
                          type="button"
                          onClick={() => {
                            if (confirm(`Delete this ${t.kind} trigger?`)) remove(t.id);
                          }}
                          className="rounded-sharp border border-rose/40 px-3 py-1.5 font-mono text-[11px] uppercase tracking-[0.24em] text-rose transition hover:bg-rose/10"
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
        </div>
      </main>
    </div>
  );
}
