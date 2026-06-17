import { useEffect, useRef, useState, type FormEvent } from 'react';
import { useChatStore } from './chatStore';
import Icon from '../ui/Icon';

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
    <aside className="flex h-full w-[26rem] flex-col border-l border-ink-500 bg-ink-700">
      <div className="border-b border-ink-500 px-5 py-3">
        <div className="flex items-center gap-2 font-mono text-[10px] uppercase tracking-[0.36em] text-cyan">
          <Icon name="signal" size={12} />
          margin notes · assistant
        </div>
        <p className="mt-1 font-mono text-[11px] text-paper-400">
          Speak the draft you want — the assistant marks it on the sheet.
        </p>
      </div>
      <div ref={listRef} className="flex-1 space-y-4 overflow-y-auto px-5 py-4">
        {messages.length === 0 && (
          <p className="font-mono text-xs text-paper-400">
            <span className="text-cyan">›</span> describe the workflow. e.g. <span className="text-paper-200">"every monday at 9am, summarize new github stars into slack"</span>
          </p>
        )}
        {messages.map((m) => (
          <div key={m.id} className={m.role === 'user' ? 'flex justify-end' : 'flex justify-start'}>
            <div
              className={
                m.role === 'user'
                  ? 'max-w-[85%] border-l-2 border-cyan bg-cyan/10 px-3 py-2 font-mono text-[12px] leading-5 text-paper-50'
                  : 'max-w-[92%] border-l-2 border-ink-500 bg-ink-600/60 px-3 py-2 font-mono text-[12px] leading-5 text-paper-200'
              }
            >
              <div className="mb-1 font-mono text-[9px] uppercase tracking-[0.32em] text-paper-400">
                {m.role === 'user' ? 'operator' : 'assistant'}
              </div>
              <div className="whitespace-pre-wrap">{m.content}</div>
              {m.proposal !== undefined && (
                <div className="mt-2 flex items-center gap-2 border border-cyan/40 bg-cyan/5 px-2 py-1 font-mono text-[10px] uppercase tracking-[0.24em] text-cyan">
                  <Icon name="check" size={10} /> proposal applied
                </div>
              )}
              {m.patch !== undefined && (
                <div className="mt-2 flex items-center gap-2 border border-cyan/40 bg-cyan/5 px-2 py-1 font-mono text-[10px] uppercase tracking-[0.24em] text-cyan">
                  <Icon name="check" size={10} /> patch applied
                </div>
              )}
            </div>
          </div>
        ))}
        {sending && (
          <div className="flex items-center gap-2 font-mono text-[11px] uppercase tracking-[0.28em] text-paper-400">
            <Icon name="spinner" size={12} className="animate-spin text-cyan" />
            drafting…
          </div>
        )}
      </div>
      <form onSubmit={onSubmit} className="border-t border-ink-500 p-4">
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
          className="block w-full resize-none rounded-sharp border border-ink-500 bg-ink-800 px-3 py-2 font-mono text-xs text-paper-50 placeholder:text-paper-600 focus:border-cyan focus:outline-none"
        />
        <div className="mt-2 flex items-center justify-between">
          <span className="font-mono text-[10px] uppercase tracking-[0.28em] text-paper-600">
            ⇧⏎ newline · ⏎ send
          </span>
          <button
            type="submit"
            disabled={!input.trim() || sending}
            className="flex items-center gap-2 rounded-sharp border border-cyan/40 bg-cyan/10 px-3 py-1.5 font-mono text-[11px] uppercase tracking-[0.24em] text-cyan transition hover:bg-cyan/20 disabled:opacity-40"
          >
            send
            <Icon name="arrow-right" size={12} />
          </button>
        </div>
      </form>
    </aside>
  );
}
