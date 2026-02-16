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
  ○ Task 2.4: Replan command (in_progress — this is your task)
Feature 3: Context Generation & Status
  ✓ Task 3.1: Context prompt assembly (completed)
  ✓ Task 3.2: Status command (completed)
  ✓ Task 3.3: List and utility commands (completed)
Feature 4: Interactive TUI Review Mode
  ✓ Task 4.1: TUI scaffold and plan rendering (completed)
  ✓ Task 4.2: Comment mode (completed)
  ✓ Task 4.3: AI refinement flow in TUI (completed)
Feature 5: Developer Experience & Polish
  ○ Task 5.1: Error handling (pending, depends on 1.1)
  ○ Task 5.2: First-run polish (pending, depends on 5.1)

## Your Task: Task 2.4 — Replan command
**Complexity:** medium
**Files in Scope:** internal/generator/replan.go, cmd/replan.go
**Depends on:** Task 2.3 (completed), Task 1.5 (completed)

Implement `etch replan <target>` — AI-powered replanning for tasks or features that need rethinking.

Target resolution (smart scope detection):
- `etch replan 1.2` → replan Task 1.2
- `etch replan feature:2` or `etch replan "Login Endpoints"` → replan all of Feature 2
- `etch replan auth-system 1.2` → replan task in specific plan
- If just a number and ambiguous, prefer task interpretation

Behavior varies by context:
- **Task with no sessions:** This is a planning issue. Prompt: "Rethink this task. Is the scope right? Are the criteria clear? Should it be split into smaller tasks?"
- **Task with failed/blocked sessions:** This is an approach issue. Include session history in prompt. Ask: "This task has been attempted N times. Here's what happened. Suggest an alternative approach or break it down differently."
- **Feature scope:** Replan all tasks within the feature. Include progress on completed tasks so Claude doesn't redo them.

Always backs up before applying changes.

### Acceptance Criteria
- [ ] Target resolution works for task IDs, feature references, and plan-scoped IDs
- [ ] Adapts prompt based on session history (planning issue vs approach issue)
- [ ] Feature-level replan preserves completed tasks
- [ ] Backs up plan before changes
- [ ] Shows diff and requires confirmation
- [ ] Can split a single task into multiple tasks
- [ ] Updated plan parses correctly
- [ ] Tests: target resolution (task ID, feature ref, plan-scoped), prompt adaptation based on session history, backup creation. Use mock API client and fixture plans.
- [ ] `go test ./internal/generator/...` passes

### Previous Sessions
None — this is session 001.

### Completed Prerequisites

**Task 2.3 (Plan refinement from review comments):**
Created `internal/generator/refine.go` — refinement logic: comment extraction, backup, API call with streaming, response validation, colored diff generation, apply function, Created `internal/generator/refine_test.go` — 18 tests covering all acceptance criteria, Modified `internal/generator/prompts.go` — added refine system prompt and user message builder. - `Refine()` returns a `RefineResult` with old/new markdown, parsed plan, backup path, and diff string. The caller handles the confirmation prompt and calls `ApplyRefinement()` — same separation-of-concerns pattern as `Generate()`/`WritePlan()`.
- Comment extraction groups comments by task ID for context in the refinement prompt.
- Diff uses LCS-based algorithm for accurate line-level comparison with ANSI color codes (red for removed, green for added).
- "No comments" check happens before backup creation to avoid unnecessary backups.
- Backup naming: `<plan-slug>-<YYYYMMDD-HHMMSS>.md` in `.etch/backups/`.
- Refine system prompt includes the full format spec to ensure the AI preserves the plan format.

**Task 1.5 (Progress file reader/writer):**
internal/progress/progress.go, internal/progress/progress_test.go. - Used `bufio.Scanner` for line-based parsing with section state tracking
- HTML comment placeholders are stripped when reading sections so unfilled sections return empty strings
- `parseListItems` skips checkbox items to avoid mixing changes with criteria
- `stripComments` removes placeholder HTML comments from section text
- Retry loop (up to 100 attempts) on O_EXCL conflict for atomic creation

## Session Progress File

Update your progress file as you work:
`.etch/progress/build-etch--task-2.4--001.md`

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
