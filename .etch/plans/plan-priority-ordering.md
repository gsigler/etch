# Plan: Plan Priority Ordering [completed]

## Overview

Add a priority field to plans so they can be ordered by importance. Currently plans are sorted alphabetically by title (`SortPlanStatuses`) and listed in filesystem glob order (`etch list`). With this change, plans gain an optional `**Priority:**` metadata line in their markdown (e.g. `**Priority:** 1`), parsed and stored in the `Plan` model. All commands that display plans (`etch list`, `etch status`) sort by priority first (lower number = higher priority), then alphabetically as tiebreaker. Plans without an explicit priority default to `0` (unset) and sort **after** all explicitly prioritized plans so that prioritized work surfaces first. The `etch plan` and `etch replan` commands get a `--priority N` flag, and an `etch priority` command is added to view and change priorities.

### Task 1: Add Priority field to Plan model and parser [completed]
**Complexity:** small
**Files:** internal/models/models.go, internal/parser/parser.go, internal/parser/parser_test.go

Add the `Priority` field to the data model and teach the parser to extract it from plan markdown.

**Model (`internal/models/models.go`):**
- Add `Priority int` field to the `Plan` struct with JSON tag `json:"priority"`
- Convention: 0 = unset, 1 = highest explicit priority, higher numbers = lower priority

**Parser (`internal/parser/parser.go`):**
- Add `priorityRe = regexp.MustCompile(`^\*\*Priority:\*\*\s*(\d+)\s*$`)` to the regex block alongside `complexityRe`, `filesRe`, etc.
- In the `statePlanLevel` case (currently has no metadata parsing — lines fall through silently), add a check for `priorityRe` before the existing `switch cur` block
- Parse the matched integer via `strconv.Atoi` into `plan.Priority`
- Only match at plan level — priority lines inside features or tasks must be ignored

**Tests (`internal/parser/parser_test.go`):**
- Table-driven tests following existing patterns: priority present, priority absent (defaults to 0), priority line inside a task description (ignored)
- Verify parsing of `**Priority:** 3` produces `plan.Priority == 3`

**Acceptance Criteria:**
- [x] `Plan` struct has `Priority int` field with JSON tag
- [x] `**Priority:** N` is parsed from plan-level section into `plan.Priority`
- [x] Missing priority defaults to 0
- [x] Priority lines inside task/feature sections are ignored
- [x] Parser tests cover present, absent, and wrong-section cases

### Task 2: Add Priority to serializer with surgical updates [completed]
**Complexity:** medium
**Files:** internal/serializer/serializer.go, internal/serializer/serializer_test.go
**Depends on:** Task 1

Update the serializer to emit and surgically update priority lines.

**Full serialization (`Serialize`):**
- After writing `# Plan: <title>\n`, emit `**Priority:** N\n` when `plan.Priority > 0`
- Omit the line entirely when priority is 0 (unset)
- The priority line goes between the `# Plan:` heading and the `## Overview` section (or first `## Feature`)

**Surgical update (`UpdatePlanPriority`):**
Add a new exported function `UpdatePlanPriority(filePath string, newPriority int) error` following the same pattern as `UpdateTaskStatus` and `UpdateCriterion`:
1. Read file into `[]string` lines
2. Find the `# Plan:` heading line
3. Find the first `## ` heading after it
4. Search lines between them for an existing `**Priority:**` line
5. Three cases:
   - **Replace:** existing line found and `newPriority > 0` — replace the line
   - **Insert:** no existing line and `newPriority > 0` — insert `**Priority:** N` on the line after `# Plan:` heading
   - **Remove:** existing line found and `newPriority == 0` — delete the line
6. Rejoin lines and write back with `os.WriteFile`
7. Use `etcherr.WrapIO` with `.WithHint()` for errors

**Tests (`internal/serializer/serializer_test.go`):**
- `Serialize` emits priority when > 0, omits when 0
- `UpdatePlanPriority`: update existing, insert when missing, remove when set to 0
- Round-trip: `Serialize` → `Parse` → `Serialize` produces identical output for plans with and without priority

**Acceptance Criteria:**
- [x] `Serialize` emits `**Priority:** N` after plan heading when priority > 0
- [x] `Serialize` omits priority line when priority is 0
- [x] `UpdatePlanPriority` replaces an existing priority line
- [x] `UpdatePlanPriority` inserts priority after `# Plan:` heading when none exists
- [x] `UpdatePlanPriority` removes the priority line when newPriority is 0
- [x] Round-trip serialization is stable

### Task 3: Update status package to sort and display by priority [completed]
**Complexity:** small
**Files:** internal/status/status.go, internal/status/status_test.go
**Depends on:** Task 1

Wire priority through the status package and update sorting.

**`PlanStatus` struct:** Add `Priority int` field with `json:"priority"` tag.

**`reconcile` function:** Populate `Priority: plan.Priority` when building `PlanStatus` from a parsed plan.

**`SortPlanStatuses` function:** Change from alphabetical-only to priority-first sorting. Unset priority (0) sorts **after** all explicit priorities so prioritized plans surface first:
```go
sort.Slice(plans, func(i, j int) bool {
    pi, pj := plans[i].Priority, plans[j].Priority
    if pi == pj {
        return plans[i].Title < plans[j].Title
    }
    if pi == 0 { return false }
    if pj == 0 { return true }
    return pi < pj
})
```

**`FormatSummary` and `FormatDetailed`:** Show priority in plan headers when set, e.g. `[1] Plan Title` vs `[ ] Plan Title` for unset.

**Tests:** Update existing status tests that assume alphabetical sort order. Add tests verifying: explicit priorities sort ascending, unset sorts last, alphabetical tiebreak within same priority.

**Acceptance Criteria:**
- [x] `PlanStatus` struct includes `Priority int` field
- [x] Priority is populated from parsed plan in `reconcile`
- [x] `SortPlanStatuses` sorts by priority first (1, 2, 3... then 0/unset)
- [x] Alphabetical tiebreak within same priority level
- [x] `FormatSummary` and `FormatDetailed` show priority numbers
- [x] Existing status tests updated and passing

### Task 4: Update list command to sort and display by priority [completed]
**Complexity:** small
**Files:** cmd/list.go
**Depends on:** Task 1

The `etch list` command uses `etchcontext.DiscoverPlans` which returns `[]*models.Plan` in glob order — it does not use the status package for sorting.

**Changes:**
- After discovering plans, sort the slice by `plan.Priority` (same logic as `SortPlanStatuses`: explicit priorities ascending, then unset, then alphabetical tiebreak)
- Update the `fmt.Printf` format to show priority: `[1]`, `[2]`, `[ ]` for unset

**Expected output:**
```
[1] Etch Go Command              3/5 tasks   60%
[2] Parallel Worktree Execution  0/8 tasks    0%
[ ] TUI Dashboard                0/7 tasks    0%
```

**Acceptance Criteria:**
- [x] `etch list` sorts plans by priority (explicit first, unset last)
- [x] Priority numbers shown next to plan names
- [x] Unset priorities display as `[ ]`

### Task 5: Add --priority flag to plan and replan commands [completed]
**Complexity:** small
**Files:** cmd/plan.go, cmd/replan.go
**Depends on:** Task 2

Add a `--priority` int flag to both `etch plan` and `etch replan` commands.

**`cmd/plan.go`:** Add `&cli.IntFlag{Name: "priority", Usage: "set plan priority (lower = higher priority)"}` to the flags list. After the plan file is generated by Claude and saved to disk, if `c.Int("priority") > 0`, call `serializer.UpdatePlanPriority(planPath, priority)` to surgically insert the priority line. This avoids modifying the generator prompt.

**`cmd/replan.go`:** Same pattern — add `--priority` flag, apply surgically after the replan completes.

**Acceptance Criteria:**
- [x] `etch plan --priority 2 "description"` creates a plan with priority 2
- [x] `etch plan "description"` (no flag) creates a plan without a priority line
- [x] `etch replan --priority 1 <plan>` sets priority on replanned file
- [x] Priority is applied surgically after file generation

### Task 6: Add etch priority command [completed]
**Complexity:** medium
**Files:** cmd/priority.go, cmd/root.go
**Depends on:** Task 2, Task 3

Add a standalone `etch priority` command for viewing and changing plan priorities.

**Three forms:**
1. `etch priority` (no args) — list all plans sorted by priority showing their current priority values. Use `etchcontext.DiscoverPlans` and sort the same way as `etch list`.
2. `etch priority <plan-slug> <N>` — set a plan's priority to N. Resolve plan slug to file path using `etchcontext.DiscoverPlans`, validate N is a positive integer, call `serializer.UpdatePlanPriority(planPath, N)`. After updating, re-read and display the new ordering.
3. `etch priority <plan-slug> --unset` — remove priority (set to 0) by calling `serializer.UpdatePlanPriority(planPath, 0)`.

**Implementation:**
- Follow CLI factory pattern: `func priorityCmd() *cli.Command`
- Use `etcherr.*` constructors with `.WithHint()` for all errors (plan not found, invalid number, etc.)
- Register in `cmd/root.go` Commands slice
- Add `--unset` as a `cli.BoolFlag`

**Acceptance Criteria:**
- [x] `etch priority` lists plans in priority order with their values
- [x] `etch priority <slug> <N>` sets a plan's priority
- [x] `etch priority <slug> --unset` removes priority (sets to 0)
- [x] Invalid inputs produce helpful error messages with hints
- [x] Command registered in root and appears in `etch --help`
