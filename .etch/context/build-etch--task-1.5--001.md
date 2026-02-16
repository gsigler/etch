# Etch Context — Implementation Task

You are working on a task as part of an implementation plan managed by Etch.

## Plan: Build Etch – AI Implementation Planning CLI
Etch is a Go CLI tool that helps developers create, review, and execute AI-generated implementation plans. It uses markdown plan files as the spec, generates context prompt files for AI coding agents, and tracks progress across sessions with file-based state.

## Current Plan State
Feature 1: Core Data Layer & Plan Parser
  ✓ Task 1.1: Project scaffold and CLI skeleton (completed)
  ✓ Task 1.2: Data models (completed)
  ✓ Task 1.3: Plan markdown parser — structure (completed)
  ✓ Task 1.3b: Plan parser — task metadata extraction (completed)
  ✓ Task 1.4: Plan markdown serializer (completed)
  ▶ Task 1.5: Progress file reader/writer (pending — this is your task)
  ✓ Task 1.6: Config management (completed)
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

## Your Task: Task 1.5 — Progress file reader/writer
**Complexity:** medium
**Files in Scope:** internal/progress/progress.go, internal/progress/progress_test.go
**Depends on:** Task 1.2 (completed)

### Writing (called by `etch context`)
- Create a new progress file with correct naming: `<plan>--task-<N.M>--<NNN>.md`
- For letter-suffix task IDs like `1.3b`, the filename uses the full ID: `<plan>--task-1.3b--001.md`
- Determine next session number by globbing existing files for that task
- Use atomic file creation (`os.OpenFile` with `O_CREATE|O_EXCL`) to prevent race conditions when two agents start the same task concurrently — if the file already exists, increment and retry
- Pre-fill headers from plan data (task title, plan slug, session number, timestamp)
- Pre-fill acceptance criteria from the plan (all with current check state)
- Leave content sections empty with placeholder comments for the agent

The template should match the session progress file format spec:

```markdown
# Session: Task 1.1 – Database Schema
**Plan:** auth-system
**Task:** 1.1
**Session:** 001
**Started:** 2026-02-16 09:30
**Status:** pending

## Changes Made
<!-- List files created or modified -->

## Acceptance Criteria Updates
- [ ] Migration file creates users table
- [ ] User model struct matches schema
- [ ] Migration runs successfully on empty database

## Decisions & Notes
<!-- Design decisions, important context for future sessions -->

## Blockers
<!-- Anything blocking progress -->

## Next
<!-- What still needs to happen -->
```

### Reading (called by `etch status`)
- Glob `.etch/progress/<plan>--*.md`
- Parse each file: extract task ID, session number, status, criteria updates
- Line-based parsing — look for `**Status:**`, `**Task:**`, `**Session:**` lines and section headers
- Parse `## Changes Made` section as list items
- Parse `## Acceptance Criteria Updates` section for `- [x]` / `- [ ]` checkboxes
- Parse `## Decisions & Notes`, `## Blockers`, `## Next` as free text
- Group by task ID, sort by session number
- Return map of task ID → `[]SessionProgress` (all sessions, sorted)

Be reasonably strict in parsing — if the agent wrote something that doesn't match the expected format, log a warning and skip that field rather than crashing.

The `SessionProgress` model is already defined in `internal/models/models.go`:

```go
type SessionProgress struct {
    PlanSlug        string      `json:"plan_slug"`
    TaskID          string      `json:"task_id"`
    SessionNumber   int         `json:"session_number"`
    Started         string      `json:"started"`
    Status          string      `json:"status"`
    ChangesMade     []string    `json:"changes_made"`
    CriteriaUpdates []Criterion `json:"criteria_updates"`
    Decisions       string      `json:"decisions"`
    Blockers        string      `json:"blockers"`
    Next            string      `json:"next"`
}
```

### Acceptance Criteria
- [ ] Creates correctly named progress files with pre-filled template
- [ ] Auto-increments session number with atomic `O_EXCL` file creation (no race conditions)
- [ ] Template matches the session format spec
- [ ] Reads and parses agent-filled progress files
- [ ] Groups by task, returns latest state per task
- [ ] Handles: missing files, partially filled files, extra content
- [ ] Warning (not crash) on unparseable fields

### Testing Requirements
This project requires tests alongside all code. For this task:
- Write tests in `internal/progress/progress_test.go`
- Use `t.TempDir()` for all file operations
- Test writing: correct filename, session auto-increment, template content, atomic creation (create two files for same task — should get 001 and 002)
- Test reading: parse a fully filled progress file, parse a partially filled one, parse one with extra content
- Test grouping: multiple tasks, multiple sessions per task, returns sorted by session number
- Test edge cases: empty progress dir, malformed files (log warning, don't crash)
- Test letter-suffix task IDs in filenames
- All tests must pass with `go test ./internal/progress/...` before the task is complete

### Previous Sessions
None — this is session 001.

### Completed Prerequisites

**Task 1.2 (completed):**
Data models in `internal/models/models.go`. Key types: Plan, Feature, Task (with Suffix field), Criterion, SessionProgress. `Task.FullID()` returns e.g. "1.3b". `internal/progress/progress.go` exists with just `package progress`.

## Session Progress File

Update your progress file as you work:
`.etch/progress/build-etch--task-1.5--001.md`

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
