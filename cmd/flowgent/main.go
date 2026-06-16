package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/efekckk/flowgent/internal/agent"
	"github.com/efekckk/flowgent/internal/api"
	"github.com/efekckk/flowgent/internal/auth"
	"github.com/efekckk/flowgent/internal/crypto"
	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/idgen"
	"github.com/efekckk/flowgent/internal/logging"
	"github.com/efekckk/flowgent/internal/provider"
	"github.com/efekckk/flowgent/internal/registry"
	"github.com/efekckk/flowgent/internal/runlog"
	"github.com/efekckk/flowgent/internal/scheduler"
	"github.com/efekckk/flowgent/internal/storage"
	"github.com/efekckk/flowgent/internal/webhook"

	corecode "github.com/efekckk/flowgent/tools/core.code"
	coreif "github.com/efekckk/flowgent/tools/core.if"
	coreloop "github.com/efekckk/flowgent/tools/core.loop"
	coremerge "github.com/efekckk/flowgent/tools/core.merge"
	coreset "github.com/efekckk/flowgent/tools/core.set"
	corewait "github.com/efekckk/flowgent/tools/core.wait"
	emailsmtp "github.com/efekckk/flowgent/tools/email.smtp_send"
	httprequest "github.com/efekckk/flowgent/tools/http.request"
	llmchat "github.com/efekckk/flowgent/tools/llm.chat"
	postgresquery "github.com/efekckk/flowgent/tools/postgres.query"
	slacksend "github.com/efekckk/flowgent/tools/slack.send_message"
	telegramsend "github.com/efekckk/flowgent/tools/telegram.send_message"
)

func main() {
	logger := logging.NewLogger(os.Stdout, envOr("LOG_LEVEL", "info"))

	dsn := envOr("DATABASE_URL", "")
	if dsn == "" {
		logger.Error("DATABASE_URL is required")
		os.Exit(1)
	}
	if err := storage.Migrate(dsn); err != nil {
		logger.Error("migrate failed", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pg, err := storage.Open(ctx, dsn)
	if err != nil {
		logger.Error("open postgres", "err", err)
		os.Exit(1)
	}
	defer pg.Close()

	masterKeyB64 := envOr("FLOWGENT_CRED_KEY", "")
	if masterKeyB64 == "" {
		logger.Error("FLOWGENT_CRED_KEY is required (base64-encoded 32-byte key)")
		os.Exit(1)
	}
	masterKey, err := crypto.ParseMasterKey(masterKeyB64)
	if err != nil {
		logger.Error("master key invalid", "err", err)
		os.Exit(1)
	}

	credRepo := storage.NewCredentialRepo(pg.Pool)

	credResolver := &workflowCredentialResolver{
		repo: credRepo,
		key:  masterKey,
	}

	reg := registry.New()
	reg.Register("core.wait", corewait.New())
	reg.Register("core.set", coreset.New())
	reg.Register("core.if", coreif.New())
	reg.Register("http.request", httprequest.New())
	reg.Register("core.merge", coremerge.New())
	reg.Register("core.loop", coreloop.New())
	reg.Register("core.code", corecode.New())
	reg.Register("slack.send_message", slacksend.New())
	reg.Register("telegram.send_message", telegramsend.New())
	reg.Register("email.smtp_send", emailsmtp.New())
	reg.Register("postgres.query", postgresquery.New())
	if err := reg.LoadFromDir(envOr("FLOWGENT_TOOLS_DIR", "./tools")); err != nil {
		logger.Error("tool registry", "err", err)
		os.Exit(1)
	}
	logger.Info("tool registry loaded", "count", len(reg.List()))

	provReg := provider.NewRegistry()
	defaultProviderSlug := envOr("FLOWGENT_DEFAULT_PROVIDER", "openai")
	var prov provider.ChatProvider
	if p, err := provReg.For(defaultProviderSlug); err == nil {
		prov = p
		logger.Info("provider configured", "slug", defaultProviderSlug)
	} else {
		logger.Warn("provider unavailable, chat endpoint will return errors", "err", err)
	}

	llmProviderResolver := &llmProviderResolverImpl{
		repo:        credRepo,
		key:         masterKey,
		providerReg: provReg,
	}

	reg.Register("llm.chat", llmchat.New(llmProviderResolver))

	knownTools := map[string]struct{}{}
	for _, m := range reg.List() {
		knownTools[m.Slug] = struct{}{}
	}
	ag := agent.New(agent.Deps{
		Provider:   prov,
		KnownTools: knownTools,
		MaxRetries: 3,
	})

	workflowRepo := storage.NewWorkflowRepo(pg.Pool)
	runRepo := storage.NewWorkflowRunRepo(pg.Pool)
	runLogRepo := storage.NewRunLogRepo(pg.Pool)

	streamer := runlog.New()
	logSink := &compositeSink{repo: runLogRepo, streamer: streamer}

	engine := executor.NewEngine(reg,
		executor.WithCredentialResolver(credResolver),
		executor.WithRunStore(&engineRunStore{workflows: workflowRepo, runs: runRepo}),
		executor.WithLogSink(logSink),
	)

	triggerRepo := storage.NewTriggerRepo(pg.Pool)
	firer := &productionFirer{
		triggers:  triggerRepo,
		workflows: workflowRepo,
		runs:      runRepo,
		engine:    engine,
	}
	sched := scheduler.New(firer)
	loadCtx, loadCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := sched.LoadFromDB(loadCtx, triggerLoader{repo: triggerRepo}); err != nil {
		logger.Warn("scheduler load failed; cron triggers will not fire until next restart", "err", err)
	}
	loadCancel()
	sched.Start()
	defer sched.Stop()

	webhookHandler := webhook.NewHandler(&triggerResolver{repo: triggerRepo}, firer)

	srv := api.NewServer(api.Deps{
		Users:         storage.NewUserRepo(pg.Pool),
		Workspaces:    storage.NewWorkspaceRepo(pg.Pool),
		Sessions:      storage.NewSessionRepo(pg.Pool),
		Workflows:     workflowRepo,
		Runs:          runRepo,
		RunLogs:       runLogRepo,
		Streamer:      streamer,
		Engine:        engine,
		ChatThreads:   storage.NewChatThreadRepo(pg.Pool),
		ChatMessages:  storage.NewChatMessageRepo(pg.Pool),
		Agent:         ag,
		Triggers:      triggerRepo,
		Scheduler:     sched,
		Throttle:      auth.NewLoginThrottle(5, 15*time.Minute, time.Now),
		CookieDomain:  envOr("SESSION_COOKIE_DOMAIN", "localhost"),
		CookieSecure:  envOr("SESSION_COOKIE_SECURE", "false") == "true",
		Credentials:    credRepo,
		CredentialKey:  masterKey,
		PublicBaseURL:  envOr("FLOWGENT_PUBLIC_BASE_URL", "http://localhost:8080"),
		WebhookHandler: webhookHandler,
	})

	addr := ":" + envOr("PORT", "8080")
	logger.Info("flowgent listening", "addr", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func envOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

type workflowCredentialResolver struct {
	repo *storage.CredentialRepo
	key  []byte
}

func (r *workflowCredentialResolver) Resolve(ctx context.Context, credentialRef string) (map[string]any, error) {
	cred, err := r.repo.Get(ctx, credentialRef)
	if err != nil {
		return nil, err
	}
	plain, err := crypto.Decrypt(cred.Encrypted, r.key)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	var secret map[string]any
	if err := json.Unmarshal(plain, &secret); err != nil {
		return nil, fmt.Errorf("parse secret: %w", err)
	}
	secret["__type"] = cred.Type
	secret["__id"] = cred.ID
	return secret, nil
}

type llmProviderResolverImpl struct {
	repo        *storage.CredentialRepo
	key         []byte
	providerReg *provider.Registry
}

// triggerLoader adapts the trigger repo to the scheduler.Loader contract. The
// scheduler only cares about (id, workflow_id, cron expression) tuples; the
// rest of the trigger row is irrelevant for boot-time registration.
type triggerLoader struct {
	repo *storage.TriggerRepo
}

func (l triggerLoader) LoadEnabledCronTriggers(ctx context.Context) ([]scheduler.LoadedTrigger, error) {
	rows, err := l.repo.ListEnabledByKind(ctx, "cron")
	if err != nil {
		return nil, err
	}
	out := make([]scheduler.LoadedTrigger, 0, len(rows))
	for _, row := range rows {
		var cfg struct {
			Cron string `json:"cron"`
		}
		_ = json.Unmarshal(row.Config, &cfg)
		out = append(out, scheduler.LoadedTrigger{ID: row.ID, WorkflowID: row.WorkflowID, Expression: cfg.Cron})
	}
	return out, nil
}

func (r *llmProviderResolverImpl) ResolveForNodeCredential(_ context.Context, input map[string]any) (provider.ChatProvider, error) {
	cred, ok := input["__credential"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing __credential")
	}
	credType, _ := cred["__type"].(string)
	if credType == "" {
		return nil, fmt.Errorf("credential missing __type")
	}
	encoded, err := json.Marshal(cred)
	if err != nil {
		return nil, err
	}
	return r.providerReg.ForCredential(credType, encoded)
}

// engineRunStore adapts the storage repos to executor.RunStore. It is the
// production hand-off point between the engine's RunFromReplay path and
// the workflow_runs / node_runs tables. The viewer's other endpoints
// (list/get/logs) talk to the repos directly; only replay needs this
// indirection because the engine creates the row itself.
type engineRunStore struct {
	workflows *storage.WorkflowRepo
	runs      *storage.WorkflowRunRepo
}

func (s *engineRunStore) LoadWorkflowForRun(ctx context.Context, workflowID string) (int, *executor.Workflow, error) {
	wf, err := s.workflows.Get(ctx, workflowID)
	if err != nil {
		return 0, nil, err
	}
	ver, err := s.workflows.GetVersion(ctx, wf.ID, wf.CurrentVersion)
	if err != nil {
		return 0, nil, err
	}
	var def executor.Workflow
	if err := json.Unmarshal(ver.Definition, &def); err != nil {
		return 0, nil, fmt.Errorf("parse definition: %w", err)
	}
	return wf.CurrentVersion, &def, nil
}

func (s *engineRunStore) InsertRun(ctx context.Context, p executor.InsertRunParams) error {
	startedAt := p.StartedAt
	return s.runs.NewRun(ctx, storage.WorkflowRun{
		ID:              p.ID,
		WorkflowID:      p.WorkflowID,
		WorkflowVersion: p.WorkflowVersion,
		Status:          "running",
		TriggerKind:     p.TriggerKind,
		TriggerPayload:  p.TriggerPayload,
		ParentRunID:     p.ParentRunID,
		StartedAt:       &startedAt,
	})
}

func (s *engineRunStore) PersistRun(ctx context.Context, runID string, wf *executor.Workflow, state *executor.RunState,
	status, errMsg string, startedAt, finishedAt time.Time) error {
	for _, node := range wf.Nodes {
		records := state.History(node.ID)
		if len(records) == 0 {
			st := state.Status(node.ID)
			if st == "" {
				st = "skipped"
			}
			_ = s.runs.InsertNodeRun(ctx, storage.NodeRun{
				ID:            idgen.NewNodeRun(),
				WorkflowRunID: runID,
				NodeID:        node.ID,
				Iteration:     0,
				Status:        st,
				Attempts:      0,
			})
			continue
		}
		inputBytes, _ := json.Marshal(state.Input(node.ID))
		for _, rec := range records {
			outputBytes, _ := json.Marshal(rec.Output)
			_ = s.runs.InsertNodeRun(ctx, storage.NodeRun{
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
	return s.runs.UpdateRunStatus(ctx, runID, status, errMsg, &startedAt, &finishedAt)
}

func (s *engineRunStore) GetTriggerPayload(ctx context.Context, runID string) (json.RawMessage, error) {
	run, err := s.runs.Get(ctx, runID)
	if err != nil {
		return nil, err
	}
	return run.TriggerPayload, nil
}

// compositeSink implements executor.LogSink by writing each event to BOTH
// the database (for replay, search, and reconnecting clients) and the
// in-process Streamer (for live SSE subscribers). Sink errors are swallowed
// to honour the LogSink contract that a sink failure never crashes a run.
type compositeSink struct {
	repo     *storage.RunLogRepo
	streamer *runlog.Streamer
}

func (c *compositeSink) Append(ctx context.Context, runID, nodeID, level, message string) {
	if c.repo != nil {
		_ = c.repo.Append(ctx, storage.RunLog{
			RunID:   runID,
			NodeID:  nodeID,
			Level:   level,
			Message: message,
		})
	}
	if c.streamer != nil {
		c.streamer.Publish(ctx, runlog.Event{
			RunID:   runID,
			NodeID:  nodeID,
			Level:   level,
			Message: message,
			At:      time.Now().UTC().Format(time.RFC3339Nano),
		})
	}
}
