import { useState, type FormEvent } from 'react';
import { useAuth } from './useAuth';
import { useNavigate, Link } from 'react-router-dom';
import { APIError } from '../api/client';
import Icon from '../ui/Icon';

export default function SignupPage() {
  const { signup } = useAuth();
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (password.length < 8) {
      setError('Password must be at least 8 characters.');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await signup(email, password);
      navigate('/workflows');
    } catch (err) {
      const msg = err instanceof APIError ? err.message : 'Signup failed';
      setError(msg);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="drafting-table relative flex h-full items-center justify-center px-4">
      <div className="pointer-events-none absolute inset-0 [background:radial-gradient(ellipse_at_top,rgba(125,211,252,0.06),transparent_60%)]" />
      <div className="relative w-full max-w-md animate-draft-in">
        <div className="mb-6 flex items-end justify-between">
          <div className="font-mono text-[10px] uppercase tracking-[0.32em] text-paper-400">
            sheet · 02 / a — enroll
          </div>
          <div className="font-mono text-[10px] uppercase tracking-[0.32em] text-paper-400">
            rev. 2026
          </div>
        </div>

        <form onSubmit={onSubmit} className="corners relative bg-ink-700/80 p-8 shadow-callout backdrop-blur-sm">
          <span className="corner-bl" />
          <span className="corner-br" />

          <div className="mb-6 border-b border-ink-500 pb-5">
            <div className="font-mono text-[10px] uppercase tracking-[0.32em] text-cyan">flowgent</div>
            <h1 className="h-display mt-1 text-3xl">New&nbsp;draftsperson</h1>
            <p className="mt-1 font-mono text-xs text-paper-400">
              Register an operator to draft workflows.
            </p>
          </div>

          <div className="space-y-5">
            <label className="block">
              <span className="font-mono text-[10px] uppercase tracking-[0.28em] text-paper-400">
                operator email
              </span>
              <input
                type="email" required value={email} onChange={(e) => setEmail(e.target.value)}
                className="mt-2 block w-full rounded-sharp border border-ink-500 bg-ink-800 px-3 py-2 font-mono text-sm text-paper-50 placeholder:text-paper-600 focus:border-cyan focus:outline-none focus:ring-0"
                placeholder="ada@flowgent.dev"
              />
            </label>
            <label className="block">
              <span className="font-mono text-[10px] uppercase tracking-[0.28em] text-paper-400">
                passphrase · min 8
              </span>
              <input
                type="password" required minLength={8} value={password} onChange={(e) => setPassword(e.target.value)}
                className="mt-2 block w-full rounded-sharp border border-ink-500 bg-ink-800 px-3 py-2 font-mono text-sm text-paper-50 placeholder:text-paper-600 focus:border-cyan focus:outline-none focus:ring-0"
                placeholder="•••••••••••"
              />
            </label>
          </div>

          {error && (
            <div className="mt-5 flex items-start gap-2 border border-rose/40 bg-rose/5 px-3 py-2 font-mono text-xs text-rose">
              <Icon name="x" size={14} className="mt-0.5 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          <button
            type="submit" disabled={submitting}
            className="group mt-6 flex w-full items-center justify-between gap-3 rounded-sharp border border-cyan/40 bg-cyan/10 px-4 py-3 font-mono text-xs uppercase tracking-[0.28em] text-cyan transition hover:bg-cyan/20 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <span className="flex items-center gap-2">
              {submitting && <Icon name="spinner" size={14} className="animate-spin" />}
              {submitting ? 'enrolling' : 'enroll & open sheet'}
            </span>
            <Icon name="arrow-right" size={14} className="transition group-hover:translate-x-0.5" />
          </button>

          <div className="mt-6 flex items-center justify-between font-mono text-[10px] uppercase tracking-[0.28em] text-paper-400">
            <span>already enrolled?</span>
            <Link to="/login" className="text-cyan hover:underline">return to drafting →</Link>
          </div>
        </form>

        <div className="mt-4 flex justify-between font-mono text-[10px] uppercase tracking-[0.32em] text-paper-600">
          <span>argon2id · server-side</span>
          <span>scale 1 : 1</span>
        </div>
      </div>
    </div>
  );
}
