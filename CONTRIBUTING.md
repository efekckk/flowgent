# Contributing to Flowgent

Thanks for taking a look. Flowgent is pre-alpha; the scope is moving and the bar for "small PR" is low — we'd rather review three focused changes than one omnibus.

## Bootstrap

```bash
git clone https://github.com/efekckk/flowgent.git
cd flowgent
make env          # writes .env with a fresh FLOWGENT_CRED_KEY
make dev          # postgres in docker, server on host
```

Open `http://localhost:8080`, sign up, add an OpenAI credential, draft a workflow.

## Layout

```
cmd/flowgent/        — main, wiring, production firer + RunStore adapter
internal/api/        — chi router, HTTP handlers, ownership guards
internal/auth/       — sessions, password hashing, login throttle
internal/agent/      — AI agent loop (provider-agnostic)
internal/executor/   — engine, expression resolver, run state, RunStore contract
internal/scheduler/  — cron scheduler with Firer interface
internal/webhook/    — inbound webhook handler with HMAC verification
internal/runlog/     — in-process SSE streamer (slow-subscriber drop, not block)
internal/storage/    — pgx repos + migrations
internal/webfs/      — embedded React SPA (vite build output goes into dist/)
internal/idgen/      — prefix-stamped ID generators
internal/provider/   — LLM provider abstraction (OpenAI / Anthropic)
tools/<slug>/        — one directory per executable tool node
web/                 — React 18 + ReactFlow + Zustand + Tailwind frontend
docs/plans/          — per-milestone implementation plans
docs/launch/         — (untracked) screencast script, Show HN draft
```

## Running tests

```bash
make test                       # everything
go test ./... -race             # backend only
npm --prefix web test           # web only
npm --prefix web run typecheck  # tsc --noEmit
```

Backend integration tests need Docker available; they auto-skip otherwise. Web tests run on jsdom via Vitest.

## Style

Backend:

- Standard library first; reach for a dependency when the alternative is hand-rolled cryptography, parsing, or pgx wiring.
- Comments explain *why*, not *what*. A non-obvious invariant, an external constraint, or a surprising design choice — yes. A one-line restatement of the next line — no.
- Errors stay specific (`fmt.Errorf("storage: insert run: %w", err)`); avoid generic `errors.New("something went wrong")`.
- Public API methods are documented with a real sentence; private helpers only when the contract isn't obvious from the signature.

Frontend:

- TypeScript strict; no `any` without a `// reason:` comment.
- Tailwind utility classes inline; no per-file CSS modules.
- Tests use Testing Library queries (`getByLabelText`, `getByRole`) — not raw selectors.
- One Zustand store per page; cross-page state lives in `AuthProvider`.

## Pull requests

1. **Branch off `main`**. PR titles use conventional-commit prefixes (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`).
2. **Keep PRs focused**. One feature or one fix per PR. If it touches more than ~500 lines or 8 files, consider splitting.
3. **Add tests for behaviour changes.** Backend: a failing test before the fix proves the bug existed. Web: at least one Testing Library check that the new copy / state renders.
4. **Run `make test` locally before pushing.** CI does the same; catching it locally is faster than a round-trip.
5. **Commit messages**: imperative mood, no trailing period (`fix(api): close orphan run-row window`, not `fix(api) Closed orphan...`). Group unrelated changes into separate commits.
6. **Don't** bump versions or edit changelog entries unless the PR is the release itself.

## Reviewing your own code first

Before opening a PR, look at your diff with `git diff main`. Ask:

- Could a future-you read this without context? Comments cover the *why*; identifier names cover the *what*.
- Is there a test that would have caught the bug you just fixed?
- Did you accidentally include a debugging `fmt.Println`, `console.log`, or commented-out block?
- Are there exported identifiers (`Foo`, `Bar`) that should be unexported, or vice versa?

## Where to start

Open issues tagged [good first issue](https://github.com/efekckk/flowgent/labels/good%20first%20issue). The roadmap items that always need help:

- **More tools**: a new entry under `tools/<slug>/` with `tool.go`, `tool_test.go`, and a registration line in `internal/registry/`. Existing tools (slack, telegram, smtp) are the template.
- **Frontend polish**: empty states, error toasts, keyboard shortcuts on the canvas.
- **Docs**: anything that confused you while reading this file is fair game.

If you're not sure whether something is worth a PR, open an issue first — fast no's beat slow merges.

## License

By contributing, you agree your changes are released under the Apache 2.0 license that covers the rest of the project.
