# Plan: Etch v2 CLI Improvements [completed]

## Overview

This plan addresses a batch of UX, correctness, and quality-of-life issues discovered during real-world usage of the etch CLI. The changes span the CLI commands, plan generation skill prompt, status display, Claude Code integration, and plan lifecycle management.

Key themes: (1) reduce friction in the setup and planning flow, (2) improve plan quality by tuning generation prompts, (3) fix status display bugs, (4) add plan auto-completion detection, (5) make slugs first-class identifiers throughout, and (6) hide finished work by default.

---

## Feature 1: Setup & Installation Improvements

### Task 1.1: Auto-install skills during etch init [completed]
**Complexity:** small
**Files:** cmd/init.go

After creating directories and writing `config.toml`, call `runSkillInstall()` (from `cmd/skill.go`) to automatically install the etch skills into the project's `.claude/skills/` directory. This removes the need for a separate `etch skill install` step. Add a message like `✓ Skills installed` to the init output. If `.claude` doesn't exist yet, create it (the existing `resolveSkillDir` logic handles this but prompts — for init, just default to creating it).

**Acceptance Criteria:**
- [x] `etch init` creates `.claude/skills/etch-plan/SKILL.md` and `.claude/skills/etch/SKILL.md` automatically
- [x] No interactive prompt is shown for the `.claude` directory during init (auto-create)
- [x] Init output includes skill installation confirmation
- [x] `etch skill install` still works independently for re-installation
- [x] Verify: run `etch init` in a temp directory and confirm both skill files exist

### Task 1.2: Require --name flag for etch plan [completed]
**Complexity:** small
**Files:** cmd/plan.go

Change `etch plan` to require the `--name` / `-n` flag. Currently it falls back to slugifying the description, which produces poor slugs. Make the `--name` flag `Required: true` in the flag definition. Update the error message to guide users: `usage: etch plan --name auth-system "add user authentication"`.

**Acceptance Criteria:**
- [x] Running `etch plan "description"` without `--name` returns a clear error
- [x] Running `etch plan --name my-plan "description"` works as before
- [x] Error hint shows the correct usage pattern with `--name`
- [x] Verify: `etch plan "test"` fails with helpful message, `etch plan -n test "test"` succeeds

---

## Feature 2: Plan Generation Quality

### Task 2.1: Tune skill prompt for larger, more thorough plans [completed]
**Complexity:** medium
**Files:** internal/skill/etch-plan.md

The current skill prompt produces plans that are too small and lack validation steps. Update the etch-plan skill to:

1. Add explicit sizing guidance: "Plans should have enough tasks to fully implement the feature. A typical plan has 5-15 tasks across 2-4 features. Don't under-scope — it's better to have more granular tasks than to combine too much into one."
2. Add a rule that every plan MUST include validation/testing tasks: "Every feature should end with a validation task that verifies the implementation works (e.g., writing tests, running the app, checking edge cases)."
3. Add a rule about acceptance criteria depth: "Each task should have 3-5 acceptance criteria. Include at least one verification criterion (e.g., 'Tests pass', 'Feature works in the UI', 'No regressions in existing tests')."
4. Add examples of validation tasks in the format spec.

**Acceptance Criteria:**
- [x] Skill prompt includes sizing guidance (5-15 tasks typical)
- [x] Skill prompt requires validation/testing tasks per feature
- [x] Skill prompt requires 3-5 acceptance criteria per task
- [x] Skill prompt requires at least one verification criterion per task
- [x] Prompt includes example of a validation task
- [x] Verify: read the updated SKILL.md and confirm all guidance is present

### Task 2.2: Update the embedded refine prompt with validation guidance [completed]
**Complexity:** small
**Files:** internal/generator/prompts.go

Update `formatSpec` in `prompts.go` to match the new sizing and validation rules added to the skill prompt. Add the same rules: tasks should have 3-5 criteria, include verification criteria, features should end with validation tasks. This ensures plans refined through the API (via `etch review`) also follow the improved standards.

**Acceptance Criteria:**
- [x] `formatSpec` includes the validation task requirement
- [x] `formatSpec` includes the 3-5 acceptance criteria rule
- [x] `formatSpec` includes the verification criterion requirement
- [x] Verify: `go build ./...` succeeds after changes

---

## Feature 3: Status Display Fixes

### Task 3.1: Show not-started plans in default status view [completed]
**Complexity:** small
**Files:** internal/status/status.go

The `IsActive()` method currently returns `false` for plans with 0 completed tasks (fully pending). Change it to also return `true` when all tasks are pending (0% complete) — these are "not started" plans that users want to see. The only plans that should be hidden by default are 100% completed plans. Update `IsActive()` to: return `true` if `CompletedTasks < TotalTasks` (i.e., only hide when fully complete).

**Acceptance Criteria:**
- [x] Plans with 0% progress appear in `etch status` (no `--all` needed)
- [x] Plans with 100% progress are hidden from `etch status` (shown with `--all`)
- [x] Plans with partial progress still appear as before
- [ ] Verify: create a new plan, run `etch status`, confirm it appears
- [x] Verify: existing tests pass with `go test ./internal/status/...`

### Task 3.2: Fix in_progress icon display in status output [completed]
**Complexity:** small
**Files:** internal/status/status.go

The `▶` icon for in-progress tasks may not be rendering in the status output. Debug the `taskIcon()` and `featureIcon()` functions to ensure `StatusInProgress` tasks display correctly. The `mapProgressStatus` function maps `"partial"` to `StatusInProgress` — verify this mapping is correct and that the icon renders properly. Also check that `FormatSummary` and `FormatDetailed` correctly show the in-progress icon.

**Acceptance Criteria:**
- [x] Tasks with `in_progress` status show the `▶` icon in `etch status`
- [x] Features with in-progress tasks show the `▶` icon
- [x] The `partial` → `in_progress` mapping in `mapProgressStatus` is correct
- [x] Verify: manually set a task to `in_progress` in a plan file and run `etch status`

### Task 3.3: Auto-detect plan completion and mark plan as done [completed]
**Complexity:** medium
**Files:** internal/status/status.go, internal/serializer/serializer.go

When `etch status` reconciles progress and finds that all tasks in a plan are `completed`, it should mark the plan's status as complete. Add a plan-level status field to the markdown format (e.g., `# Plan: Title [completed]`) and update the serializer to write it. In `reconcile()`, after aggregating totals, if `CompletedTasks == TotalTasks && TotalTasks > 0`, update the plan file with a completion marker. Also update the parser to read this status.

**Acceptance Criteria:**
- [x] When all tasks are completed, `etch status` marks the plan as `[completed]` in the plan file
- [x] The plan title line is updated from `# Plan: Title` to `# Plan: Title [completed]`
- [x] Parser reads the `[completed]` status from the plan title
- [x] A completed plan shows a `✓` icon in the status summary instead of `📋`
- [x] Verify: complete all tasks in a test plan, run `etch status`, confirm plan file is updated

---

## Feature 4: Claude Code Integration

### Task 4.1: Pass --dangerously-skip-permissions when launching Claude [completed]
**Complexity:** small
**Files:** internal/claude/claude.go

Update both `Run()` and `RunWithStdin()` to pass `--dangerously-skip-permissions` as a CLI flag to the `claude` command. This allows the generated sessions to auto-accept file edits without prompting the user. In `Run()`, change `exec.Command(path, prompt)` to `exec.Command(path, "--dangerously-skip-permissions", prompt)`. In `RunWithStdin()`, change `exec.Command(path)` to `exec.Command(path, "--dangerously-skip-permissions")`.

**Acceptance Criteria:**
- [x] `claude.Run()` passes `--dangerously-skip-permissions` flag
- [x] `claude.RunWithStdin()` passes `--dangerously-skip-permissions` flag
- [x] Verify: `go build ./...` succeeds
- [ ] Verify: run `etch plan -n test "test feature"` and confirm Claude starts without permission prompts

### Task 4.2: Auto-allow etch commands in Claude Code sessions [completed]
**Complexity:** medium
**Files:** internal/claude/claude.go, internal/skill/etch.md

Update `claude.Run()` and `claude.RunWithStdin()` to also pass `--allowedTools` with the etch binary commands so that `etch progress`, `etch run`, and `etch context` commands are automatically permitted. Add `--allowedTools "Bash(etch *)"` to the Claude invocation. Also update the etch skill reference (`etch.md`) to note that etch commands are auto-approved.

**Acceptance Criteria:**
- [x] Claude sessions launched by etch auto-approve `etch` CLI commands
- [x] The `--allowedTools` flag is passed correctly to the claude CLI
- [x] The etch skill reference notes that etch commands are auto-approved
- [x] Verify: run `etch run` and confirm etch progress commands don't prompt for approval

---

## Feature 5: Slug & List Improvements

### Task 5.1: Show slugs in etch list and make them first-class [completed]
**Complexity:** small
**Files:** cmd/list.go

Update `runList()` to display the slug for each plan, similar to how `etch status` shows `slug: <name>`. Change the format to show the slug prominently so users know what identifier to use for other commands. Format: `[priority] slug  Title  N/M tasks  XX%`. Also update the list command to accept a `--all` flag to include completed plans (default: hide completed).

**Acceptance Criteria:**
- [x] `etch list` output shows the slug for each plan
- [x] Slugs are visually prominent and easy to copy
- [x] `etch list` hides 100% completed plans by default
- [x] `etch list --all` shows all plans including completed ones
- [x] Verify: run `etch list` and confirm slugs appear in the output

### Task 5.2: Hide finished plans from status by default [completed]
**Complexity:** small
**Files:** cmd/status.go

The `--all` flag description says "show all plans including fully pending and completed" but after Task 3.1 changes, pending plans will show by default. Update the flag description to "show all plans including completed" and update the "No active plans" message to "No active plans. Use --all to see completed plans."

**Acceptance Criteria:**
- [x] `--all` flag description is updated to mention only completed plans
- [x] "No active plans" message mentions completed plans specifically
- [x] Verify: the help text (`etch status --help`) shows the updated description

---

## Feature 6: Replan Improvements

### Task 6.1: Add --reason flag to etch replan [completed]
**Complexity:** small
**Files:** cmd/replan.go

Add a `--reason` / `-r` string flag to the replan command. When provided, include the reason in the prompt sent to Claude Code so it knows why the user wants to replan. Append to the prompt: `\n\n**Reason for replanning:** <reason>`. Update the command description/examples to show the new flag.

**Acceptance Criteria:**
- [x] `etch replan --reason "tasks are too granular"` passes the reason to the Claude prompt
- [x] The reason appears in the prompt as `**Reason for replanning:** <text>`
- [x] The flag has both `--reason` and `-r` aliases
- [x] Replanning without `--reason` still works as before (no change in behavior)
- [ ] Verify: run `etch replan --reason "test"` and confirm the prompt includes the reason
