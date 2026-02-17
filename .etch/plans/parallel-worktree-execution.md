# Plan: Parallel Worktree Execution for Etch Go

## Overview

Extend `etch go` with git worktree-based parallel task execution. When a plan runs, etch creates a feature branch for the plan and spins up isolated git worktrees for each active task. Each Claude Code agent runs inside its own worktree, enabling true parallel execution without file conflicts. When a task completes, its worktree branch is merged back into the feature branch and cleaned up. A `--workers N` flag (default 3) controls concurrency, and `--cleanup` handles orphaned worktrees from interrupted runs.

This plan depends on the sequential `etch go` command from the `etch-go-command` plan being implemented first. It refactors the runner from sequential to parallel execution while preserving the existing outcome handling, signal management, and status reconciliation.

---

## Feature 1: Git Worktree Management

### Task 1.1: Create the worktree lifecycle package [pending]
**Complexity:** medium
**Files:** internal/worktree/worktree.go

Create `internal/worktree` package that wraps git worktree operations. All git commands should use `os/exec` to call the `git` binary. The package should provide:

1. `CreateFeatureBranch(rootDir, planSlug string) (branchName string, err error)` — creates `etch/<plan-slug>` branch from current HEAD if it doesn't already exist. If it exists, verify it's a valid branch and return its name. Use `git branch --list` to check existence and `git branch <name>` to create.

2. `CreateWorktree(rootDir, planSlug, taskID, featureBranch string) (worktreePath string, err error)` — creates a worktree at `.etch/worktrees/task-<taskID>/` branched from the feature branch. The worktree branch should be named `etch/<plan-slug>/task-<taskID>`. Use `git worktree add -b <branch> <path> <start-point>`. Return the absolute path to the worktree.

3. `RemoveWorktree(rootDir, taskID string) error` — removes the worktree directory and prunes with `git worktree remove <path>` followed by `git worktree prune`.

4. `ListWorktrees(rootDir string) ([]WorktreeInfo, error)` — parses output of `git worktree list --porcelain` to return active worktrees with their paths and branches.

5. All functions should use `etcherr.*` constructors with `.WithHint()` for error handling.

**Acceptance Criteria:**
- [ ] Feature branch is created from current HEAD with `etch/<slug>` naming
- [ ] Idempotent: re-running with existing branch reuses it
- [ ] Worktrees are created at `.etch/worktrees/task-N.M/` with correct branch names
- [ ] Worktree removal cleans up both directory and git metadata
- [ ] ListWorktrees correctly parses `git worktree list --porcelain` output

### Task 1.2: Add merge-back logic [pending]
**Complexity:** medium
**Files:** internal/worktree/merge.go
**Depends on:** Task 1.1

Add merge functionality to the worktree package:

1. `MergeTaskBranch(rootDir, featureBranch, taskBranch string) error` — merges the task branch into the feature branch. Steps:
   - `git checkout <featureBranch>` (in the main repo)
   - `git merge --no-ff <taskBranch> -m "etch: merge task <taskID>"`
   - If merge fails (exit code != 0), return a typed `MergeConflictError` containing the conflicting files (parse `git diff --name-only --diff-filter=U`)
   - On success, delete the task branch with `git branch -d <taskBranch>`

2. `MergeConflictError` struct with `ConflictFiles []string` and `TaskID string` fields, implementing the `error` interface.

3. `AbortMerge(rootDir string) error` — runs `git merge --abort` for recovery from conflicts.

Important: The merge must happen in the main repository's working directory, not in the worktree. The worktree should be removed before merging to avoid git lock issues.

**Acceptance Criteria:**
- [ ] Successful merge creates a merge commit on the feature branch
- [ ] Task branch is deleted after successful merge
- [ ] Merge conflicts return a typed MergeConflictError with file list
- [ ] AbortMerge recovers cleanly from conflict state

### Task 1.3: Add cleanup command [pending]
**Complexity:** small
**Files:** internal/worktree/cleanup.go, cmd/go.go
**Depends on:** Task 1.1

Add cleanup functionality:

1. `Cleanup(rootDir string) (removed []string, err error)` in the worktree package — lists all worktrees under `.etch/worktrees/`, removes each one with `RemoveWorktree`, prunes with `git worktree prune`, and returns the list of removed paths.

2. Wire `--cleanup` flag into `cmd/go.go`. When `--cleanup` is passed, run cleanup instead of the normal execution flow. Print each removed worktree path and a summary count.

**Acceptance Criteria:**
- [ ] `etch go --cleanup` removes all worktrees under `.etch/worktrees/`
- [ ] Orphaned git worktree metadata is pruned
- [ ] Output lists each removed worktree and total count
- [ ] No error if no worktrees exist

---

## Feature 2: Parallel Execution Engine

### Task 2.1: Refactor runner for parallel dispatch [pending]
**Complexity:** large
**Files:** internal/runner/runner.go, internal/runner/worker.go
**Depends on:** Task 1.2

Refactor the sequential runner into a parallel execution engine. Create `internal/runner/worker.go` with:

1. A `WorkerPool` struct that manages N worker goroutines. Each worker:
   - Pulls tasks from a shared channel
   - Creates a worktree for the task via `worktree.CreateWorktree`
   - Assembles context (note: context assembly writes to `.etch/context/` and `.etch/progress/` which are in the main repo — these paths need to be accessible from the worktree, so pass the main repo's rootDir for context assembly but the worktree path for claude execution)
   - Launches Claude Code inside the worktree directory
   - Reads the progress file after completion
   - Sends the result back on a results channel

2. A `Scheduler` that manages the task DAG:
   - Maintains sets: `completed`, `running`, `ready`, `blocked`
   - Initially populates `ready` with tasks that have no unmet dependencies
   - When a worker reports a task completed, moves it to `completed`, re-evaluates `blocked` tasks to see if any become `ready`, and dispatches newly ready tasks to available workers
   - When a worker reports failure/partial/blocked, removes the task from `running` and records it (does NOT dispatch new tasks for that slot until the issue is addressed — actually, other independent tasks should still run)

3. The scheduler loop runs until: all tasks are completed, OR all remaining tasks are blocked/failed and no workers are running.

Use `sync.WaitGroup` for worker lifecycle and channels for task dispatch and result collection. The scheduler should be the only goroutine that mutates shared state (no locks needed if dispatch/results go through channels).

**Acceptance Criteria:**
- [ ] Worker pool spawns N goroutines controlled by `--workers` flag
- [ ] Tasks dispatch to workers as they become eligible
- [ ] Completed tasks trigger dependency re-evaluation and new dispatches
- [ ] Failed/partial/blocked tasks don't block other independent tasks from running
- [ ] Scheduler exits when all tasks complete or all remaining are stuck

### Task 2.2: Add --workers flag and integrate worktrees [pending]
**Complexity:** medium
**Files:** cmd/go.go, internal/runner/runner.go
**Depends on:** Task 2.1

Wire the parallel execution into the CLI:

1. Add `--workers` flag (int, default 3) to `cmd/go.go`
2. At the start of `etch go`, create the feature branch via `worktree.CreateFeatureBranch`
3. Pass the worker count and feature branch name into the runner config
4. When `--workers 1` is passed, the system should still work correctly (sequential behavior via single worker)
5. After each successful task completion, the worker should:
   - Remove the worktree
   - Merge the task branch into the feature branch
   - If merge conflict: send a conflict result back to the scheduler, which pauses that task's merge but continues other workers
6. Update the status output to show which tasks are running in parallel (e.g. "Workers: 3/3 active — Task 1.1, Task 1.2, Task 2.1")

**Acceptance Criteria:**
- [ ] `--workers N` flag controls concurrency (default 3)
- [ ] Feature branch is created at start of run
- [ ] Each task runs in its own worktree
- [ ] Successful tasks are merged back and worktrees cleaned up
- [ ] Merge conflicts pause the affected task without stopping other workers
- [ ] Status output shows active worker count and running tasks

### Task 2.3: Update signal handling for parallel workers [pending]
**Complexity:** medium
**Files:** internal/runner/runner.go, internal/runner/worker.go
**Depends on:** Task 2.1

Update the ctrl+c handling from the sequential plan to work with parallel workers:

1. When context is cancelled (ctrl+c), the scheduler should:
   - Stop dispatching new tasks
   - Wait for all running workers to finish their current claude sessions (claude handles its own ctrl+c)
   - After all workers stop, read progress files for each running task
   - Clean up worktrees for tasks that didn't complete (leave the worktree branches intact so work isn't lost)
   - Print summary: which tasks completed, which were interrupted mid-execution, which never started

2. A second ctrl+c should force-kill: cancel all worker contexts immediately and exit. Worktrees are left in place (user can run `etch go --cleanup` later).

3. The scheduler should track which worktrees are active so cleanup knows what to handle.

**Acceptance Criteria:**
- [ ] First ctrl+c stops new dispatches and waits for running workers
- [ ] Second ctrl+c force-exits, leaving worktrees for later cleanup
- [ ] Progress files from interrupted tasks are read and reported
- [ ] Summary shows completed, interrupted, and pending tasks
- [ ] Worktree branches are preserved (not deleted) for interrupted tasks

---

## Feature 3: Testing

### Task 3.1: Test worktree package [pending]
**Complexity:** medium
**Files:** internal/worktree/worktree_test.go
**Depends on:** Task 1.2

Write tests for the worktree package using `os.MkdirTemp` and `git init` to create temporary git repos:

1. Test `CreateFeatureBranch`: creates branch, idempotent on second call
2. Test `CreateWorktree`: creates directory and branch, verify files are accessible
3. Test `RemoveWorktree`: cleans up directory and git metadata
4. Test `MergeTaskBranch`: commit a file in the worktree branch, merge succeeds, branch deleted
5. Test merge conflict: create conflicting commits on both branches, verify `MergeConflictError` is returned with correct files
6. Test `Cleanup`: create multiple worktrees, verify all removed

Each test should create a fresh temp directory with `git init`, make an initial commit (git worktree requires at least one commit), and clean up after.

**Acceptance Criteria:**
- [ ] Tests cover create, remove, list, merge, conflict, and cleanup operations
- [ ] Tests use real git repos (not mocks) for correctness
- [ ] Tests are isolated with temp directories
- [ ] Tests pass with `go test ./internal/worktree`

### Task 3.2: Test parallel runner [pending]
**Complexity:** medium
**Files:** internal/runner/runner_test.go
**Depends on:** Task 2.1

Extend the runner tests to cover parallel execution:

1. Test that independent tasks (no dependencies between them) run concurrently — use a fake claude launcher that records start/end times and verify overlap
2. Test that dependent tasks wait: task B depends on task A, verify B doesn't start until A completes
3. Test that a failed task doesn't block independent tasks from running
4. Test worker count limiting: with 2 workers and 4 independent tasks, verify at most 2 run concurrently
5. Test merge conflict handling: fake a conflict and verify the scheduler continues other tasks

Use the existing fake claude launcher interface, extended to simulate worktree operations.

**Acceptance Criteria:**
- [ ] Tests verify concurrent execution of independent tasks
- [ ] Tests verify dependency ordering is respected under parallelism
- [ ] Tests verify worker count is respected
- [ ] Tests verify merge conflict handling doesn't stop other workers
- [ ] Tests pass with `go test ./internal/runner`
