import { useEffect, useState, type FormEvent } from 'react';
import { useCredentialsStore } from './credentialsStore';
import { CREDENTIAL_TYPES, fieldsForType } from './credentialTypes';
import { SectionSidebar, PageHeader } from '../ui/SectionShell';
import Icon from '../ui/Icon';

export default function CredentialsPage() {
  const { items, fetch, create, remove, error } = useCredentialsStore();
  const [name, setName] = useState('');
  const [type, setType] = useState(CREDENTIAL_TYPES[0].value);
  const [fields, setFields] = useState<Record<string, string>>(() => {
    const init: Record<string, string> = {};
    for (const f of fieldsForType(CREDENTIAL_TYPES[0].value)) init[f.key] = '';
    return init;
  });
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  useEffect(() => { fetch(); }, [fetch]);

  useEffect(() => {
    const next: Record<string, string> = {};
    for (const f of fieldsForType(type)) next[f.key] = '';
    setFields(next);
    setFormError(null);
  }, [type]);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!name.trim()) {
      setFormError('Name is required');
      return;
    }
    const required = fieldsForType(type);
    const secret: Record<string, string> = {};
    for (const f of required) {
      const v = (fields[f.key] || '').trim();
      if (!v) {
        setFormError(`${f.label} is required`);
        return;
      }
      secret[f.key] = v;
    }
    setSubmitting(true); setFormError(null);
    try {
      await create(name.trim(), type, secret);
      setName('');
      const cleared: Record<string, string> = {};
      for (const f of required) cleared[f.key] = '';
      setFields(cleared);
    } catch (err) {
      setFormError(String(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="flex h-full bg-ink-900">
      <SectionSidebar active="credentials" />

      <main className="flex-1 overflow-y-auto">
        <PageHeader
          eyebrow="vault · operator credentials"
          title="Credentials"
          description="Keys, tokens and DSNs are encrypted at rest with AES-256-GCM. Once saved, secret material never leaves the server."
        />

        <div className="px-8 py-6">
          <form onSubmit={onSubmit} className="corners relative max-w-lg space-y-4 bg-ink-700/60 p-5">
            <span className="corner-bl" />
            <span className="corner-br" />
            <div className="flex items-center gap-2 border-b border-ink-500 pb-3 font-mono text-[10px] uppercase tracking-[0.32em] text-cyan">
              <Icon name="key" size={12} />
              Add a credential
            </div>
            <label className="block">
              <span className="font-mono text-[10px] uppercase tracking-[0.28em] text-paper-400">Name</span>
              <input
                type="text" required value={name} onChange={(e) => setName(e.target.value)}
                placeholder="e.g. openai_default"
                className="mt-1.5 block w-full rounded-sharp border border-ink-500 bg-ink-800 px-3 py-2 font-mono text-sm text-paper-50 placeholder:text-paper-600 focus:border-cyan focus:outline-none"
              />
            </label>
            <label className="block">
              <span className="font-mono text-[10px] uppercase tracking-[0.28em] text-paper-400">Type</span>
              <select
                value={type} onChange={(e) => setType(e.target.value)}
                className="mt-1.5 block w-full rounded-sharp border border-ink-500 bg-ink-800 px-3 py-2 font-mono text-sm text-paper-50 focus:border-cyan focus:outline-none"
              >
                {CREDENTIAL_TYPES.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
              </select>
            </label>
            {fieldsForType(type).map((f) => (
              <label key={f.key} className="block">
                <span className="font-mono text-[10px] uppercase tracking-[0.28em] text-paper-400">{f.label}</span>
                <input
                  type={f.type}
                  required
                  value={fields[f.key] || ''}
                  onChange={(e) => setFields((prev) => ({ ...prev, [f.key]: e.target.value }))}
                  placeholder={f.placeholder}
                  className="mt-1.5 block w-full rounded-sharp border border-ink-500 bg-ink-800 px-3 py-2 font-mono text-sm text-paper-50 placeholder:text-paper-600 focus:border-cyan focus:outline-none"
                />
              </label>
            ))}
            {formError && (
              <p className="flex items-center gap-2 font-mono text-xs text-rose">
                <Icon name="x" size={12} /> {formError}
              </p>
            )}
            <button
              type="submit" disabled={submitting}
              className="flex items-center gap-2 rounded-sharp border border-cyan/40 bg-cyan/10 px-4 py-2 font-mono text-[11px] uppercase tracking-[0.24em] text-cyan transition hover:bg-cyan/20 disabled:opacity-50"
            >
              {submitting ? <Icon name="spinner" size={12} className="animate-spin" /> : <Icon name="plus" size={12} />}
              {submitting ? 'Saving…' : 'Save credential'}
            </button>
          </form>

          <section className="mt-10 max-w-3xl">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="font-mono text-[10px] uppercase tracking-[0.32em] text-paper-400">Saved credentials</h2>
              <span className="font-mono text-[10px] text-paper-600">{items.length.toString().padStart(2, '0')} sealed</span>
            </div>
            {error && (
              <p className="mt-2 flex items-center gap-2 font-mono text-xs text-rose">
                <Icon name="x" size={12} /> {error}
              </p>
            )}
            {items.length === 0 ? (
              <p className="border border-dashed border-ink-500 bg-ink-700/30 px-5 py-10 text-center font-mono text-xs uppercase tracking-[0.28em] text-paper-400">
                No credentials yet.
              </p>
            ) : (
              <div className="corners relative overflow-hidden border border-ink-500 bg-ink-700/40">
                <span className="corner-bl" />
                <span className="corner-br" />
                <table className="w-full">
                  <thead className="border-b border-ink-500 bg-ink-700/80">
                    <tr>
                      {['name', 'type', 'created', ''].map((h, i) => (
                        <th key={i} className="px-3 py-2 text-left font-mono text-[10px] uppercase tracking-[0.28em] text-paper-400">{h}</th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {items.map((c) => (
                      <tr key={c.id} className="border-b border-ink-500/60">
                        <td className="px-3 py-2.5 font-mono text-sm text-paper-50">{c.name}</td>
                        <td className="px-3 py-2.5 font-mono text-xs text-paper-200">{c.type}</td>
                        <td className="px-3 py-2.5 font-mono text-xs text-paper-400 tabular-nums">{new Date(c.created_at).toLocaleString()}</td>
                        <td className="px-3 py-2.5 text-right">
                          <button
                            onClick={() => { if (confirm(`Delete credential "${c.name}"?`)) remove(c.id); }}
                            className="font-mono text-xs uppercase tracking-[0.24em] text-rose hover:underline"
                          >
                            Delete
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        </div>
      </main>
    </div>
  );
}
