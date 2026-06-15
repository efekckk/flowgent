import { useCallback } from 'react';
import { useChatStore, type ChatMessage } from './chatStore';
import type { SSEEvent, WorkflowDefinition } from '../api/types';

function uid(): string {
  return Math.random().toString(36).slice(2, 12);
}

export function useChat(workflowId: string, onUpdateDefinition: (def: WorkflowDefinition) => void) {
  const { append, setSending } = useChatStore();

  const send = useCallback(async (message: string) => {
    append({ id: uid(), role: 'user', content: message });
    setSending(true);

    let assistantContent = '';
    let proposal: unknown = undefined;
    let patch: unknown = undefined;
    const assistantId = uid();

    const res = await fetch(`/v1/workflows/${workflowId}/chat`, {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ message, model: 'gpt-4o' }),
    });
    if (!res.ok || !res.body) {
      append({ id: assistantId, role: 'assistant', content: `Error: ${res.statusText}` });
      setSending(false);
      return;
    }

    const reader = res.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      let nl: number;
      while ((nl = buffer.indexOf('\n\n')) !== -1) {
        const chunk = buffer.slice(0, nl);
        buffer = buffer.slice(nl + 2);
        const lines = chunk.split('\n');
        for (const line of lines) {
          if (!line.startsWith('data: ')) continue;
          try {
            const ev = JSON.parse(line.slice(6)) as SSEEvent;
            switch (ev.type) {
              case 'text':
                assistantContent += ev.content;
                break;
              case 'proposal':
                proposal = ev.payload;
                onUpdateDefinition({
                  nodes: ev.payload.nodes,
                  edges: ev.payload.edges,
                });
                break;
              case 'patch':
                patch = ev.payload;
                break;
              case 'error':
                assistantContent = ev.error;
                break;
              case 'done':
                break;
            }
          } catch { /* malformed chunk; skip */ }
        }
      }
    }

    const msg: ChatMessage = {
      id: assistantId,
      role: 'assistant',
      content: assistantContent || '(no reply)',
      proposal,
      patch,
    };
    append(msg);
    setSending(false);
  }, [workflowId, append, setSending, onUpdateDefinition]);

  return { send };
}
