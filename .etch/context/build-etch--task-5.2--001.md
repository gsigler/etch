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
  ✓ Task 2.1: API client (completed)
  ✓ Task 2.2: Plan generation command (completed)
  ✓ Task 2.3: Plan refinement from review comments (completed)
  ✓ Task 2.4: Replan command (completed)
Feature 3: Context Generation & Status
  ✓ Task 3.1: Context prompt assembly (completed)
  ✓ Task 3.2: Status command (completed)
  ✓ Task 3.3: List and utility commands (completed)
Feature 4: Interactive TUI Review Mode
  ✓ Task 4.1: TUI scaffold and plan rendering (completed)
  ✓ Task 4.2: Comment mode (completed)
  ✓ Task 4.3: AI refinement flow in TUI (completed)
Feature 5: Developer Experience & Polish
  ✓ Task 5.1: Error handling (completed)
  ○ Task 5.2: First-run polish (in_progress — this is your task)

## Your Task: Task 5.2 — First-run polish
**Complexity:** small
**Files in Scope:** cmd/init.go
**Depends on:** Task 5.1 (completed)

Polish `etch init` output with clear quickstart messaging:

`etch init` prints:
```
✓ Etch initialized!

Next steps:
  etch plan "describe your feature"    Generate an implementation plan
  etch review <plan>                   Review and refine with AI
  etch context <task>                  Generate context prompt file for a task
  etch status                          Check progress across all plans
```


**Note:** README, CONTRIBUTING.md, and LICENSE are documentation tasks that can be written anytime and don't block shipping. They are intentionally excluded from this plan's critical path.

### Acceptance Criteria
- [ ] `etch init` prints clear quickstart

### Previous Sessions
None — this is session 001.

### Completed Prerequisites

**Task 5.1 (Error handling):**
Created `internal/errors/errors.go` — custom Error type with Category (config, api, parse, project, usage, io), Message, Hint, Cause fields. Convenience constructors and `Format()`/`Render()` for colored terminal output using Lipgloss., Updated `main.go` — panic recovery via `defer recover()`, colored error output via `etcherr.Render()`, verbose flag forwarded from `cmd.Execute()`., Updated `cmd/root.go` — `Execute()` returns `(bool, error)` with verbose flag value, `Before` hook captures `--verbose`., Updated all cmd/*.go files — replaced `fmt.Errorf` with typed `etcherr.*` errors with actionable hints., Updated `internal/api/client.go` — all errors use `etcherr.WrapAPI`/`etcherr.API` with hints., Updated `internal/config/config.go` — config errors use `etcherr.WrapConfig`/`etcherr.Config`., Updated `internal/generator/generator.go`, `refine.go`, `replan.go` — all errors typed., Updated `internal/context/context.go` — project/IO errors typed with hints., Updated `internal/status/status.go` — IO/parse errors typed., Updated tests: `cmd/init_test.go`, `internal/api/client_test.go`, `internal/config/config_test.go` to match new error types.. - Error type: single `errors.Error` struct with a `Category` field rather than separate types per category. Simpler, composable via `WithHint()`.
- Lipgloss colors: red bold for "error", yellow for category label, dim for hints and causes.
- `--verbose` flag shows the `Cause` chain; non-verbose shows only message + hint.
- Panic recovery in main.go catches unexpected panics and renders them as errors with exit code 2.
- The old `api.APIError` type is kept for backward compat but the client no longer returns it — it returns `etcherr.Error` instead.
- Lower-level packages (parser, serializer, progress, tui) still use `fmt.Errorf` — their errors bubble up through higher-level packages that wrap them in typed errors.

## Session Progress File

Update your progress file as you work:
`.etch/progress/build-etch--task-5.2--001.md`

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
