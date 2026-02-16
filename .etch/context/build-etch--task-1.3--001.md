# Etch Context — Implementation Task

You are working on a task as part of an implementation plan managed by Etch.

## Plan: Build Etch – AI Implementation Planning CLI
Etch is a Go CLI tool that helps developers create, review, and execute AI-generated implementation plans. It uses markdown plan files as the spec, generates context prompt files for AI coding agents, and tracks progress across sessions with file-based state.

## Current Plan State
Feature 1: Core Data Layer & Plan Parser
  ✓ Task 1.1: Project scaffold and CLI skeleton (completed)
  ✓ Task 1.2: Data models (completed)
  ▶ Task 1.3: Plan markdown parser — structure (pending — this is your task)
  ○ Task 1.3b: Plan parser — task metadata extraction (pending, depends on 1.3)
  ○ Task 1.4: Plan markdown serializer (pending, depends on 1.3b)
  ○ Task 1.5: Progress file reader/writer (pending, depends on 1.2)
  ▶ Task 1.6: Config management (in_progress — parallel session)
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

## Your Task: Task 1.3 — Plan markdown parser (structure)
**Complexity:** medium
**Files in Scope:** internal/parser/parser.go, internal/parser/parser_test.go
**Depends on:** Task 1.2 (completed)

Parse plan markdown files into `Plan` structs using a line-based state machine. No external markdown library — the plan format is a strict subset of markdown with known heading patterns, so a state machine tracking heading depth via `#` prefixes is simpler, faster, and dependency-free.

Parsing strategy — line-based state machine:
- Read file line by line, track current state (plan-level, feature-level, task-level)
- `# Plan:` → extract plan title, switch to plan state
- `## Feature N:` → create new feature, switch to feature state
- `## Overview` → plan overview section
- `### Task N.M:` or `### Task N:` → create new task, switch to task state
- `---` → skip (section separator)
- Anything else → accumulate into current section's description
- Heading depth determines hierarchy: H1=plan, H2=feature/overview, H3=task

Handle single-feature detection: if tasks appear (H3) without any prior `## Feature` heading, wrap them in an implicit feature.

Edge cases:
- Single-feature plan (no `## Feature` headings)
- Empty features (heading but no tasks yet)
- Malformed or missing status tag → default to `pending`
- `---` separators (skip)

**Important:** This task is structure-only. Extract the heading hierarchy, plan title, overview, feature numbers/titles/overviews, and task titles. Task-level metadata (complexity, files, depends_on, criteria, comments) will be extracted in Task 1.3b. For now, task sections should capture the raw body text in the `Description` field for 1.3b to parse later. You should still extract the status tag from the `### Task` heading line since it's part of the structure.

The data models you'll use are in `internal/models/models.go`:
- `Plan` — Title, Overview, Features, FilePath, Slug
- `Feature` — Number, Title, Overview, Tasks
- `Task` — FeatureNumber, TaskNumber, Title, Status, Description (and more fields for 1.3b)
- Helper: `models.ParseStatus(s string) Status`

A test fixture is available: the plan spec file itself at `.etch/plans/build-etch.md` — use it as a real-world parsing test.

### Acceptance Criteria
- [ ] Parses multi-feature plans into correct Plan/Feature/Task hierarchy
- [ ] Parses single-feature plan shorthand (implicit feature wrapping)
- [ ] Extracts plan title and overview
- [ ] Extracts feature numbers, titles, and overviews
- [ ] Returns clear error for fundamentally broken files (no `# Plan:` heading)
- [ ] Gracefully handles empty features and missing sections
- [ ] Test: parses the spec file at `.etch/plans/build-etch.md` and produces correct hierarchy

### Testing Requirements
This project requires tests alongside all code. For this task:
- Write tests in `internal/parser/parser_test.go`
- Include a test that parses `.etch/plans/build-etch.md` and verifies: plan title, feature count (5), task count per feature, task titles
- Test single-feature plan (no `## Feature` heading — tasks wrapped in implicit feature)
- Test empty/broken input (no `# Plan:` heading → error)
- Test empty features, missing overview sections
- Table-driven tests where appropriate
- All tests must pass with `go test ./internal/parser/...` before the task is complete

### Previous Sessions
None — this is session 001.

### Completed Prerequisites

**Task 1.1 (completed):**
Project scaffold created with `urfave/cli/v2`. All 8 subcommands registered as stubs. `internal/parser/parser.go` exists with just `package parser`.

**Task 1.2 (completed):**
All data models defined in `internal/models/models.go` with JSON tags. Types: Status, Complexity, Plan, Feature, Task, Criterion, SessionProgress. Helper methods: `Task.FullID()`, `Status.Icon()`, `Plan.TaskByID()`, `ParseStatus()`. All tested with table-driven tests.

## Session Progress File

Update your progress file as you work:
`.etch/progress/build-etch--task-1.3--001.md`

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
