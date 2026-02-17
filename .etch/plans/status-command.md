# Plan: Enhance Status Command

## Overview

Improve the `etch status` command to be more useful and visually appealing. By default, only show plans that are in progress, with an `--all` flag to show everything. Beautify the output with prominent slug display, overall percentage with a progress bar, and blocked/unblocked task indicators.

### Task 1: Filter to active plans and beautify output [completed]
**Complexity:** large
**Files:** cmd/status.go, internal/status/status.go

Add an `--all` flag to the status command. By default, filter to only "active" plans — those with at least one in-progress, failed, or blocked task, or partially completed. Fully pending and fully completed plans are hidden unless `--all` is passed. When no active plans exist, print "No active plans. Use --all to see all plans."

Simultaneously redesign the formatting in `FormatSummary` and `FormatDetailed`:

1. Add `CompletedTasks` and `TotalTasks` fields to `PlanStatus`, populated during `reconcile`. Display overall percentage next to the plan title.
2. Add a 10-char progress bar using block characters, e.g. `[████░░░░░░] 45%`.
3. Show the slug prominently on its own line as `  slug: my-plan-slug` so users can copy it for `etch run`, `etch context`, etc.
4. Add separator lines between plans in summary view.
5. Align task output consistently.

**Acceptance Criteria:**
- [X] `--all` flag added; default output filters to active plans only
- [X] When no active plans, helpful message is printed
- [X] Overall percentage and progress bar displayed per plan
- [X] Slug shown prominently and clearly labeled
- [X] Plans visually separated in summary view

### Task 2: Blocked/unblocked task indicators and tests [completed]
**Complexity:** large
**Files:** internal/status/status.go, internal/status/status_test.go
**Depends on:** Task 1

Add dependency-aware blocked task detection. Enrich `TaskStatus` with `DependsOn []string` and a computed `IsBlocked bool` during reconcile. A pending task is blocked if any of its `DependsOn` tasks are not completed. Update the output so blocked tasks show `⊘` and ready-to-work pending tasks show `○`. In detailed view, show which tasks a blocked task is waiting on.

Write table-driven tests covering: `IsActive` filtering, percentage calculation edge cases, progress bar rendering, blocked task resolution with dependency chains, and format output spot-checks.

**Acceptance Criteria:**
- [x] Pending tasks with unmet dependencies show as blocked `⊘`
- [x] Pending tasks with no deps or all deps met show as ready `○`
- [x] Detailed view shows which tasks a blocked task depends on
- [x] Tests cover filtering, percentage, progress bar, and blocked detection
- [x] All tests pass with `go test ./internal/status/...`
