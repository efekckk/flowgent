# Flowgent

> **Workflow automation, drafted in plain language.**
> You describe what you want; an AI agent wires the nodes, validates the graph, and hands it back as an editable workflow.

<p align="center">
  <em>Hero screencast — coming with the launch post.</em>
  <!-- TODO: drop hero.gif into docs/launch/assets/ and reference here -->
</p>

[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26-00ADD8)](go.mod)
[![Tests](https://img.shields.io/badge/tests-passing-success)](#test)

---

## Why

Zapier is great until you outgrow it. n8n is great until you have to draw every node by hand. Flowgent sits in the middle: an **open-source, self-hostable** automation engine where the **interface is your sentence**. The agent proposes a workflow, you edit the graph, you ship.

| | Zapier | n8n | **Flowgent** |
|---|:---:|:---:|:---:|
| Self-host | – | ✓ | ✓ |
| Open source | – | ✓ | ✓ (Apache 2.0) |
| AI-native authoring | partial | – | **first-class** |
| Live log streaming | ✓ | ✓ | ✓ (SSE + full-text search) |
| Cron + Webhook + Manual triggers | ✓ | ✓ | ✓ |
| Replay past runs | – | – | ✓ |

## 60-second quickstart

```bash
git clone https://github.com/efekckk/flowgent.git
cd flowgent
make demo
```

`make demo` writes a `.env` with a fresh `FLOWGENT_CRED_KEY`, builds the multi-stage Docker image (web + Go), and starts Postgres + the API on `http://localhost:8080`.

Open it, sign up, add an OpenAI credential, then tell the assistant what you want:

> *"Every time someone posts to /webhook, summarise the payload with GPT-4o, then ping `#general` on Slack."*

The agent proposes a workflow. You hit **Fire Run**, watch the live log stream, then click **Replay** to re-run with the same input.

Want hot reload? `make dev` runs Postgres in Docker and the API on the host with `go run`.

## What's inside

- **Visual editor** built on ReactFlow with a deliberate engineering-blueprint UI (dark canvas, monospace typography, corner brackets, isometric grid). It looks like a drafting table, not a clone of every other SaaS dashboard.
- **AI agent** (OpenAI + Anthropic) that validates each proposal (tool slugs known, edge targets valid, no cycles) and self-corrects up to 3 times.
- **Executor** with parallel branches (goroutine pool, 8 nodes per wave), loops, merges, classified retries (`rate_limited`, `transient_5xx`, `auth_failed`, `validation`), JS sandbox (goja, CPU+mem-limited).
- **Triggers**: manual, cron (5-field + `@hourly`/`@daily`/`@every Nm`), and webhook (with optional HMAC-SHA256 verification, constant-time).
- **Run viewer**: read-only canvas with per-node status colors, live SSE log stream (reconnect-safe via `?since=`), node IO inspection, replay.
- **Log search**: workspace-scoped full-text search backed by a Postgres generated `tsvector` column + GIN index, with `ts_headline` snippet highlighting.
- **Credentials**: encrypted at rest with AES-256-GCM. Plaintext never leaves the server's process memory.
- **Workspace isolation**: every workflow / run / trigger / credential endpoint resolves to a workspace and rejects cross-tenant access with a flat 404.

## Status

| Milestone | Done | What landed |
|---|:---:|---|
| M1 Foundation | ✓ | Auth, sessions, base storage |
| M2 Engine core | ✓ | Tool registry, expression engine |
| M3 DAG + parallel | ✓ | Loop, merge, JS sandbox, parallel branches |
| M4 AI agent | ✓ | Provider abstraction, generation loop |
| M5 UI shell | ✓ | React + ReactFlow + chat |
| M6 Credentials + LLM | ✓ | Encrypted store, `llm.chat` |
| M7 Integrations wave 1 | ✓ | Slack, Telegram, Email, Postgres |
| M8 Triggers + run viewer | ✓ | Cron, webhook, live log, full-text search |
| M9 Ownership + hardening | ✓ | Workspace IDOR sweep, replay orphan-row fix, audit-trail ctx fix |
| M10 Self-host & launch | ✓ | Docker compose, Makefile, multi-stage image, launch artifacts |

This is pre-alpha — schemas may still shift. The code is open while it bakes.

---

## Reference

### Manual setup (without Docker)

```bash
docker compose up -d postgres
npm --prefix web install
npm --prefix web run build
cp -r web/dist/. internal/webfs/dist/

export DATABASE_URL="postgres://flowgent:flowgent@localhost:5432/flowgent?sslmode=disable"
export FLOWGENT_CRED_KEY="$(openssl rand -base64 32)"
export FLOWGENT_OPENAI_KEY="sk-..."
export FLOWGENT_PUBLIC_BASE_URL="http://localhost:8080"
go run ./cmd/flowgent
```

### Make targets

| Target | What it does |
|---|---|
| `make env` | Generate `.env` from `.env.example` with a fresh `FLOWGENT_CRED_KEY` |
| `make demo` | `docker compose up --build` — full stack at `:8080` |
| `make dev` | Postgres in Docker, server on host with `go run` |
| `make build` | Web build + embedded Go binary at `./flowgent` |
| `make test` | `go test ./... -race` + `npm --prefix web test` |
| `make clean` | Drop binary + web `dist/` + node_modules |

### Test

```bash
go test ./... -race
npm --prefix web test
```

Backend integration tests use [dockertest](https://github.com/ory/dockertest); they auto-skip on machines without Docker.

### API surface

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
| POST   | /v1/workflows/{id}/triggers   | Create trigger (cron or webhook)         |
| GET    | /v1/workflows/{id}/triggers   | List triggers                            |
| PATCH  | /v1/triggers/{id}             | Update trigger (config, enabled)         |
| DELETE | /v1/triggers/{id}             | Delete trigger                           |
| GET    | /v1/workflows/{id}/runs       | List runs (status, from, to, limit, cursor) |
| GET    | /v1/runs/{id}                 | Run + node runs snapshot                 |
| GET    | /v1/runs/{id}/logs            | Run logs (polling fallback)              |
| GET    | /v1/runs/{id}/stream          | SSE log stream                           |
| POST   | /v1/runs/{id}/replay          | Replay a past run                        |
| GET    | /v1/workspaces/{id}/runs/search | Workspace-scoped log full-text search  |
| POST   | /webhooks/{trigger_id}/{token} | Inbound webhook trigger (no auth)       |

Everything else under `/` is served by the React SPA.

### Available tools

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

### Credential types

| Type            | Fields                                          | Used by                |
|-----------------|-------------------------------------------------|------------------------|
| openai          | api_key                                         | llm.chat               |
| anthropic       | api_key                                         | llm.chat               |
| slack_webhook   | url                                             | slack.send_message     |
| telegram_bot    | bot_token                                       | telegram.send_message  |
| smtp            | host, port, username, password, from            | email.smtp_send        |
| postgres        | dsn                                             | postgres.query         |

All credentials are encrypted at rest with AES-256-GCM (master key from `FLOWGENT_CRED_KEY`). Plaintext never leaves process memory.

### Triggers

| Kind     | Config                                  |
|----------|-----------------------------------------|
| manual   | none — Run button only                  |
| cron     | `cron` (5-field crontab or @shortcut)   |
| webhook  | optional `secret` (HMAC-SHA256)         |

Cron expressions support standard 5-field syntax (`0 9 * * *`) plus shortcuts (`@hourly`, `@daily`, `@weekly`, `@every 5m`). The scheduler boots from the `triggers` table; expressions are validated at create time so malformed configurations fail fast.

Webhooks are addressable at `<public_base>/webhooks/{trigger_id}/{token}`. When a secret is configured, requests must include `X-Flowgent-Signature: sha256=<hex>` computed over the raw body. Verification is constant-time.

### Run viewer

Every fire produces a `workflow_run` row plus a `node_run` per node executed. The Runs page lists history with filters (status, date range) and cursor pagination. Clicking a run opens the canvas in read-only mode with per-node status colors plus a tabbed right panel:

- **Logs** — live SSE stream from `/v1/runs/:id/stream`; reconnect-safe via `?since=<last_id>` catch-up from the database
- **Node IO** — pretty-printed input/output JSON for the selected node

The **Replay** button re-executes a past run with its original `trigger_payload`. Replayed runs carry `parent_run_id` so the history page can flag them.

### Log search

Run logs are indexed in Postgres via a generated `tsvector` column with a GIN index. The search bar in the app header queries `GET /v1/workspaces/:id/runs/search?q=<query>` and returns up to 100 matches with `ts_headline` snippets (server-side highlighting). Queries shorter than 3 characters are rejected to keep latency predictable.

### Engine capabilities

- Parallel branch execution: up to 8 nodes per wave on a goroutine pool
- Loop coordination: `core.loop` with explicit body node list
- Merge synchronisation: `core.merge` waits for all upstream branches
- Error edge routing: any node failure with an `error` outgoing edge is routed instead of failing the run
- Retry policy: classified errors (`rate_limited`, `transient_5xx`) trigger exponential backoff; `auth_failed` and `validation` short-circuit
- Credential injection: any node with a `credential` reference receives the decrypted secret in `input.__credential`

### AI agent

Send a chat message via the UI or `POST /v1/workflows/:id/chat`. The agent validates each proposal (tool slugs known, edge targets valid, no cycles) and re-prompts the model up to 3 times if validation fails. Supported providers: **OpenAI** and **Anthropic**.

### Expressions

Inside node params: `{{ $trigger.field }}`, `{{ $nodes.<id>.<field> }}`, `{{ $now }}`, or any expr-lang expression. A param whose entire value is a single `{{ ... }}` keeps its native type. Inside a loop body, `$nodes.<loop_id>.current` and `$nodes.<loop_id>.index` carry the iteration value and index.

### Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the build / test / PR loop.

## License

Apache 2.0. See [LICENSE](LICENSE).
