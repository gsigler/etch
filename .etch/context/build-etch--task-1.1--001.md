# Etch Context — Implementation Task

You are working on a task as part of an implementation plan managed by Etch.

## Plan: Build Etch – AI Implementation Planning CLI
Etch is a Go CLI tool that helps developers create, review, and execute AI-generated implementation plans. It uses markdown plan files as the spec, generates context prompt files for AI coding agents, and tracks progress across sessions with file-based state.

## Current Plan State
Feature 1: Core Data Layer & Plan Parser
  ▶ Task 1.1: Project scaffold and CLI skeleton (pending — this is your task)
  ○ Task 1.2: Data models (pending, depends on 1.1)
  ○ Task 1.3: Plan markdown parser — structure (pending, depends on 1.2)
  ○ Task 1.3b: Plan parser — task metadata extraction (pending, depends on 1.3)
  ○ Task 1.4: Plan markdown serializer (pending, depends on 1.3b)
  ○ Task 1.5: Progress file reader/writer (pending, depends on 1.2)
  ○ Task 1.6: Config management (pending, depends on 1.1)
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

## Your Task: Task 1.1 — Project scaffold and CLI skeleton
**Complexity:** small
**Files in Scope:** go.mod, main.go, cmd/*.go, internal/etch/*.go
**Depends on:** none

Initialize the Go project:

etch/
├── go.mod
├── go.sum
├── main.go                    # Entry point, calls cmd.Execute()
├── cmd/
│   ├── root.go                # Root command dispatch, global flags
│   ├── init.go                # etch init
│   ├── plan.go                # etch plan
│   ├── review.go              # etch review
│   ├── status.go              # etch status
│   ├── context.go             # etch context
│   ├── replan.go              # etch replan
│   ├── list.go                # etch list
│   └── open.go                # etch open
├── internal/
│   ├── parser/                # Plan markdown parsing
│   ├── serializer/            # Plan markdown writing
│   ├── progress/              # Progress file read/write
│   ├── generator/             # AI plan generation + refinement
│   ├── context/               # Context prompt assembly
│   ├── config/                # Config management
│   └── models/                # Data structures
└── README.md

Use `urfave/cli/v2` for CLI framework — lightweight, sufficient for ~8 subcommands, no code generation overhead. All subcommands print "not yet implemented" stubs.

`etch init`:
- Creates `.etch/plans/`, `.etch/progress/`, `.etch/context/`, `.etch/backups/`
- Creates `.etch/config.toml` with documented defaults
- Asks: "Track progress files in git? (y/N)" → if no, adds `.etch/progress/` to `.gitignore`
- Always adds `.etch/backups/`, `.etch/context/`, and `.etch/config.toml` to `.gitignore`
- Prints quickstart message

### Acceptance Criteria
- [ ] `go build` produces a working binary
- [ ] `etch init` creates directory structure and config
- [ ] `.gitignore` handling based on user choice
- [ ] All subcommands exist as stubs with `--help`
- [ ] Package structure matches the layout above
- [ ] `go test ./...` passes
- [ ] Tests for `etch init`: directory creation, config file content, gitignore entries (yes and no paths), idempotent re-init

### Testing Requirements
This project requires tests alongside all code. For this task:
- Write tests in `cmd/init_test.go` for the `etch init` command
- Use `t.TempDir()` for isolated filesystem tests
- Verify: directories created, config.toml contents, .gitignore entries for both "track progress" yes/no paths
- Test idempotent re-init (running init twice doesn't break anything)
- All tests must pass with `go test ./...` before the task is complete

### Previous Sessions
None — this is session 001.

### Completed Prerequisites
None — this task has no dependencies.

## Session Progress File

Update your progress file as you work:
`.etch/progress/build-etch--task-1.1--001.md`

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
