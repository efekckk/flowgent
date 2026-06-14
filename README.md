# Flowgent

AI-first workflow automation. You describe what you want in plain language, AI builds the workflow.

> Status: pre-alpha. **Foundation (M1) complete.** See `docs/plans/` for the next milestone.

## Run locally

```bash
docker compose up -d postgres
export DATABASE_URL="postgres://flowgent:flowgent@localhost:5432/flowgent?sslmode=disable"
go run ./cmd/flowgent
```

Health check:

```
curl localhost:8080/health
# {"status":"ok"}
```

## Test

```bash
go test ./... -race
```

Integration tests use [dockertest](https://github.com/ory/dockertest); they auto-skip on machines without Docker.

## Current API surface

| Method | Path                | Description                              |
|--------|---------------------|------------------------------------------|
| GET    | /health             | Liveness probe                           |
| POST   | /v1/auth/signup     | Create user + default workspace + session |
| POST   | /v1/auth/login      | Issue session for existing user          |
| POST   | /v1/auth/logout     | Invalidate session                       |
| GET    | /v1/me              | Current user (auth required)             |

## Roadmap

See `docs/specs/2026-06-12-flowgent-design.md` for full design. Milestone status:

- [x] M1 Foundation — auth, sessions, base storage
- [ ] M2 Engine core — tool registry, expression engine, first primitive nodes
- [ ] M3 DAG + paralel — loop, merge, code sandbox, parallel branches
- [ ] M4 AI agent — provider abstraction, workflow generation loop
- [ ] M5 UI shell — React + ReactFlow + chat panel
- [ ] M6 Credentials + LLM integration
- [ ] M7 Integrations wave 1 — Slack, Telegram, Email, Postgres, Sheets
- [ ] M8 Triggers + run viewer — cron, webhook, live log
- [ ] M9 Polish & onboarding
- [ ] M10 Self-host & launch

## License

Apache 2.0. See [LICENSE](LICENSE).
