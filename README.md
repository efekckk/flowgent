# Flowgent

AI-first workflow automation. You describe what you want in plain language, AI builds the workflow.

> Status: pre-alpha. **Web UI (M5) complete.** See `docs/plans/` for the next milestone.

## Run locally

```bash
docker compose up -d postgres
npm --prefix web install
npm --prefix web run build
cp -r web/dist/. internal/webfs/dist/

export DATABASE_URL="postgres://flowgent:flowgent@localhost:5432/flowgent?sslmode=disable"
export FLOWGENT_OPENAI_KEY="sk-..."        # or FLOWGENT_ANTHROPIC_KEY
export FLOWGENT_DEFAULT_PROVIDER="openai"  # or "anthropic"
go run ./cmd/flowgent
```

Open `http://localhost:8080` in a browser. Sign up, click **+ New workflow**, then chat with the assistant ("Build me a workflow that pings example.com every minute and emails me on failure"). The proposal lands on the canvas; click **Run now** to execute.

## Test

```bash
go test ./... -race
npm --prefix web test
```

Backend integration tests use [dockertest](https://github.com/ory/dockertest); they auto-skip on machines without Docker.

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
| POST   | /v1/workflows/{id}/chat       | Chat with AI agent; SSE stream of events |

Everything else under `/` is served by the React SPA.

## Available tools

| Slug          | Category       | What it does |
|---------------|----------------|--------------|
| http.request  | core           | HTTP GET/POST/etc. with classified retry errors |
| core.if       | control        | Compares two values, routes to true/false port  |
| core.set      | transform/data | Emits its `values` map as the node output       |
| core.wait     | control        | Sleeps N ms, honours ctx cancellation           |
| core.merge    | control        | Combines upstream outputs (append/merge/first)  |
| core.loop     | control        | Iterates over an array, runs body per item      |
| core.code     | transform/code | Sandboxed JS snippet (goja, CPU+mem-limited)    |

## Engine capabilities

- Parallel branch execution: up to 8 nodes per wave on a goroutine pool
- Loop coordination: `core.loop` with explicit body node list
- Merge synchronisation: `core.merge` waits for all upstream branches
- Error edge routing: any node failure with an `error` outgoing edge is routed instead of failing the run
- Retry policy: classified errors (`rate_limited`, `transient_5xx`) trigger exponential backoff; `auth_failed` and `validation` short-circuit

## AI agent

Send a chat message via the UI or `POST /v1/workflows/:id/chat`. The agent validates each proposal (tool slugs known, edge targets valid, no cycles) and re-prompts the model up to 3 times if validation fails. Supported providers: **OpenAI** (`FLOWGENT_OPENAI_KEY`) and **Anthropic** (`FLOWGENT_ANTHROPIC_KEY`). Gemini and Ollama are queued for v1.x.

## Expressions

Inside node params: `{{ $trigger.field }}`, `{{ $nodes.<id>.<field> }}`, `{{ $now }}`, or any expr-lang expression. A param whose entire value is a single `{{ ... }}` keeps its native type. Inside a loop body, `$nodes.<loop_id>.current` and `$nodes.<loop_id>.index` carry the iteration value and index.

## Roadmap

See `docs/specs/2026-06-12-flowgent-design.md` for full design. Milestone status:

- [x] M1 Foundation — auth, sessions, base storage
- [x] M2 Engine core — tool registry, expression engine, first primitive nodes
- [x] M3 DAG + paralel — loop, merge, code sandbox, parallel branches
- [x] M4 AI agent — provider abstraction, workflow generation loop
- [x] M5 UI shell — React + ReactFlow + chat panel
- [ ] M6 Credentials + LLM integration
- [ ] M7 Integrations wave 1 — Slack, Telegram, Email, Postgres, Sheets
- [ ] M8 Triggers + run viewer — cron, webhook, live log
- [ ] M9 Polish & onboarding
- [ ] M10 Self-host & launch

## License

Apache 2.0. See [LICENSE](LICENSE).
