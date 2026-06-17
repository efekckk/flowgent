import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import type { SearchHit } from '../api/types';
import Icon from '../ui/Icon';

interface LogSearchBarProps {
  workspaceId: string | null;
}

export default function LogSearchBar({ workspaceId }: LogSearchBarProps) {
  const [q, setQ] = useState('');
  const [hits, setHits] = useState<SearchHit[]>([]);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();
  const timerRef = useRef<number | undefined>(undefined);
  const rootRef = useRef<HTMLDivElement | null>(null);

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
      <div className="relative">
        <Icon name="search" size={12} className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-paper-400" />
        <input
          type="search"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          onFocus={() => { if (hits.length > 0) setOpen(true); }}
          onKeyDown={onKeyDown}
          placeholder="Search logs…"
          aria-label="Search run logs"
          className="w-full rounded-sharp border border-ink-500 bg-ink-800 px-3 py-1.5 pl-8 font-mono text-xs text-paper-50 placeholder:text-paper-600 focus:border-cyan focus:outline-none"
        />
      </div>
      {open && (
        <div className="corners absolute left-0 right-0 z-30 mt-1 max-h-96 overflow-auto border border-ink-500 bg-ink-700/95 shadow-callout backdrop-blur-sm">
          <span className="corner-bl" />
          <span className="corner-br" />
          {loading && (
            <div className="flex items-center gap-2 px-3 py-2 font-mono text-[11px] uppercase tracking-[0.24em] text-paper-400">
              <Icon name="spinner" size={12} className="animate-spin text-cyan" />
              searching…
            </div>
          )}
          {error && (
            <div className="px-3 py-2 font-mono text-[11px] text-rose">{error}</div>
          )}
          {!loading && !error && hits.length === 0 && (
            <div className="px-3 py-2 font-mono text-[11px] uppercase tracking-[0.24em] text-paper-400">No matches.</div>
          )}
          {hits.map((h, i) => (
            <button
              key={`${h.run_id}-${h.at}-${i}`}
              onClick={() => onHit(h)}
              className="block w-full border-b border-ink-500/40 px-3 py-2 text-left transition hover:bg-cyan/5"
              type="button"
            >
              <div className="font-mono text-[10px] uppercase tracking-[0.24em] text-paper-400">
                {new Date(h.at).toLocaleString()} · run <span className="text-cyan">{h.run_id.slice(0, 12)}</span>
              </div>
              <div
                className="mt-0.5 font-mono text-[11px] text-paper-200 [&_mark]:bg-cyan/25 [&_mark]:text-cyan [&_mark]:px-0.5"
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
