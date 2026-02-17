# Plan: Etch Skill Install & Update Commands

## Overview

Add `etch skill install` and `etch skill update` CLI commands that manage a Claude Code skill file (`.claude/skills/etch-plan/SKILL.md`) in the current project. The skill file teaches AI agents how to create etch plans: the correct markdown format, task structure, statuses, acceptance criteria, and plan conventions.

The skill content is embedded in the etch binary using Go's `embed` package, so it works offline and stays versioned with the etch release. `install` creates the file (refusing if it already exists), and `update` overwrites it with the latest embedded version.

---

## Feature 1: Embedded Skill Content

### Task 1.1: Embed the existing skill file in the binary [completed]
**Complexity:** small
**Files:** internal/skill/etch-plan.md, internal/skill/embed.go

Copy the current `.claude/skills/etch-plan/SKILL.md` into `internal/skill/etch-plan.md` as the embedded source of truth. This file is a standalone markdown skill (with frontmatter, step-by-step workflow, format examples, and `$ARGUMENTS` placeholder) that will be copied verbatim into target projects by the install command.

Create `internal/skill/embed.go` with a `//go:embed etch-plan.md` directive exposing the content as a `string` variable. The skill content lives as a plain `.md` file in its own package so it's easy to edit and iterate on without touching Go code.

**Acceptance Criteria:**
- [x] `internal/skill/etch-plan.md` exists and matches the current `.claude/skills/etch-plan/SKILL.md` content
- [x] `internal/skill/embed.go` exposes `Content` as a `string` variable using `//go:embed`
- [x] The skill file is a plain markdown file in its own package, easy to edit independently

---

## Feature 2: CLI Commands

### Task 2.1: Add `etch skill install` and `etch skill update` commands [completed]
**Complexity:** medium
**Files:** cmd/skill.go, cmd/skill_test.go, cmd/root.go
**Depends on:** Task 1.1

Create the `skill` command group with `install` and `update` subcommands, register `skillCmd()` in `cmd/root.go`, and add tests.

**Install** subcommand:

1. Uses the current working directory (does NOT require `etch init` / `.etch/` directory)
2. Creates `.claude/skills/etch-plan/` directory if it doesn't exist
3. Writes the embedded skill content to `.claude/skills/etch-plan/SKILL.md`
4. If the file already exists, prints a message suggesting `etch skill update` instead and exits without overwriting
5. Prints a success message with the path

**Update** subcommand:

1. Overwrites `.claude/skills/etch-plan/SKILL.md` with the latest embedded content (or creates it if missing)
2. Prints a success message

Follow the existing CLI patterns: factory function returning `*cli.Command`, errors via `etcherr.*` constructors with `.WithHint()`.

Tests use `os.MkdirTemp()` for filesystem isolation and cover:
- Install creates the file with expected content
- Install refuses to overwrite existing file
- Update overwrites existing file
- Both commands create directories as needed

**Acceptance Criteria:**
- [x] `etch skill install` creates `.claude/skills/etch-plan/SKILL.md` with the embedded content
- [x] Running install when the file already exists prints a helpful message and does not overwrite
- [x] `etch skill update` overwrites the existing skill file with the latest embedded content
- [x] Update works even if the file doesn't exist yet (creates it)
- [x] Neither command requires `.etch/` to exist — works in any directory
- [x] `skillCmd()` is registered in `cmd/root.go`
- [x] Uses `etcherr` for all error handling with user-actionable hints
- [x] `go test ./cmd/...` passes
- [x] `go build ./...` succeeds
