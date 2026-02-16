# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test

```bash
go build ./...                    # Build all packages
go build -o etch .                # Build binary
go test ./...                     # Run all tests
go test ./internal/parser         # Test a single package
go test -run TestParsePlan ./internal/parser  # Run a single test
```

No Makefile — standard Go tooling only. Go 1.24+ required.

## Architecture

Etch is a file-based planning system for AI coding agents. Plans, progress, and context are all markdown files in `.etch/`.

**Data flow:** Plan files (source of truth) → context generation → agent execution → progress files → status reconciliation back into plan files.

**Package dependency layers:**

```
cmd/*  (CLI commands, urfave/cli)
  ↓
internal/generator   (plan generation/refinement, calls API)
internal/context     (context prompt assembly)
internal/status      (progress → plan reconciliation)
internal/tui         (bubbletea TUI for review)
  ↓
internal/api         (Anthropic streaming client)
internal/parser      (markdown → models, line-based state machine)
internal/serializer  (models → markdown, full + surgical modes)
internal/progress    (progress file reader/writer)
internal/config      (TOML config)
  ↓
internal/plan        (data models — zero internal deps)
internal/errors      (typed errors — zero internal deps)
```

`plan/` and `errors/` are leaf packages with no internal dependencies.

## Key Patterns

**Error handling:** Always use `etcherr.*` constructors (`etcherr.WrapAPI`, `etcherr.Config`, `etcherr.WrapIO`, etc.), never `fmt.Errorf` or `errors.New`. Every error gets a `.WithHint()` for user-actionable messaging. Errors bubble up to `main.go` where `etcherr.Render()` formats them. `--verbose` shows the full cause chain.

**Parser is a line-based state machine**, not an AST parser. It uses anchored regexes (`^` prefix) and tracks code fences to avoid false matches. It's intentionally forgiving (unknown statuses default to `pending`).

**Serializer has two modes:**
- `Serialize(plan)` — full markdown generation (for new plans or complete rewrites)
- `UpdateTaskStatus()` / `UpdateCriterion()` — surgical regex-based line replacement that preserves manual formatting and comments

Use surgical updates for status changes. Only use full serialization when replacing an entire plan.

**Single vs multi-feature plans:** When a plan has one feature whose title matches the plan title, the serializer omits `## Feature` headings. Adding a second feature requires full re-serialization.

**Task IDs can have letter suffixes** (e.g., `1.3b`). Use `task.FullID()` for string representation. The regex handles `(\d+)(?:\.(\d+)([a-z])?)?`.

**Progress files use atomic creation** (`O_CREATE|O_EXCL`) with auto-incrementing session numbers to support concurrent agents without conflicts.

**Generator prompts** are Go string constants in `internal/generator/prompts.go`, embedded in the binary by design.

**CLI commands** follow a factory pattern: `func planCmd() *cli.Command` returning a `*cli.Command` with logic in the `Action` closure. Commands return errors, never call `os.Exit`.

**API client** uses an `APIClient` interface so tests can inject fakes. The concrete client streams responses from the Anthropic Messages API with exponential backoff on 429s.

**TUI has 7 modes** (normal, search, comment, etc.) each with its own keybinding handler. Pre-renders plan as styled `lineEntry` arrays for fast scrolling.

## Testing

Tests use table-driven patterns and `os.MkdirTemp()` for filesystem isolation. The parser test suite includes a smoke test against the actual `build-etch.md` plan file. No shared test helper package — helpers are local to each test file.
