# Plan: Run or Context an Entire Feature at Once [completed]

## Overview

Add a `--feature` / `-f` flag to both `etch run` and `etch context` commands, allowing users to target an entire feature instead of a single task. When `--feature` is provided (e.g., `--feature 2`), the command assembles a combined context prompt containing all tasks in that feature (respecting dependency order and skipping completed tasks), then either prints the context path (`etch context`) or launches Claude Code (`etch run`) with the combined prompt. This enables tackling a cohesive feature in a single agent session rather than task-by-task.

The implementation adds a new `AssembleFeature` function in `internal/context` that produces a unified context file covering multiple tasks, reuses the existing progress infrastructure per-task, and extends the CLI flag handling in `cmd/context.go` and `cmd/run.go`.

### Task 1: Add ResolveFeature function to internal/context [completed]
**Complexity:** medium
**Files:** internal/context/context.go
**Depends on:** (none)

Add a `ResolveFeature` function that takes plans, planSlug, and featureNumber, then returns the plan and the `*models.Feature`. It should:

1. Filter plans by slug (like `ResolveTask` does)
2. Find the feature by number within the resolved plan
3. Return an error with a helpful hint if the feature is not found (e.g., "run 'etch status' to see available features")

Also add a helper `featurePendingTasks` that returns the ordered list of pending tasks in a feature whose dependencies are satisfied or are within the feature itself, skipping completed tasks.

**Acceptance Criteria:**
- [x] `ResolveFeature` resolves a feature by number within a plan
- [x] `ResolveFeature` returns typed errors with hints when feature not found
- [x] `featurePendingTasks` returns pending tasks in dependency order, skipping completed ones

### Task 2: Add AssembleFeature function to internal/context [completed]
**Complexity:** large
**Files:** internal/context/context.go
**Depends on:** Task 1

Add an `AssembleFeature` function that builds a combined context prompt for all actionable tasks in a feature. It should:

1. Accept `rootDir`, `plan`, and `feature` as arguments
2. Get all pending tasks in the feature (using `featurePendingTasks` from Task 1)
3. Create a progress file for each pending task (using existing `progress.WriteSession`)
4. Build a combined template that includes:
   - The plan overview and current plan state (same as single-task context)
   - A "Your Feature" section listing all tasks being worked on with their descriptions, files, and criteria
   - Each task's full details (description, acceptance criteria, review comments, previous sessions)
   - Progress reporting instructions for each task (with their individual `etch progress` commands)
5. Write a single context file named `<slug>--feature-<N>--<session>.md`
6. Return a `FeatureResult` struct containing the context path, a map of task IDs to progress paths, session number, and token estimate

The template should instruct the agent to work through tasks in order, marking each done via `etch progress done` before moving to the next.

**Acceptance Criteria:**
- [x] `AssembleFeature` creates progress files for each pending task in the feature
- [x] Combined context includes all task details, criteria, and progress instructions
- [x] Context file is written with feature-based naming convention
- [x] Returns `FeatureResult` with all progress paths and token estimate
- [x] Template instructs the agent to work tasks in order and report progress per-task

### Task 3: Add --feature flag to etch context and etch run commands [completed]
**Complexity:** medium
**Files:** cmd/context.go, cmd/run.go
**Depends on:** Task 2

Add a `--feature` / `-f` string flag to both commands. Update the resolution logic:

1. Add the flag definition to both `contextCmd()` and `runCmd()`
2. When `--feature` is provided, call `etchcontext.ResolveFeature` and `etchcontext.AssembleFeature` instead of the task-based flow
3. The `--feature` flag is mutually exclusive with `--task` — return a usage error if both are provided
4. `--plan` works with `--feature` the same way it works with `--task`
5. Update output messages to show feature info (e.g., "Context assembled for Feature 2 — Auth System (3 tasks, session 001)")
6. For `etch run`, read the combined context file and pipe it to `claude.RunWithStdin` just like the task flow

Factor the shared feature resolution logic into a `resolveFeatureArgs` helper in `cmd/context.go` (parallel to `resolveContextArgs`).

**Acceptance Criteria:**
- [x] `--feature` / `-f` flag is available on both `etch context` and `etch run`
- [x] `--feature` and `--task` are mutually exclusive with a clear error message
- [x] `etch context --feature 2` prints context file path and token estimate
- [x] `etch run --feature 2` launches Claude Code with the combined feature context
- [x] `--plan` flag works in combination with `--feature`

### Task 4: Add tests for feature context assembly [completed]
**Complexity:** medium
**Files:** internal/context/context_test.go
**Depends on:** Task 2

Write tests for the new feature-level functions using table-driven patterns and `os.MkdirTemp`:

1. Test `ResolveFeature` — valid feature number, invalid feature number, feature in specific plan
2. Test `featurePendingTasks` — all pending, some completed, dependency ordering, all completed (empty result)
3. Test `AssembleFeature` — verify the context file is created, contains all task details, progress files are created for each task, and token estimate is reasonable
4. Test edge cases: feature with a single task (should work identically to task-level context), feature where all tasks are completed (should return an error)

**Acceptance Criteria:**
- [x] Tests cover `ResolveFeature` with valid and invalid inputs
- [x] Tests cover `featurePendingTasks` with various task states
- [x] Tests cover `AssembleFeature` output content and file creation
- [x] Tests cover edge cases (single task, all completed)
- [x] Tests pass with `go test ./internal/context`
