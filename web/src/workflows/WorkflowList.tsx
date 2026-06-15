import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useWorkflowsStore } from './workflowsStore';
import { useAuth } from '../auth/useAuth';

export default function WorkflowList() {
  const navigate = useNavigate();
  const { id: currentId } = useParams<{ id: string }>();
  const { list, createEmpty } = useWorkflowsStore();
  const { user, logout } = useAuth();
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
    <aside className="flex h-full w-64 flex-col border-r border-slate-200 bg-white">
      <div className="border-b border-slate-200 px-4 py-3">
        <div className="text-xs uppercase tracking-wide text-slate-500">Signed in as</div>
        <div className="truncate text-sm font-medium text-slate-700">{user?.email}</div>
      </div>
      <div className="flex-1 overflow-y-auto">
        <div className="border-b border-slate-100 px-4 py-2 text-xs font-semibold uppercase tracking-wide text-slate-500">
          Workflows
        </div>
        {list.length === 0 && (
          <p className="px-4 py-3 text-sm text-slate-400">No workflows yet.</p>
        )}
        <ul>
          {list.map((wf) => (
            <li key={wf.id}>
              <button
                onClick={() => navigate(`/workflows/${wf.id}`)}
                className={`block w-full truncate px-4 py-2 text-left text-sm hover:bg-slate-50 ${
                  wf.id === currentId ? 'bg-indigo-50 text-indigo-700' : 'text-slate-700'
                }`}
              >
                {wf.name}
              </button>
            </li>
          ))}
        </ul>
      </div>
      <div className="border-t border-slate-200 p-3 space-y-2">
        {creating ? (
          <div className="space-y-2">
            <input
              type="text" value={name} placeholder="New workflow name"
              onChange={(e) => setName(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter') onCreate(); }}
              className="block w-full rounded-md border border-slate-300 px-2 py-1 text-sm"
              autoFocus
            />
            <div className="flex gap-2">
              <button onClick={onCreate} className="flex-1 rounded bg-indigo-600 px-2 py-1 text-sm font-medium text-white hover:bg-indigo-700">
                Create
              </button>
              <button onClick={() => { setCreating(false); setName(''); }} className="rounded border border-slate-300 px-2 py-1 text-sm text-slate-700 hover:bg-slate-50">
                Cancel
              </button>
            </div>
          </div>
        ) : (
          <button
            onClick={() => setCreating(true)}
            className="w-full rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700"
          >
            + New workflow
          </button>
        )}
        <button
          onClick={logout}
          className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm text-slate-600 hover:bg-slate-50"
        >
          Sign out
        </button>
      </div>
    </aside>
  );
}
