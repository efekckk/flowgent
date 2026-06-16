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
	"github.com/efekckk/flowgent/internal/logging"
	"github.com/efekckk/flowgent/internal/provider"
	"github.com/efekckk/flowgent/internal/registry"
	"github.com/efekckk/flowgent/internal/storage"

	corecode "github.com/efekckk/flowgent/tools/core.code"
	coreif "github.com/efekckk/flowgent/tools/core.if"
	coreloop "github.com/efekckk/flowgent/tools/core.loop"
	coremerge "github.com/efekckk/flowgent/tools/core.merge"
	coreset "github.com/efekckk/flowgent/tools/core.set"
	corewait "github.com/efekckk/flowgent/tools/core.wait"
	httprequest "github.com/efekckk/flowgent/tools/http.request"
	llmchat "github.com/efekckk/flowgent/tools/llm.chat"
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

	engine := executor.NewEngine(reg, executor.WithCredentialResolver(credResolver))

	srv := api.NewServer(api.Deps{
		Users:         storage.NewUserRepo(pg.Pool),
		Workspaces:    storage.NewWorkspaceRepo(pg.Pool),
		Sessions:      storage.NewSessionRepo(pg.Pool),
		Workflows:     storage.NewWorkflowRepo(pg.Pool),
		Runs:          storage.NewWorkflowRunRepo(pg.Pool),
		Engine:        engine,
		ChatThreads:   storage.NewChatThreadRepo(pg.Pool),
		ChatMessages:  storage.NewChatMessageRepo(pg.Pool),
		Agent:         ag,
		Throttle:      auth.NewLoginThrottle(5, 15*time.Minute, time.Now),
		CookieDomain:  envOr("SESSION_COOKIE_DOMAIN", "localhost"),
		CookieSecure:  envOr("SESSION_COOKIE_SECURE", "false") == "true",
		Credentials:   credRepo,
		CredentialKey: masterKey,
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
