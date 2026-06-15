package storage_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/efekckk/flowgent/internal/idgen"
	"github.com/efekckk/flowgent/internal/storage"
)

func setupWorkflow(t *testing.T) (*storage.WorkflowRepo, *storage.WorkflowRunRepo, storage.Workflow, *storage.UserRepo, storage.User) {
	t.Helper()
	pool := openFresh(t)
	users := storage.NewUserRepo(pool)
	workspaces := storage.NewWorkspaceRepo(pool)
	workflows := storage.NewWorkflowRepo(pool)
	runs := storage.NewWorkflowRunRepo(pool)
	ctx := context.Background()

	u := storage.User{ID: idgen.NewUser(), Email: "r@example.com", PasswordHash: "h"}
	_ = users.Insert(ctx, u)
	ws := storage.Workspace{ID: idgen.NewWorkspace(), OwnerUserID: u.ID, Name: "x"}
	_ = workspaces.Insert(ctx, ws)
	wf := storage.Workflow{ID: idgen.NewWorkflow(), WorkspaceID: ws.ID, Name: "wf", Status: "active"}
	_ = workflows.Insert(ctx, wf)
	def, _ := json.Marshal(map[string]any{"nodes": []any{}, "edges": []any{}})
	_ = workflows.SaveVersion(ctx, storage.WorkflowVersion{
		ID: idgen.NewWorkflowVersion(), WorkflowID: wf.ID, Version: 1, Definition: def,
	})
	return workflows, runs, wf, users, u
}

func TestWorkflowRunRepo_NewRun(t *testing.T) {
	_, runs, wf, _, _ := setupWorkflow(t)
	ctx := context.Background()
	run := storage.WorkflowRun{
		ID:              idgen.NewRun(),
		WorkflowID:      wf.ID,
		WorkflowVersion: 1,
		Status:          "queued",
		TriggerKind:     "manual",
		TriggerPayload:  json.RawMessage(`{"hello":"world"}`),
	}
	if err := runs.NewRun(ctx, run); err != nil {
		t.Fatalf("new run: %v", err)
	}
	got, err := runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != "queued" {
		t.Errorf("status: %s", got.Status)
	}
}

func TestWorkflowRunRepo_UpdateRunStatus(t *testing.T) {
	_, runs, wf, _, _ := setupWorkflow(t)
	ctx := context.Background()
	run := storage.WorkflowRun{
		ID: idgen.NewRun(), WorkflowID: wf.ID, WorkflowVersion: 1,
		Status: "queued", TriggerKind: "manual",
	}
	_ = runs.NewRun(ctx, run)
	now := time.Now().UTC()
	if err := runs.UpdateRunStatus(ctx, run.ID, "succeeded", "", &now, &now); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ := runs.Get(ctx, run.ID)
	if got.Status != "succeeded" || got.FinishedAt == nil {
		t.Errorf("got %+v", got)
	}
}

func TestWorkflowRunRepo_InsertAndUpdateNodeRun(t *testing.T) {
	_, runs, wf, _, _ := setupWorkflow(t)
	ctx := context.Background()
	run := storage.WorkflowRun{
		ID: idgen.NewRun(), WorkflowID: wf.ID, WorkflowVersion: 1,
		Status: "running", TriggerKind: "manual",
	}
	_ = runs.NewRun(ctx, run)

	nr := storage.NodeRun{
		ID:            idgen.NewNodeRun(),
		WorkflowRunID: run.ID,
		NodeID:        "n1",
		Iteration:     0,
		Status:        "running",
		Attempts:      1,
	}
	if err := runs.InsertNodeRun(ctx, nr); err != nil {
		t.Fatalf("insert node run: %v", err)
	}
	now := time.Now().UTC()
	if err := runs.UpdateNodeRun(ctx, nr.ID, storage.NodeRunUpdate{
		Status:     "succeeded",
		Output:     json.RawMessage(`{"ok":true}`),
		Attempts:   1,
		StartedAt:  &now,
		FinishedAt: &now,
		DurationMs: intPtr(12),
	}); err != nil {
		t.Fatalf("update node run: %v", err)
	}
	rows, err := runs.ListNodeRuns(ctx, run.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 1 || rows[0].Status != "succeeded" {
		t.Errorf("got %+v", rows)
	}
}

func intPtr(v int) *int { return &v }
