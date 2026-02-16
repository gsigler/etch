# Etch Context — Implementation Task

You are working on a task as part of an implementation plan managed by Etch.

## Plan: Build Etch – AI Implementation Planning CLI
Etch is a Go CLI tool that helps developers create, review, and execute AI-generated implementation plans. It uses markdown plan files as the spec, generates context prompt files for AI coding agents, and tracks progress across sessions with file-based state.

## Current Plan State
Feature 1: Core Data Layer & Plan Parser
  ✓ Task 1.1: Project scaffold and CLI skeleton (completed)
  ▶ Task 1.2: Data models (pending — this is your task)
  ○ Task 1.3: Plan markdown parser — structure (pending, depends on 1.2)
  ○ Task 1.3b: Plan parser — task metadata extraction (pending, depends on 1.3)
  ○ Task 1.4: Plan markdown serializer (pending, depends on 1.3b)
  ○ Task 1.5: Progress file reader/writer (pending, depends on 1.2)
  ○ Task 1.6: Config management (pending, depends on 1.1)
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

## Your Task: Task 1.2 — Data models
**Complexity:** small
**Files in Scope:** internal/models/models.go, internal/models/models_test.go
**Depends on:** Task 1.1 (completed)

Define core data structures:

```go
type Status string
const (
    StatusPending    Status = "pending"
    StatusInProgress Status = "in_progress"
    StatusBlocked    Status = "blocked"
    StatusCompleted  Status = "completed"
    StatusFailed     Status = "failed"
)

type Complexity string
const (
    ComplexitySmall  Complexity = "small"
    ComplexityMedium Complexity = "medium"
    ComplexityLarge  Complexity = "large"
)

type Plan struct {
    Title    string
    Overview string
    Features []Feature
    FilePath string    // path to the .md file
    Slug     string    // derived from filename
}

type Feature struct {
    Number   int
    Title    string
    Overview string
    Tasks    []Task
}

type Task struct {
    FeatureNumber int
    TaskNumber    int
    Title         string
    Status        Status
    Complexity    Complexity
    Files         []string
    DependsOn     []string
    Description   string
    Criteria      []Criterion
    Comments      []string   // 💬 review comments
}

type Criterion struct {
    Description string
    IsMet       bool
}

type SessionProgress struct {
    PlanSlug      string
    TaskID        string    // "1.1"
    SessionNumber int
    Started       string
    Status        string    // completed, partial, failed, blocked
    ChangesMade   []string
    CriteriaUpdates []Criterion
    Decisions     string
    Blockers      string
    Next          string
}
```

Include helper methods: `Task.FullID()` → `"1.2"`, `Status.Icon()` → `"✓"/"▶"/"○"/"✗"/"⊘"`, `Plan.TaskByID(id string)`, `ParseStatus(s string) Status`.

### Acceptance Criteria
- [ ] All types defined with JSON tags for potential future use
- [ ] Helper methods implemented with table-driven tests in `internal/models/models_test.go`
- [ ] `ParseStatus` handles unknown strings gracefully (defaults to Pending) — tested
- [ ] `Task.FullID()` returns correct format for single and multi-feature plans — tested
- [ ] `Status.Icon()` returns correct icon for all status values — tested
- [ ] `go test ./internal/models/...` passes

### Testing Requirements
This project requires tests alongside all code. For this task:
- Write table-driven tests in `internal/models/models_test.go`
- Cover all helper methods: `Task.FullID()`, `Status.Icon()`, `ParseStatus()`, `Plan.TaskByID()`
- Test edge cases: unknown status strings, task not found, single-feature vs multi-feature FullID
- All tests must pass with `go test ./internal/models/...` before the task is complete

### Previous Sessions
None — this is session 001.

### Completed Prerequisites

**Task 1.1 (completed):**
Project scaffold created with `urfave/cli/v2`. All 8 subcommands registered as stubs. `etch init` fully implemented (creates .etch dirs, config.toml, gitignore handling). Internal packages exist as empty placeholders. `internal/models/models.go` exists with just `package models`.

## Session Progress File

Update your progress file as you work:
`.etch/progress/build-etch--task-1.2--001.md`

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
