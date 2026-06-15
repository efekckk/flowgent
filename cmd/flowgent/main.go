package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/efekckk/flowgent/internal/api"
	"github.com/efekckk/flowgent/internal/auth"
	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/logging"
	"github.com/efekckk/flowgent/internal/registry"
	"github.com/efekckk/flowgent/internal/storage"

	coreif "github.com/efekckk/flowgent/tools/core.if"
	coreset "github.com/efekckk/flowgent/tools/core.set"
	corewait "github.com/efekckk/flowgent/tools/core.wait"
	httprequest "github.com/efekckk/flowgent/tools/http.request"
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

	reg := registry.New()
	reg.Register("core.wait", corewait.New())
	reg.Register("core.set", coreset.New())
	reg.Register("core.if", coreif.New())
	reg.Register("http.request", httprequest.New())
	if err := reg.LoadFromDir(envOr("FLOWGENT_TOOLS_DIR", "./tools")); err != nil {
		logger.Error("tool registry", "err", err)
		os.Exit(1)
	}
	logger.Info("tool registry loaded", "count", len(reg.List()))

	engine := executor.NewEngine(reg)

	srv := api.NewServer(api.Deps{
		Users:        storage.NewUserRepo(pg.Pool),
		Workspaces:   storage.NewWorkspaceRepo(pg.Pool),
		Sessions:     storage.NewSessionRepo(pg.Pool),
		Workflows:    storage.NewWorkflowRepo(pg.Pool),
		Runs:         storage.NewWorkflowRunRepo(pg.Pool),
		Engine:       engine,
		Throttle:     auth.NewLoginThrottle(5, 15*time.Minute, time.Now),
		CookieDomain: envOr("SESSION_COOKIE_DOMAIN", "localhost"),
		CookieSecure: envOr("SESSION_COOKIE_SECURE", "false") == "true",
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
