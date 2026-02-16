# Session: Task 5.1 – Error handling
**Plan:** build-etch
**Task:** 5.1
**Session:** 001
**Started:** 2026-02-15 18:50
**Status:** completed

## Changes Made
- Created `internal/errors/errors.go` — custom Error type with Category (config, api, parse, project, usage, io), Message, Hint, Cause fields. Convenience constructors and `Format()`/`Render()` for colored terminal output using Lipgloss.
- Updated `main.go` — panic recovery via `defer recover()`, colored error output via `etcherr.Render()`, verbose flag forwarded from `cmd.Execute()`.
- Updated `cmd/root.go` — `Execute()` returns `(bool, error)` with verbose flag value, `Before` hook captures `--verbose`.
- Updated all cmd/*.go files — replaced `fmt.Errorf` with typed `etcherr.*` errors with actionable hints.
- Updated `internal/api/client.go` — all errors use `etcherr.WrapAPI`/`etcherr.API` with hints.
- Updated `internal/config/config.go` — config errors use `etcherr.WrapConfig`/`etcherr.Config`.
- Updated `internal/generator/generator.go`, `refine.go`, `replan.go` — all errors typed.
- Updated `internal/context/context.go` — project/IO errors typed with hints.
- Updated `internal/status/status.go` — IO/parse errors typed.
- Updated tests: `cmd/init_test.go`, `internal/api/client_test.go`, `internal/config/config_test.go` to match new error types.

## Acceptance Criteria Updates
- [x] Error types defined for all categories
- [x] Actionable messages on every error
- [x] Colored output
- [x] `--verbose` for debug info
- [x] No raw panics

## Decisions & Notes
- Error type: single `errors.Error` struct with a `Category` field rather than separate types per category. Simpler, composable via `WithHint()`.
- Lipgloss colors: red bold for "error", yellow for category label, dim for hints and causes.
- `--verbose` flag shows the `Cause` chain; non-verbose shows only message + hint.
- Panic recovery in main.go catches unexpected panics and renders them as errors with exit code 2.
- The old `api.APIError` type is kept for backward compat but the client no longer returns it — it returns `etcherr.Error` instead.
- Lower-level packages (parser, serializer, progress, tui) still use `fmt.Errorf` — their errors bubble up through higher-level packages that wrap them in typed errors.

## Blockers
None.

## Next
Task 5.2: First-run polish.
