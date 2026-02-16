# Session: Task 1.4 – Plan markdown serializer
**Plan:** build-etch
**Task:** 1.4
**Session:** 001
**Started:** 2026-02-15
**Status:** completed

## Changes Made
- Created `internal/serializer/serializer.go` — full serializer + targeted update functions
- Created `internal/serializer/serializer_test.go` — 14 tests covering all acceptance criteria
- Modified `internal/parser/parser.go` — added `criteriaHeadingRe` to skip `**Acceptance Criteria:**` heading lines (needed for clean round-tripping)

## Acceptance Criteria Updates
- [x] Full serialize produces valid markdown matching the format spec
- [x] Targeted update: changes a task's status tag without touching other content
- [x] Targeted update: flips acceptance criteria checkboxes
- [x] Preserves all unrelated content (descriptions, comments, blank lines)
- [x] Round-trip test: parse → full serialize → parse produces equivalent Plan
- [x] Targeted update doesn't introduce formatting drift

## Decisions & Notes
- Single-feature detection: `len(plan.Features) == 1` → omit `## Feature N:` heading, use `### Task N:` format
- Parser required a minor fix: `**Acceptance Criteria:**` heading was being accumulated into task description. Added `criteriaHeadingRe` pattern to skip it. This doesn't change existing parser test behavior (all 17 parser tests still pass).
- Targeted updates (`UpdateTaskStatus`, `UpdateCriterion`) operate on raw file lines with regex, never re-serializing, so they preserve all formatting exactly.
- `UpdateCriterion` scopes its search to the target task section (between `### Task` headings) to avoid accidentally flipping identically-named criteria in other tasks.

## Blockers
None.

## Next
Task complete. All 31 tests pass (17 parser + 14 serializer).
