import { useEffect } from 'react';
import { useRunDetailStore } from './runDetailStore';
import type { RunLogEvent } from '../api/types';

// Opens an EventSource against the SSE endpoint for the given run and
// pipes "log" frames into the store. When a log line announces a node
// terminal event we re-fetch the run so the canvas can recolor itself.
// The browser auto-reconnects on transient errors; the backend honours
// ?since= to backfill missed lines on reconnect.
export function useRunLogStream(runId: string | null) {
  const appendLog = useRunDetailStore((s) => s.appendLog);
  const refreshNodes = useRunDetailStore((s) => s.refreshNodes);

  useEffect(() => {
    if (!runId) return;
    const url = `/v1/runs/${runId}/stream`;
    const es = new EventSource(url, { withCredentials: true });

    es.addEventListener('log', (msg) => {
      try {
        const data = JSON.parse((msg as MessageEvent).data) as RunLogEvent;
        appendLog(data);
        const m = (data.message || '').toLowerCase();
        if (m.includes('succeeded') || m.includes('failed') || m.includes('error')) {
          void refreshNodes();
        }
      } catch {
        // ignore malformed frames
      }
    });

    es.onerror = () => {
      // The browser handles reconnect automatically; nothing to do here.
    };

    return () => es.close();
  }, [runId, appendLog, refreshNodes]);
}
