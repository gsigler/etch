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
  ✓ Task 2.2: Plan generation command (completed)
  ○ Task 2.3: Plan refinement from review comments (in_progress — this is your task)
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

## Your Task: Task 2.3 — Plan refinement from review comments
**Complexity:** medium
**Files in Scope:** internal/generator/refine.go, internal/generator/prompts.go
**Depends on:** Task 2.2 (completed)

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

### Acceptance Criteria
- [ ] Extracts 💬 comments from plan
- [ ] Backs up plan before making changes
- [ ] Sends plan + comments to Claude API
- [ ] Validates response parses correctly
- [ ] Shows colored terminal diff
- [ ] Confirmation prompt before applying
- [ ] Addressed comments removed, unaddressed preserved
- [ ] "No comments found" message if plan has no 💬 comments
- [ ] Tests: backup creation, comment extraction, "no comments" path, diff generation. Use mock API client.
- [ ] `go test ./internal/generator/...` passes

### Previous Sessions
None — this is session 001.

### Completed Prerequisites

**Task 2.2 (Plan generation command):**
Created `internal/generator/generator.go` — main generator logic: project context gathering (file tree, config files, CLAUDE.md, existing plan titles), slug generation with collision handling, plan generation flow with streaming, markdown extraction, validation via parser, file writing, Created `internal/generator/prompts.go` — system prompt with plan format specification, complexity guide injection, user message construction with project context, Modified `cmd/plan.go` — full `etch plan "description"` command: loads config, resolves API key, generates slug, handles slug collisions with interactive prompt (create/replan/quit), streams tokens to terminal, writes plan file, prints summary, Created `internal/generator/generator_test.go` — 22 tests covering all acceptance criteria. - `APIClient` interface allows mocking the API client in tests — only requires `SendStream` method
- `extractMarkdown` handles three cases: fenced markdown blocks, preamble before `# Plan:`, and raw plan output
- File tree uses `os.ReadDir` recursion with depth limit (3) and line limit (2000), skipping excluded dirs and hidden files
- Slug collision prompts offer (c)reate with `-2` suffix, (r)eplan redirect, or (q)uit
- `Result` struct holds plan, filepath, slug, and markdown separately so command can set filepath after writing
- Generator does NOT write the file — `WritePlan` is separate so the command controls when/where to write

## Session Progress File

Update your progress file as you work:
`.etch/progress/build-etch--task-2.3--001.md`

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
