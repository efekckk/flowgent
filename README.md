# Flowgent

AI-first workflow automation. You describe what you want in plain language, AI builds the workflow.

> Status: pre-alpha. **Integrations wave 1 (M7) complete.** See `docs/plans/` for the next milestone.

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

Open `http://localhost:8080`. Sign up, add credentials (OpenAI/Anthropic for the LLM; Slack incoming-webhook URL, Telegram bot token, SMTP creds, or a Postgres DSN for outputs), and describe a workflow ("Every time someone posts on /webhook, summarise with GPT-4o, then ping #general on Slack"). The agent wires it up; click Run.

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

| Slug                   | Category       | What it does |
|------------------------|----------------|--------------|
| http.request           | core           | HTTP GET/POST/etc. with classified retry errors |
| core.if                | control        | Compares two values, routes to true/false port  |
| core.set               | transform/data | Emits its `values` map as the node output       |
| core.wait              | control        | Sleeps N ms, honours ctx cancellation           |
| core.merge             | control        | Combines upstream outputs (append/merge/first)  |
| core.loop              | control        | Iterates over an array, runs body per item      |
| core.code              | transform/code | Sandboxed JS snippet (goja, CPU+mem-limited)    |
| llm.chat               | ai             | Calls a chat provider via a workspace credential|
| slack.send_message     | communication  | Posts to a Slack incoming webhook               |
| telegram.send_message  | communication  | Sends via Telegram Bot API                      |
| email.smtp_send        | communication  | SMTP PLAIN auth send via Go stdlib              |
| postgres.query         | data           | Parameterised SQL query via pgx                 |

## Credential types

| Type            | Fields                                          | Used by                |
|-----------------|-------------------------------------------------|------------------------|
| openai          | api_key                                         | llm.chat               |
| anthropic       | api_key                                         | llm.chat               |
| slack_webhook   | url                                             | slack.send_message     |
| telegram_bot    | bot_token                                       | telegram.send_message  |
| smtp            | host, port, username, password, from            | email.smtp_send        |
| postgres        | dsn                                             | postgres.query         |

All credentials are encrypted at rest with AES-256-GCM (master key from `FLOWGENT_CRED_KEY`). Plaintext never leaves process memory.

## Engine capabilities

- Parallel branch execution: up to 8 nodes per wave on a goroutine pool
- Loop coordination: `core.loop` with explicit body node list
- Merge synchronisation: `core.merge` waits for all upstream branches
- Error edge routing: any node failure with an `error` outgoing edge is routed instead of failing the run
- Retry policy: classified errors (`rate_limited`, `transient_5xx`) trigger exponential backoff; `auth_failed` and `validation` short-circuit
- Credential injection: any node with a `credential` reference receives the decrypted secret in `input.__credential`

## AI agent

Send a chat message via the UI or `POST /v1/workflows/:id/chat`. The agent validates each proposal (tool slugs known, edge targets valid, no cycles) and re-prompts the model up to 3 times if validation fails. Supported providers: **OpenAI** and **Anthropic**.

## Expressions

Inside node params: `{{ $trigger.field }}`, `{{ $nodes.<id>.<field> }}`, `{{ $now }}`, or any expr-lang expression. A param whose entire value is a single `{{ ... }}` keeps its native type. Inside a loop body, `$nodes.<loop_id>.current` and `$nodes.<loop_id>.index` carry the iteration value and index.

## Roadmap

Milestone status:

- [x] M1 Foundation — auth, sessions, base storage
- [x] M2 Engine core — tool registry, expression engine, first primitive nodes
- [x] M3 DAG + paralel — loop, merge, code sandbox, parallel branches
- [x] M4 AI agent — provider abstraction, workflow generation loop
- [x] M5 UI shell — React + ReactFlow + chat panel
- [x] M6 Credentials + LLM integration — encrypted credentials store, llm.chat node
- [x] M7 Integrations wave 1 — Slack, Telegram, Email, Postgres
- [ ] M8 Triggers + run viewer — cron, webhook, live log
- [ ] M9 Polish & onboarding
- [ ] M10 Self-host & launch

> M7.1 will add Google Sheets via service-account JWT; OAuth-backed integrations (Slack OAuth bot mode, Google Workspace) target v1.x.

## License

Apache 2.0. See [LICENSE](LICENSE).
