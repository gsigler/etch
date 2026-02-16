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
  ✓ Task 1.5: Progress file reader/writer (completed)
  ✓ Task 1.6: Config management (completed)
Feature 2: Plan Generation & AI Integration
  ○ Task 2.1: API client (pending, depends on 1.6)
  ○ Task 2.2: Plan generation command (pending, depends on 1.3b, 1.4, 2.1)
  ○ Task 2.3: Plan refinement from review comments (pending, depends on 2.2)
  ○ Task 2.4: Replan command (pending, depends on 2.3, 1.5)
Feature 3: Context Generation & Status
  ▶ Task 3.1: Context prompt assembly (pending — this is your task)
  ▶ Task 3.2: Status command (in_progress — parallel session)
  ○ Task 3.3: List and utility commands (pending, depends on 1.3b)
Feature 4: Interactive TUI Review Mode
  ○ Task 4.1: TUI scaffold and plan rendering (pending, depends on 1.3b)
  ○ Task 4.2: Comment mode (pending, depends on 4.1)
  ○ Task 4.3: AI refinement flow in TUI (pending, depends on 4.2, 2.3)
Feature 5: Developer Experience & Polish
  ○ Task 5.1: Error handling (pending, depends on 1.1)
  ○ Task 5.2: First-run polish (pending, depends on 5.1)

## Your Task: Task 3.1 — Context prompt assembly
**Complexity:** medium
**Files in Scope:** internal/context/context.go, internal/context/context_test.go, cmd/context.go
**Depends on:** Task 1.3b (completed), Task 1.5 (completed)

Implement `etch context [task_id]`.

### Task identification
- `etch context` (no args) → auto-select next pending task respecting dependency order. If multiple candidates, show picker.
- `etch context 1.2` → Feature 1, Task 2
- `etch context 2` → Task 2 in single-feature plan
- `etch context auth-system 1.2` → specific plan
- If multiple plans exist and no plan specified, show picker: "Which plan? (1) auth-system (2) api-refactor"

### Context prompt template

The assembled context must follow this exact template structure (written to `.etch/context/<plan>--task-<N.M>--<session>.md`):

```
# Etch Context — Implementation Task

You are working on a task as part of an implementation plan managed by Etch.

## Plan: <Plan Title>
<Plan overview, condensed to 2-3 sentences>

## Current Plan State
Feature 1: <Title>
  ✓ Task 1.1: <Title> (completed)
  ▶ Task 1.2: <Title> (in_progress — this is your task)
  ○ Task 1.3: <Title> (pending, depends on 1.2)
Feature 2: <Title>
  ○ Task 2.1: <Title> (pending)

## Your Task: Task <N.M> — <Title>
**Complexity:** <complexity>
**Files in Scope:** <files>
**Depends on:** <dependencies>

<Full task description from the plan>

### Acceptance Criteria
- [ ] <criterion>
- [x] <already completed criterion>

### Previous Sessions
<If not session 001, summaries from prior sessions for this task>

**Session 001 (2026-02-16, partial):**
Changes: <changes made summary>
Decisions: <key decisions>
Blockers: <what blocked>
Next: <what to do next>

### Completed Prerequisites
<For each completed dependency, what was done>

**Task 1.1 (completed):**
Created users table migration and User model. Using sqlx with async.

## Session Progress File

Update your progress file as you work:
`.etch/progress/<plan>--task-<N.M>--<session>.md`

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
```

### Assembly steps
1. Plan overview (condensed)
2. Current plan state (all tasks with status icons, incorporating progress files)
3. Full current task spec (description, metadata, criteria, comments)
4. Previous session summaries for this task (if any) — read from progress files
5. Completed prerequisite summaries — for each dependency that's completed, include a summary from its latest progress file
6. Agent instructions including progress file path

### Output
- Write the assembled context to `.etch/context/<plan>--task-<N.M>--<session>.md`
- Create empty session progress file using `progress.CreateFile()` from internal/progress
- Print confirmation with:
  - Token estimate (chars / 3.5)
  - Session progress file path
  - Ready-to-run command: `cat .etch/context/<file> | claude`
- Warning if estimate > 80K tokens

### Auto-select logic (bare `etch context`)
Find the next pending task where all dependencies are satisfied (all deps have status completed). If multiple candidates, pick the first by task ID order. If still ambiguous across plans, show a picker.

### Available packages
- `internal/parser` — `ParseFile(path) (*models.Plan, error)` to read plan files
- `internal/progress` — `ReadAll(dir, planSlug) (map[string][]models.SessionProgress, error)` to read progress files, `CreateFile(dir, planSlug, taskID string, criteria []models.Criterion) (string, int, error)` to create new progress files
- `internal/models` — all data types, `Task.FullID()`, `Status.Icon()`, `Plan.TaskByID()`

### Acceptance Criteria
- [ ] Context follows the template spec
- [ ] Bare `etch context` auto-selects next pending task by dependency order
- [ ] Task ID resolution handles all formats
- [ ] Previous sessions included when they exist
- [ ] Prerequisite summaries included for completed dependencies
- [ ] Creates session progress file with correct template
- [ ] Writes context prompt file to `.etch/context/`
- [ ] Prints ready-to-run `cat .etch/context/<file> | claude` command
- [ ] Token estimate printed
- [ ] Plan picker when ambiguous
- [ ] Warning on large context
- [ ] Tests in `internal/context/context_test.go`: template output matches spec, auto-select next task logic, task ID resolution, previous session inclusion, prerequisite summaries, context file written correctly, token estimate calculation
- [ ] `go test ./internal/context/...` passes

### Testing Requirements
- Write tests in `internal/context/context_test.go`
- Use `t.TempDir()` for all file operations
- Create fixture plan files and progress files in temp dirs
- Test: template structure matches spec, auto-select picks correct next task, task ID resolution (all formats), previous session summaries included, prerequisite summaries included, context file written to correct path, token estimate math
- Test edge cases: first task (no prereqs, no sessions), task with multiple prior sessions, all tasks completed
- All tests must pass with `go test ./internal/context/...` before the task is complete

### Previous Sessions
None — this is session 001.

### Completed Prerequisites

**Task 1.3b (completed):**
Full plan parser with metadata extraction. `parser.ParseFile(path)` returns a complete `*models.Plan` with all features, tasks, metadata, criteria, and comments populated.

**Task 1.5 (completed):**
Progress file reader/writer. `progress.CreateFile(dir, planSlug, taskID, criteria)` creates atomic progress files with auto-incrementing session numbers. `progress.ReadAll(dir, planSlug)` returns `map[string][]SessionProgress` grouped by task ID, sorted by session number.

## Session Progress File

Update your progress file as you work:
`.etch/progress/build-etch--task-3.1--001.md`

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
