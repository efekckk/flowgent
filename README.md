# Flowgent

AI-first workflow automation. You describe what you want in plain language, AI builds the workflow.

> Status: pre-alpha. **DAG + Parallel (M3) complete.** See `docs/plans/` for the next milestone.

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

| Method | Path                          | Description                              |
|--------|-------------------------------|------------------------------------------|
| GET    | /health                       | Liveness probe                           |
| POST   | /v1/auth/signup               | Create user + default workspace + session |
| POST   | /v1/auth/login                | Issue session for existing user          |
| POST   | /v1/auth/logout               | Invalidate session                       |
| GET    | /v1/me                        | Current user (auth required)             |
| POST   | /v1/workflows                 | Create workflow (draft, v1)              |
| GET    | /v1/workflows/{id}            | Read workflow with current definition    |
| POST   | /v1/workflows/{id}/run        | Execute workflow manually                |

## Available tools

| Slug          | Category   | What it does |
|---------------|------------|--------------|
| http.request  | core       | HTTP GET/POST/etc. with classified retry errors |
| core.if       | control    | Compares two values, routes to true/false port  |
| core.set      | transform  | Emits its `values` map as the node output       |
| core.wait     | control    | Sleeps N ms, honours ctx cancellation           |
| core.merge    | control    | Combines upstream outputs (append/merge/first)  |
| core.loop     | control    | Iterates over an array, runs body per item      |
| core.code     | transform  | Sandboxed JS snippet (goja, CPU+mem-limited)    |

## Engine capabilities

- Parallel branch execution: up to 8 nodes per wave on a goroutine pool
- Loop coordination: `core.loop` with explicit body node list
- Merge synchronisation: `core.merge` waits for all upstream branches
- Error edge routing: any node failure with an `error` outgoing edge is routed instead of failing the run
- Retry policy: classified errors (`rate_limited`, `transient_5xx`) trigger exponential backoff up to N attempts; `auth_failed` and `validation` short-circuit

## Expressions

Inside node params, use `{{ $trigger.field }}`, `{{ $nodes.<id>.<field> }}`, `{{ $now }}`, or any expr-lang expression. A param whose entire value is a single `{{ ... }}` keeps its native type (int / map / etc.); embedded expressions are stringified into the surrounding template. Inside a loop body, `$nodes.<loop_id>.current` and `$nodes.<loop_id>.index` carry the current iteration's value and index.

## Roadmap

See `docs/specs/2026-06-12-flowgent-design.md` for full design. Milestone status:

- [x] M1 Foundation — auth, sessions, base storage
- [x] M2 Engine core — tool registry, expression engine, first primitive nodes
- [x] M3 DAG + paralel — loop, merge, code sandbox, parallel branches
- [ ] M4 AI agent — provider abstraction, workflow generation loop
- [ ] M5 UI shell — React + ReactFlow + chat panel
- [ ] M6 Credentials + LLM integration
- [ ] M7 Integrations wave 1 — Slack, Telegram, Email, Postgres, Sheets
- [ ] M8 Triggers + run viewer — cron, webhook, live log
- [ ] M9 Polish & onboarding
- [ ] M10 Self-host & launch

## License

Apache 2.0. See [LICENSE](LICENSE).
