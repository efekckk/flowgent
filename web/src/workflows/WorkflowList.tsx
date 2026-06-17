import { useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useWorkflowsStore } from './workflowsStore';
import { useAuth } from '../auth/useAuth';
import LogSearchBar from '../search/LogSearchBar';
import Icon from '../ui/Icon';

export default function WorkflowList() {
  const navigate = useNavigate();
  const { id: currentId } = useParams<{ id: string }>();
  const { list, createEmpty } = useWorkflowsStore();
  const { user, logout, workspace } = useAuth();
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState('');

  useEffect(() => {
    // No list endpoint yet; placeholder.
  }, []);

  async function onCreate() {
    if (!name.trim()) return;
    const wf = await createEmpty(name.trim());
    setCreating(false);
    setName('');
    navigate(`/workflows/${wf.id}`);
  }

  return (
    <aside className="flex h-full w-72 flex-col border-r border-ink-500 bg-ink-700">
      <div className="border-b border-ink-500 px-5 py-4">
        <div className="flex items-baseline justify-between font-mono text-[10px] uppercase tracking-[0.32em] text-paper-400">
          <span>flowgent</span>
          <span className="text-cyan">v0.8</span>
        </div>
        <h1 className="h-display mt-1 text-xl text-paper-50">Drafting room</h1>
      </div>

      <div className="border-b border-ink-500 px-5 py-3">
        <div className="font-mono text-[10px] uppercase tracking-[0.28em] text-paper-600">operator</div>
        <div className="mt-1 truncate font-mono text-xs text-paper-200">{user?.email}</div>
        {workspace && (
          <div className="mt-2 flex items-center justify-between font-mono text-[10px] uppercase tracking-[0.28em] text-paper-600">
            <span>workspace</span>
            <span className="truncate text-paper-400">{workspace.name}</span>
          </div>
        )}
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="flex items-center justify-between px-5 pb-2 pt-4">
          <span className="font-mono text-[10px] uppercase tracking-[0.32em] text-paper-400">
            sheets · workflows
          </span>
          <span className="font-mono text-[10px] text-paper-600">{list.length.toString().padStart(2, '0')}</span>
        </div>
        {list.length === 0 && (
          <p className="px-5 py-3 font-mono text-xs text-paper-600">
            — empty room. draft a sheet to begin.
          </p>
        )}
        <ul className="space-y-px">
          {list.map((wf, idx) => {
            const active = wf.id === currentId;
            return (
              <li key={wf.id}>
                <button
                  onClick={() => navigate(`/workflows/${wf.id}`)}
                  className={`group flex w-full items-center gap-3 border-l-2 px-5 py-2.5 text-left font-mono text-xs transition ${
                    active
                      ? 'border-cyan bg-cyan/5 text-cyan'
                      : 'border-transparent text-paper-200 hover:border-ink-400 hover:bg-ink-600/60 hover:text-paper-50'
                  }`}
                >
                  <span className={`text-[10px] tabular-nums ${active ? 'text-cyan' : 'text-paper-600'}`}>
                    {(idx + 1).toString().padStart(2, '0')}
                  </span>
                  <span className="truncate">{wf.name}</span>
                  {active && <Icon name="caret-right" size={12} className="ml-auto" />}
                </button>
              </li>
            );
          })}
        </ul>
      </div>

      <div className="border-t border-ink-500 px-5 py-4 space-y-3">
        <LogSearchBar workspaceId={workspace?.id ?? null} />

        {creating ? (
          <div className="space-y-2">
            <input
              type="text" value={name} placeholder="new sheet title…"
              onChange={(e) => setName(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter') onCreate(); }}
              className="block w-full rounded-sharp border border-ink-500 bg-ink-800 px-2 py-1.5 font-mono text-xs text-paper-50 placeholder:text-paper-600 focus:border-cyan focus:outline-none"
              autoFocus
            />
            <div className="flex gap-2">
              <button
                onClick={onCreate}
                className="flex-1 rounded-sharp border border-cyan/40 bg-cyan/10 px-2 py-1.5 font-mono text-[10px] uppercase tracking-[0.24em] text-cyan hover:bg-cyan/20"
              >
                draft
              </button>
              <button
                onClick={() => { setCreating(false); setName(''); }}
                className="rounded-sharp border border-ink-500 px-2 py-1.5 font-mono text-[10px] uppercase tracking-[0.24em] text-paper-400 hover:bg-ink-600"
              >
                cancel
              </button>
            </div>
          </div>
        ) : (
          <button
            onClick={() => setCreating(true)}
            className="group flex w-full items-center justify-between rounded-sharp border border-cyan/40 bg-cyan/10 px-3 py-2 font-mono text-[11px] uppercase tracking-[0.24em] text-cyan transition hover:bg-cyan/20"
          >
            <span className="flex items-center gap-2">
              <Icon name="plus" size={12} />
              draft new sheet
            </span>
            <span className="text-paper-400 group-hover:text-cyan">⏎</span>
          </button>
        )}

        <Link
          to="/credentials"
          className="flex w-full items-center gap-2 rounded-sharp border border-ink-500 px-3 py-2 font-mono text-[11px] uppercase tracking-[0.24em] text-paper-200 transition hover:border-ink-400 hover:bg-ink-600"
        >
          <Icon name="key" size={12} className="text-paper-400" />
          credentials
        </Link>
        <button
          onClick={logout}
          className="flex w-full items-center gap-2 rounded-sharp px-3 py-1.5 font-mono text-[10px] uppercase tracking-[0.24em] text-paper-600 transition hover:text-paper-200"
        >
          <Icon name="logout" size={12} />
          sign out
        </button>
      </div>
    </aside>
  );
}
