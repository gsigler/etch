# Plan: Etch Go Command — Autonomous Plan Execution

## Overview

Add an `etch go` command that autonomously executes an entire plan by iterating through tasks in dependency order. For each task, it assembles context, launches Claude Code via shell, waits for completion, reads the progress file to determine the outcome, and decides whether to continue or pause. The command runs tasks sequentially (parallel execution is out of scope), supports graceful ctrl+c shutdown that preserves progress, and prints clear status output between tasks showing what completed, what's next, and overall progress.

This builds on top of the existing `etch run` (single-task launcher), `internal/context` (context assembly and dependency resolution), `internal/progress` (progress file reading), and `internal/status` (reconciliation) packages. The new command orchestrates these into a loop with outcome-based control flow.

### Task 1: Create the go command skeleton and task ordering [pending]
**Complexity:** medium
**Files:** cmd/go.go, cmd/root.go
**Depends on:** (none)

Create `cmd/go.go` with the `goCmd()` factory function following the existing CLI pattern. Register it in `cmd/root.go`. The command accepts an optional `[plan-slug]` argument (or auto-selects like `etch run` does). It should:

1. Resolve the project root and discover plans (reuse `findProjectRoot`, `etchcontext.DiscoverPlans`)
2. Parse the plan and read all progress to determine current state
3. Build an ordered list of actionable tasks: skip already-completed tasks, identify pending tasks whose dependencies are all satisfied, and detect blocked tasks
4. Print the execution plan showing which tasks will run and which are already done
5. If no actionable tasks remain, print a summary and exit

The task ordering logic should use the existing `effectiveStatus` and `allDepsCompleted` functions from `internal/context` — either by exporting them or by duplicating the logic in a new `internal/runner` package. Prefer creating `internal/runner/runner.go` to keep orchestration logic testable and separate from CLI wiring.

**Acceptance Criteria:**
- [ ] `etch go` command is registered and appears in `etch --help`
- [ ] Accepts optional `[plan-slug]` argument, auto-selects if only one plan exists
- [ ] Correctly identifies completed, pending-ready, and blocked tasks
- [ ] Prints execution plan summary before starting (e.g. "3 tasks to run, 2 already completed, 1 blocked")
- [ ] Exits cleanly when all tasks are already completed

### Task 2: Implement the task execution loop [pending]
**Complexity:** large
**Files:** internal/runner/runner.go, cmd/go.go
**Depends on:** Task 1

Implement the core execution loop in `internal/runner`. For each ready task in order:

1. Print a header showing task ID, title, and position (e.g. "Task 1.2 — Add auth middleware [2/5]")
2. Assemble context using `etchcontext.Assemble` (which creates both context and progress files)
3. Launch Claude Code with the context via `claude.RunWithStdin`, connecting stdout/stderr to the terminal so the user can watch
4. After Claude exits, read the progress file that was created during assembly
5. Determine the outcome from the progress file's `**Status:**` field:
   - `completed` → mark task done, print success, continue to next task
   - `partial` → pause execution, print what was done and what remains
   - `blocked` → pause execution, print blockers
   - `failed` → pause execution, print failure info
   - Progress file still says `pending` (agent didn't update it) → treat as failed, pause
6. After marking a task complete, re-evaluate which tasks are now unblocked (a completed task may satisfy dependencies for later tasks)
7. When pausing, show a summary: what completed this session, what caused the pause, and remaining tasks

The runner should accept a `RunConfig` struct with the root dir, plan, and an interface for launching claude (to enable testing with fakes).

**Acceptance Criteria:**
- [ ] Tasks execute sequentially in dependency order
- [ ] Context is assembled and Claude is launched for each task
- [ ] Progress file is read after each task completes
- [ ] Completed tasks advance the loop; partial/blocked/failed/no-update tasks pause execution
- [ ] Dependencies are re-evaluated after each completion (newly unblocked tasks become eligible)
- [ ] Clear status output between tasks showing progress

### Task 3: Add graceful ctrl+c shutdown [pending]
**Complexity:** medium
**Files:** internal/runner/runner.go, cmd/go.go
**Depends on:** Task 2

Add signal handling for graceful shutdown:

1. Set up a `signal.Notify` channel for `os.Interrupt` (ctrl+c) in the command action
2. Pass a `context.Context` (with cancel) into the runner
3. Between tasks (not during claude execution), check if context is cancelled
4. If cancelled between tasks: print what was completed so far, what was interrupted, and remaining tasks. Exit with code 0 since progress is preserved
5. During claude execution: the first ctrl+c should be forwarded to claude (it handles its own graceful shutdown). A second ctrl+c should kill the process group
6. After claude exits due to interrupt, read the progress file as normal — claude may have written partial progress

**Acceptance Criteria:**
- [ ] Ctrl+c between tasks stops the loop and prints a summary
- [ ] Ctrl+c during claude execution forwards to claude first
- [ ] Progress is never lost — whatever was written to progress files is preserved
- [ ] Exit message shows what completed and what remains

### Task 4: Add status reconciliation and summary output [pending]
**Complexity:** medium
**Files:** internal/runner/runner.go, cmd/go.go
**Depends on:** Task 2

After each task completes (and at the end of the run), reconcile progress back into the plan file using the existing `status.Run` logic, and print a clear summary:

1. After each completed task, call `status.Run` to reconcile progress files into the plan markdown (updating `[pending]` → `[completed]`, checking off acceptance criteria)
2. Print a between-task status block showing: completed tasks (with checkmark), current task (with arrow), remaining tasks (with circle), and a progress bar
3. At the end of a full run (all tasks complete), print a final summary: total tasks completed, total time elapsed, and a success message
4. At the end of a partial run (paused), print: tasks completed this session, reason for pause, and suggested next step (e.g. "fix the blocker and run `etch go` again")

**Acceptance Criteria:**
- [ ] Plan file is updated after each task completion (status tags and criteria checkboxes)
- [ ] Between-task output shows clear progress with icons matching `etch status` style
- [ ] Final summary distinguishes between full completion and partial runs
- [ ] Progress bar shows overall plan completion percentage

### Task 5: Add tests for the runner [pending]
**Complexity:** medium
**Files:** internal/runner/runner_test.go
**Depends on:** Task 2

Write tests for the runner package using table-driven patterns and `os.MkdirTemp` for filesystem isolation:

1. Test task ordering: verify tasks are executed in correct dependency order, and already-completed tasks are skipped
2. Test outcome handling: mock the claude launcher to simulate completed/partial/failed/blocked outcomes by writing different progress file statuses
3. Test dependency re-evaluation: when task A completes, verify task B (which depends on A) becomes eligible
4. Test all-complete scenario: verify clean exit when no tasks need running
5. Test all-blocked scenario: verify appropriate messaging when remaining tasks are all blocked

Use an `APIClient`-style interface for the claude launcher so tests can inject a fake that writes to the progress file and returns.

**Acceptance Criteria:**
- [ ] Tests cover task ordering with dependencies
- [ ] Tests cover all four outcome types (completed, partial, failed, blocked)
- [ ] Tests cover dependency unblocking
- [ ] Tests cover edge cases (no tasks, all complete, all blocked)
- [ ] Tests pass with `go test ./internal/runner`
