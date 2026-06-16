package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/idgen"
	"github.com/efekckk/flowgent/internal/registry"
	"github.com/efekckk/flowgent/internal/storage"
	"github.com/efekckk/flowgent/internal/storage/storagetest"
	"github.com/efekckk/flowgent/internal/webhook"

	coreset "github.com/efekckk/flowgent/tools/core.set"
)

// seedFireableWorkflow inserts the minimal user/workspace/workflow/version
// tree the firer needs so a fire produces a real workflow_runs row. The
// returned workflow has a single core.set node so engine.Run executes
// without external dependencies.
func seedFireableWorkflow(t *testing.T, pool *pgxpool.Pool) storage.Workflow {
	t.Helper()
	ctx := context.Background()
	users := storage.NewUserRepo(pool)
	workspaces := storage.NewWorkspaceRepo(pool)
	workflows := storage.NewWorkflowRepo(pool)

	u := storage.User{ID: idgen.NewUser(), Email: "firer@example.com", PasswordHash: "h"}
	if err := users.Insert(ctx, u); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	ws := storage.Workspace{ID: idgen.NewWorkspace(), OwnerUserID: u.ID, Name: "firer-ws"}
	if err := workspaces.Insert(ctx, ws); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	wf := storage.Workflow{ID: idgen.NewWorkflow(), WorkspaceID: ws.ID, Name: "wf", Status: "active"}
	if err := workflows.Insert(ctx, wf); err != nil {
		t.Fatalf("insert workflow: %v", err)
	}
	def, _ := json.Marshal(map[string]any{
		"id":   wf.ID,
		"name": "fireable",
		"nodes": []map[string]any{
			{
				"id":     "n1",
				"tool":   "core.set",
				"params": map[string]any{"values": map[string]any{"hello": "world"}},
			},
		},
		"edges": []map[string]any{},
	})
	if err := workflows.SaveVersion(ctx, storage.WorkflowVersion{
		ID: idgen.NewWorkflowVersion(), WorkflowID: wf.ID, Version: 1, Definition: def,
	}); err != nil {
		t.Fatalf("save version: %v", err)
	}
	return wf
}

func buildEngine(t *testing.T) *executor.Engine {
	t.Helper()
	reg := registry.New()
	reg.Register("core.set", coreset.New())
	return executor.NewEngine(reg)
}

func newFirerTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if _, err := os.Stat("/var/run/docker.sock"); err != nil {
		t.Skip("docker unavailable")
	}
	if err := storagetest.Start(); err != nil {
		t.Skipf("dockertest: %v", err)
	}
	t.Cleanup(storagetest.Stop)
	dsn := storagetest.Fresh(t)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestProductionFirer_CronFireRecordsTriggerProvenance(t *testing.T) {
	pool := newFirerTestPool(t)
	ctx := context.Background()

	wf := seedFireableWorkflow(t, pool)
	triggerRepo := storage.NewTriggerRepo(pool)
	runRepo := storage.NewWorkflowRunRepo(pool)
	workflowRepo := storage.NewWorkflowRepo(pool)

	trgID := idgen.NewTrigger()
	cfg := json.RawMessage(`{"cron":"@every 1h"}`)
	if err := triggerRepo.Insert(ctx, storage.Trigger{
		ID: trgID, WorkflowID: wf.ID, Kind: "cron", Config: cfg, Enabled: true,
	}); err != nil {
		t.Fatalf("insert trigger: %v", err)
	}

	f := &productionFirer{
		triggers:  triggerRepo,
		workflows: workflowRepo,
		runs:      runRepo,
		engine:    buildEngine(t),
	}

	if err := f.FireTrigger(ctx, trgID, wf.ID, map[string]any{"cron": "@every 1h"}); err != nil {
		t.Fatalf("fire: %v", err)
	}

	runs, _, err := runRepo.ListForWorkflow(ctx, wf.ID, storage.RunFilter{Limit: 10})
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	got := runs[0]
	if got.TriggerKind != "cron" {
		t.Fatalf("trigger_kind: want cron, got %q", got.TriggerKind)
	}
	if got.TriggerID == nil || *got.TriggerID != trgID {
		t.Fatalf("trigger_id: want %q, got %v", trgID, got.TriggerID)
	}
	if got.Status != "succeeded" {
		t.Fatalf("status: want succeeded, got %q (err=%q)", got.Status, got.Error)
	}

	trg, err := triggerRepo.Get(ctx, trgID)
	if err != nil {
		t.Fatalf("get trigger: %v", err)
	}
	if trg.LastFiredAt == nil {
		t.Fatalf("last_fired_at should be non-nil after fire")
	}
}

func TestProductionFirer_WebhookFireRecordsTriggerProvenance(t *testing.T) {
	pool := newFirerTestPool(t)
	ctx := context.Background()

	wf := seedFireableWorkflow(t, pool)
	triggerRepo := storage.NewTriggerRepo(pool)
	runRepo := storage.NewWorkflowRunRepo(pool)
	workflowRepo := storage.NewWorkflowRepo(pool)

	trgID := idgen.NewTrigger()
	cfg := json.RawMessage(`{"token":"tok-abc","secret":""}`)
	if err := triggerRepo.Insert(ctx, storage.Trigger{
		ID: trgID, WorkflowID: wf.ID, Kind: "webhook", Config: cfg, Enabled: true,
	}); err != nil {
		t.Fatalf("insert trigger: %v", err)
	}

	f := &productionFirer{
		triggers:  triggerRepo,
		workflows: workflowRepo,
		runs:      runRepo,
		engine:    buildEngine(t),
	}

	payload := map[string]any{"event": "test"}
	if err := f.FireTrigger(ctx, trgID, wf.ID, payload); err != nil {
		t.Fatalf("fire: %v", err)
	}

	runs, _, err := runRepo.ListForWorkflow(ctx, wf.ID, storage.RunFilter{Limit: 10})
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	got := runs[0]
	if got.TriggerKind != "webhook" {
		t.Fatalf("trigger_kind: want webhook, got %q", got.TriggerKind)
	}
	if got.TriggerID == nil || *got.TriggerID != trgID {
		t.Fatalf("trigger_id: want %q, got %v", trgID, got.TriggerID)
	}
	var stored map[string]any
	if err := json.Unmarshal(got.TriggerPayload, &stored); err != nil {
		t.Fatalf("payload unmarshal: %v", err)
	}
	if stored["event"] != "test" {
		t.Fatalf("trigger_payload event: want %q, got %v", "test", stored["event"])
	}
}

func TestTriggerResolver_ReturnsNotFoundForCronRows(t *testing.T) {
	pool := newFirerTestPool(t)
	ctx := context.Background()

	wf := seedFireableWorkflow(t, pool)
	triggerRepo := storage.NewTriggerRepo(pool)

	cronID := idgen.NewTrigger()
	if err := triggerRepo.Insert(ctx, storage.Trigger{
		ID: cronID, WorkflowID: wf.ID, Kind: "cron",
		Config: json.RawMessage(`{"cron":"@every 1h"}`), Enabled: true,
	}); err != nil {
		t.Fatalf("insert cron trigger: %v", err)
	}

	disabledHookID := idgen.NewTrigger()
	if err := triggerRepo.Insert(ctx, storage.Trigger{
		ID: disabledHookID, WorkflowID: wf.ID, Kind: "webhook",
		Config: json.RawMessage(`{"token":"x"}`), Enabled: false,
	}); err != nil {
		t.Fatalf("insert disabled webhook: %v", err)
	}

	enabledHookID := idgen.NewTrigger()
	if err := triggerRepo.Insert(ctx, storage.Trigger{
		ID: enabledHookID, WorkflowID: wf.ID, Kind: "webhook",
		Config: json.RawMessage(`{"token":"good","secret":"s"}`), Enabled: true,
	}); err != nil {
		t.Fatalf("insert enabled webhook: %v", err)
	}

	res := &triggerResolver{repo: triggerRepo}

	// Cron rows must never resolve via the webhook resolver.
	if _, ok, err := res.ResolveWebhook(ctx, cronID); err != nil || ok {
		t.Fatalf("cron row should not resolve: ok=%v err=%v", ok, err)
	}
	// Disabled webhook rows must not resolve.
	if _, ok, err := res.ResolveWebhook(ctx, disabledHookID); err != nil || ok {
		t.Fatalf("disabled webhook should not resolve: ok=%v err=%v", ok, err)
	}
	// Unknown ids: ok=false, err=nil.
	if _, ok, err := res.ResolveWebhook(ctx, "nope"); err != nil || ok {
		t.Fatalf("unknown id should be ok=false,nil err, got ok=%v err=%v", ok, err)
	}
	// Enabled webhook resolves with token + secret.
	got, ok, err := res.ResolveWebhook(ctx, enabledHookID)
	if err != nil || !ok {
		t.Fatalf("enabled webhook should resolve: ok=%v err=%v", ok, err)
	}
	if got.Token != "good" {
		t.Fatalf("token: want good, got %q", got.Token)
	}
	if string(got.Secret) != "s" {
		t.Fatalf("secret: want s, got %q", string(got.Secret))
	}
	// Sanity: WebhookTrigger zero value test compiles.
	_ = webhook.WebhookTrigger{}
	// Ensure errors.Is wiring is reachable for the type-checker.
	_ = errors.Is(nil, storage.ErrNotFound)
}
