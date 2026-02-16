# Session: Task 1.6 – Config management
**Plan:** build-etch
**Task:** 1.6
**Session:** 001
**Started:** 2026-02-15
**Status:** completed

## Changes Made
- `internal/config/config.go` — Config struct, Load(), ResolveAPIKey()
- `internal/config/config_test.go` — 8 tests covering all acceptance criteria
- `go.mod` / `go.sum` — upgraded BurntSushi/toml v1.4.0 → v1.6.0 (was indirect dep)

## Acceptance Criteria Updates
- [x] Reads `.etch/config.toml`
- [x] API key resolution chain works correctly
- [x] Missing config file → sensible defaults (don't crash)
- [x] Helpful error when no API key found anywhere
- [x] Tests in `internal/config/config_test.go`: valid config, missing file, env var override, missing API key error
- [x] `go test ./internal/config/...` passes

## Decisions & Notes
- Config uses `[api]` and `[defaults]` sections (canonical format per task spec), not the `[ai]`/`[plan]`/`[context]` sections from init.go's defaultConfig. Init.go update deferred per instructions.
- API key resolution: env var > config file > error. Env var always wins when set.
- `Load()` takes a `projectRoot` string parameter for testability — callers pass "." for normal use.
- `ResolveAPIKey()` is a separate method so callers can defer the check until an API key is actually needed (e.g., not needed for `etch status`).
- Invalid TOML returns an error (tested).

## Blockers
None.

## Next
Task complete. Downstream tasks 2.1 (API client) can use `config.Load()` and `cfg.ResolveAPIKey()`.
