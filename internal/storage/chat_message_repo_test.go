package storage_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/efekckk/flowgent/internal/idgen"
	"github.com/efekckk/flowgent/internal/storage"
)

func TestChatMessageRepo_InsertAndListInOrder(t *testing.T) {
	pool := openFresh(t)
	users := storage.NewUserRepo(pool)
	workspaces := storage.NewWorkspaceRepo(pool)
	workflows := storage.NewWorkflowRepo(pool)
	threads := storage.NewChatThreadRepo(pool)
	msgs := storage.NewChatMessageRepo(pool)
	ctx := context.Background()

	u := storage.User{ID: idgen.NewUser(), Email: "m@example.com", PasswordHash: "h"}
	_ = users.Insert(ctx, u)
	ws := storage.Workspace{ID: idgen.NewWorkspace(), OwnerUserID: u.ID, Name: "x"}
	_ = workspaces.Insert(ctx, ws)
	wf := storage.Workflow{ID: idgen.NewWorkflow(), WorkspaceID: ws.ID, Name: "w", Status: "draft"}
	_ = workflows.Insert(ctx, wf)
	def, _ := json.Marshal(map[string]any{})
	_ = workflows.SaveVersion(ctx, storage.WorkflowVersion{
		ID: idgen.NewWorkflowVersion(), WorkflowID: wf.ID, Version: 1, Definition: def,
	})
	thr := storage.ChatThread{ID: idgen.NewChatThread(), WorkflowID: wf.ID, UserID: u.ID}
	_ = threads.Insert(ctx, thr)

	for i := 1; i <= 3; i++ {
		err := msgs.Insert(ctx, storage.ChatMessage{
			ID: idgen.NewChatMessage(), ThreadID: thr.ID, Role: "user",
			Content: fmt.Sprintf("msg %d", i),
		})
		if err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	rows, err := msgs.ListByThread(ctx, thr.ID, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 3 {
		t.Errorf("count: %d", len(rows))
	}
}

func TestChatMessageRepo_AssistantToolCallsPreserved(t *testing.T) {
	pool := openFresh(t)
	users := storage.NewUserRepo(pool)
	workspaces := storage.NewWorkspaceRepo(pool)
	workflows := storage.NewWorkflowRepo(pool)
	threads := storage.NewChatThreadRepo(pool)
	msgs := storage.NewChatMessageRepo(pool)
	ctx := context.Background()

	u := storage.User{ID: idgen.NewUser(), Email: "tc@example.com", PasswordHash: "h"}
	_ = users.Insert(ctx, u)
	ws := storage.Workspace{ID: idgen.NewWorkspace(), OwnerUserID: u.ID, Name: "x"}
	_ = workspaces.Insert(ctx, ws)
	wf := storage.Workflow{ID: idgen.NewWorkflow(), WorkspaceID: ws.ID, Name: "w", Status: "draft"}
	_ = workflows.Insert(ctx, wf)
	def, _ := json.Marshal(map[string]any{})
	_ = workflows.SaveVersion(ctx, storage.WorkflowVersion{
		ID: idgen.NewWorkflowVersion(), WorkflowID: wf.ID, Version: 1, Definition: def,
	})
	thr := storage.ChatThread{ID: idgen.NewChatThread(), WorkflowID: wf.ID, UserID: u.ID}
	_ = threads.Insert(ctx, thr)

	toolCalls := json.RawMessage(`[{"name":"propose_workflow","args":{"x":1}}]`)
	_ = msgs.Insert(ctx, storage.ChatMessage{
		ID: idgen.NewChatMessage(), ThreadID: thr.ID, Role: "assistant",
		Content: "Building workflow.", ToolCalls: toolCalls,
		Model: "claude-3-7-sonnet",
	})
	rows, _ := msgs.ListByThread(ctx, thr.ID, 10)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	var gotTC, wantTC any
	if err := json.Unmarshal(rows[0].ToolCalls, &gotTC); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}
	if err := json.Unmarshal(toolCalls, &wantTC); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	gotBytes, _ := json.Marshal(gotTC)
	wantBytes, _ := json.Marshal(wantTC)
	if string(gotBytes) != string(wantBytes) {
		t.Errorf("tool_calls round-trip: got %s want %s", gotBytes, wantBytes)
	}
}
