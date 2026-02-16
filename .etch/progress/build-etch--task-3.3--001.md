# Session: Task 3.3 – List and utility commands
**Plan:** build-etch
**Task:** 3.3
**Session:** 001
**Started:** 2026-02-15 15:41
**Status:** completed

## Changes Made
- `cmd/list.go` — Implemented `etch list` with plan title, task counts, completion %
- `cmd/open.go` — Implemented `etch open <plan>` with $EDITOR fallback to vi, `findPlanBySlug` helper
- `cmd/delete.go` — New file. Implemented `etch delete <plan>` with confirmation prompt, `--yes` flag, removes plan + progress + context files
- `cmd/root.go` — Registered `deleteCmd()` in app commands
- `cmd/list_test.go` — New file. Tests for list (multiple plans, no plans), delete (removes files, missing plan, cancellation), findPlanBySlug

## Acceptance Criteria Updates
- [x] List shows summary for all plans
- [x] Open launches editor
- [x] Delete requires confirmation, removes plan + progress files
- [x] Missing plan handled gracefully
- [x] Tests: list output with multiple plans, delete removes plan + progress files, missing plan error
- [x] `go test ./cmd/...` passes

## Decisions & Notes
- `delete` command also removes context files (`.etch/context/<slug>--*.md`) since they're derived artifacts
- Added `--yes`/`-y` flag to skip confirmation for scripting
- `findPlanBySlug` shared helper in `open.go` — discovers plans via `DiscoverPlans`, falls back to direct file check
- Reused `askYesNo` from `init.go` for delete confirmation
- Test helpers (`setupEtchProject`, `writePlan`, `writeProgress`, `captureStdout`, `chdirTo`) in list_test.go for reuse

## Blockers
None.

## Next
All acceptance criteria met. Task complete.
