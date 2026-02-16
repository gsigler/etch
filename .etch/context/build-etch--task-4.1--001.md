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
  ○ Task 4.1: TUI scaffold and plan rendering (in_progress — this is your task)
  ○ Task 4.2: Comment mode (pending, depends on 4.1)
  ○ Task 4.3: AI refinement flow in TUI (pending, depends on 4.2, 2.3)
Feature 5: Developer Experience & Polish
  ○ Task 5.1: Error handling (pending, depends on 1.1)
  ○ Task 5.2: First-run polish (pending, depends on 5.1)

## Your Task: Task 4.1 — TUI scaffold and plan rendering
**Complexity:** medium
**Files in Scope:** internal/tui/model.go, internal/tui/view.go, internal/tui/keys.go
**Depends on:** Task 1.3b (completed)

Bubbletea app that renders a plan as a scrollable, color-coded document.

Layout (Lipgloss styling):
- Top bar: plan title, task counts, key hints (dim)
- Main area: scrollable plan content
- Bottom bar: current position (Feature X / Task Y), mode indicator

Rendering with Lipgloss:
- Task headers colored by status (green/yellow/dim/red)
- Acceptance criteria: green ✓ or dim ○
- `> 💬` comments: amber/yellow background, visually distinct
- `---` as styled horizontal rule
- Bold/code rendered appropriately

Navigation (Bubbletea key handling):
- `j`/`k` or arrows: scroll
- `d`/`u`: half-page scroll
- `gg`/`G`: top/bottom
- `n`/`p`: next/previous task
- `f`/`F`: next/previous feature
- `/`: search mode (highlight matches, `n` for next match)
- `q`: quit

### Acceptance Criteria
- [ ] `etch review <plan>` opens full-screen TUI
- [ ] Plan rendered with status coloring
- [ ] 💬 comments visually highlighted
- [ ] All navigation keys work
- [ ] Position indicator in bottom bar
- [ ] Clean exit on `q` (terminal restored)
- [ ] Handles long plans (smooth scrolling)

### Previous Sessions
None — this is session 001.

### Completed Prerequisites

**Task 1.3b (Plan parser — task metadata extraction):**
`internal/models/models.go` — Added `Suffix string` field to `Task`, updated `FullID()` to append suffix, `internal/models/models_test.go` — Added suffix test cases to `TestTaskFullID` and `TestPlanTaskByID`, `internal/parser/parser.go` — Fixed task heading regex to support letter suffixes (`[a-z]`), added metadata extraction (complexity, files, depends_on, criteria, comments) with line-level pattern matching in `stateTask`, `internal/parser/parser_test.go` — Updated `TestParse_TaskDescriptionCapture` → `TestParse_TaskMetadataExtraction`, updated spec file test to expect 7 tasks in Feature 1, added tests: `TestParse_LetterSuffixTaskID`, `TestParse_ReviewComments`, `TestParse_TaskNoMetadata`, `TestParse_TaskNoCriteria`, `TestParse_FilesInScope`. - Metadata lines (Complexity, Files, Depends on) are extracted and removed from Description; only non-metadata text goes into Description
- `**Files:**` and `**Files in Scope:**` both supported via regex alternation
- Multi-line `> 💬` comments: consecutive `>` lines after a `> 💬` line are joined with newlines into a single comment string
- `inComment` flag tracks multi-line comment state, reset on flush or when a non-`>` line is encountered

## Session Progress File

Update your progress file as you work:
`.etch/progress/build-etch--task-4.1--001.md`

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
