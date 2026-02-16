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
  ▶ Task 1.4: Plan markdown serializer (pending — this is your task)
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

## Your Task: Task 1.4 — Plan markdown serializer
**Complexity:** medium
**Files in Scope:** internal/serializer/serializer.go, internal/serializer/serializer_test.go
**Depends on:** Task 1.3b (completed)

Two modes:

**Full serialization:** `Plan` struct → complete markdown string. Used for new plans from AI generation.

Output must match the plan markdown format spec:
```
# Plan: <Plan Title>

## Overview
<overview text>

---

## Feature N: <Feature Title>

### Overview
<feature overview>

### Task N.M: <Task Title> [status]
**Complexity:** <complexity>
**Files:** <comma-separated>
**Depends on:** <dependencies or "none">

<description>

> 💬 <review comment>

**Acceptance Criteria:**
- [ ] <criterion>
- [x] <completed criterion>

---
```

For single-feature plans (only one feature), omit the `## Feature N:` heading — use `### Task N:` format directly.

**Targeted update:** Modify specific fields in an existing plan file without rewriting everything. Used by `etch status` to update status tags and checkboxes.

Targeted update approach:
- Read the file as lines
- For status update: find the line matching `### Task N.M:` (or `### Task N.Mb:` for letter-suffix IDs) and replace the `[old_status]` with `[new_status]`
- For checkbox update: within a task's section, find the criterion line and flip `[ ]` to `[x]`
- Write the file back

This preserves all formatting, descriptions, comments, and whitespace that the full serializer might subtly alter.

### Acceptance Criteria
- [ ] Full serialize produces valid markdown matching the format spec
- [ ] Targeted update: changes a task's status tag without touching other content
- [ ] Targeted update: flips acceptance criteria checkboxes
- [ ] Preserves all unrelated content (descriptions, comments, blank lines)
- [ ] Round-trip test: parse → full serialize → parse produces equivalent Plan
- [ ] Targeted update doesn't introduce formatting drift

### Testing Requirements
This project requires tests alongside all code. For this task:
- Write tests in `internal/serializer/serializer_test.go`
- Test full serialization: multi-feature plan, single-feature plan, plan with comments, plan with mixed criteria states
- Test targeted status update: change one task's status, verify rest of file unchanged
- Test targeted checkbox update: flip criteria, verify rest of file unchanged
- Round-trip test: build a Plan struct, serialize it, parse it back with the parser, compare
- Test with letter-suffix task IDs (e.g. Task 1.3b)
- All tests must pass with `go test ./internal/serializer/...` before the task is complete

### Previous Sessions
None — this is session 001.

### Completed Prerequisites

**Task 1.2 (completed):**
Data models in `internal/models/models.go`. Key types: Plan, Feature, Task (with Suffix field for letter IDs like "b"), Criterion, Status, Complexity. `Task.FullID()` returns e.g. "1.3b". `Status.Icon()` returns unicode icons.

**Task 1.3 (completed):**
Line-based state machine parser in `internal/parser/parser.go`. `Parse(io.Reader)` and `ParseFile(path)` functions. Handles code fence tracking, single-feature detection, separator skipping.

**Task 1.3b (completed):**
Metadata extraction added to parser. Extracts: complexity, files, depends_on, acceptance criteria (with check state), review comments (single and multi-line). Letter-suffix task IDs now supported (`### Task 1.3b:` parses correctly). The parser is the inverse of what you're building — use it for round-trip testing.

## Session Progress File

Update your progress file as you work:
`.etch/progress/build-etch--task-1.4--001.md`

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
