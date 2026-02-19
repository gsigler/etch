---
name: etch
description: Reference for the etch CLI — a file-based planning system for AI coding agents. Use when you need to check plan status, report progress on tasks, run tasks, or manage plans.
---

# Etch CLI Reference

Etch is a file-based planning system for AI coding agents. Plans, progress, and context are all markdown files in `.etch/`.

## Setup

```bash
etch init                        # Initialize etch in the current project
etch skill install               # Install the etch-plan skill for Claude Code
```

## Plan lifecycle

### `etch plan <description> [--name <slug>]`

Generate an implementation plan from a description. Uses the Anthropic API to create a structured plan with tasks, acceptance criteria, and file scopes. Saves to `.etch/plans/<slug>.md`.

```bash
etch plan "Add user authentication"
etch plan "Add auth" --name auth-system
```

### `etch list`

List all available plans.

### `etch status [plan-slug] [--all] [--json]`

Show progress for plans. Displays task statuses and completion percentage (computed from checked vs total acceptance criteria). Without a slug, shows all in-progress plans.

- `--all` — include fully pending and fully completed plans
- `--json` — output in JSON format

```bash
etch status                      # all active plans
etch status auth-system          # specific plan
etch status --json               # machine-readable output
```

### `etch review <plan-name>`

Open the interactive TUI to review and annotate a plan.

### `etch open <plan-name>`

Open a plan file in your editor.

### `etch delete <plan-name> [--yes]`

Delete a plan and its progress files. Prompts for confirmation unless `--yes` is passed.

### `etch replan [-p <plan>] [--target <target>]`

Regenerate part of a plan incorporating progress and feedback.

```bash
etch replan                              # replan the only plan (or pick from list)
etch replan -p my-plan                   # replan entire plan by name
etch replan --target 1.2                 # replan Task 1.2
etch replan --target feature:2           # replan Feature 2
etch replan --target "Login System"      # replan feature by title
etch replan -p my-plan --target 1.2      # replan task in a specific plan
```

## Running tasks

### `etch context [-p <plan>] [-t <task-id>]`

Generate the context prompt for a task. Outputs the assembled context that would be passed to an AI agent, including plan state, task details, and session history.

### `etch run [-p <plan>] [-t <task-id>]`

Launch Claude Code with assembled context for a task. Automatically resolves the next pending task if no task ID is given.

## Reporting progress

Use `etch progress` subcommands to report work on tasks. These commands update both the plan file (task status, criteria) and session progress files in `.etch/progress/`.

All subcommands use flags: `--task, -t` (required) for the task ID and `--plan, -p` (required) for the plan slug.

### `etch progress start -p <plan> -t <task-id>`

Mark a task as in-progress. Creates a session progress file if one doesn't exist, or reuses the latest session. Always run this before beginning work on a task.

```bash
etch progress start -p my-plan -t 1.3
```

### `etch progress update -p <plan> -t <task-id> -m "text"`

Log a progress note. Appends a timestamped entry to the "Changes Made" section of the session progress file.

- `--message, -m` (required) — the update message

```bash
etch progress update -p my-plan -t 1.3 -m "Added validation to signup form"
```

### `etch progress criteria -p <plan> -t <task-id> --check "text"`

Check off acceptance criteria in the plan file. Supports exact or case-insensitive substring matching. Can be specified multiple times.

- `--check` (required, repeatable) — criterion text to match

```bash
etch progress criteria -p my-plan -t 1.3 --check "Unit tests pass"
etch progress criteria -p my-plan -t 1.3 --check "validation" --check "error messages"
```

### `etch progress done -p <plan> -t <task-id>`

Mark a task as completed. Updates plan and session status. Warns if any acceptance criteria are still unchecked.

```bash
etch progress done -p my-plan -t 1.3
```

### `etch progress block -p <plan> -t <task-id> --reason "text"`

Mark a task as blocked. Appends the reason to the "Blockers" section of the progress file.

- `--reason` (required) — why the task is blocked

```bash
etch progress block -p my-plan -t 1.3 --reason "Waiting on API schema from backend team"
```

### `etch progress fail -p <plan> -t <task-id> --reason "text"`

Mark a task as failed. Appends the reason to the "Blockers" section of the progress file.

- `--reason` (required) — why the task failed

```bash
etch progress fail -p my-plan -t 1.3 --reason "Approach is not viable, needs redesign"
```

### Typical progress workflow

```bash
# 1. Start working on a task
etch progress start -p my-plan -t 1.3

# 2. Log updates as you work
etch progress update -p my-plan -t 1.3 -m "Implemented input validation"
etch progress update -p my-plan -t 1.3 -m "Added error message display"

# 3. Check off acceptance criteria as you meet them
etch progress criteria -p my-plan -t 1.3 --check "Validates email format"
etch progress criteria -p my-plan -t 1.3 --check "Shows inline errors"

# 4. Mark the task as done
etch progress done -p my-plan -t 1.3
```

## Permissions

When launched via `etch run`, Claude Code sessions automatically approve all `etch` CLI commands (e.g., `etch progress`, `etch context`, `etch status`). You do not need to ask for permission to run etch commands — they are pre-allowed via `--allowedTools`.

## Key concepts

- **Completion percentage** is computed by `etch status` from checked vs total acceptance criteria. It is not stored anywhere — you don't need to track or report it.
- **Session progress files** live in `.etch/progress/` and are named `<plan-slug>--task-<id>--<session>.md`. They are created by `etch progress start`.
- **Plan files** in `.etch/plans/` are the source of truth for task status and acceptance criteria.
- Always run `start` before other progress subcommands — they expect a session file to exist.
- `criteria --check` uses substring matching, so a unique fragment of the criterion text is enough.
