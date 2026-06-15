export default function ChatPanel({ workflowId }: { workflowId: string }) {
  return (
    <aside className="flex h-full w-96 flex-col border-l border-slate-200 bg-white">
      <div className="border-b border-slate-200 px-4 py-3 text-sm font-semibold text-slate-700">
        Chat
      </div>
      <div className="flex-1 overflow-y-auto px-4 py-3 text-sm text-slate-400">
        Workflow: {workflowId} (chat coming in next task)
      </div>
    </aside>
  );
}
