# Flowgent

AI-first workflow automation. You describe what you want in plain language, AI builds the workflow.

> Status: pre-alpha. **Credentials + LLM node (M6) complete.** See `docs/plans/` for the next milestone.

## Run locally

```bash
docker compose up -d postgres
npm --prefix web install
npm --prefix web run build
cp -r web/dist/. internal/webfs/dist/

export DATABASE_URL="postgres://flowgent:flowgent@localhost:5432/flowgent?sslmode=disable"
export FLOWGENT_CRED_KEY="$(openssl rand -base64 32)"
export FLOWGENT_OPENAI_KEY="sk-..."        # optional: env fallback for the agent chat endpoint
export FLOWGENT_DEFAULT_PROVIDER="openai"
go run ./cmd/flowgent
```

Open `http://localhost:8080`. Sign up, add an OpenAI or Anthropic key under **Credentials**, create a workflow, and chat with the assistant. Reference the credential in your prompt ("Use credential openai_default for the LLM node") and the assistant will wire it into the proposed workflow.

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
| POST   | /v1/credentials               | Create credential (encrypted at rest)    |
| GET    | /v1/credentials               | List workspace credentials               |
| DELETE | /v1/credentials/{id}          | Delete credential                        |

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
| llm.chat      | ai             | Calls a chat provider via a workspace credential|

## Credentials

API keys live in the `credentials` table, encrypted at rest with AES-256-GCM. The master key comes from `FLOWGENT_CRED_KEY` (base64-encoded 32 raw bytes). Workflows reference a credential by id; the engine decrypts the secret just before invoking the tool and injects it as `__credential` in the node input. Plaintext keys never leave the server's process memory.

Supported credential types in M6: **openai**, **anthropic**. OAuth-backed types (Slack, Google, Twilio) ship in v1.x.

## Engine capabilities

- Parallel branch execution: up to 8 nodes per wave on a goroutine pool
- Loop coordination: `core.loop` with explicit body node list
- Merge synchronisation: `core.merge` waits for all upstream branches
- Error edge routing: any node failure with an `error` outgoing edge is routed instead of failing the run
- Retry policy: classified errors (`rate_limited`, `transient_5xx`) trigger exponential backoff; `auth_failed` and `validation` short-circuit
- Credential injection: any node with a `credential` reference receives the decrypted secret in `input.__credential`

## AI agent

Send a chat message via the UI or `POST /v1/workflows/:id/chat`. The agent validates each proposal (tool slugs known, edge targets valid, no cycles) and re-prompts the model up to 3 times if validation fails. Supported providers: **OpenAI** (`FLOWGENT_OPENAI_KEY` for agent fallback) and **Anthropic** (`FLOWGENT_ANTHROPIC_KEY`). Workflows reference credentials by id; the env fallback is only used by the chat endpoint when no per-workflow credential is configured.

## Expressions

Inside node params: `{{ $trigger.field }}`, `{{ $nodes.<id>.<field> }}`, `{{ $now }}`, or any expr-lang expression. A param whose entire value is a single `{{ ... }}` keeps its native type. Inside a loop body, `$nodes.<loop_id>.current` and `$nodes.<loop_id>.index` carry the iteration value and index.

## Roadmap

See `docs/specs/2026-06-12-flowgent-design.md` for full design. Milestone status:

- [x] M1 Foundation — auth, sessions, base storage
- [x] M2 Engine core — tool registry, expression engine, first primitive nodes
- [x] M3 DAG + paralel — loop, merge, code sandbox, parallel branches
- [x] M4 AI agent — provider abstraction, workflow generation loop
- [x] M5 UI shell — React + ReactFlow + chat panel
- [x] M6 Credentials + LLM integration — encrypted credentials store, llm.chat node
- [ ] M7 Integrations wave 1 — Slack, Telegram, Email, Postgres, Sheets
- [ ] M8 Triggers + run viewer — cron, webhook, live log
- [ ] M9 Polish & onboarding
- [ ] M10 Self-host & launch

## License

Apache 2.0. See [LICENSE](LICENSE).
