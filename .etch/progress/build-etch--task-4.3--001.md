# Session: Task 4.3 – AI refinement flow in TUI
**Plan:** build-etch
**Task:** 4.3
**Session:** 001
**Started:** 2026-02-15 16:47
**Status:** completed

## Changes Made
- Created `internal/tui/diff.go` — LCS-based line diff computation, colored diff line rendering (green added, red removed, gray context), diffStats helper
- Created `internal/tui/review.go` — Full refinement orchestration: `RefineFunc` type + `Option`/`WithRefineFunc` for DI, `collectComments` (gathers all 💬 across tasks), `backupPlan`/`restoreBackup` (timestamped .bak files), `startRefinement` (async tea.Cmd), spinner setup, mode handlers (`enterApplyMode`, `updateApplyConfirm`, `handleRefinementResult`, `updateLoadingKey`, `updateDiff`), cleanup, `viewLoading` (centered spinner), `viewDiff` (scrollable colored diff with stats)
- Modified `internal/tui/model.go` — Added modes `modeApplyConfirm`, `modeLoading`, `modeDiff`; added refinement state fields (refineFn, spinner, diffLines, diffOffset, oldPlanContent, newPlanContent, backupPath, applyCommentCount); changed `New()` to accept variadic `Option`s (backwards-compatible); updated `Update()` to handle `refinementResultMsg`, `spinner.TickMsg`, and new mode dispatching; added `a` key in `updateNormal`; updated `viewHeight` for apply confirm; refactored `View()` to dispatch loading/diff views and show apply confirm prompt
- Modified `internal/tui/view.go` — Added `a:apply` to top bar hints; added `APPLY REFINEMENT` mode indicator in bottom bar

## Acceptance Criteria Updates
- [x] `a` triggers refinement with confirmation
- [x] Loading spinner while waiting
- [x] Colored diff view
- [x] Accept/reject flow
- [x] Plan backup happens before changes
- [x] API errors shown gracefully
- [x] "No comments to send" if no 💬 found

## Decisions & Notes
- `RefineFunc` is injected via functional options pattern (`tui.New(plan, path, tui.WithRefineFunc(fn))`) — backwards-compatible with existing `cmd/review.go` which passes no options
- When `refineFn` is nil, pressing `a` shows "Refinement not configured" — Task 2.3 will wire up the real API client
- Backup uses timestamped filename (`plan.md.bak.20260215-164700`) to avoid overwriting previous backups
- Diff uses LCS algorithm (O(n*m)) — fine for plan files which are typically <500 lines
- Accept writes new content and reloads plan; reject restores from backup — backup is cleaned up in both cases
- ESC cancels during loading mode (though the async API call still completes)
- `collectComments` prefixes each comment with `[Task X.Y]` for API context

## Blockers
None — Task 2.3 (refinement API logic) is pending but the TUI flow is fully implemented with a pluggable `RefineFunc` interface.

## Next
- Task 2.3 needs to implement the actual `RefineFunc` and wire it into `cmd/review.go` via `tui.WithRefineFunc()`
- Consider adding context-only diff mode (suppress unchanged lines far from changes) for large plans
