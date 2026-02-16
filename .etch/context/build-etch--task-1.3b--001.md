# Etch Context — Implementation Task

You are working on a task as part of an implementation plan managed by Etch.

## Plan: Build Etch – AI Implementation Planning CLI
Etch is a Go CLI tool that helps developers create, review, and execute AI-generated implementation plans. It uses markdown plan files as the spec, generates context prompt files for AI coding agents, and tracks progress across sessions with file-based state.

## Current Plan State
Feature 1: Core Data Layer & Plan Parser
  ✓ Task 1.1: Project scaffold and CLI skeleton (completed)
  ✓ Task 1.2: Data models (completed)
  ✓ Task 1.3: Plan markdown parser — structure (completed)
  ▶ Task 1.3b: Plan parser — task metadata extraction (pending — this is your task)
  ○ Task 1.4: Plan markdown serializer (pending, depends on 1.3b)
  ○ Task 1.5: Progress file reader/writer (pending, depends on 1.2)
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

## Your Task: Task 1.3b — Plan parser: task metadata extraction
**Complexity:** medium
**Files in Scope:** internal/parser/parser.go, internal/parser/parser_test.go
**Depends on:** Task 1.3 (completed)

Within each task section identified by the structure parser, extract all task-level metadata using line-level pattern matching:

- Regex for status tag: `\[(\w+)\]$` at end of heading
- Prefix match for metadata: `**Complexity:**`, `**Files:**`, `**Depends on:**`
- Checkbox match: `^- \[([ x])\] (.+)$`
- Comment match: `^> 💬 (.+)$` (possibly multi-line blockquote)
- Everything else → description text

Edge cases:
- Task with no metadata lines
- Task with no acceptance criteria
- Multi-line `> 💬` comments (consecutive `>` lines)

### Bug fix required: letter-suffix task IDs

The structure parser from Task 1.3 has a bug: the task heading regex `(\d+)(?:\.(\d+))?` skips letter-suffix task IDs like `### Task 1.3b:`. This means our own plan file doesn't fully parse (Task 1.3b is skipped).

**Fix the regex** to support letter suffixes: `(\d+)(?:\.(\d+)([a-z])?)?`

This also requires updating the `Task` model or `FullID()` method to handle the suffix. Options:
- Add a `Suffix string` field to `Task` in `internal/models/models.go` (e.g. `Suffix: "b"`)
- Update `FullID()` to return `"1.3b"` when suffix is present
- Update existing tests in `parser_test.go` that hardcode Feature 1 having 6 tasks — it should be 7 after the fix

### Acceptance Criteria
- [ ] Extracts all task metadata (status, complexity, files, depends_on)
- [ ] Extracts acceptance criteria with completion state
- [ ] Extracts `> 💬` review comments (single and multi-line)
- [ ] Gracefully handles missing/optional metadata fields
- [ ] Test: parses this spec file and produces correct task details
- [ ] Test: round-trip parse → serialize → parse produces equivalent result (skip this if serializer isn't built yet — just note it)
- [ ] Fixes letter-suffix task ID regex so `### Task 1.3b:` parses correctly
- [ ] Spec file test updated: Feature 1 has 7 tasks (not 6), Task 1.3b is present

### Testing Requirements
This project requires tests alongside all code. For this task:
- Write/update tests in `internal/parser/parser_test.go`
- Test metadata extraction: complexity, files list, depends_on parsing
- Test acceptance criteria: `- [ ]` and `- [x]` patterns
- Test review comments: single-line `> 💬`, multi-line blockquotes
- Test edge cases: task with no metadata, task with no criteria
- Test the letter-suffix fix: parse a plan with `### Task 1.3b:` heading
- Update the spec file test to verify Feature 1 has 7 tasks
- Table-driven tests where appropriate
- All tests must pass with `go test ./internal/parser/... ./internal/models/...` before the task is complete

### Previous Sessions
None — this is session 001.

### Completed Prerequisites

**Task 1.2 (completed):**
All data models defined in `internal/models/models.go` with JSON tags. Types: Status, Complexity, Plan, Feature, Task, Criterion, SessionProgress. Helper methods: `Task.FullID()`, `Status.Icon()`, `Plan.TaskByID()`, `ParseStatus()`. You'll need to add a `Suffix string` field to `Task` and update `FullID()`.

**Task 1.3 (completed):**
Line-based state machine parser in `internal/parser/parser.go`. Handles plan/feature/task hierarchy, single-feature detection, code fence tracking. Task body text is captured in `Description` field. The parser currently skips `### Task 1.3b:` headings (regex bug). 12 existing tests — update the spec file test's expected task count after fixing the regex.

## Session Progress File

Update your progress file as you work:
`.etch/progress/build-etch--task-1.3b--001.md`

This file has been created for you. Fill in each section:
- **Changes Made:** files created or modified
- **Acceptance Criteria Updates:** check off what you completed
- **Decisions & Notes:** design decisions, important context
- **Blockers:** anything blocking progress
- **Next:** what still needs to happen
- **Status:** update to completed, partial, failed, or blocked

Rules:
- Stay within the files listed in scope. You MAY also modify `internal/models/models.go` and `internal/models/models_test.go` to add the Suffix field.
- Do NOT modify the plan file. Only update your progress file.
- Keep notes concise but useful — future sessions depend on them.
