import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import type { SearchHit } from '../api/types';

interface LogSearchBarProps {
  workspaceId: string | null;
}

// LogSearchBar is a workspace-scoped run-log search input. Typing fires a
// 500ms-debounced request against /v1/workspaces/{wsID}/runs/search; the
// dropdown shows up to 20 hits with ts_headline-rendered snippets where
// matched terms are wrapped in «…» on the server.
export default function LogSearchBar({ workspaceId }: LogSearchBarProps) {
  const [q, setQ] = useState('');
  const [hits, setHits] = useState<SearchHit[]>([]);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();
  const timerRef = useRef<number | undefined>(undefined);
  const rootRef = useRef<HTMLDivElement | null>(null);

  // Debounced search on q change. The backend rejects queries under 3 chars
  // with 400, so we short-circuit client-side to avoid the wasted round-trip.
  useEffect(() => {
    if (timerRef.current) window.clearTimeout(timerRef.current);
    if (!workspaceId || q.trim().length < 3) {
      setHits([]);
      setError(null);
      return;
    }
    timerRef.current = window.setTimeout(async () => {
      setLoading(true);
      setError(null);
      try {
        const res = await api.searchRunLogs(workspaceId, q.trim(), 20);
        setHits(res.hits ?? []);
        setOpen(true);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
        setHits([]);
        setOpen(true);
      } finally {
        setLoading(false);
      }
    }, 500);
    return () => {
      if (timerRef.current) window.clearTimeout(timerRef.current);
    };
  }, [q, workspaceId]);

  // Close dropdown on outside click so the menu doesn't trap focus when
  // the user moves on to other parts of the page.
  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', onClick);
    return () => document.removeEventListener('mousedown', onClick);
  }, []);

  function onHit(hit: SearchHit) {
    setOpen(false);
    setQ('');
    navigate(`/runs/${hit.run_id}`);
  }

  function onKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Escape') {
      setOpen(false);
    }
  }

  if (!workspaceId) return null;

  return (
    <div ref={rootRef} className="relative w-full">
      <input
        type="search"
        value={q}
        onChange={(e) => setQ(e.target.value)}
        onFocus={() => { if (hits.length > 0) setOpen(true); }}
        onKeyDown={onKeyDown}
        placeholder="Search logs…"
        aria-label="Search run logs"
        className="w-full rounded border border-slate-300 px-3 py-1.5 text-sm focus:border-indigo-500 focus:outline-none"
      />
      {open && (
        <div className="absolute left-0 right-0 z-30 mt-1 max-h-96 overflow-auto rounded border border-slate-200 bg-white shadow-lg">
          {loading && (
            <div className="px-3 py-2 text-sm text-slate-500">Searching…</div>
          )}
          {error && (
            <div className="px-3 py-2 text-sm text-red-600">{error}</div>
          )}
          {!loading && !error && hits.length === 0 && (
            <div className="px-3 py-2 text-sm text-slate-500">No matches.</div>
          )}
          {hits.map((h, i) => (
            <button
              key={`${h.run_id}-${h.at}-${i}`}
              onClick={() => onHit(h)}
              className="block w-full px-3 py-2 text-left hover:bg-indigo-50"
              type="button"
            >
              <div className="text-xs text-slate-400">
                {new Date(h.at).toLocaleString()} · run {h.run_id.slice(0, 12)}
              </div>
              <div
                className="text-sm text-slate-700"
                // The snippet is server-controlled but may still contain
                // user-supplied log text. We escape HTML defensively and
                // only re-allow our own «…» → <mark> substitution.
                dangerouslySetInnerHTML={{ __html: renderSnippet(h.snippet) }}
              />
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

function renderSnippet(snippet: string): string {
  const escaped = snippet
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
  return escaped.replaceAll('«', '<mark>').replaceAll('»', '</mark>');
}
