import { useParams } from 'react-router-dom';
import { useEffect } from 'react';
import WorkflowList from './WorkflowList';
import { useWorkflowsStore } from './workflowsStore';

export default function WorkflowsPage() {
  const { id } = useParams<{ id: string }>();
  const { current, fetchOne } = useWorkflowsStore();

  useEffect(() => {
    if (id) {
      fetchOne(id);
    }
  }, [id, fetchOne]);

  return (
    <div className="flex h-full bg-slate-50">
      <WorkflowList />
      <main className="flex-1 overflow-hidden">
        {!id && (
          <div className="flex h-full items-center justify-center text-slate-400">
            Select or create a workflow to begin.
          </div>
        )}
        {id && !current && (
          <div className="flex h-full items-center justify-center text-slate-400">
            Loading workflow…
          </div>
        )}
        {id && current && (
          <div className="flex h-full items-center justify-center text-slate-400">
            {current.name} (canvas + chat coming in next task)
          </div>
        )}
      </main>
    </div>
  );
}
