package storage_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/efekckk/flowgent/internal/idgen"
	"github.com/efekckk/flowgent/internal/storage"
)

func TestChatThreadRepo_InsertAndGet(t *testing.T) {
	pool := openFresh(t)
	users := storage.NewUserRepo(pool)
	workspaces := storage.NewWorkspaceRepo(pool)
	workflows := storage.NewWorkflowRepo(pool)
	threads := storage.NewChatThreadRepo(pool)
	ctx := context.Background()

	u := storage.User{ID: idgen.NewUser(), Email: "c@example.com", PasswordHash: "h"}
	_ = users.Insert(ctx, u)
	ws := storage.Workspace{ID: idgen.NewWorkspace(), OwnerUserID: u.ID, Name: "x"}
	_ = workspaces.Insert(ctx, ws)
	wf := storage.Workflow{ID: idgen.NewWorkflow(), WorkspaceID: ws.ID, Name: "w", Status: "draft"}
	_ = workflows.Insert(ctx, wf)
	def, _ := json.Marshal(map[string]any{"nodes": []any{}, "edges": []any{}})
	_ = workflows.SaveVersion(ctx, storage.WorkflowVersion{
		ID: idgen.NewWorkflowVersion(), WorkflowID: wf.ID, Version: 1, Definition: def,
	})

	thr := storage.ChatThread{
		ID:         idgen.NewChatThread(),
		WorkflowID: wf.ID,
		UserID:     u.ID,
	}
	if err := threads.Insert(ctx, thr); err != nil {
		t.Fatalf("insert: %v", err)
	}
	got, err := threads.GetByWorkflowAndUser(ctx, wf.ID, u.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != thr.ID {
		t.Errorf("got %+v", got)
	}
}
