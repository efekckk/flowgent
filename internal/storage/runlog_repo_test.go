package storage_test

import (
	"context"
	"testing"

	"github.com/efekckk/flowgent/internal/idgen"
	"github.com/efekckk/flowgent/internal/storage"
)

func setupRunWithLogs(t *testing.T) (*storage.RunLogRepo, *storage.WorkflowRunRepo, storage.Workflow, string) {
	t.Helper()
	pool := openFresh(t)
	users := storage.NewUserRepo(pool)
	workspaces := storage.NewWorkspaceRepo(pool)
	workflows := storage.NewWorkflowRepo(pool)
	runs := storage.NewWorkflowRunRepo(pool)
	logs := storage.NewRunLogRepo(pool)
	ctx := context.Background()

	u := storage.User{ID: idgen.NewUser(), Email: "logs@example.com", PasswordHash: "h"}
	_ = users.Insert(ctx, u)
	ws := storage.Workspace{ID: idgen.NewWorkspace(), OwnerUserID: u.ID, Name: "ws"}
	_ = workspaces.Insert(ctx, ws)
	wf := storage.Workflow{ID: idgen.NewWorkflow(), WorkspaceID: ws.ID, Name: "logs-host", Status: "active"}
	_ = workflows.Insert(ctx, wf)

	runID := idgen.NewRun()
	_ = runs.NewRun(ctx, storage.WorkflowRun{
		ID: runID, WorkflowID: wf.ID, WorkflowVersion: 1,
		Status: "running", TriggerKind: "manual",
	})
	return logs, runs, wf, runID
}

func TestRunLogRepo_AppendAndListByRun(t *testing.T) {
	logs, _, _, runID := setupRunWithLogs(t)
	ctx := context.Background()

	for _, msg := range []string{"first line", "second line", "third line"} {
		if err := logs.Append(ctx, storage.RunLog{
			RunID: runID, Level: "info", Message: msg,
		}); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	got, err := logs.ListByRun(ctx, runID, 0, 100)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(got))
	}
	want := []string{"first line", "second line", "third line"}
	for i, row := range got {
		if row.Message != want[i] {
			t.Errorf("row %d: got %q want %q", i, row.Message, want[i])
		}
	}
}

func TestRunLogRepo_ListByRun_sinceID(t *testing.T) {
	logs, _, _, runID := setupRunWithLogs(t)
	ctx := context.Background()

	for _, msg := range []string{"a", "b", "c", "d"} {
		_ = logs.Append(ctx, storage.RunLog{RunID: runID, Level: "info", Message: msg})
	}
	// First page: everything.
	all, err := logs.ListByRun(ctx, runID, 0, 100)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("expected 4, got %d", len(all))
	}
	cursor := all[1].ID

	tail, err := logs.ListByRun(ctx, runID, cursor, 100)
	if err != nil {
		t.Fatalf("list tail: %v", err)
	}
	if len(tail) != 2 {
		t.Fatalf("expected 2 rows after id %d, got %d", cursor, len(tail))
	}
	if tail[0].Message != "c" || tail[1].Message != "d" {
		t.Errorf("tail content: %+v", tail)
	}
}

func TestRunLogRepo_Search_workspaceScoped(t *testing.T) {
	logs, _, wf, runID := setupRunWithLogs(t)
	ctx := context.Background()

	_ = logs.Append(ctx, storage.RunLog{
		RunID: runID, NodeID: "n1", Level: "error",
		Message: "credential validation failed for user alice",
	})
	_ = logs.Append(ctx, storage.RunLog{
		RunID: runID, NodeID: "n2", Level: "info",
		Message: "unrelated success message",
	})

	hits, err := logs.Search(ctx, wf.WorkspaceID, "credential validation", 50)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d (%+v)", len(hits), hits)
	}
	if hits[0].RunID != runID || hits[0].WorkflowID != wf.ID {
		t.Errorf("hit metadata: %+v", hits[0])
	}
	if hits[0].Snippet == "" {
		t.Errorf("expected ts_headline snippet, got empty")
	}

	// A different workspace must not see those logs.
	otherWS := idgen.NewWorkspace()
	hits2, err := logs.Search(ctx, otherWS, "credential", 50)
	if err != nil {
		t.Fatalf("search other ws: %v", err)
	}
	if len(hits2) != 0 {
		t.Errorf("workspace scoping leaked %d rows", len(hits2))
	}
}
