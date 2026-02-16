# Etch Context — Implementation Task

You are working on a task as part of an implementation plan managed by Etch.

## Plan: Build Etch – AI Implementation Planning CLI
Etch is a Go CLI tool that helps developers create, review, and execute AI-generated implementation plans. It uses markdown plan files as the spec, generates context prompt files for AI coding agents, and tracks progress across sessions with file-based state.

## Current Plan State
Feature 1: Core Data Layer & Plan Parser
  ✓ Task 1.1: Project scaffold and CLI skeleton (completed)
  ▶ Task 1.2: Data models (in_progress — parallel session)
  ○ Task 1.3: Plan markdown parser — structure (pending, depends on 1.2)
  ○ Task 1.3b: Plan parser — task metadata extraction (pending, depends on 1.3)
  ○ Task 1.4: Plan markdown serializer (pending, depends on 1.3b)
  ○ Task 1.5: Progress file reader/writer (pending, depends on 1.2)
  ▶ Task 1.6: Config management (pending — this is your task)
Feature 2: Plan Generation & AI Integration
  ○ Task 2.1: API client (pending, depends on 1.6)
  ○ Task 2.2: Plan generation command (pending, depends on 1.3b, 1.4, 2.1)
  ○ Task 2.3: Plan refinement from review comments (pending, depends on 2.2)
  ○ Task 2.4: Replan command (pending, depends on 2.3, 1.5)
Feature 3: Context Generation & Status
  ○ Task 3.1: Context prompt assembly (pending, depends on 1.3b, 1.5)
  ○ Task 3.2: Status command (pending, depends on 1.3b, 1.4, 1.5)
  ○ Task 3.3: List and utility commands (pending, depends on 1.3b)
Feature 4: Interactive TUI Review Mode
  ○ Task 4.1: TUI scaffold and plan rendering (pending, depends on 1.3b)
  ○ Task 4.2: Comment mode (pending, depends on 4.1)
  ○ Task 4.3: AI refinement flow in TUI (pending, depends on 4.2, 2.3)
Feature 5: Developer Experience & Polish
  ○ Task 5.1: Error handling (pending, depends on 1.1)
  ○ Task 5.2: First-run polish (pending, depends on 5.1)

## Your Task: Task 1.6 — Config management
**Complexity:** small
**Files in Scope:** internal/config/config.go, internal/config/config_test.go
**Depends on:** Task 1.1 (completed)

Implement config management for Etch. The config file is `.etch/config.toml`:

```toml
[api]
model = "claude-sonnet-4-20250514"
# API key: set ANTHROPIC_API_KEY env var (preferred) or uncomment below
# api_key = "sk-ant-..."

[defaults]
complexity_guide = "small = single focused session, medium = may need iteration, large = multiple sessions likely"
```

Requirements:
- Read with `BurntSushi/toml`
- API key resolution chain: `ANTHROPIC_API_KEY` env var → config file `api_key` → error with helpful message
- Defaults for all optional fields
- `Config` struct passed explicitly (no global state)

Note: `etch init` (Task 1.1) already creates a config.toml with a slightly different structure (`[ai]` section). Your implementation is the canonical config format — `etch init` can be updated later to match. Don't modify `cmd/init.go` in this task.

### Acceptance Criteria
- [ ] Reads `.etch/config.toml`
- [ ] API key resolution chain works correctly
- [ ] Missing config file → sensible defaults (don't crash)
- [ ] Helpful error when no API key found anywhere
- [ ] Tests in `internal/config/config_test.go`: valid config, missing file, env var override, missing API key error
- [ ] `go test ./internal/config/...` passes

### Testing Requirements
This project requires tests alongside all code. For this task:
- Write tests in `internal/config/config_test.go`
- Use `t.TempDir()` for isolated filesystem tests
- Use `t.Setenv()` for env var tests (auto-restores after test)
- Table-driven tests where appropriate
- Cover: valid config parsing, missing file defaults, env var overrides config file, missing API key error message
- All tests must pass with `go test ./internal/config/...` before the task is complete

### Previous Sessions
None — this is session 001.

### Completed Prerequisites

**Task 1.1 (completed):**
Project scaffold created with `urfave/cli/v2`. All 8 subcommands registered as stubs. `etch init` fully implemented (creates .etch dirs, config.toml, gitignore handling). `internal/config/config.go` exists with just `package config`. Uses `BurntSushi/toml` — you'll need to `go get` this dependency.

## Session Progress File

Update your progress file as you work:
`.etch/progress/build-etch--task-1.6--001.md`

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
