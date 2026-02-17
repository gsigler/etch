# Plan: Live TUI Dashboard for Etch Go

## Overview

Replace the sequential terminal output in `etch go` with a live bubbletea TUI dashboard that shows real-time execution status. The dashboard displays all tasks in a list with status icons, criteria-based progress percentages, worker assignments for active tasks, and dependency info for waiting tasks. A bottom bar shows completed/total count, active worker count, and keybindings. The dashboard polls progress files every 2 seconds to refresh, highlights task state transitions, and pauses prominently on failure/block to collect human input before continuing or aborting.

This builds on the existing TUI patterns in `internal/tui/` (lipgloss styles, bubbletea model/update/view, keybindings, line rendering) and integrates with the runner from the `etch-go-command` and `parallel-worktree-execution` plans. The dashboard replaces the runner's stdout printing with a bubbletea program that receives task state updates via messages.

---

## Feature 1: Dashboard Model and View

### Task 1.1: Create the dashboard bubbletea model [pending]
**Complexity:** large
**Files:** internal/tui/dashboard/model.go, internal/tui/dashboard/keys.go

Create a new `internal/tui/dashboard` package (separate from the existing review TUI) with a bubbletea `Model` struct. The model should hold:

1. `plan *models.Plan` — the plan being executed
2. `tasks []DashTask` — enriched task list with runtime state:
   - `Task *models.Task` — reference to the plan task
   - `Status models.Status` — current status (mirrors progress file)
   - `WorkerID int` — which worker is running it (0 = not assigned)
   - `CriteriaMet int` / `CriteriaTotal int` — for progress percentage
   - `LastMessage string` — latest note from progress file (decisions or next)
   - `Highlight time.Time` — when the last state transition occurred (for brief highlight)
   - `DependsOnNames []string` — human-readable dependency list
3. `completedCount`, `totalCount`, `activeWorkers`, `maxWorkers int`
4. `mode dashMode` — one of: `modeRunning`, `modePaused`, `modeComplete`
5. `pauseReason string` — what caused the pause (failure message, blocker, etc.)
6. `pauseTaskID string` — which task caused the pause
7. `width`, `height int` — terminal dimensions
8. `selectedIdx int` — cursor position in task list for scrolling

Create `keys.go` with the dashboard-specific keymap:
- `q` — quit (graceful shutdown)
- `p` — pause all workers
- `s` — select task to view agent output (placeholder for now)
- `j/k` — scroll task list
- `enter` — when paused, continue execution
- `esc` — when paused, abort

The `Init()` method should return a `tea.Batch` with a tick command for the 2-second polling interval.

**Acceptance Criteria:**
- [ ] Dashboard model holds all necessary state for rendering tasks
- [ ] DashTask struct captures runtime state per task (status, worker, criteria progress, highlight)
- [ ] Keymap defines q, p, s, j/k, enter, esc bindings
- [ ] Init returns a tick command for polling

### Task 1.2: Implement the dashboard view renderer [pending]
**Complexity:** medium
**Files:** internal/tui/dashboard/view.go
**Depends on:** Task 1.1

Implement the `View()` method following the existing TUI's lipgloss styling patterns. Layout:

```
┌─────────────────────────────────────────────────┐
│ Etch Go: Plan Title        [████░░░░░░] 40%     │  ← top bar
├─────────────────────────────────────────────────┤
│ ✓ 1.1  Setup database schema          100%      │  ← completed
│ ✓ 1.2  Add migration scripts          100%      │
│ ▶ 1.3  Implement user model      W1   60%      │  ← active, worker 1
│        "Added User struct, working on tests"    │  ← latest message
│ ▶ 2.1  Add auth middleware       W2   20%      │  ← active, worker 2
│        "Setting up JWT validation"              │
│ ○ 2.2  Add login endpoint              0%      │  ← pending
│        waiting on: 1.3, 2.1                     │  ← deps
│ ⊘ 3.1  E2E tests                       0%      │  ← blocked
│        blocked: needs API keys configured       │
├─────────────────────────────────────────────────┤
│ 2/6 done  Workers: 2/3  q:quit p:pause s:view  │  ← bottom bar
└─────────────────────────────────────────────────┘
```

Reuse styles from `internal/tui/view.go`: `taskCompleted`, `taskInProgress`, `taskPending`, `taskFailed`, `taskBlocked`, `barStyle`, `hintStyle`. Import them by moving shared styles to a `internal/tui/styles` sub-package OR by referencing them directly (since they're package-level vars in `internal/tui`). Prefer creating `internal/tui/styles/styles.go` with shared styles to avoid import cycles.

When a task was recently highlighted (within 3 seconds of `Highlight` time), render its line with a brief flash style (e.g., bright background that fades back to normal on next tick).

When in `modePaused`, render a prominent overlay or section:
```
⚠ PAUSED — Task 2.1 failed: "compilation error in user_test.go"
  [Enter] retry/continue  [Esc] abort
```

**Acceptance Criteria:**
- [ ] Task list renders with status icons, progress percentages, and worker assignments
- [ ] Active tasks show latest progress message below the task line
- [ ] Waiting tasks show dependency information
- [ ] Recently-transitioned tasks are briefly highlighted
- [ ] Pause mode shows reason prominently with continue/abort keybindings
- [ ] Top bar shows plan title and overall progress bar
- [ ] Bottom bar shows completed/total, worker count, and keybindings

---

## Feature 2: Progress Polling and State Updates

### Task 2.1: Implement progress file polling [pending]
**Complexity:** medium
**Files:** internal/tui/dashboard/poll.go
**Depends on:** Task 1.1

Create the polling mechanism that reads progress files every 2 seconds and sends bubbletea messages to update the model:

1. Define a `tickMsg` type returned by a `tea.Tick` command every 2 seconds
2. Define a `taskUpdateMsg` struct: `{ TaskID string, Status models.Status, CriteriaMet int, CriteriaTotal int, LastMessage string }`
3. In the `Update` handler for `tickMsg`:
   - Call `progress.ReadAll(rootDir, plan.Slug)` to get all progress files
   - For each task, compute `effectiveStatus` and criteria completion counts
   - Compare with current `DashTask` state — if anything changed, emit a `taskUpdateMsg`
   - If status changed, set `Highlight` to `time.Now()`
   - Schedule the next tick
4. Use a `tea.Cmd` that wraps the file reading in a goroutine (bubbletea pattern) to avoid blocking the UI

The polling should be resilient: if a progress file is being written (partial read), skip it and retry next tick.

**Acceptance Criteria:**
- [ ] Progress files are read every 2 seconds via tea.Tick
- [ ] Task state transitions are detected and trigger visual updates
- [ ] Criteria completion counts are computed from progress files
- [ ] Highlight timestamps are set on state transitions
- [ ] Partial/corrupt progress files don't crash the poll loop

### Task 2.2: Connect runner events to dashboard messages [pending]
**Complexity:** large
**Files:** internal/tui/dashboard/model.go, internal/runner/runner.go
**Depends on:** Task 2.1

Bridge the runner's execution events to the dashboard's bubbletea message loop. The runner needs to send events instead of printing to stdout:

1. Define an `EventSink` interface in the runner package:
   ```go
   type EventSink interface {
       TaskStarted(taskID string, workerID int)
       TaskCompleted(taskID string)
       TaskFailed(taskID string, reason string)
       TaskBlocked(taskID string, reason string)
       RunComplete()
   }
   ```
2. The dashboard model implements `EventSink` by sending corresponding `tea.Msg` types through a channel that the bubbletea program reads
3. In the `Update` handler, process runner events:
   - `TaskStarted` → set task to in_progress, assign worker ID
   - `TaskCompleted` → set task to completed, increment counter, clear worker
   - `TaskFailed` → enter `modePaused`, show failure reason
   - `TaskBlocked` → enter `modePaused`, show blocker
   - `RunComplete` → enter `modeComplete`, show final summary
4. When in `modePaused` and user presses Enter, send a resume signal back to the runner (via a channel). When user presses Esc, send an abort signal
5. The runner should run in a separate goroutine, communicating with the TUI via channels. Use `tea.Program.Send()` to inject messages from the runner goroutine into the bubbletea event loop

**Acceptance Criteria:**
- [ ] Runner events are delivered to the dashboard as bubbletea messages
- [ ] Task state updates in real-time as the runner progresses
- [ ] Pause/resume flow works: failure pauses, Enter resumes, Esc aborts
- [ ] Runner runs in a goroutine without blocking the TUI
- [ ] Clean shutdown: quitting the TUI signals the runner to stop

---

## Feature 3: CLI Integration

### Task 3.1: Wire dashboard into etch go command [pending]
**Complexity:** medium
**Files:** cmd/go.go
**Depends on:** Task 2.2

Replace the current stdout-based execution in `cmd/go.go` with the dashboard TUI:

1. After resolving the plan and building the task list, create a `dashboard.Model` with the plan and initial task states
2. Create a `tea.Program` with `tea.WithAltScreen()` for full-screen mode
3. Start the runner in a goroutine, passing the `tea.Program` reference for sending events
4. Run `program.Run()` (blocking) — returns when the user quits or execution completes
5. After the TUI exits, print a final text summary to stdout (since alt-screen content disappears)
6. Add a `--no-tui` flag that falls back to the original sequential output for CI/headless environments. When stdout is not a TTY (`!term.IsTerminal()`), default to no-tui mode

**Acceptance Criteria:**
- [ ] `etch go` launches the TUI dashboard by default
- [ ] `--no-tui` flag falls back to text output
- [ ] Non-TTY environments automatically use text output
- [ ] Final summary prints after TUI exits
- [ ] Clean exit on q, ctrl+c, or run completion

---

## Feature 4: Testing

### Task 4.1: Test dashboard model state transitions [pending]
**Complexity:** medium
**Files:** internal/tui/dashboard/model_test.go
**Depends on:** Task 2.2

Test the dashboard model's Update logic with synthetic messages:

1. Test `tickMsg` processing: create a model with known progress files, send a tick, verify task states update correctly
2. Test state transition highlighting: send a `taskUpdateMsg` that changes status, verify `Highlight` is set
3. Test pause flow: send a `TaskFailed` event, verify mode transitions to `modePaused` with correct reason. Send Enter keypress, verify mode returns to `modeRunning`
4. Test abort flow: from paused, send Esc, verify mode transitions to abort
5. Test completion: send `RunComplete`, verify `modeComplete`
6. Test criteria progress: verify percentages compute correctly from criteria counts

Use `os.MkdirTemp` for progress file fixtures. Test the model directly by calling `Update()` with synthetic messages — no need to run a full `tea.Program`.

**Acceptance Criteria:**
- [ ] Tests cover tick-based progress polling state updates
- [ ] Tests cover pause/resume/abort flows
- [ ] Tests cover task state transition highlighting
- [ ] Tests cover criteria progress calculation
- [ ] Tests pass with `go test ./internal/tui/dashboard`
