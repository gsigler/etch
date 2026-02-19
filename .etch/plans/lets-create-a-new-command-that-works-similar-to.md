# Plan: Add `etch run` Command to Launch Claude with Context [completed]

## Overview

The existing `etch context` command assembles a context prompt for a task and writes it to a file, then tells the user to manually pipe it to Claude (`cat ... | claude`). This plan adds a new top-level `etch run` command that assembles context and immediately launches an interactive Claude Code session with that context — eliminating the manual step.

The existing `etch context` command remains unchanged for users who want the manual workflow (inspect context, pipe it themselves, use a different agent, etc.). The new `etch run` command reuses the same context assembly logic but adds a stdin-piping path through the `internal/claude` package.

---

## Feature 1: Claude Stdin Support

### Task 1.1: Add `RunWithStdin` function to internal/claude [completed]
**Complexity:** small
**Files:** internal/claude/claude.go, internal/claude/claude_test.go

The current `claude.Run()` passes the prompt as a positional CLI argument, which won't work for large context prompts (can exceed OS argument length limits). Add a new `RunWithStdin(prompt, workDir string) error` function that pipes the prompt content to Claude's stdin while still connecting stdout/stderr to the terminal for interactive use.

Implementation:
- Create `RunWithStdin` that uses `cmd.StdinPipe()` to write the prompt, then waits for the process
- Use `claude --prompt -` or pipe directly to stdin (Claude Code reads from stdin when it detects piped input)
- Keep stdout and stderr connected to the terminal for interactivity
- Use the same error handling pattern as the existing `Run` function

**Acceptance Criteria:**
- [x] `RunWithStdin` function exists and compiles
- [x] Prompt content is piped via stdin, not passed as a CLI argument
- [x] stdout/stderr remain connected to the user's terminal
- [x] Error handling follows existing `etcherr.*` patterns

---

## Feature 2: Run Command

### Task 2.1: Create `etch run` CLI command [completed]
**Complexity:** medium
**Files:** cmd/run.go, cmd/root.go
**Depends on:** Task 1.1

Add a new `etch run [plan-name] [task-id]` command that:
1. Reuses the existing argument parsing and plan/task resolution logic from `cmd/context.go` (the `findProjectRoot`, `looksLikeTaskID`, `pickPlan`, and `etchcontext.ResolveTask` flow)
2. Calls `etchcontext.Assemble()` to generate context and progress files
3. Reads the generated context file content
4. Launches Claude Code with the context via `claude.RunWithStdin()`
5. Prints confirmation before launching (task name, session number, context/progress file paths)

The command accepts the same arguments as `etch context`: `[plan-name] [task-id]`, with the same auto-selection and picker behavior.

Implementation:
- Create `cmd/run.go` with `runCmd() *cli.Command`
- Extract shared argument-parsing logic from `cmd/context.go` into a helper (e.g. `resolveContextArgs`) that both `contextCmd` and `runCmd` can call, to avoid duplicating the plan-slug/task-ID resolution
- Register `runCmd()` in `cmd/root.go`
- Read the assembled context file and pass it to `claude.RunWithStdin()`
- Print a brief summary before launching (task ID, session number, file paths)

**Acceptance Criteria:**
- [x] `etch run` command is registered and appears in help output
- [x] Accepts same `[plan-name] [task-id]` arguments as `etch context`
- [x] Assembles context and creates progress file (same as `etch context`)
- [x] Launches interactive Claude Code session with the assembled context
- [x] Prints task and session info before launching
- [x] Argument parsing logic is shared with `etch context` (no copy-paste duplication)
