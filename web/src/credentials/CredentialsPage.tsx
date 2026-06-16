import { useEffect, useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { useCredentialsStore } from './credentialsStore';
import { useAuth } from '../auth/useAuth';
import { CREDENTIAL_TYPES, fieldsForType } from './credentialTypes';
import LogSearchBar from '../search/LogSearchBar';

export default function CredentialsPage() {
  const { items, fetch, create, remove, error } = useCredentialsStore();
  const { logout, workspace } = useAuth();
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
    <div className="flex h-full bg-slate-50">
      <aside className="flex h-full w-64 flex-col border-r border-slate-200 bg-white p-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">Account</h2>
        <Link to="/workflows" className="mt-3 rounded-md px-3 py-2 text-sm text-slate-700 hover:bg-slate-50">← Back to workflows</Link>
        <Link to="/credentials" className="rounded-md bg-indigo-50 px-3 py-2 text-sm font-medium text-indigo-700">Credentials</Link>
        <div className="mt-auto space-y-2">
          <LogSearchBar workspaceId={workspace?.id ?? null} />
          <button onClick={logout} className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm text-slate-600 hover:bg-slate-50">
            Sign out
          </button>
        </div>
      </aside>
      <main className="flex-1 overflow-y-auto p-8">
        <h1 className="text-2xl font-semibold text-slate-800">Credentials</h1>
        <p className="mt-1 text-sm text-slate-500">
          API keys you've added are encrypted at rest with AES-256-GCM and never shown again after creation.
        </p>

        <form onSubmit={onSubmit} className="mt-6 max-w-lg space-y-3 rounded-md border border-slate-200 bg-white p-4">
          <h2 className="text-sm font-semibold text-slate-700">Add a credential</h2>
          <label className="block">
            <span className="text-sm font-medium text-slate-700">Name</span>
            <input
              type="text" required value={name} onChange={(e) => setName(e.target.value)}
              placeholder="e.g. openai_default"
              className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm"
            />
          </label>
          <label className="block">
            <span className="text-sm font-medium text-slate-700">Type</span>
            <select
              value={type} onChange={(e) => setType(e.target.value)}
              className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm"
            >
              {CREDENTIAL_TYPES.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
            </select>
          </label>
          {fieldsForType(type).map((f) => (
            <label key={f.key} className="block">
              <span className="text-sm font-medium text-slate-700">{f.label}</span>
              <input
                type={f.type}
                required
                value={fields[f.key] || ''}
                onChange={(e) => setFields((prev) => ({ ...prev, [f.key]: e.target.value }))}
                placeholder={f.placeholder}
                className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 font-mono text-sm"
              />
            </label>
          ))}
          {formError && <p className="text-sm text-red-600">{formError}</p>}
          <button
            type="submit" disabled={submitting}
            className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
          >
            {submitting ? 'Saving…' : 'Save credential'}
          </button>
        </form>

        <section className="mt-8 max-w-3xl">
          <h2 className="text-sm font-semibold text-slate-700">Saved credentials</h2>
          {error && <p className="mt-2 text-sm text-red-600">{error}</p>}
          {items.length === 0 ? (
            <p className="mt-2 text-sm text-slate-400">No credentials yet.</p>
          ) : (
            <table className="mt-3 w-full divide-y divide-slate-200 rounded-md border border-slate-200 bg-white text-sm">
              <thead className="bg-slate-50">
                <tr>
                  <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">Name</th>
                  <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">Type</th>
                  <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">Created</th>
                  <th className="px-3 py-2"></th>
                </tr>
              </thead>
              <tbody>
                {items.map((c) => (
                  <tr key={c.id} className="border-t border-slate-100">
                    <td className="px-3 py-2 font-mono text-slate-700">{c.name}</td>
                    <td className="px-3 py-2 text-slate-600">{c.type}</td>
                    <td className="px-3 py-2 text-slate-500">{new Date(c.created_at).toLocaleString()}</td>
                    <td className="px-3 py-2 text-right">
                      <button
                        onClick={() => { if (confirm(`Delete credential "${c.name}"?`)) remove(c.id); }}
                        className="text-sm text-red-600 hover:underline"
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      </main>
    </div>
  );
}
