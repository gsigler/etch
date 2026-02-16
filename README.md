# Etch

Etch is a CLI tool that helps developers create, review, and execute AI-generated implementation plans. It solves the problem of context loss across multiple AI coding sessions by structuring work into plans, features, and agent-sized tasks — all stored as plain markdown.

## Why Etch?

When a feature takes multiple AI coding sessions to complete, you lose context between sessions, manually re-explain state, and have no structured way to track progress. Etch fixes this:

- **Plans are markdown** — readable, diffable, version-controlled
- **Progress is separate** — per-session files let multiple agents work simultaneously without conflicts
- **Context is generated** — pipe a task's full context (plan state, dependencies, prior session notes) directly into your AI agent
- **Claude Code integration** — generate plans and launch agents directly through Claude Code
- **No database** — everything lives in `.etch/` as files

## Installation

### Prerequisites

Etch requires [Claude Code](https://docs.anthropic.com/en/docs/claude-code) to be installed and authenticated. Claude Code handles all AI interactions — plan generation, replanning, and task execution.

### From GitHub Releases

Download the latest binary for your platform from the [Releases](https://github.com/gsigler/etch/releases) page.

Available for: Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64, arm64).

### From Source

```bash
go install github.com/gsigler/etch@latest
```

Or clone and build:

```bash
git clone https://github.com/gsigler/etch.git
cd etch
go build -o etch .
```

## Getting Started

### 1. Initialize

```bash
cd your-project
etch init
```

This creates the `.etch/` directory structure and a config file.

### 2. Install the Claude Code skill

```bash
etch skill install
```

This installs the `etch-plan` skill into your project's `.claude/skills/` directory, enabling Claude Code to generate and modify plans in the correct format.

### 3. Generate a plan

```bash
etch plan "Add user authentication with JWT tokens"
```

Etch launches Claude Code with the `etch-plan` skill to generate a structured plan in `.etch/plans/`.

### 4. Review and refine

```bash
etch review auth-system
```

Opens an interactive TUI where you can scroll through the plan, leave comments, and trigger AI refinement.

### 5. Run a task with Claude Code

```bash
etch run 1.1
```

This assembles the context and launches Claude Code with the full plan state, task details, completed prerequisites, and prior session notes — everything the agent needs to pick up where work left off.

You can also generate the context file separately and pipe it manually:

```bash
etch context 1.1
cat .etch/context/auth-system--task-1.1--001.md | claude
```

### 6. Check progress

```bash
etch status
```

Reads progress files written by agents, reconciles them with the plan, and shows a summary.

## Commands

### `etch init`

Initialize etch in the current project. Creates `.etch/` directory structure, config file, and updates `.gitignore`.

### `etch plan <description>`

Generate an implementation plan by launching Claude Code with the `etch-plan` skill. Claude Code creates the plan interactively and saves it as markdown.

```bash
etch plan "Add rate limiting to the API endpoints"
```

### `etch review <plan-name>`

Open the interactive TUI to review a plan. Browse tasks, leave comments, and refine with AI.

**Key bindings:**

| Key | Action |
|-----|--------|
| `j`/`k` or arrows | Scroll |
| `d`/`u` | Half-page scroll |
| `gg`/`G` | Jump to top/bottom |
| `n`/`p` | Next/previous task |
| `f`/`F` | Next/previous feature |
| `/` | Search |
| `c` | Add comment |
| `C` | Add multi-line comment (opens `$EDITOR`) |
| `x` | Delete comment |
| `a` | Apply AI refinement |
| `q` | Quit |

### `etch run [plan-name] [task-id]`

Assemble context and launch Claude Code to execute a task. If no task is specified, auto-selects the next pending task.

```bash
etch run 1.2                  # Run task 1.2
etch run auth-system 1.2      # Specify plan and task
etch run                      # Auto-select next task
```

### `etch replan [plan-name] <target>`

Regenerate part of a plan by launching Claude Code, incorporating progress and feedback.

```bash
etch replan 1.2              # Replan task 1.2
etch replan feature:2        # Replan all of feature 2
etch replan "Feature Title"  # Replan by title
```

Creates a backup before making changes.

### `etch context [plan-name] [task-id]`

Generate a context prompt file for an AI agent. If no task is specified, auto-selects the next pending task.

```bash
etch context 1.2
etch context auth-system 1.2
etch context                  # Auto-select next task
```

### `etch status [plan-slug]`

Show progress across all plans, or detailed status for a specific plan. Updates plan file status based on progress files.

```bash
etch status
etch status auth-system
etch status --json
```

### `etch list`

List all available plans with task counts and completion percentages.

### `etch open <plan-name>`

Open a plan file in your editor (`$EDITOR`, defaults to `vi`).

### `etch delete <plan-name>`

Delete a plan and all its associated progress and context files.

```bash
etch delete auth-system
etch delete auth-system -y    # Skip confirmation
```

### `etch skill install`

Install or update the `etch-plan` Claude Code skill in the current project. This writes the skill definition to `.claude/skills/etch-plan/SKILL.md`.

```bash
etch skill install
```

### Global Flags

| Flag | Description |
|------|-------------|
| `--verbose` | Show full error chains for debugging |
| `--help, -h` | Show help |
| `--version, -v` | Show version |

## Configuration

Etch reads configuration from `.etch/config.toml`:

```toml
[defaults]
complexity_guide = "small = single focused session, medium = may need iteration, large = multiple sessions likely"
```

### Prerequisites

- **Claude Code** must be installed and authenticated. Etch delegates plan generation, replanning, and task execution to Claude Code.
- Run `etch skill install` to install the `etch-plan` skill that teaches Claude Code the etch plan format.

## Project Structure

```
your-project/
└── .etch/
    ├── config.toml        # Project configuration
    ├── plans/             # Plan markdown files (source of truth)
    │   └── auth-system.md
    ├── progress/          # Per-session execution logs
    │   ├── auth-system--task-1.1--001.md
    │   └── auth-system--task-1.1--002.md
    ├── context/           # Generated prompt files (gitignored)
    │   └── auth-system--task-1.1--001.md
    └── backups/           # Auto-backups before AI refinement (gitignored)
```

**What gets tracked in git:**
- `plans/` — always (it's your spec)
- `progress/` — your choice at `etch init`
- `context/` — never (regenerable)
- `backups/` — never
- `config.toml` — never (project-specific settings)

## Plan Format

Plans are structured markdown with a specific format that etch parses:

```markdown
# Plan: Add Authentication

## Overview
High-level description of the feature.

---

## Feature 1: User Model

### Task 1.1: Database schema [pending]
**Complexity:** small
**Files:** db/migrations/001_users.sql, db/models/user.go
**Depends on:** none

Create the users table migration and Go model.

**Acceptance Criteria:**
- [ ] Migration creates users table
- [ ] User model struct matches schema
```

**Task statuses:** `pending`, `in_progress`, `completed`, `blocked`, `failed`

## Workflow

The typical etch workflow looks like this:

```
etch plan → etch review → etch run → etch status → repeat
```

1. **Plan** — Generate a structured implementation plan via Claude Code
2. **Review** — Read through, leave comments, refine with AI
3. **Run** — Launch Claude Code with assembled context to execute the next task
4. **Status** — Reconcile progress, update the plan, see what's next
5. **Repeat** — Run the next task

## Contributing

### Prerequisites

- Go 1.24+
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed and authenticated

### Development

```bash
git clone https://github.com/gsigler/etch.git
cd etch
go build ./...
go test ./...
```

### Project Layout

```
cmd/           CLI command definitions (urfave/cli)
internal/
  api/         Anthropic API client
  claude/      Claude Code subprocess runner
  config/      TOML config management
  context/     Context prompt assembly
  errors/      Typed errors with hints
  generator/   Slug generation, target resolution, backups
  parser/      Plan markdown parser
  plan/        Data models
  progress/    Progress file reader/writer
  serializer/  Plan markdown serializer
  skill/       Embedded etch-plan skill content
  status/      Status reconciliation
  tui/         Bubbletea TUI for review
```

### Submitting Changes

1. Fork the repository
2. Create a feature branch (`git checkout -b my-feature`)
3. Make your changes
4. Run tests (`go test ./...`)
5. Commit and push
6. Open a pull request

## License

MIT
