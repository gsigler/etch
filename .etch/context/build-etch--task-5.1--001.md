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
  ○ Task 5.1: Error handling (in_progress — this is your task)
  ○ Task 5.2: First-run polish (pending, depends on 5.1)

## Your Task: Task 5.1 — Error handling
**Complexity:** small
**Files in Scope:** internal/errors/errors.go, all packages
**Depends on:** Task 1.1 (completed)

Consistent errors across all commands:
- Custom error types with categories (ConfigError, APIError, ParseError, etc.)
- Every error includes what went wrong + what to do next
- Colored output: red errors, yellow warnings, dim hints (use Lipgloss)
- `--verbose` global flag for debug output
- No panics reach the user

### Acceptance Criteria
- [ ] Error types defined for all categories
- [ ] Actionable messages on every error
- [ ] Colored output
- [ ] `--verbose` for debug info
- [ ] No raw panics

### Previous Sessions
None — this is session 001.

### Completed Prerequisites

**Task 1.1 (Project scaffold and CLI skeleton):**
`go.mod` — module definition with urfave/cli/v2 dependency, `main.go` — entry point calling `cmd.Execute()`, `cmd/root.go` — root command with global `--verbose` flag, registers all 8 subcommands, `cmd/init.go` — full `etch init` implementation (dirs, config, gitignore, user prompt), `cmd/plan.go`, `cmd/review.go`, `cmd/status.go`, `cmd/context.go`, `cmd/replan.go`, `cmd/list.go`, `cmd/open.go` — stub subcommands printing "not yet implemented", `cmd/init_test.go` — 5 tests covering dirs, config, gitignore (track/no-track), idempotency, `internal/models/models.go` — empty package placeholder, `internal/parser/parser.go` — empty package placeholder, `internal/serializer/serializer.go` — empty package placeholder, `internal/progress/progress.go` — empty package placeholder, `internal/generator/generator.go` — empty package placeholder, `internal/context/context.go` — empty package placeholder, `internal/config/config.go` — empty package placeholder. - Used `urfave/cli/v2` v2.27.5 as specified in the plan
- `etch init` uses stdin prompt for git tracking question (bufio.NewReader)
- Config file uses TOML format with commented-out defaults
- `.gitignore` append logic avoids duplicates on repeated `etch init` runs
- Internal packages are empty placeholders — ready for Task 1.2 (data models)

## Session Progress File

Update your progress file as you work:
`.etch/progress/build-etch--task-5.1--001.md`

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
