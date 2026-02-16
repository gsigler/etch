# Session: Task 2.3 – Plan refinement from review comments
**Plan:** build-etch
**Task:** 2.3
**Session:** 001
**Started:** 2026-02-15 18:33
**Status:** completed

## Changes Made
- Created `internal/generator/refine.go` — refinement logic: comment extraction, backup, API call with streaming, response validation, colored diff generation, apply function
- Created `internal/generator/refine_test.go` — 18 tests covering all acceptance criteria
- Modified `internal/generator/prompts.go` — added refine system prompt and user message builder

## Acceptance Criteria Updates
- [x] Extracts 💬 comments from plan
- [x] Backs up plan before making changes
- [x] Sends plan + comments to Claude API
- [x] Validates response parses correctly
- [x] Shows colored terminal diff
- [x] Confirmation prompt before applying
- [x] Addressed comments removed, unaddressed preserved
- [x] "No comments found" message if plan has no 💬 comments
- [x] Tests: backup creation, comment extraction, "no comments" path, diff generation. Use mock API client.
- [x] `go test ./internal/generator/...` passes

## Decisions & Notes
- `Refine()` returns a `RefineResult` with old/new markdown, parsed plan, backup path, and diff string. The caller handles the confirmation prompt and calls `ApplyRefinement()` — same separation-of-concerns pattern as `Generate()`/`WritePlan()`.
- Comment extraction groups comments by task ID for context in the refinement prompt.
- Diff uses LCS-based algorithm for accurate line-level comparison with ANSI color codes (red for removed, green for added).
- "No comments" check happens before backup creation to avoid unnecessary backups.
- Backup naming: `<plan-slug>-<YYYYMMDD-HHMMSS>.md` in `.etch/backups/`.
- Refine system prompt includes the full format spec to ensure the AI preserves the plan format.

## Blockers
None.

## Next
Task is complete. All 40 generator tests pass (22 existing + 18 new).
