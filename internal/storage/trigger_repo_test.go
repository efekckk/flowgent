package storage_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/efekckk/flowgent/internal/idgen"
	"github.com/efekckk/flowgent/internal/storage"
)

func setupTriggerContext(t *testing.T) (*storage.TriggerRepo, *pgxpool.Pool, storage.Workflow) {
	t.Helper()
	pool := openFresh(t)
	users := storage.NewUserRepo(pool)
	workspaces := storage.NewWorkspaceRepo(pool)
	workflows := storage.NewWorkflowRepo(pool)
	repo := storage.NewTriggerRepo(pool)
	ctx := context.Background()

	u := storage.User{ID: idgen.NewUser(), Email: "trg@example.com", PasswordHash: "h"}
	if err := users.Insert(ctx, u); err != nil {
		t.Fatalf("user: %v", err)
	}
	ws := storage.Workspace{ID: idgen.NewWorkspace(), OwnerUserID: u.ID, Name: "x"}
	if err := workspaces.Insert(ctx, ws); err != nil {
		t.Fatalf("workspace: %v", err)
	}
	wf := storage.Workflow{ID: idgen.NewWorkflow(), WorkspaceID: ws.ID, Name: "wf", Status: "draft"}
	if err := workflows.Insert(ctx, wf); err != nil {
		t.Fatalf("workflow: %v", err)
	}
	return repo, pool, wf
}

func TestTriggerRepo_InsertAndList(t *testing.T) {
	repo, _, wf := setupTriggerContext(t)
	ctx := context.Background()

	in := storage.Trigger{
		ID:         "trg_a",
		WorkflowID: wf.ID,
		Kind:       "cron",
		Config:     json.RawMessage(`{"cron":"@every 5m"}`),
		Enabled:    true,
	}
	if err := repo.Insert(ctx, in); err != nil {
		t.Fatalf("insert: %v", err)
	}
	list, err := repo.ListByWorkflow(ctx, wf.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].ID != "trg_a" {
		t.Fatalf("list: %+v", list)
	}
}

func TestTriggerRepo_ListEnabledByKind(t *testing.T) {
	repo, _, wf := setupTriggerContext(t)
	ctx := context.Background()

	_ = repo.Insert(ctx, storage.Trigger{ID: "t1", WorkflowID: wf.ID, Kind: "cron", Config: json.RawMessage(`{}`), Enabled: true})
	_ = repo.Insert(ctx, storage.Trigger{ID: "t2", WorkflowID: wf.ID, Kind: "cron", Config: json.RawMessage(`{}`), Enabled: false})
	_ = repo.Insert(ctx, storage.Trigger{ID: "t3", WorkflowID: wf.ID, Kind: "webhook", Config: json.RawMessage(`{}`), Enabled: true})

	crons, err := repo.ListEnabledByKind(ctx, "cron")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(crons) != 1 || crons[0].ID != "t1" {
		t.Fatalf("crons: %+v", crons)
	}
}

func TestTriggerRepo_UpdateAndGet(t *testing.T) {
	repo, _, wf := setupTriggerContext(t)
	ctx := context.Background()
	_ = repo.Insert(ctx, storage.Trigger{ID: "trg", WorkflowID: wf.ID, Kind: "cron", Config: json.RawMessage(`{}`), Enabled: true})

	if err := repo.UpdateConfig(ctx, "trg", json.RawMessage(`{"cron":"0 9 * * *"}`), false); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := repo.Get(ctx, "trg")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Enabled {
		t.Errorf("expected disabled")
	}
	if string(got.Config) != `{"cron": "0 9 * * *"}` && string(got.Config) != `{"cron":"0 9 * * *"}` {
		t.Errorf("config: %s", got.Config)
	}
}

func TestTriggerRepo_Delete(t *testing.T) {
	repo, _, wf := setupTriggerContext(t)
	ctx := context.Background()
	_ = repo.Insert(ctx, storage.Trigger{ID: "trg", WorkflowID: wf.ID, Kind: "webhook", Config: json.RawMessage(`{}`), Enabled: true})

	if err := repo.Delete(ctx, "trg"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.Get(ctx, "trg"); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
	if err := repo.Delete(ctx, "trg"); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("expected ErrNotFound on second delete, got %v", err)
	}
}

func TestTriggerRepo_TouchLastFired(t *testing.T) {
	repo, _, wf := setupTriggerContext(t)
	ctx := context.Background()
	_ = repo.Insert(ctx, storage.Trigger{ID: "trg", WorkflowID: wf.ID, Kind: "cron", Config: json.RawMessage(`{}`), Enabled: true})

	if err := repo.TouchLastFired(ctx, "trg"); err != nil {
		t.Fatalf("touch: %v", err)
	}
	got, _ := repo.Get(ctx, "trg")
	if got.LastFiredAt == nil {
		t.Errorf("expected last_fired_at to be set")
	}
}
