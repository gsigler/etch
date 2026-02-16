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
  ○ Task 2.1: API client (pending, depends on 1.6)
  ○ Task 2.2: Plan generation command (pending, depends on 1.3b, 1.4, 2.1)
  ○ Task 2.3: Plan refinement from review comments (pending, depends on 2.2)
  ○ Task 2.4: Replan command (pending, depends on 2.3, 1.5)
Feature 3: Context Generation & Status
  ✓ Task 3.1: Context prompt assembly (completed)
  ✓ Task 3.2: Status command (completed)
  ✓ Task 3.3: List and utility commands (completed)
Feature 4: Interactive TUI Review Mode
  ✓ Task 4.1: TUI scaffold and plan rendering (completed)
  ✓ Task 4.2: Comment mode (completed)
  ○ Task 4.3: AI refinement flow in TUI (in_progress — this is your task)
Feature 5: Developer Experience & Polish
  ○ Task 5.1: Error handling (pending, depends on 1.1)
  ○ Task 5.2: First-run polish (pending, depends on 5.1)

## Your Task: Task 4.3 — AI refinement flow in TUI
**Complexity:** medium
**Files in Scope:** internal/tui/review.go, internal/tui/diff.go
**Depends on:** Task 4.2 (completed), Task 2.3 (pending)

The apply flow within the TUI.

1. `a` → "Send N comments for refinement? (y/n)"
2. Confirm → loading spinner (Bubbletea spinner bubble)
3. Response → switch to diff view (red/green coloring)
4. `y` to accept, `n` to reject
5. Accept → plan updated, TUI refreshes
6. Reject → return to review mode

Uses the refinement logic from Task 2.3 (backup, API call, parse, diff).

### Acceptance Criteria
- [ ] `a` triggers refinement with confirmation
- [ ] Loading spinner while waiting
- [ ] Colored diff view
- [ ] Accept/reject flow
- [ ] Plan backup happens before changes
- [ ] API errors shown gracefully
- [ ] "No comments to send" if no 💬 found

### Previous Sessions
None — this is session 001.

### Completed Prerequisites

**Task 4.2 (Comment mode):**
Created `internal/tui/comments.go` — `AddComment` and `DeleteComment` for line-level plan file editing, Created `internal/tui/input.go` — `commentInput` (bubbles textinput wrapper), `openEditor()` via `tea.ExecProcess`, Modified `internal/tui/model.go` — `tuiMode` enum, comment/confirm modes, reloadPlan, ANSI stripping, updated `New()` signature, Modified `internal/tui/view.go` — prompt styles, updated bottom bar for modes/status, updated top bar hints, Modified `cmd/review.go` — pass `plan.FilePath` to `tui.New()`, Modified `go.mod`/`go.sum` — added transitive dep `github.com/atotto/clipboard`. - Replaced boolean `searchMode` with `tuiMode` enum for cleaner multi-mode state
- Comments added/deleted via direct file manipulation (same pattern as serializer's UpdateTaskStatus), then reloadPlan() re-parses
- $EDITOR uses `tea.ExecProcess` to suspend TUI, hand control to editor, resume on exit
- ANSI stripping needed for matching rendered comment lines back to model data
- viewHeight reduces by 1 when prompt is visible

## Session Progress File

Update your progress file as you work:
`.etch/progress/build-etch--task-4.3--001.md`

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
