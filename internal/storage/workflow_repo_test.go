package storage_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/efekckk/flowgent/internal/idgen"
	"github.com/efekckk/flowgent/internal/storage"
)

func TestWorkflowRepo_InsertAndGet(t *testing.T) {
	pool := openFresh(t)
	users := storage.NewUserRepo(pool)
	workspaces := storage.NewWorkspaceRepo(pool)
	workflows := storage.NewWorkflowRepo(pool)
	ctx := context.Background()

	u := storage.User{ID: idgen.NewUser(), Email: "a@example.com", PasswordHash: "h"}
	_ = users.Insert(ctx, u)
	ws := storage.Workspace{ID: idgen.NewWorkspace(), OwnerUserID: u.ID, Name: "x"}
	_ = workspaces.Insert(ctx, ws)

	wf := storage.Workflow{
		ID:          idgen.NewWorkflow(),
		WorkspaceID: ws.ID,
		Name:        "first",
		Status:      "draft",
	}
	if err := workflows.Insert(ctx, wf); err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, err := workflows.Get(ctx, wf.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "first" || got.CurrentVersion != 1 {
		t.Errorf("got %+v", got)
	}
}

func TestWorkflowRepo_SaveVersionAndGet(t *testing.T) {
	pool := openFresh(t)
	users := storage.NewUserRepo(pool)
	workspaces := storage.NewWorkspaceRepo(pool)
	workflows := storage.NewWorkflowRepo(pool)
	ctx := context.Background()

	u := storage.User{ID: idgen.NewUser(), Email: "b@example.com", PasswordHash: "h"}
	_ = users.Insert(ctx, u)
	ws := storage.Workspace{ID: idgen.NewWorkspace(), OwnerUserID: u.ID, Name: "x"}
	_ = workspaces.Insert(ctx, ws)

	wf := storage.Workflow{ID: idgen.NewWorkflow(), WorkspaceID: ws.ID, Name: "vv", Status: "draft"}
	_ = workflows.Insert(ctx, wf)

	def, _ := json.Marshal(map[string]any{"nodes": []any{}, "edges": []any{}})
	if err := workflows.SaveVersion(ctx, storage.WorkflowVersion{
		ID:              idgen.NewWorkflowVersion(),
		WorkflowID:      wf.ID,
		Version:         1,
		Definition:      def,
		Message:         "initial",
		CreatedByUserID: &u.ID,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	ver, err := workflows.GetVersion(ctx, wf.ID, 1)
	if err != nil {
		t.Fatalf("get version: %v", err)
	}
	if ver.Message != "initial" {
		t.Errorf("got %+v", ver)
	}
}

func TestWorkflowRepo_ListByWorkspace(t *testing.T) {
	pool := openFresh(t)
	users := storage.NewUserRepo(pool)
	workspaces := storage.NewWorkspaceRepo(pool)
	workflows := storage.NewWorkflowRepo(pool)
	ctx := context.Background()

	u := storage.User{ID: idgen.NewUser(), Email: "c@example.com", PasswordHash: "h"}
	_ = users.Insert(ctx, u)
	ws := storage.Workspace{ID: idgen.NewWorkspace(), OwnerUserID: u.ID, Name: "x"}
	_ = workspaces.Insert(ctx, ws)

	for i := 0; i < 3; i++ {
		_ = workflows.Insert(ctx, storage.Workflow{
			ID: idgen.NewWorkflow(), WorkspaceID: ws.ID, Name: "wf", Status: "draft",
		})
	}
	rows, err := workflows.ListByWorkspace(ctx, ws.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 3 {
		t.Errorf("count: %d", len(rows))
	}
}
