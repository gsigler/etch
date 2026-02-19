# Plan: Replace API Calls with Claude Code CLI for Plan Generation [completed]

## Overview

Replace the direct Anthropic API integration in `etch plan` and `etch replan` with interactive Claude Code CLI (`claude`) invocations. Instead of managing API keys and streaming SSE responses, etch will exec the `claude` command with a prompt that instructs it to use the `/etch-plan` skill. This lets the user iterate interactively with Claude Code during plan creation, avoids separate API billing, and leverages Claude Code's built-in tools for project exploration.

The `etch review` command will continue using the direct API for now. The `internal/api` package stays, but plan and replan commands no longer depend on it or require an API key.

---

## Feature 1: Update `etch plan` to Use Claude Code CLI

### Task 1.1: Add a Claude Code CLI executor utility [completed]
**Complexity:** medium
**Files:** internal/claude/claude.go, internal/claude/claude_test.go

Create a new `internal/claude` package that wraps `os/exec` to launch the `claude` CLI interactively. The key function should:

- Accept a prompt string and working directory
- Verify `claude` is on PATH (return a clear error with hint if not found)
- Exec `claude` with the prompt passed via stdin or `-p` flag for the initial message, but in interactive mode (not `--print`)
- Connect stdin/stdout/stderr to the terminal so the user can interact
- Return the exit code after the session ends

Use `claude` with the `--initial-prompt` flag (or equivalent) so it starts with the prompt but remains interactive. Research the correct CLI flags for this.

**Acceptance Criteria:**
- [x] `claude` package provides a `Run(prompt, workDir string) error` function
- [x] Returns a descriptive `etcherr` if `claude` is not found on PATH
- [x] User can interact with Claude Code during the session
- [x] Session runs in the project root directory

### Task 1.2: Rewrite `cmd/plan.go` to use Claude Code CLI [completed]
**Complexity:** medium
**Files:** cmd/plan.go
**Depends on:** Task 1.1

Rewrite the plan command's Action to:

1. Parse description and find project root (unchanged)
2. Load config for slug generation only (no API key needed)
3. Handle slug collisions (unchanged logic)
4. Build a prompt string that tells Claude to create a plan: include the description and instruct it to use `/etch-plan` with the description as the argument. Include the target slug so Claude writes to `.etch/plans/<slug>.md`
5. Call `claude.Run(prompt, rootDir)` to launch the interactive session
6. After the session ends, check if `.etch/plans/<slug>.md` exists
7. Validate the plan parses correctly using `parser.ParseFile()`
8. Print a summary

Remove imports of `internal/api` and `internal/generator` (for the Generate function). The streaming callback, API client creation, and manual file writing are all removed.

**Acceptance Criteria:**
- [x] `etch plan "description"` launches an interactive Claude Code session
- [x] No API key is required
- [x] After the session, the plan file is validated
- [x] Slug collision handling still works
- [x] If the plan file wasn't created, a helpful error is shown

### Task 1.3: Update config to make API key optional [completed]
**Complexity:** small
**Files:** internal/config/config.go
**Depends on:** Task 1.2

The `ResolveAPIKey()` method currently returns an error if no API key is found. Plan and replan no longer need it. Either:
- Make `ResolveAPIKey()` return `("", nil)` when no key is set (callers that need it check for empty string)
- Or don't call `ResolveAPIKey()` from plan/replan at all (simpler, preferred approach)

Since plan.go and replan.go will no longer call `ResolveAPIKey()`, this task is mainly about ensuring the config loading doesn't fail when no API key is configured. Verify that `config.Load()` works without an API key set.

**Acceptance Criteria:**
- [x] `etch plan` works without `ANTHROPIC_API_KEY` set or configured
- [x] `etch review` still requires and validates the API key
- [x] Config loading doesn't error when API key is absent

---

## Feature 2: Update `etch replan` to Use Claude Code CLI

### Task 2.1: Rewrite `cmd/replan.go` to use Claude Code CLI [completed]
**Complexity:** medium
**Files:** cmd/replan.go
**Depends on:** Task 1.1

Rewrite the replan command's Action to:

1. Parse args, discover plans, resolve target (unchanged logic)
2. Read the current plan file content
3. Build a prompt that includes:
   - The current plan markdown
   - What target is being replanned (task ID or feature number + title)
   - Instructions to modify the plan file in place, preserving completed tasks
   - Reference to the etch plan format rules
4. Call `claude.Run(prompt, rootDir)` to launch the interactive session
5. After the session ends, validate the plan file still parses correctly
6. Print success message

Remove the diff/confirm flow, API client, and streaming. The user controls changes interactively. The backup creation can optionally be kept as a safety measure before launching claude.

**Acceptance Criteria:**
- [x] `etch replan 1.2` launches an interactive Claude Code session with the current plan context
- [x] No API key is required
- [x] The prompt includes the current plan content and target information
- [x] After the session, the plan is validated
- [x] A backup of the original plan is created before the session starts

---

## Feature 3: Cleanup and Testing

### Task 3.1: Remove unused generator functions and update tests [completed]
**Complexity:** medium
**Files:** internal/generator/generator.go, internal/generator/generator_test.go, internal/generator/prompts.go
**Depends on:** Task 1.2, Task 2.1

The `generator.Generate()` function and its context-gathering helpers are no longer called by plan. The `generator.Replan()` function is no longer called by replan. Clean up:

- Remove `Generate()`, `gatherContext()`, and related helpers that are now unused
- Remove `Replan()` and `ApplyReplan()` functions
- Remove the system/user prompt templates for plan generation and replan (keep refine prompts for review)
- Keep `Slugify()`, `SlugExists()`, `ResolveSlug()`, `WritePlan()`, and `ResolveTarget()` as they're still used
- Update or remove tests that tested the removed functions

**Acceptance Criteria:**
- [x] No dead code remains in the generator package
- [x] `Slugify`, `SlugExists`, `ResolveSlug`, `WritePlan`, `ResolveTarget` are preserved
- [x] Remaining tests pass
- [x] `go build ./...` succeeds with no unused import errors

### Task 3.2: Verify end-to-end flow and update error handling [completed]
**Complexity:** small
**Files:** cmd/plan.go, cmd/replan.go, internal/claude/claude.go
**Depends on:** Task 3.1

Test the full flow manually:
- `etch plan "test feature"` — should launch claude, create plan, validate
- `etch replan 1.1` — should launch claude with plan context, validate after
- Missing `claude` binary — should show helpful error
- No `.etch/` directory — existing error handling should still work

Ensure all error paths use `etcherr.*` constructors with `.WithHint()`.

**Acceptance Criteria:**
- [x] `go test ./...` passes
- [x] `go build ./...` succeeds
- [x] Error messages follow etch conventions (etcherr + hints)
