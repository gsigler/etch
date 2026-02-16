# Etch Context — Implementation Task

You are working on a task as part of an implementation plan managed by Etch.

## Plan: Build Etch – AI Implementation Planning CLI
Etch is a Go CLI tool that helps developers create, review, and execute AI-generated implementation plans. The plan markdown file is the center of everything — it serves as the spec, the context source, and the shared document that humans review and approve. Progress tracking lives in separate per-session files so multiple agents can work simultaneously without conflicts.

## Current Plan State
Feature 1: Core Data Layer & Plan Parser
  ✓ Task 1.1: Project scaffold and CLI skeleton (completed)
  ✓ Task 1.2: Data models (completed)
  ✓ Task 1.3: Plan markdown parser — structure (completed)
  ✓ Task 1.3b: Plan parser — task metadata extraction (completed)
  ✓ Task 1.4: Plan markdown serializer (completed)
  ✓ Task 1.5: Progress file reader/writer (completed)
  ✓ Task 1.6: Config management (completed)
Feature 2: Plan Generation & AI Integration
  ✓ Task 2.1: API client (completed)
  ○ Task 2.2: Plan generation command (in_progress — this is your task)
  ○ Task 2.3: Plan refinement from review comments (pending, depends on 2.2)
  ○ Task 2.4: Replan command (pending, depends on 2.3, 1.5)
Feature 3: Context Generation & Status
  ✓ Task 3.1: Context prompt assembly (completed)
  ✓ Task 3.2: Status command (completed)
  ✓ Task 3.3: List and utility commands (completed)
Feature 4: Interactive TUI Review Mode
  ✓ Task 4.1: TUI scaffold and plan rendering (completed)
  ✓ Task 4.2: Comment mode (completed)
  ✓ Task 4.3: AI refinement flow in TUI (completed)
Feature 5: Developer Experience & Polish
  ○ Task 5.1: Error handling (pending, depends on 1.1)
  ○ Task 5.2: First-run polish (pending, depends on 5.1)

## Your Task: Task 2.2 — Plan generation command
**Complexity:** medium
**Files in Scope:** internal/generator/generator.go, internal/generator/prompts.go, cmd/plan.go
**Depends on:** Task 1.3b (completed), Task 1.4 (completed), Task 2.1 (completed)

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

### Acceptance Criteria
- [ ] Generates valid plan from description
- [ ] Project context gathered and included
- [ ] Streaming shows real-time generation
- [ ] Output parses successfully
- [ ] Written to correctly named file
- [ ] Summary printed (feature count, task count)
- [ ] Handles: empty project, very large tree (truncate at ~2000 lines), duplicate slugs
- [ ] Tests: project context gathering (file tree, config detection), slug generation and collision handling, prompt construction (verify format spec included). Use mock API client for generation tests.
- [ ] `go test ./internal/generator/...` passes

### Previous Sessions
None — this is session 001.

### Completed Prerequisites

**Task 1.3b (Plan parser — task metadata extraction):**
`internal/models/models.go` — Added `Suffix string` field to `Task`, updated `FullID()` to append suffix, `internal/models/models_test.go` — Added suffix test cases to `TestTaskFullID` and `TestPlanTaskByID`, `internal/parser/parser.go` — Fixed task heading regex to support letter suffixes (`[a-z]`), added metadata extraction (complexity, files, depends_on, criteria, comments) with line-level pattern matching in `stateTask`, `internal/parser/parser_test.go` — Updated `TestParse_TaskDescriptionCapture` → `TestParse_TaskMetadataExtraction`, updated spec file test to expect 7 tasks in Feature 1, added tests: `TestParse_LetterSuffixTaskID`, `TestParse_ReviewComments`, `TestParse_TaskNoMetadata`, `TestParse_TaskNoCriteria`, `TestParse_FilesInScope`. - Metadata lines (Complexity, Files, Depends on) are extracted and removed from Description; only non-metadata text goes into Description
- `**Files:**` and `**Files in Scope:**` both supported via regex alternation
- Multi-line `> 💬` comments: consecutive `>` lines after a `> 💬` line are joined with newlines into a single comment string
- `inComment` flag tracks multi-line comment state, reset on flush or when a non-`>` line is encountered

**Task 1.4 (Plan markdown serializer):**
Created `internal/serializer/serializer.go` — full serializer + targeted update functions, Created `internal/serializer/serializer_test.go` — 14 tests covering all acceptance criteria, Modified `internal/parser/parser.go` — added `criteriaHeadingRe` to skip `**Acceptance Criteria:**` heading lines (needed for clean round-tripping). - Single-feature detection: `len(plan.Features) == 1` → omit `## Feature N:` heading, use `### Task N:` format
- Parser required a minor fix: `**Acceptance Criteria:**` heading was being accumulated into task description. Added `criteriaHeadingRe` pattern to skip it. This doesn't change existing parser test behavior (all 17 parser tests still pass).
- Targeted updates (`UpdateTaskStatus`, `UpdateCriterion`) operate on raw file lines with regex, never re-serializing, so they preserve all formatting exactly.
- `UpdateCriterion` scopes its search to the target task section (between `### Task` headings) to avoid accidentally flipping identically-named criteria in other tasks.

**Task 2.1 (API client):**
Created `internal/api/client.go` — Anthropic Messages API client, Created `internal/api/client_test.go` — 9 tests, all passing. - `Client` struct has public fields: APIKey, Model, BaseURL, MaxTokens, HTTPClient, InitialBackoff
- `BaseURL` and `InitialBackoff` are overridable for testing (BaseURL points to httptest server, InitialBackoff speeds up retry tests)
- `Send()` for non-streaming, `SendStream()` for streaming with a `StreamCallback func(text string)` — both return full accumulated text
- SSE parsing handles content_block_delta events with text_delta type, skips all other event types
- `APIError` type with StatusCode and Message for structured error handling
- 401 returns immediately with helpful message about checking API key
- 429 retries with exponential backoff (configurable initial, doubles each time, max 3 attempts)
- 500+ returns immediately with response body as error message
- Network errors wrapped with "sending request:" prefix
- Default max_tokens: 8192, timeout: 5 minutes

## Session Progress File

Update your progress file as you work:
`.etch/progress/build-etch--task-2.2--001.md`

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
