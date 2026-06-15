import { useEffect, useRef, useState, type FormEvent } from 'react';
import { useChatStore } from './chatStore';

interface Props {
  workflowId: string;
  onSend?: (message: string) => void;
}

export default function ChatPanel({ workflowId, onSend }: Props) {
  const { messages, sending, reset } = useChatStore();
  const [input, setInput] = useState('');
  const listRef = useRef<HTMLDivElement>(null);

  useEffect(() => { reset(); }, [workflowId, reset]);
  useEffect(() => {
    listRef.current?.scrollTo({ top: listRef.current.scrollHeight, behavior: 'smooth' });
  }, [messages]);

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!input.trim() || sending) return;
    const text = input.trim();
    setInput('');
    onSend?.(text);
  }

  return (
    <aside className="flex h-full w-96 flex-col border-l border-slate-200 bg-white">
      <div className="border-b border-slate-200 px-4 py-3 text-sm font-semibold text-slate-700">
        Chat
      </div>
      <div ref={listRef} className="flex-1 space-y-3 overflow-y-auto px-4 py-3">
        {messages.length === 0 && (
          <p className="text-sm text-slate-400">Describe what you want — the assistant will propose a workflow.</p>
        )}
        {messages.map((m) => (
          <div key={m.id} className={m.role === 'user' ? 'flex justify-end' : 'flex justify-start'}>
            <div
              className={
                m.role === 'user'
                  ? 'max-w-[80%] rounded-lg bg-indigo-600 px-3 py-2 text-sm text-white'
                  : 'max-w-[90%] rounded-lg bg-slate-100 px-3 py-2 text-sm text-slate-800'
              }
            >
              <div className="whitespace-pre-wrap">{m.content}</div>
              {m.proposal !== undefined && (
                <div className="mt-2 rounded border border-slate-300 bg-white px-2 py-1 text-xs text-slate-600">
                  📋 Proposal applied to canvas
                </div>
              )}
              {m.patch !== undefined && (
                <div className="mt-2 rounded border border-slate-300 bg-white px-2 py-1 text-xs text-slate-600">
                  ✏ Patch applied to canvas
                </div>
              )}
            </div>
          </div>
        ))}
        {sending && (
          <p className="text-xs text-slate-400">Thinking…</p>
        )}
      </div>
      <form onSubmit={onSubmit} className="border-t border-slate-200 p-3">
        <textarea
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault();
              onSubmit(e as unknown as FormEvent);
            }
          }}
          placeholder="Tell the assistant what you want…"
          rows={3}
          className="block w-full resize-none rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
        />
        <div className="mt-2 flex justify-end">
          <button
            type="submit"
            disabled={!input.trim() || sending}
            className="rounded-md bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
          >
            Send
          </button>
        </div>
      </form>
    </aside>
  );
}
