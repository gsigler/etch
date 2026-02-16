# Session: Task 2.4 – Replan command
**Plan:** build-etch
**Task:** 2.4
**Session:** 001
**Started:** 2026-02-15 18:36
**Status:** completed

## Changes Made
- internal/generator/replan.go (created) — core replan logic: target resolution, scope building, session history formatting, Replan() and ApplyReplan() functions
- internal/generator/replan_test.go (created) — 28 tests covering target resolution, prompt adaptation, integration with mock API, backup creation, task splitting
- internal/generator/prompts.go (modified) — added replan system prompt and buildReplanUserMessage()
- cmd/replan.go (modified) — full CLI command with target parsing, plan discovery, diff display, confirmation prompt

## Acceptance Criteria Updates
- [x] Target resolution works for task IDs, feature references, and plan-scoped IDs
- [x] Adapts prompt based on session history (planning issue vs approach issue)
- [x] Feature-level replan preserves completed tasks
- [x] Backs up plan before changes
- [x] Shows diff and requires confirmation
- [x] Can split a single task into multiple tasks
- [x] Updated plan parses correctly
- [x] Tests: target resolution (task ID, feature ref, plan-scoped), prompt adaptation based on session history, backup creation. Use mock API client and fixture plans.
- [x] `go test ./internal/generator/...` passes

## Decisions & Notes
- ReplanTarget struct uses Type field ("task" or "feature") to distinguish scope, with pointers to the resolved Task or Feature.
- ResolveTarget supports: "1.2" (task ID), "1.3b" (with suffix), "feature:2" (feature ref), "Frontend UI" (title match), "frontend" (partial title), bare numbers (prefer task, fall back to feature).
- BuildReplanScope adapts prompt: no sessions = "planning issue" with scope/criteria questions; has sessions = "approach issue" with full session history and suggestion to restructure.
- Feature-level replan explicitly lists completed tasks with "MUST be preserved" instruction, and pending tasks as "tasks to replan".
- Reuses BackupPlan, GenerateDiff, ApplyRefinement from refine.go (ApplyReplan delegates to ApplyRefinement).
- Replan system prompt includes instruction about task splitting with suffixes (1.3a, 1.3b).
- cmd/replan.go handles plan-scoped targets via 2-arg form: `etch replan my-plan 1.2`.

## Blockers
None.

## Next
All acceptance criteria met. Task complete.
