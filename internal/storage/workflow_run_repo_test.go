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

func TestWorkflowRunRepo_ListForWorkflow_paginates(t *testing.T) {
	_, runs, wf, _, _ := setupWorkflow(t)
	ctx := context.Background()

	// Insert 7 runs; ListForWorkflow with Limit=3 should walk three pages
	// (3 + 3 + 1) with cursors threading them together.
	for i := 0; i < 7; i++ {
		err := runs.NewRun(ctx, storage.WorkflowRun{
			ID:              idgen.NewRun(),
			WorkflowID:      wf.ID,
			WorkflowVersion: 1,
			Status:          "succeeded",
			TriggerKind:     "manual",
			TriggerPayload:  json.RawMessage(`{"i":` + itoa(i) + `}`),
		})
		if err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
		// Tiny sleep so created_at advances and the (created_at, id)
		// ordering is deterministic; the tests below depend on
		// monotone creation order being reflected in DESC listing.
		time.Sleep(2 * time.Millisecond)
	}

	page1, cur1, err := runs.ListForWorkflow(ctx, wf.ID, storage.RunFilter{Limit: 3})
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1) != 3 || cur1 == "" {
		t.Fatalf("page1 len=%d cursor=%q", len(page1), cur1)
	}

	page2, cur2, err := runs.ListForWorkflow(ctx, wf.ID, storage.RunFilter{Limit: 3, Cursor: cur1})
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2) != 3 || cur2 == "" {
		t.Fatalf("page2 len=%d cursor=%q", len(page2), cur2)
	}

	page3, cur3, err := runs.ListForWorkflow(ctx, wf.ID, storage.RunFilter{Limit: 3, Cursor: cur2})
	if err != nil {
		t.Fatalf("page3: %v", err)
	}
	if len(page3) != 1 || cur3 != "" {
		t.Fatalf("page3 len=%d cursor=%q (want 1, \"\")", len(page3), cur3)
	}

	// Pages must not overlap.
	seen := map[string]bool{}
	for _, r := range append(append(page1, page2...), page3...) {
		if seen[r.ID] {
			t.Fatalf("duplicate run id across pages: %s", r.ID)
		}
		seen[r.ID] = true
	}
	if len(seen) != 7 {
		t.Errorf("expected 7 unique rows, got %d", len(seen))
	}
}

func TestWorkflowRunRepo_ListForWorkflow_filterByStatus(t *testing.T) {
	_, runs, wf, _, _ := setupWorkflow(t)
	ctx := context.Background()

	for _, st := range []string{"succeeded", "failed", "succeeded", "running"} {
		_ = runs.NewRun(ctx, storage.WorkflowRun{
			ID: idgen.NewRun(), WorkflowID: wf.ID, WorkflowVersion: 1,
			Status: st, TriggerKind: "manual",
		})
	}

	got, _, err := runs.ListForWorkflow(ctx, wf.ID, storage.RunFilter{Status: "succeeded"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 succeeded rows, got %d", len(got))
	}
	for _, r := range got {
		if r.Status != "succeeded" {
			t.Errorf("filter leaked: %s", r.Status)
		}
	}
}

func TestWorkflowRunRepo_ListForWorkflow_filterByDateRange(t *testing.T) {
	_, runs, wf, _, _ := setupWorkflow(t)
	ctx := context.Background()

	_ = runs.NewRun(ctx, storage.WorkflowRun{
		ID: idgen.NewRun(), WorkflowID: wf.ID, WorkflowVersion: 1,
		Status: "succeeded", TriggerKind: "manual",
	})
	time.Sleep(20 * time.Millisecond)
	cutoff := time.Now().UTC()
	time.Sleep(20 * time.Millisecond)
	keepID := idgen.NewRun()
	_ = runs.NewRun(ctx, storage.WorkflowRun{
		ID: keepID, WorkflowID: wf.ID, WorkflowVersion: 1,
		Status: "succeeded", TriggerKind: "manual",
	})

	got, _, err := runs.ListForWorkflow(ctx, wf.ID, storage.RunFilter{From: &cutoff})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].ID != keepID {
		t.Fatalf("expected only the post-cutoff run, got %d rows", len(got))
	}
}

func TestWorkflowRunRepo_GetWithNodes(t *testing.T) {
	_, runs, wf, _, _ := setupWorkflow(t)
	ctx := context.Background()

	runID := idgen.NewRun()
	_ = runs.NewRun(ctx, storage.WorkflowRun{
		ID: runID, WorkflowID: wf.ID, WorkflowVersion: 1,
		Status: "running", TriggerKind: "manual",
	})
	for _, nid := range []string{"a", "b", "c"} {
		_ = runs.InsertNodeRun(ctx, storage.NodeRun{
			ID:            idgen.NewNodeRun(),
			WorkflowRunID: runID,
			NodeID:        nid,
			Status:        "succeeded",
			Attempts:      1,
		})
	}

	got, nodes, err := runs.GetWithNodes(ctx, runID)
	if err != nil {
		t.Fatalf("GetWithNodes: %v", err)
	}
	if got.ID != runID {
		t.Fatalf("got run id %s", got.ID)
	}
	if len(nodes) != 3 {
		t.Fatalf("expected 3 node runs, got %d", len(nodes))
	}
}

// itoa avoids dragging in strconv just for the tiny seed loop above.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
