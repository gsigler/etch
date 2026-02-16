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
  ○ Task 4.2: Comment mode (in_progress — this is your task)
  ○ Task 4.3: AI refinement flow in TUI (pending, depends on 4.2, 2.3)
Feature 5: Developer Experience & Polish
  ○ Task 5.1: Error handling (pending, depends on 1.1)
  ○ Task 5.2: First-run polish (pending, depends on 5.1)

## Your Task: Task 4.2 — Comment mode
**Complexity:** medium
**Files in Scope:** internal/tui/comments.go, internal/tui/input.go
**Depends on:** Task 4.1 (completed)

Leave 💬 comments from the TUI.

Workflow:
1. Navigate to a task/feature heading
2. `c` → text input appears (Bubbletea text input bubble)
3. Type comment, Enter to submit
4. Inserted as `> 💬 <text>` below the heading in the plan file
5. TUI re-renders with new comment highlighted

Multi-line: `C` opens `$EDITOR` with temp file. On save+close, content becomes the comment.

Delete: `x` on a 💬 line → confirmation → removed from plan file.

### Acceptance Criteria
- [ ] `c` opens text input at current section
- [ ] Comment saved to plan file as `> 💬`
- [ ] `C` opens $EDITOR for multi-line
- [ ] New comments appear immediately
- [ ] `x` deletes comment with confirmation
- [ ] File saved after each comment operation

### Previous Sessions
None — this is session 001.

### Completed Prerequisites

**Task 4.1 (TUI scaffold and plan rendering):**
Created `internal/tui/keys.go` — Bubbletea key bindings (j/k, d/u, gg/G, n/p, f/F, /, q), Created `internal/tui/model.go` — Bubbletea Model with Init/Update/View, scroll state, search mode, task/feature navigation, Created `internal/tui/view.go` — Lipgloss styled rendering: top bar, plan content with status colors, criteria, comments, bottom bar with position, Modified `cmd/review.go` — Replaced stub with plan loading + `tea.NewProgram` launch with alt screen, Modified `go.mod` / `go.sum` — Added bubbletea, lipgloss, bubbles dependencies. - Used manual viewport scrolling (offset-based) rather than bubbles/viewport for simpler integration with line-level feature/task tracking
- `lineEntry` struct tracks which feature/task each rendered line belongs to, enabling position indicator and task/feature jumping
- Search uses case-insensitive matching with yellow highlight on matches; `n` key navigates search results when matches exist, otherwise jumps to next task
- `gg` for top uses a `lastKeyG` flag to detect the two-key sequence
- `G` (shift+g) detected via raw string match since key.Binding doesn't distinguish case well
- Alt screen mode ensures clean terminal restore on quit

## Session Progress File

Update your progress file as you work:
`.etch/progress/build-etch--task-4.2--001.md`

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
