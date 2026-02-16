# Etch Context — Implementation Task

You are working on a task as part of an implementation plan managed by Etch.

## Plan: Build Etch – AI Implementation Planning CLI
Etch is a Go CLI tool that helps developers create, review, and execute AI-generated implementation plans. The plan markdown file is the center of everything — it serves as the spec, the context source, and the shared document that humans review and approve. Progress tracking lives in separate per-session files so multiple agents can work simultaneously without conflicts.

## Current Plan State
Feature 1: Core Data Layer & Plan Parser
  ✓ Task 1.1: Project scaffold and CLI skeleton (completed)
  ✓ Task 1.2: Data models (completed)
  ✓ Task 1.3: Plan markdown parser — structure (completed)
  ✓ Task 1.3b: Plan parser — task metadata extraction (completed)
  ✓ Task 1.4: Plan markdown serializer (completed)
  ✓ Task 1.5: Progress file reader/writer (completed)
  ✓ Task 1.6: Config management (completed)
Feature 2: Plan Generation & AI Integration
  ○ Task 2.1: API client (in_progress — this is your task)
  ○ Task 2.2: Plan generation command (pending, depends on 1.3b, 1.4, 2.1)
  ○ Task 2.3: Plan refinement from review comments (pending, depends on 2.2)
  ○ Task 2.4: Replan command (pending, depends on 2.3, 1.5)
Feature 3: Context Generation & Status
  ✓ Task 3.1: Context prompt assembly (completed)
  ✓ Task 3.2: Status command (completed)
  ○ Task 3.3: List and utility commands (pending, depends on 1.3b)
Feature 4: Interactive TUI Review Mode
  ○ Task 4.1: TUI scaffold and plan rendering (pending, depends on 1.3b)
  ○ Task 4.2: Comment mode (pending, depends on 4.1)
  ○ Task 4.3: AI refinement flow in TUI (pending, depends on 4.2, 2.3)
Feature 5: Developer Experience & Polish
  ○ Task 5.1: Error handling (pending, depends on 1.1)
  ○ Task 5.2: First-run polish (pending, depends on 5.1)

## Your Task: Task 2.1 — API client
**Complexity:** small
**Files in Scope:** internal/api/client.go, internal/api/client_test.go
**Depends on:** Task 1.6 (completed)

HTTP client for the Anthropic Messages API.

- POST to `https://api.anthropic.com/v1/messages`
- Headers: `x-api-key`, `anthropic-version: 2023-06-01`, `content-type: application/json`
- Support streaming (SSE) for showing generation progress
- Errors: 401 (bad key), 429 (rate limit — retry with backoff, max 3), 500, network
- Keep generic: accepts system prompt + user message + model, returns response text

Use `net/http` from stdlib. Parse SSE `data:` lines for streaming.

### Acceptance Criteria
- [ ] Makes Messages API requests successfully
- [ ] Streaming SSE output works (yields text chunks via channel or callback)
- [ ] Handles auth, rate limit, server, and network errors with clear messages
- [ ] Respects model setting from config
- [ ] Exponential backoff on 429 (1s, 2s, 4s, give up)
- [ ] Tests using `httptest.NewServer` — no real API calls: success response, streaming chunks, 401/429/500 error handling, backoff retry count
- [ ] `go test ./internal/api/...` passes

### Previous Sessions
None — this is session 001.

### Completed Prerequisites

**Task 1.6 (Config management):**
`internal/config/config.go` — Config struct, Load(), ResolveAPIKey(), `internal/config/config_test.go` — 8 tests covering all acceptance criteria, `go.mod` / `go.sum` — upgraded BurntSushi/toml v1.4.0 → v1.6.0 (was indirect dep). - Config uses `[api]` and `[defaults]` sections (canonical format per task spec), not the `[ai]`/`[plan]`/`[context]` sections from init.go's defaultConfig. Init.go update deferred per instructions.
- API key resolution: env var > config file > error. Env var always wins when set.
- `Load()` takes a `projectRoot` string parameter for testability — callers pass "." for normal use.
- `ResolveAPIKey()` is a separate method so callers can defer the check until an API key is actually needed (e.g., not needed for `etch status`).
- Invalid TOML returns an error (tested).

## Session Progress File

Update your progress file as you work:
`.etch/progress/build-etch--task-2.1--001.md`

This file has been created for you. Fill in each section:
- **Changes Made:** files created or modified
- **Acceptance Criteria Updates:** check off what you completed
- **Decisions & Notes:** design decisions, important context
- **Blockers:** anything blocking progress
- **Next:** what still needs to happen
- **Status:** update to completed, partial, failed, or blocked

Rules:
- Stay within the files listed in scope. Ask before modifying others.
- Do NOT modify the plan file. Only update your progress file.
- Keep notes concise but useful — future sessions depend on them.
