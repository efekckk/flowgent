package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/efekckk/flowgent/internal/api"
	"github.com/efekckk/flowgent/internal/auth"
	"github.com/efekckk/flowgent/internal/logging"
	"github.com/efekckk/flowgent/internal/storage"
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

	srv := api.NewServer(api.Deps{
		Users:        storage.NewUserRepo(pg.Pool),
		Workspaces:   storage.NewWorkspaceRepo(pg.Pool),
		Sessions:     storage.NewSessionRepo(pg.Pool),
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
