# Plan: Build Etch – AI Implementation Planning CLI

## Overview

Etch is a Go CLI tool that helps developers create, review, and execute AI-generated implementation plans. The plan markdown file is the center of everything — it serves as the spec, the context source, and the shared document that humans review and approve. Progress tracking lives in separate per-session files so multiple agents can work simultaneously without conflicts.

**Core Problem:** When a feature takes multiple AI coding sessions to complete, developers lose context between sessions, manually re-explain state, and have no structured way to track progress. Etch solves this with a planning layer that decomposes work, generates context prompts, and tracks progress across sessions and agents.

**Core Loop:**
1. `etch plan "description"` → Claude API generates a structured plan
2. `etch review` → Interactive TUI to scroll through the plan and leave 💬 comments for AI refinement
3. Iterate until the plan is solid (share with teammates for sign-off — it's just a markdown file)
4. `etch context <task>` → Assembles context prompt file, creates empty session progress file, prints ready-to-run command
5. `claude --append-system-prompt-file .etch/context/<file>` → Agent works on the task and fills in the session progress file
6. `etch status` → Reads plan + all progress files, shows current state, updates plan status tags and checkboxes
7. Repeat from step 4 for the next task

**Technology:**
- **Language:** Go
- **TUI:** Bubbletea + Lipgloss (Charm ecosystem)
- **Markdown parsing:** Line-based state machine (no external markdown library needed)
- **AI:** Claude API (Anthropic) via direct HTTP for plan generation/refinement; Claude Code CLI for task execution via context prompt files
- **Context delivery:** Prompt files in `.etch/context/` + ready-to-run `claude` command
- **Config:** TOML (BurntSushi/toml)
- **State:** All files, no database

**What Etch is NOT:** No GUI, no board view, no process management, no database. Etch is a CLI tool. The `etch status` command is your dashboard. The TUI review mode is your review interface. Everything else happens in your editor and your terminal.

---

## Architecture Decisions

### State is all files. No database.

```
your-project/
├── .etch/
│   ├── config.toml                          # API key ref, model prefs, defaults
│   ├── backups/                             # Auto-backup before AI refinement
│   │   └── auth-system-2026-02-16T10-30.md
│   ├── context/                             # Generated prompt files (latest per task)
│   │   └── auth-system--task-1.2--002.md    # Context prompt for task 1.2, session 2
│   ├── plans/
│   │   ├── auth-system.md                   # Plan file (source of truth for spec)
│   │   └── api-refactor.md                  # Another plan
│   └── progress/
│       ├── auth-system--task-1.1--001.md    # Session 1 for task 1.1
│       ├── auth-system--task-1.1--002.md    # Session 2 for task 1.1
│       ├── auth-system--task-1.2--001.md    # Session 1 for task 1.2
│       └── auth-system--task-2.1--001.md    # Session 1 for task 2.1 (maybe a different agent)
├── src/
└── ...
```

**Progress file naming:** `<plan-slug>--task-<N.M>--<session-number>.md`

Flat files. No nesting. Easy to glob, sort, and understand at a glance.

### Hierarchy: Plan → Feature → Task

- A **plan file** is the top-level container
- **Features** are major sections within a plan (one or many)
- **Tasks** are agent-sized units of work within a feature
- Single-feature plans can omit the feature heading

### Separation of concerns

| File | Purpose | Who writes it | Git tracked? |
|------|---------|---------------|-------------|
| `.etch/plans/*.md` | The spec — what to build | Human + AI (via etch plan/review) | Always yes |
| `.etch/progress/*.md` | Execution log — what happened | AI agents (during sessions) | User chooses at init |
| `.etch/context/*.md` | Generated prompt files for agents | Etch (`etch context`) | No (gitignored, regenerable) |
| `.etch/backups/*.md` | Safety net before AI refinement | Etch (automatic) | No (gitignored) |
| `.etch/config.toml` | Settings | Human | User chooses (usually no — contains API key ref) |

### Review comments are inline

Comments use `> 💬` blockquote syntax directly in the plan markdown:

```markdown
### Task 1.2: Auth Endpoints [completed]

> 💬 This should handle token rotation too. What happens on expiry?

> 💬 Middleware or per-route guards?

Implement POST /auth/register and POST /auth/login...
```

Comments are refinement instructions for AI. `etch review --apply` sends them to Claude, Claude addresses them and removes the resolved ones. No sidecar files, no JSON.

### Progress is separate from the plan

Agents write to `.etch/progress/<plan>--task-<N.M>--<session>.md`. The plan stays clean as a readable spec. `etch status` reads progress files and updates the plan's status tags and checkboxes.

### Testing as we go

Every task ships with tests. No code lands without coverage for its core behavior.

- **Unit tests:** Every `internal/` package gets `*_test.go` files alongside the code. Test the public API of each package.
- **Integration tests for `etch init`:** Use `t.TempDir()` to create isolated filesystem environments. Verify directory creation, config generation, and gitignore handling.
- **Table-driven tests:** Use Go's table-driven test pattern for parsing, serialization, and status mapping — easy to add cases as edge cases surface.
- **Test fixtures:** Store sample plan and progress files in `testdata/` directories within each package. Include the plan spec itself as a fixture.
- **CI-friendly:** All tests run with `go test ./...`. No external dependencies, no network calls in unit tests. API client tests use a local HTTP test server (`httptest.NewServer`).
- **Run tests before marking any task complete.** If tests don't pass, the task isn't done.

### Backups before AI refinement

Every time `etch review --apply` or `etch replan` modifies the plan, Etch copies the current version to `.etch/backups/<plan>-<timestamp>.md` first. No silent data loss.

---

## Plan Markdown Format Specification

This is the contract between the CLI, the AI, and the developer.

### Full format

```markdown
# Plan: <Plan Title>

## Overview
<High-level description. Why are we doing this? What's the end state?>

---

## Feature 1: <Feature Title>

### Overview
<What this feature accomplishes>

### Task 1.1: <Task Title> [completed]
**Complexity:** small | medium | large
**Files:** <comma-separated file paths or globs>
**Depends on:** none | Task X.Y, Task X.Z

<Detailed description — multiple paragraphs allowed>

> 💬 <Review comment from human — AI refinement instruction>

**Acceptance Criteria:**
- [ ] <Concrete, verifiable outcome>
- [x] <Completed criterion>

---

## Feature 2: <Feature Title>
...
```

### Single-feature shorthand

When a plan has only one feature, omit the feature heading:

```markdown
# Plan: Add Rate Limiting

## Overview
Add rate limiting to all API endpoints...

### Task 1: <Title> [status]
...

### Task 2: <Title> [status]
...
```

### Parsing rules

- Plan title: `# Plan: <title>`
- Features: `## Feature N: <title>` (number is parsed from the heading)
- Tasks: `### Task N.M: <title> [status]` (multi-feature) or `### Task N: <title> [status]` (single-feature)
- Status values: `pending` | `in_progress` | `blocked` | `completed` | `failed`
- Metadata lines: `**Complexity:**`, `**Files:**`, `**Depends on:**`
- Acceptance criteria: `- [x]` = met, `- [ ]` = not met
- Review comments: `> 💬 <text>` (may be multi-line blockquote)
- Section separators: `---` (ignored during parsing)
- Anything not matching a known pattern → part of the task/feature description

### Status tag is canonical

The `[status]` in the task heading is the source of truth. `etch status` updates it based on progress files. Humans can also manually change it.

---

## Session Progress File Format

Created by `etch context` with headers pre-filled. Agent fills in the content.

**File:** `.etch/progress/<plan>--task-<N.M>--<session>.md`

```markdown
# Session: Task 1.1 – Database Schema
**Plan:** auth-system
**Task:** 1.1
**Session:** 001
**Started:** 2026-02-16 09:30
**Status:** completed

## Changes Made
- Created src/db/migrations/001_users.sql
- Created src/db/models/user.rs
- Added sqlx dependency to Cargo.toml

## Acceptance Criteria Updates
- [x] Migration file creates users table
- [x] User model struct matches schema
- [x] Migration runs successfully on empty database

## Decisions & Notes
Chose sqlx over diesel for async support. The migration uses
IF NOT EXISTS for idempotency. Password hash field is VARCHAR(255)
to accommodate bcrypt output.

## Blockers
None.

## Next
All criteria met. Task complete.
```

**Status values in progress files:** `completed` | `partial` | `failed` | `blocked`

**How `etch status` reconciles:**
1. Reads all progress files for the plan
2. For each task, finds the latest session (highest session number)
3. Maps progress status to plan status: `completed` → `completed`, `partial` → `in_progress`, `failed` → `failed`, `blocked` → `blocked`
4. Merges acceptance criteria: if any session marks `[x]`, the plan gets `[x]`
5. Updates the plan file (status tags + checkboxes)
6. Displays summary

---

## Context Prompt Template

Assembled by `etch context <task>` and written to `.etch/context/<plan>--task-<N.M>--<session>.md`:

```
# Etch Context — Implementation Task

You are working on a task as part of an implementation plan managed by Etch.

## Plan: <Plan Title>
<Plan overview, condensed to 2-3 sentences>

## Current Plan State
Feature 1: <Title>
  ✓ Task 1.1: <Title> (completed)
  ▶ Task 1.2: <Title> (in_progress — this is your task)
  ○ Task 1.3: <Title> (pending, depends on 1.2)
Feature 2: <Title>
  ○ Task 2.1: <Title> (pending)

## Your Task: Task <N.M> — <Title>
**Complexity:** <complexity>
**Files in Scope:** <files>
**Depends on:** <dependencies>

<Full task description from the plan>

### Acceptance Criteria
- [ ] <criterion>
- [x] <already completed criterion>

### Previous Sessions
<If not session 001, summaries from prior sessions for this task>

**Session 001 (2026-02-16, partial):**
Changes: <changes made summary>
Decisions: <key decisions>
Blockers: <what blocked>
Next: <what to do next>

### Completed Prerequisites
<For each completed dependency, what was done>

**Task 1.1 (completed):**
Created users table migration and User model. Using sqlx with async.

## Session Progress File

Update your progress file as you work:
`.etch/progress/<plan>--task-<N.M>--<session>.md`

This file has been created for you. Fill in each section:
- **Changes Made:** files created or modified
- **Acceptance Criteria Updates:** check off what you completed
- **Decisions & Notes:** design decisions, important context
- **Blockers:** anything blocking progress
- **Next:** what still needs to happen
- **Status:** update to completed, partial, failed, or blocked

Rules:
- Stay within the files listed in scope. Ask before modifying others.
- Do NOT modify the plan file. Only update your progress file.
- Keep notes concise but useful — future sessions depend on them.
```

---

## Feature 1: Core Data Layer & Plan Parser

### Overview
Build the Go modules for parsing plan markdown, reading/writing progress files, and managing the `.etch/` directory. Every other feature depends on these.

### Task 1.1: Project scaffold and CLI skeleton [completed]
**Complexity:** small
**Files:** go.mod, main.go, cmd/*.go, internal/etch/*.go
**Depends on:** none

Initialize the Go project:

```
etch/
├── go.mod
├── go.sum
├── main.go                    # Entry point, calls cmd.Execute()
├── cmd/
│   ├── root.go                # Root command dispatch, global flags
│   ├── init.go                # etch init
│   ├── plan.go                # etch plan
│   ├── review.go              # etch review
│   ├── status.go              # etch status
│   ├── context.go             # etch context
│   ├── replan.go              # etch replan
│   ├── list.go                # etch list
│   └── open.go                # etch open
├── internal/
│   ├── parser/                # Plan markdown parsing
│   ├── serializer/            # Plan markdown writing
│   ├── progress/              # Progress file read/write
│   ├── generator/             # AI plan generation + refinement
│   ├── context/               # Context prompt assembly
│   ├── config/                # Config management
│   └── models/                # Data structures
└── README.md
```

Use `urfave/cli/v2` for CLI framework — lightweight, sufficient for ~8 subcommands, no code generation overhead. All subcommands print "not yet implemented" stubs.

`etch init`:
- Creates `.etch/plans/`, `.etch/progress/`, `.etch/context/`, `.etch/backups/`
- Creates `.etch/config.toml` with documented defaults
- Asks: "Track progress files in git? (y/N)" → if no, adds `.etch/progress/` to `.gitignore`
- Always adds `.etch/backups/`, `.etch/context/`, and `.etch/config.toml` to `.gitignore`
- Prints quickstart message

**Acceptance Criteria:**
- [x] `go build` produces a working binary
- [x] `etch init` creates directory structure and config
- [x] `.gitignore` handling based on user choice
- [x] All subcommands exist as stubs with `--help`
- [x] Package structure matches the layout above
- [x] `go test ./...` passes
- [x] Tests for `etch init`: directory creation, config file content, gitignore entries (yes and no paths), idempotent re-init

### Task 1.2: Data models [completed]
**Complexity:** small
**Files:** internal/models/models.go
**Depends on:** Task 1.1

Define core data structures:

```go
type Status string
const (
    StatusPending    Status = "pending"
    StatusInProgress Status = "in_progress"
    StatusBlocked    Status = "blocked"
    StatusCompleted  Status = "completed"
    StatusFailed     Status = "failed"
)

type Complexity string
const (
    ComplexitySmall  Complexity = "small"
    ComplexityMedium Complexity = "medium"
    ComplexityLarge  Complexity = "large"
)

type Plan struct {
    Title    string
    Overview string
    Features []Feature
    FilePath string    // path to the .md file
    Slug     string    // derived from filename
}

type Feature struct {
    Number   int
    Title    string
    Overview string
    Tasks    []Task
}

type Task struct {
    FeatureNumber int
    TaskNumber    int
    Title         string
    Status        Status
    Complexity    Complexity
    Files         []string
    DependsOn     []string
    Description   string
    Criteria      []Criterion
    Comments      []string   // 💬 review comments
}

type Criterion struct {
    Description string
    IsMet       bool
}

type SessionProgress struct {
    PlanSlug      string
    TaskID        string    // "1.1"
    SessionNumber int
    Started       string
    Status        string    // completed, partial, failed, blocked
    ChangesMade   []string
    CriteriaUpdates []Criterion
    Decisions     string
    Blockers      string
    Next          string
}
```

Include helper methods: `Task.FullID()` → `"1.2"`, `Status.Icon()` → `"✓"/"▶"/"○"/"✗"/"⊘"`, `Plan.TaskByID(id string)`, `ParseStatus(s string) Status`.

**Acceptance Criteria:**
- [x] All types defined with JSON tags for potential future use
- [x] Helper methods implemented with table-driven tests in `internal/models/models_test.go`
- [x] `ParseStatus` handles unknown strings gracefully (defaults to Pending) — tested
- [x] `Task.FullID()` returns correct format for single and multi-feature plans — tested
- [x] `Status.Icon()` returns correct icon for all status values — tested
- [x] `go test ./internal/models/...` passes

### Task 1.3: Plan markdown parser — structure [completed]
**Complexity:** medium
**Files:** internal/parser/parser.go, internal/parser/parser_test.go
**Depends on:** Task 1.2

Parse plan markdown files into `Plan` structs using a line-based state machine. No external markdown library — the plan format is a strict subset of markdown with known heading patterns, so a state machine tracking heading depth via `#` prefixes is simpler, faster, and dependency-free.

Parsing strategy — line-based state machine:
- Read file line by line, track current state (plan-level, feature-level, task-level)
- `# Plan:` → extract plan title, switch to plan state
- `## Feature N:` → create new feature, switch to feature state
- `## Overview` → plan overview section
- `### Task N.M:` or `### Task N:` → create new task, switch to task state
- `---` → skip (section separator)
- Anything else → accumulate into current section's description
- Heading depth determines hierarchy: H1=plan, H2=feature/overview, H3=task

Handle single-feature detection: if tasks appear (H3) without any prior `## Feature` heading, wrap them in an implicit feature.

Edge cases:
- Single-feature plan (no `## Feature` headings)
- Empty features (heading but no tasks yet)
- Malformed or missing status tag → default to `pending`
- `---` separators (skip)

**Acceptance Criteria:**
- [x] Parses multi-feature plans into correct Plan/Feature/Task hierarchy
- [x] Parses single-feature plan shorthand (implicit feature wrapping)
- [x] Extracts plan title and overview
- [x] Extracts feature numbers, titles, and overviews
- [x] Returns clear error for fundamentally broken files (no `# Plan:` heading)
- [x] Gracefully handles empty features and missing sections
- [x] Test: parses this spec file and produces correct hierarchy

> 💬 The parser skips letter-suffix task IDs like `### Task 1.3b:` — the regex `(\d+)(?:\.(\d+))?` doesn't handle them. Task 1.3b should update the regex to support `(\d+)(?:\.(\d+)([a-z])?)?` so our own plan parses fully.

### Task 1.3b: Plan parser — task metadata extraction [completed]
**Complexity:** medium
**Files:** internal/parser/parser.go, internal/parser/parser_test.go
**Depends on:** Task 1.3

Within each task section identified by the structure parser, extract all task-level metadata using line-level pattern matching:

- Regex for status tag: `\[(\w+)\]$` at end of heading
- Prefix match for metadata: `**Complexity:**`, `**Files:**`, `**Depends on:**`
- Checkbox match: `^- \[([ x])\] (.+)$`
- Comment match: `^> 💬 (.+)$` (possibly multi-line blockquote)
- Everything else → description text

Edge cases:
- Task with no metadata lines
- Task with no acceptance criteria
- Multi-line `> 💬` comments (consecutive `>` lines)

**Acceptance Criteria:**
- [x] Extracts all task metadata (status, complexity, files, depends_on)
- [x] Extracts acceptance criteria with completion state
- [x] Extracts `> 💬` review comments (single and multi-line)
- [x] Gracefully handles missing/optional metadata fields
- [x] Test: parses this spec file and produces correct task details
- [ ] Test: round-trip parse → serialize → parse produces equivalent result (deferred to Task 1.4)

### Task 1.4: Plan markdown serializer [completed]
**Complexity:** medium
**Files:** internal/serializer/serializer.go, internal/serializer/serializer_test.go
**Depends on:** Task 1.3b

Two modes:

**Full serialization:** `Plan` struct → complete markdown string. Used for new plans from AI generation.

**Targeted update:** Modify specific fields in an existing plan file without rewriting everything. Used by `etch status` to update status tags and checkboxes.

Targeted update approach:
- Read the file as lines
- For status update: find the line matching `### Task N.M:` and replace the `[old_status]` with `[new_status]`
- For checkbox update: within a task's section, find the criterion line and flip `[ ]` to `[x]`
- Write the file back

This preserves all formatting, descriptions, comments, and whitespace that the full serializer might subtly alter.

**Acceptance Criteria:**
- [x] Full serialize produces valid markdown matching the format spec
- [x] Targeted update: changes a task's status tag without touching other content
- [x] Targeted update: flips acceptance criteria checkboxes
- [x] Preserves all unrelated content (descriptions, comments, blank lines)
- [x] Round-trip test: parse → full serialize → parse produces equivalent Plan
- [x] Targeted update doesn't introduce formatting drift

### Task 1.5: Progress file reader/writer [completed]
**Complexity:** medium
**Files:** internal/progress/progress.go, internal/progress/progress_test.go
**Depends on:** Task 1.2

**Writing (called by `etch context`):**
- Create a new progress file with correct naming: `<plan>--task-<N.M>--<NNN>.md`
- Determine next session number by globbing existing files for that task
- Use atomic file creation (`os.OpenFile` with `O_CREATE|O_EXCL`) to prevent race conditions when two agents start the same task concurrently — if the file already exists, increment and retry
- Pre-fill headers from plan data (task title, plan slug, session number, timestamp)
- Pre-fill acceptance criteria from the plan (all with current check state)
- Leave content sections empty with placeholder comments for the agent

**Reading (called by `etch status`):**
- Glob `.etch/progress/<plan>--*.md`
- Parse each file: extract task ID, session number, status, criteria updates
- Line-based parsing — look for `**Status:**`, `**Task:**`, `**Session:**` lines and section headers
- Group by task ID, sort by session number
- Return map of task ID → latest session state

Be reasonably strict in parsing — if the agent wrote something that doesn't match the expected format, log a warning and skip that field rather than crashing.

**Acceptance Criteria:**
- [x] Creates correctly named progress files with pre-filled template
- [x] Auto-increments session number with atomic `O_EXCL` file creation (no race conditions)
- [x] Template matches the session format spec
- [x] Reads and parses agent-filled progress files
- [x] Groups by task, returns latest state per task
- [x] Handles: missing files, partially filled files, extra content
- [x] Warning (not crash) on unparseable fields

### Task 1.6: Config management [completed]
**Complexity:** small
**Files:** internal/config/config.go
**Depends on:** Task 1.1

```toml
[api]
model = "claude-sonnet-4-20250514"
# API key: set ANTHROPIC_API_KEY env var (preferred) or uncomment below
# api_key = "sk-ant-..."

[defaults]
complexity_guide = "small = single focused session, medium = may need iteration, large = multiple sessions likely"
```

- Read with `BurntSushi/toml`
- API key: `ANTHROPIC_API_KEY` env var → config file `api_key` → error with helpful message
- Defaults for all optional fields
- `Config` struct passed explicitly (no global state)

**Acceptance Criteria:**
- [x] Reads `.etch/config.toml`
- [x] API key resolution chain works correctly
- [x] Missing config file → sensible defaults (don't crash)
- [x] Helpful error when no API key found anywhere
- [x] Tests in `internal/config/config_test.go`: valid config, missing file, env var override, missing API key error
- [x] `go test ./internal/config/...` passes

---

## Feature 2: Plan Generation & AI Integration

### Overview
Claude API integration for generating plans, refining them from review comments, and replanning tasks/features that need rethinking. Uses the Anthropic Messages API directly with streaming for real-time output.

### Task 2.1: API client [completed]
**Complexity:** small
**Files:** internal/api/client.go, internal/api/client_test.go
**Depends on:** Task 1.6

HTTP client for the Anthropic Messages API.

- POST to `https://api.anthropic.com/v1/messages`
- Headers: `x-api-key`, `anthropic-version: 2023-06-01`, `content-type: application/json`
- Support streaming (SSE) for showing generation progress
- Errors: 401 (bad key), 429 (rate limit — retry with backoff, max 3), 500, network
- Keep generic: accepts system prompt + user message + model, returns response text

Use `net/http` from stdlib. Parse SSE `data:` lines for streaming.

**Acceptance Criteria:**
- [x] Makes Messages API requests successfully
- [x] Streaming SSE output works (yields text chunks via channel or callback)
- [x] Handles auth, rate limit, server, and network errors with clear messages
- [x] Respects model setting from config
- [x] Exponential backoff on 429 (1s, 2s, 4s, give up)
- [x] Tests using `httptest.NewServer` — no real API calls: success response, streaming chunks, 401/429/500 error handling, backoff retry count
- [x] `go test ./internal/api/...` passes

### Task 2.2: Plan generation command [completed]
**Complexity:** medium
**Files:** internal/generator/generator.go, internal/generator/prompts.go, cmd/plan.go
**Depends on:** Task 1.3b, Task 1.4, Task 2.1

Implement `etch plan "description"`.

Flow:
1. Gather project context:
   - File tree (3 levels deep, excluding .git, node_modules, target, __pycache__, .etch, vendor, dist, build)
   - Key config files (first 100 lines): package.json, Cargo.toml, go.mod, pyproject.toml, tsconfig.json, Makefile, docker-compose.yml
   - CLAUDE.md if present
   - Titles of existing plans in `.etch/plans/`
2. Construct system prompt:
   - Include the plan format specification
   - Instructions: agent-sized tasks, specific files, verifiable criteria, consider dependencies, realistic complexity
   - Complexity guide from config
3. Construct user message: feature description + project context
4. Stream to terminal — show tokens as they arrive
5. Extract markdown from response
6. Validate by parsing
7. Write to `.etch/plans/<slug>.md`
8. Print summary

Slug: lowercase, hyphens for spaces, strip non-alphanumeric, max 50 chars. If slug exists, prompt: "Plan '<slug>' already exists. Create '<slug>-2'? Or did you mean `etch replan <slug>`? (c)reate / (r)eplan / (q)uit"

**Acceptance Criteria:**
- [x] Generates valid plan from description
- [x] Project context gathered and included
- [x] Streaming shows real-time generation
- [x] Output parses successfully
- [x] Written to correctly named file
- [x] Summary printed (feature count, task count)
- [x] Handles: empty project, very large tree (truncate at ~2000 lines), duplicate slugs
- [x] Tests: project context gathering (file tree, config detection), slug generation and collision handling, prompt construction (verify format spec included). Use mock API client for generation tests.
- [x] `go test ./internal/generator/...` passes

### Task 2.3: Plan refinement from review comments [completed]
**Complexity:** medium
**Files:** internal/generator/refine.go, internal/generator/prompts.go
**Depends on:** Task 2.2

Send plan + `> 💬` comments to Claude for refinement. Called by the TUI's apply action or `etch review --apply <plan>`.

Flow:
1. Parse the plan, verify there are `> 💬` comments
2. **Backup** the current plan to `.etch/backups/<plan>-<timestamp>.md`
3. Construct prompt: "Revise this plan to address the 💬 review comments. Remove comments you've addressed. Preserve the format exactly. Only change what the feedback asks for."
4. Stream the response
5. Parse the response to validate it's a valid plan
6. Show a colored diff in the terminal (red removed, green added)
7. Prompt: "Apply changes? (y/n)"
8. Yes → overwrite plan file
9. No → no changes (backup remains for safety)

**Acceptance Criteria:**
- [x] Extracts 💬 comments from plan
- [x] Backs up plan before making changes
- [x] Sends plan + comments to Claude API
- [x] Validates response parses correctly
- [x] Shows colored terminal diff
- [x] Confirmation prompt before applying
- [x] Addressed comments removed, unaddressed preserved
- [x] "No comments found" message if plan has no 💬 comments
- [x] Tests: backup creation, comment extraction, "no comments" path, diff generation. Use mock API client.
- [x] `go test ./internal/generator/...` passes

### Task 2.4: Replan command [completed]
**Complexity:** medium
**Files:** internal/generator/replan.go, cmd/replan.go
**Depends on:** Task 2.3, Task 1.5

Implement `etch replan <target>` — AI-powered replanning for tasks or features that need rethinking.

Target resolution (smart scope detection):
- `etch replan 1.2` → replan Task 1.2
- `etch replan feature:2` or `etch replan "Login Endpoints"` → replan all of Feature 2
- `etch replan auth-system 1.2` → replan task in specific plan
- If just a number and ambiguous, prefer task interpretation

Behavior varies by context:
- **Task with no sessions:** This is a planning issue. Prompt: "Rethink this task. Is the scope right? Are the criteria clear? Should it be split into smaller tasks?"
- **Task with failed/blocked sessions:** This is an approach issue. Include session history in prompt. Ask: "This task has been attempted N times. Here's what happened. Suggest an alternative approach or break it down differently."
- **Feature scope:** Replan all tasks within the feature. Include progress on completed tasks so Claude doesn't redo them.

Always backs up before applying changes.

**Acceptance Criteria:**
- [x] Target resolution works for task IDs, feature references, and plan-scoped IDs
- [x] Adapts prompt based on session history (planning issue vs approach issue)
- [x] Feature-level replan preserves completed tasks
- [x] Backs up plan before changes
- [x] Shows diff and requires confirmation
- [x] Can split a single task into multiple tasks
- [x] Updated plan parses correctly
- [x] Tests: target resolution (task ID, feature ref, plan-scoped), prompt adaptation based on session history, backup creation. Use mock API client and fixture plans.
- [x] `go test ./internal/generator/...` passes

---

## Feature 3: Context Generation & Status

### Overview
The commands that make plans actionable: `etch context` for prompt file generation, `etch status` for current state.

### Task 3.1: Context prompt assembly [completed]
**Complexity:** medium
**Files:** internal/context/context.go, cmd/context.go
**Depends on:** Task 1.3b, Task 1.5

Implement `etch context [task_id]`.

Task identification:
- `etch context` (no args) → auto-select next pending task respecting dependency order. If multiple candidates, show picker.
- `etch context 1.2` → Feature 1, Task 2
- `etch context 2` → Task 2 in single-feature plan
- `etch context auth-system 1.2` → specific plan
- If multiple plans exist and no plan specified, show picker: "Which plan? (1) auth-system (2) api-refactor"

Assemble the context prompt following the template in this spec:
1. Plan overview (condensed)
2. Current plan state (all tasks with status icons, incorporating progress files)
3. Full current task spec
4. Previous session summaries for this task (if any)
5. Completed prerequisite summaries
6. Agent instructions including progress file path

Then:
- Write the assembled context to `.etch/context/<plan>--task-<N.M>--<session>.md`
- Create empty session progress file with next session number
- Print confirmation with:
  - Token estimate (chars / 3.5, which better reflects Claude's tokenizer on code-heavy content)
  - Session progress file path
  - Ready-to-run command: `claude --append-system-prompt-file .etch/context/<file>`
- Warning if estimate > 80K tokens

**Acceptance Criteria:**
- [x] Context follows the template spec
- [x] Bare `etch context` auto-selects next pending task by dependency order
- [x] Task ID resolution handles all formats
- [x] Previous sessions included when they exist
- [x] Prerequisite summaries included for completed dependencies
- [x] Creates session progress file with correct template
- [x] Writes context prompt file to `.etch/context/`
- [x] Prints ready-to-run `cat .etch/context/<file> | claude` command
- [x] Token estimate printed
- [x] Plan picker when ambiguous
- [x] Warning on large context
- [x] Tests in `internal/context/context_test.go`: template output matches spec, auto-select next task logic, task ID resolution, previous session inclusion, prerequisite summaries, context file written correctly, token estimate calculation
- [x] `go test ./internal/context/...` passes

### Task 3.2: Status command [completed]
**Complexity:** medium
**Files:** internal/status/status.go, cmd/status.go
**Depends on:** Task 1.3b, Task 1.4, Task 1.5

**Known limitation:** If two agents finish tasks simultaneously and `etch status` runs while another instance is also running, the targeted plan file update could conflict. The targeted update approach (line-level edits to different task headings) minimizes this risk, but there is no file locking. For v1 this is acceptable — concurrent `etch status` runs are unlikely in practice.

Implement `etch status`.

Flow:
1. Read all plans in `.etch/plans/`
2. For each plan, read progress files
3. Determine current task statuses from latest sessions
4. Update plan files (status tags + checkboxes) if progress files show changes
5. Display summary

Display:
```
📋 Auth System Rebuild
   ✓ Feature 1: JWT Token Management [3/3 tasks]
   ▶ Feature 2: Login Endpoints [1/3 tasks]
     ✓ 2.1: Registration endpoint
     ▶ 2.2: Login endpoint (2 sessions, last: partial)
     ○ 2.3: Password validation
   ○ Feature 3: Password Reset [0/2 tasks]

📋 API Refactor
   ○ Feature 1: GraphQL Migration [0/4 tasks]
```

Variations:
- `etch status` → all plans, summary
- `etch status <plan>` → single plan, detailed (criteria + last session notes)
- `etch status --json` → machine-readable

**Acceptance Criteria:**
- [x] Reads all plans and progress files
- [x] Updates plan files with current status from progress
- [x] Status icons (✓ ▶ ○ ✗ ⊘)
- [x] Shows session count and last outcome for in-progress tasks
- [x] Detailed view for single plan
- [x] `--json` output
- [x] Handles: no plans, no progress, orphaned progress files
- [x] Only touches status tags and checkboxes (no other plan content modified)
- [x] Tests in `internal/status/status_test.go`: status reconciliation from progress files, plan file update (status tags + checkboxes only), JSON output, edge cases (no plans, no progress, orphaned files). Use fixture plans and progress files.
- [x] `go test ./internal/status/...` passes

### Task 3.3: List and utility commands [completed]
**Complexity:** small
**Files:** cmd/list.go, cmd/open.go
**Depends on:** Task 1.3b

- `etch list` → all plans with title, task counts, completion %
- `etch open <plan>` → opens in `$EDITOR` (fallback: vi)
- `etch delete <plan>` → removes plan + matching progress files (confirmation required)

**Acceptance Criteria:**
- [x] List shows summary for all plans
- [x] Open launches editor
- [x] Delete requires confirmation, removes plan + progress files
- [x] Missing plan handled gracefully
- [x] Tests: list output with multiple plans, delete removes plan + progress files, missing plan error
- [x] `go test ./cmd/...` passes

---

## Feature 4: Interactive TUI Review Mode

### Overview
Bubbletea-powered TUI for scrolling through plans, leaving 💬 comments, and triggering AI refinement. This is "PR review for implementation plans."

### Task 4.1: TUI scaffold and plan rendering [completed]
**Complexity:** medium
**Files:** internal/tui/model.go, internal/tui/view.go, internal/tui/keys.go
**Depends on:** Task 1.3b

Bubbletea app that renders a plan as a scrollable, color-coded document.

Layout (Lipgloss styling):
- Top bar: plan title, task counts, key hints (dim)
- Main area: scrollable plan content
- Bottom bar: current position (Feature X / Task Y), mode indicator

Rendering with Lipgloss:
- Task headers colored by status (green/yellow/dim/red)
- Acceptance criteria: green ✓ or dim ○
- `> 💬` comments: amber/yellow background, visually distinct
- `---` as styled horizontal rule
- Bold/code rendered appropriately

Navigation (Bubbletea key handling):
- `j`/`k` or arrows: scroll
- `d`/`u`: half-page scroll
- `gg`/`G`: top/bottom
- `n`/`p`: next/previous task
- `f`/`F`: next/previous feature
- `/`: search mode (highlight matches, `n` for next match)
- `q`: quit

**Acceptance Criteria:**
- [x] `etch review <plan>` opens full-screen TUI
- [x] Plan rendered with status coloring
- [x] 💬 comments visually highlighted
- [x] All navigation keys work
- [x] Position indicator in bottom bar
- [x] Clean exit on `q` (terminal restored)
- [x] Handles long plans (smooth scrolling)

### Task 4.2: Comment mode [completed]
**Complexity:** medium
**Files:** internal/tui/comments.go, internal/tui/input.go
**Depends on:** Task 4.1

Leave 💬 comments from the TUI.

Workflow:
1. Navigate to a task/feature heading
2. `c` → text input appears (Bubbletea text input bubble)
3. Type comment, Enter to submit
4. Inserted as `> 💬 <text>` below the heading in the plan file
5. TUI re-renders with new comment highlighted

Multi-line: `C` opens `$EDITOR` with temp file. On save+close, content becomes the comment.

Delete: `x` on a 💬 line → confirmation → removed from plan file.

**Acceptance Criteria:**
- [x] `c` opens text input at current section
- [x] Comment saved to plan file as `> 💬`
- [x] `C` opens $EDITOR for multi-line
- [x] New comments appear immediately
- [x] `x` deletes comment with confirmation
- [x] File saved after each comment operation

### Task 4.3: AI refinement flow in TUI [completed]
**Complexity:** medium
**Files:** internal/tui/review.go, internal/tui/diff.go
**Depends on:** Task 4.2, Task 2.3

The apply flow within the TUI.

1. `a` → "Send N comments for refinement? (y/n)"
2. Confirm → loading spinner (Bubbletea spinner bubble)
3. Response → switch to diff view (red/green coloring)
4. `y` to accept, `n` to reject
5. Accept → plan updated, TUI refreshes
6. Reject → return to review mode

Uses the refinement logic from Task 2.3 (backup, API call, parse, diff).

**Acceptance Criteria:**
- [x] `a` triggers refinement with confirmation
- [x] Loading spinner while waiting
- [x] Colored diff view
- [x] Accept/reject flow
- [x] Plan backup happens before changes
- [x] API errors shown gracefully
- [x] "No comments to send" if no 💬 found

---

## Feature 5: Developer Experience & Polish

### Overview
Error handling, help text, first-run experience, documentation for open source launch.

### Task 5.1: Error handling [completed]
**Complexity:** small
**Files:** internal/errors/errors.go, all packages
**Depends on:** Task 1.1

Consistent errors across all commands:
- Custom error types with categories (ConfigError, APIError, ParseError, etc.)
- Every error includes what went wrong + what to do next
- Colored output: red errors, yellow warnings, dim hints (use Lipgloss)
- `--verbose` global flag for debug output
- No panics reach the user

**Acceptance Criteria:**
- [x] Error types defined for all categories
- [x] Actionable messages on every error
- [x] Colored output
- [x] `--verbose` for debug info
- [x] No raw panics

### Task 5.2: First-run polish [completed]
**Complexity:** small
**Files:** cmd/init.go
**Depends on:** Task 5.1

Polish `etch init` output with clear quickstart messaging:

`etch init` prints:
```
✓ Etch initialized!

Next steps:
  etch plan "describe your feature"    Generate an implementation plan
  etch review <plan>                   Review and refine with AI
  etch context <task>                  Generate context prompt file for a task
  etch status                          Check progress across all plans
```

**Acceptance Criteria:**
- [x] `etch init` prints clear quickstart

**Note:** README, CONTRIBUTING.md, and LICENSE are documentation tasks that can be written anytime and don't block shipping. They are intentionally excluded from this plan's critical path.

---

## Go Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/urfave/cli/v2` | CLI framework (lightweight, no codegen) |
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/lipgloss` | TUI styling |
| `github.com/charmbracelet/bubbles` | TUI components (text input, spinner, viewport) |
| `github.com/BurntSushi/toml` | Config file parsing |
| `github.com/muesli/termenv` | Terminal color detection |

**Removed dependencies:**
- `goldmark` — plan format is a strict markdown subset; a line-based state machine is simpler and dependency-free
- `golang.design/x/clipboard` — replaced with prompt file approach; no clipboard dependency needed at all
- `cobra` — replaced with `urfave/cli/v2`, sufficient for ~8 subcommands without the weight

---

## Build Order

**Phase 1: Foundation (get to "I can generate plans")**
1. Task 1.1 — Project scaffold ✓
2. Task 1.2 — Data models ✓
3. Task 1.6 — Config ✓
4. Task 1.3 — Parser structure ✓
5. Task 1.3b — Parser metadata extraction ✓
6. Task 1.4 — Serializer ✓
7. Task 2.1 — API client *(can parallelize with 1.5 — depends only on 1.6, already completed)*
8. Task 2.2 — Plan generation

**→ Checkpoint: `etch init` + `etch plan` work. You can generate real plans.**

**Phase 2: Core loop (get to "I'm using this daily")**
9. Task 1.5 — Progress file read/write *(can parallelize with 2.1)*
10. Task 3.1 — Context prompt file generation
11. Task 3.2 — Status with plan sync
12. Task 3.3 — List/open/delete

**→ Checkpoint: Full loop works. Plan → context → agent → status → repeat. Start dogfooding.**

**Phase 3: Review experience**
13. Task 4.1 — TUI scaffold + rendering
14. Task 4.2 — Comment mode
15. Task 2.3 — Refinement logic *(can parallelize with 4.2 — independent dependency chain)*
16. Task 4.3 — AI refinement in TUI

**→ Checkpoint: Full review loop in TUI. Comments → AI refines → diff → accept.**

**Phase 4: Polish & ship**
17. Task 2.4 — Replan command
18. Task 5.1 — Error handling pass
19. Task 5.2 — First-run polish

**→ Ship it. Open source. Post on HN.**

**Parallelization opportunities:** Tasks 2.1 and 1.5 are both unblocked now and can run in parallel. Task 2.3 can parallelize with 4.2. This compresses the critical path significantly if using multiple agents.