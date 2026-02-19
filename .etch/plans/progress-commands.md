# Plan: Progress Commands [completed]

## Overview

Add `etch progress` subcommands that let AI agents report their work on tasks programmatically. These commands update both the session progress markdown files (in `.etch/progress/`) and the plan files (in `.etch/plans/`), keeping them in sync. This replaces the current workflow where agents manually edit progress files, giving etch direct control over status transitions, criterion checking, and progress logging.

The commands follow the existing CLI patterns: factory functions returning `*cli.Command`, task ID resolution via `etchcontext.ResolveTask`, and error handling via `etcherr.*` constructors. The progress file format and plan file surgical updates (via `serializer.UpdateTaskStatus` and `serializer.UpdateCriterion`) already exist — these commands wire them together behind a clean CLI interface.

### Task 1: Add progress command scaffold and `start` subcommand [completed]
**Complexity:** medium
**Files:** cmd/progress.go, cmd/root.go
**Depends on:** (none)

Create `cmd/progress.go` with the top-level `etch progress` command and the `start` subcommand. The `progress` command itself is a container with no action — it just groups subcommands. Register it in `cmd/root.go`.

The `start` subcommand:
- Takes `[plan-name] <task-id>` arguments, resolved the same way as `etch context` (reuse `findProjectRoot`, `etchcontext.DiscoverPlans`, `etchcontext.ResolveTask`).
- Updates the task status to `in_progress` in the plan file using `serializer.UpdateTaskStatus`.
- Finds or creates a session progress file for the task. Look for the latest existing session file first (use `progress.ReadAll` to find it); if none exists, create one with `progress.WriteSession`.
- Updates the `**Status:**` line in the progress file to `in_progress`.
- Prints confirmation: `"Task <id> started (session <NNN>)"`.

**Acceptance Criteria:**
- [x] `etch progress start <task-id>` sets plan file status to `in_progress`
- [x] Creates a new session file if none exists for the task
- [x] Reuses the latest session file if one already exists
- [x] Updates the progress file status to `in_progress`
- [x] Prints confirmation with task ID and session number
- [x] Fails gracefully with hint if task not found

### Task 2: Add `update` and `done` subcommands [completed]
**Complexity:** medium
**Files:** cmd/progress.go, internal/progress/progress.go
**Depends on:** Task 1

Add the `update` and `done` subcommands to the progress command.

**`update` subcommand:**
- Takes `[plan-name] <task-id> --message "text"` — the `--message` flag is required.
- Finds the latest session progress file for the task (error if none exists — tell user to run `etch progress start` first).
- Appends a timestamped entry to the `## Changes Made` section of the progress file, formatted as `- [HH:MM] <message>`.
- Does NOT change task status.
- Prints: `"Logged update for Task <id>"`.

Add a new exported function `progress.FindLatestSessionPath(rootDir, planSlug, taskID) (string, int, error)` that returns the path and session number of the latest session file, or an error if none exists. This will be reused by other subcommands.

Add a new exported function `progress.AppendToSection(path, sectionName, content string) error` that appends a line to a named `## Section` in a progress file. This is a surgical line-based edit similar to the serializer pattern.

**`done` subcommand:**
- Takes `[plan-name] <task-id>`.
- Updates plan file status to `completed` via `serializer.UpdateTaskStatus`.
- Updates progress file `**Status:**` to `completed`.
- Checks unchecked acceptance criteria in the plan file. If any exist, prints a warning listing them but still completes.
- Prints: `"Task <id> completed"` (with warning if unchecked criteria).

**Acceptance Criteria:**
- [x] `etch progress update <task-id> --message "msg"` appends timestamped entry to session file
- [x] `update` errors if no session file exists with helpful hint
- [x] `etch progress done <task-id>` sets plan and progress status to `completed`
- [x] `done` warns about unchecked acceptance criteria but still completes
- [x] `progress.FindLatestSessionPath` is exported and reusable
- [x] `progress.AppendToSection` is exported and reusable

### Task 3: Add `criteria` subcommand [completed]
**Complexity:** medium
**Files:** cmd/progress.go, internal/progress/progress.go
**Depends on:** Task 1

Add the `criteria` subcommand for checking off acceptance criteria.

- Takes `[plan-name] <task-id> --check "criterion text"`. The `--check` flag is a `cli.StringSlice` to support multiple checks in one call.
- For each `--check` value:
  1. First try exact match against the task's criteria in the plan file using `serializer.UpdateCriterion`.
  2. If exact match fails, try substring match: find criteria where the check text is a substring of the criterion description (case-insensitive). Use `serializer.UpdateCriterion` with the full matched text.
  3. Also update the matching criterion in the progress file's `## Acceptance Criteria Updates` section.
- Report results: list which criteria were matched and checked, and which `--check` values didn't match anything.
- Print summary: `"Checked N/M criteria for Task <id>"`.

**Acceptance Criteria:**
- [x] `etch progress criteria <task-id> --check "text"` marks matching criterion in plan file
- [x] Also marks criterion in progress file
- [x] Supports multiple `--check` flags in one call
- [x] Substring matching works when exact match fails (case-insensitive)
- [x] Reports which criteria matched and which didn't
- [x] Prints summary with count

### Task 4: Add `block` and `fail` subcommands [completed]
**Complexity:** small
**Files:** cmd/progress.go
**Depends on:** Task 1

Add the `block` and `fail` subcommands.

**`block` subcommand:**
- Takes `[plan-name] <task-id> --reason "text"` — `--reason` is required.
- Updates plan file status to `blocked` via `serializer.UpdateTaskStatus`.
- Updates progress file `**Status:**` to `blocked`.
- Appends the reason to the `## Blockers` section in the progress file using `progress.AppendToSection`.
- Prints: `"Task <id> blocked: <reason>"`.

**`fail` subcommand:**
- Takes `[plan-name] <task-id> --reason "text"` — `--reason` is required.
- Updates plan file status to `failed` via `serializer.UpdateTaskStatus`.
- Updates progress file `**Status:**` to `failed`.
- Appends the reason to the `## Blockers` section in the progress file.
- Prints: `"Task <id> failed: <reason>"`.

Both commands should find the latest session file (error with hint if none exists).

**Acceptance Criteria:**
- [x] `etch progress block <task-id> --reason "text"` sets status to blocked in plan and progress
- [x] Block reason is appended to Blockers section
- [x] `etch progress fail <task-id> --reason "text"` sets status to failed in plan and progress
- [x] Fail reason is appended to Blockers section
- [x] Both error with helpful hint if no session file exists

### Task 5: Add progress file update helpers and tests [completed]
**Complexity:** medium
**Files:** internal/progress/progress.go, internal/progress/progress_test.go, cmd/progress_test.go
**Depends on:** Task 2, Task 3, Task 4

Add helper functions to `internal/progress/progress.go` for updating progress file metadata and write tests for all progress commands.

Add `progress.UpdateStatus(path string, newStatus string) error` — surgically replaces the `**Status:**` line in a progress file. Similar to `serializer.UpdateTaskStatus` but for progress files.

Add `progress.UpdateCriterion(path string, criterionText string, met bool) error` — surgically updates a criterion checkbox in a progress file's acceptance criteria section.

Write tests:
- `progress_test.go`: Test `FindLatestSessionPath`, `AppendToSection`, `UpdateStatus`, `UpdateCriterion` with temp dirs and fixture files.
- `cmd/progress_test.go`: Integration-style tests that create temp `.etch/` directories with plan and progress files, run progress subcommands, and verify both plan and progress files are updated correctly.

**Acceptance Criteria:**
- [x] `progress.UpdateStatus` surgically updates the Status line
- [x] `progress.UpdateCriterion` updates criterion checkbox in progress files
- [x] Unit tests for all new progress package functions
- [x] Integration tests for start, update, done, criteria, block, fail subcommands
- [x] All tests pass with `go test ./...`

### Task 6: Update etch skill documentation [completed]
**Complexity:** small
**Files:** .claude/skills/etch-plan/SKILL.md
**Depends on:** Task 1, Task 2, Task 3, Task 4

Update the etch skill file to document the new `etch progress` commands so that AI agents know how to use them. Add a section explaining each subcommand, its arguments, flags, and expected behavior. Include examples showing the typical workflow: start → update → criteria → done.

**Acceptance Criteria:**
- [x] SKILL.md documents all six progress subcommands
- [x] Includes argument and flag descriptions for each command
- [x] Shows example workflow from start to done
- [x] Explains that percentage is computed by `etch status`, not stored
