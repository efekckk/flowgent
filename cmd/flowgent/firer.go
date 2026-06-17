package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/idgen"
	"github.com/efekckk/flowgent/internal/storage"
	"github.com/efekckk/flowgent/internal/webhook"
)

// productionFirer satisfies both scheduler.Firer and webhook.Firer (their
// method signatures are identical by design). For each fire it resolves the
// trigger row to derive trigger kind, loads the latest workflow version,
// inserts a workflow_runs row stamped with trigger_id + trigger_kind so the
// run viewer can attribute the run, dispatches engine.Run, and persists
// node_runs + the final status the same way the synchronous /run handler does.
//
// last_fired_at is touched on a defer so a dispatch failure still records the
// attempt — operators care that "the trigger tried", not just that it
// succeeded.
type productionFirer struct {
	triggers  *storage.TriggerRepo
	workflows *storage.WorkflowRepo
	runs      *storage.WorkflowRunRepo
	engine    *executor.Engine
}

func (f *productionFirer) FireTrigger(ctx context.Context, triggerID, workflowID string, payload map[string]any) error {
	defer func() {
		// last_fired_at is bookkeeping that must survive the caller's
		// context cancellation — a scheduler tick cleanup or a webhook
		// client hang-up shouldn't drop the audit trail. Use a fresh
		// background context with a tight deadline.
		touchCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = f.triggers.TouchLastFired(touchCtx, triggerID)
	}()

	trg, err := f.triggers.Get(ctx, triggerID)
	if err != nil {
		return fmt.Errorf("firer: load trigger: %w", err)
	}
	if trg.WorkflowID != workflowID {
		// Scheduler/handler can technically pass a stale workflow id; trust
		// the row over the caller to avoid firing into the wrong workflow
		// after a config edit.
		workflowID = trg.WorkflowID
	}

	wf, err := f.workflows.Get(ctx, workflowID)
	if err != nil {
		return fmt.Errorf("firer: load workflow: %w", err)
	}
	ver, err := f.workflows.GetVersion(ctx, wf.ID, wf.CurrentVersion)
	if err != nil {
		return fmt.Errorf("firer: load workflow version: %w", err)
	}
	var def executor.Workflow
	if err := json.Unmarshal(ver.Definition, &def); err != nil {
		return fmt.Errorf("firer: parse definition: %w", err)
	}

	runID := idgen.NewRun()
	startedAt := time.Now().UTC()
	payloadBytes, _ := json.Marshal(payload)
	if len(payloadBytes) == 0 || string(payloadBytes) == "null" {
		payloadBytes = json.RawMessage(`{}`)
	}
	triggerIDCopy := triggerID
	if err := f.runs.NewRun(ctx, storage.WorkflowRun{
		ID:              runID,
		WorkflowID:      wf.ID,
		WorkflowVersion: wf.CurrentVersion,
		Status:          "running",
		TriggerKind:     trg.Kind,
		TriggerID:       &triggerIDCopy,
		TriggerPayload:  payloadBytes,
		StartedAt:       &startedAt,
	}); err != nil {
		return fmt.Errorf("firer: insert run: %w", err)
	}

	// Mirror the engine's defensive defer: if we exit before the final
	// UpdateRunStatus call (panic, ctx cancellation, etc.) the row stays
	// in 'running' forever. The CAS guard in FailRunIfRunning ensures the
	// fallback never overwrites a real terminal status.
	settled := false
	defer func() {
		if settled {
			return
		}
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = f.runs.FailRunIfRunning(bgCtx, runID, "firer exited before final status", time.Now().UTC())
	}()

	state, runErr := f.engine.Run(ctx, &def, executor.RunOptions{
		TriggerKind:    trg.Kind,
		TriggerPayload: payload,
		RunID:          runID,
	})
	finishedAt := time.Now().UTC()
	persistFiredNodeRuns(ctx, f.runs, runID, &def, state)
	runStatus, runErrMsg := state.RunStatus()
	errText := runErrMsg
	if runErr != nil && errText == "" {
		errText = runErr.Error()
	}
	_ = f.runs.UpdateRunStatus(ctx, runID, runStatus, errText, &startedAt, &finishedAt)
	settled = true
	return runErr
}

// persistFiredNodeRuns mirrors api.persistNodeRuns. We duplicate the few
// lines here rather than export the API package's helper because the firer
// is in main and pulling the api package in just for one helper would invert
// the import direction.
func persistFiredNodeRuns(ctx context.Context, repo *storage.WorkflowRunRepo, runID string, wf *executor.Workflow, state *executor.RunState) {
	for _, node := range wf.Nodes {
		records := state.History(node.ID)
		if len(records) == 0 {
			status := state.Status(node.ID)
			if status == "" {
				status = "skipped"
			}
			_ = repo.InsertNodeRun(ctx, storage.NodeRun{
				ID:            idgen.NewNodeRun(),
				WorkflowRunID: runID,
				NodeID:        node.ID,
				Iteration:     0,
				Status:        status,
				Attempts:      0,
			})
			continue
		}
		inputBytes, _ := json.Marshal(state.Input(node.ID))
		for _, rec := range records {
			outputBytes, _ := json.Marshal(rec.Output)
			_ = repo.InsertNodeRun(ctx, storage.NodeRun{
				ID:            idgen.NewNodeRun(),
				WorkflowRunID: runID,
				NodeID:        node.ID,
				Iteration:     rec.Iteration,
				Status:        rec.Status,
				Input:         inputBytes,
				Output:        outputBytes,
				Attempts:      rec.Attempts,
			})
		}
	}
}

// triggerResolver answers webhook.Handler lookups against the triggers table.
// It only returns enabled webhook rows so a misconfigured URL cannot fire a
// disabled trigger and so a cron row can never be reached via /webhooks/...
// even if the caller happens to know its id.
type triggerResolver struct {
	repo *storage.TriggerRepo
}

func (r *triggerResolver) ResolveWebhook(ctx context.Context, id string) (webhook.WebhookTrigger, bool, error) {
	trg, err := r.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return webhook.WebhookTrigger{}, false, nil
		}
		return webhook.WebhookTrigger{}, false, err
	}
	if trg.Kind != "webhook" || !trg.Enabled {
		return webhook.WebhookTrigger{}, false, nil
	}
	var cfg struct {
		Token  string `json:"token"`
		Secret string `json:"secret"`
	}
	_ = json.Unmarshal(trg.Config, &cfg)
	var secret []byte
	if cfg.Secret != "" {
		secret = []byte(cfg.Secret)
	}
	return webhook.WebhookTrigger{
		ID:         trg.ID,
		WorkflowID: trg.WorkflowID,
		Token:      cfg.Token,
		Secret:     secret,
	}, true, nil
}
