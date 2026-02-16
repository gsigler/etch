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
  ▶ Task 3.1: Context prompt assembly (in_progress — parallel session)
  ▶ Task 3.2: Status command (pending — this is your task)
  ○ Task 3.3: List and utility commands (pending, depends on 1.3b)
Feature 4: Interactive TUI Review Mode
  ○ Task 4.1: TUI scaffold and plan rendering (pending, depends on 1.3b)
  ○ Task 4.2: Comment mode (pending, depends on 4.1)
  ○ Task 4.3: AI refinement flow in TUI (pending, depends on 4.2, 2.3)
Feature 5: Developer Experience & Polish
  ○ Task 5.1: Error handling (pending, depends on 1.1)
  ○ Task 5.2: First-run polish (pending, depends on 5.1)

## Your Task: Task 3.2 — Status command
**Complexity:** medium
**Files in Scope:** internal/status/status.go, internal/status/status_test.go, cmd/status.go
**Depends on:** Task 1.3b (completed), Task 1.4 (completed), Task 1.5 (completed)

**Known limitation:** If two agents finish tasks simultaneously and `etch status` runs while another instance is also running, the targeted plan file update could conflict. For v1 this is acceptable.

Implement `etch status`.

### Flow
1. Read all plans in `.etch/plans/`
2. For each plan, read progress files using `progress.ReadAll()`
3. Determine current task statuses from latest sessions
4. Update plan files (status tags + checkboxes) using `serializer.UpdateTaskStatus()` and `serializer.UpdateCriterion()` if progress files show changes
5. Display summary

### Status reconciliation logic
- For each task, find the latest session (highest session number)
- Map progress status to plan status: `completed` → `completed`, `partial` → `in_progress`, `failed` → `failed`, `blocked` → `blocked`
- Merge acceptance criteria: if any session marks `[x]`, the plan gets `[x]`
- Only update the plan file if something actually changed

### Display format
```
📋 Auth System Rebuild
   ✓ Feature 1: JWT Token Management [3/3 tasks]
   ▶ Feature 2: Login Endpoints [1/3 tasks]
     ✓ 2.1: Registration endpoint
     ▶ 2.2: Login endpoint (2 sessions, last: partial)
     ○ 2.3: Password validation
   ○ Feature 3: Password Reset [0/2 tasks]

📋 API Refactor
   ○ Feature 1: GraphQL Migration [0/4 tasks]
```

### Variations
- `etch status` → all plans, summary view
- `etch status <plan>` → single plan, detailed view (show criteria + last session notes for each task)
- `etch status --json` → machine-readable JSON output

### Available packages
- `internal/parser` — `ParseFile(path) (*models.Plan, error)` to read plan files
- `internal/serializer` — `UpdateTaskStatus(path, taskID, newStatus) error` and `UpdateCriterion(path, taskID, criterionText string, met bool) error` for targeted plan updates
- `internal/progress` — `ReadAll(dir, planSlug) (map[string][]models.SessionProgress, error)` to read progress files
- `internal/models` — all data types, `Task.FullID()`, `Status.Icon()`

### Acceptance Criteria
- [ ] Reads all plans and progress files
- [ ] Updates plan files with current status from progress
- [ ] Status icons (✓ ▶ ○ ✗ ⊘)
- [ ] Shows session count and last outcome for in-progress tasks
- [ ] Detailed view for single plan
- [ ] `--json` output
- [ ] Handles: no plans, no progress, orphaned progress files
- [ ] Only touches status tags and checkboxes (no other plan content modified)
- [ ] Tests in `internal/status/status_test.go`: status reconciliation from progress files, plan file update (status tags + checkboxes only), JSON output, edge cases (no plans, no progress, orphaned files). Use fixture plans and progress files.
- [ ] `go test ./internal/status/...` passes

### Testing Requirements
- Write tests in `internal/status/status_test.go`
- Use `t.TempDir()` for all file operations
- Create fixture plan files (valid etch format) and progress files in temp dirs
- Test: reconciliation maps progress status correctly (completed/partial/failed/blocked), criteria merging, plan file updated with new status tags
- Test: display output contains correct icons and task counts
- Test: JSON output is valid and contains expected fields
- Test: no plans dir → graceful empty output
- Test: orphaned progress files (no matching task) → ignored without error
- Test: plan with no progress files → all tasks show as-is from plan
- Verify plan file content is preserved except status tags and checkboxes
- All tests must pass with `go test ./internal/status/...` before the task is complete

### Previous Sessions
None — this is session 001.

### Completed Prerequisites

**Task 1.3b (completed):**
Full plan parser with metadata extraction. `parser.ParseFile(path)` returns a complete `*models.Plan` with all features, tasks, metadata, criteria, and comments.

**Task 1.4 (completed):**
Plan serializer with two modes. Full serialization (`Serialize(*Plan) string`) and targeted updates (`UpdateTaskStatus(path, taskID, status)` and `UpdateCriterion(path, taskID, criterion, met)`). Targeted updates preserve all formatting.

**Task 1.5 (completed):**
Progress file reader/writer. `progress.ReadAll(dir, planSlug)` returns `map[string][]SessionProgress` grouped by task ID, sorted by session number. Each SessionProgress has: Status, ChangesMade, CriteriaUpdates, Decisions, Blockers, Next.

## Session Progress File

Update your progress file as you work:
`.etch/progress/build-etch--task-3.2--001.md`

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
