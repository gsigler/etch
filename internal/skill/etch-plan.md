---
name: etch-plan
description: Create an etch implementation plan for a feature or task. Use when the user wants to plan work, break down a feature, or create a structured implementation plan with tasks and acceptance criteria.
argument-hint: <feature description>
---

Create an implementation plan using the etch plan format. The plan will be saved as a markdown file in `.etch/plans/`.

## Step 1: Gather context

Before writing the plan, understand the project:

1. Read `CLAUDE.md` and key config files (`go.mod`, `package.json`, `Cargo.toml`, etc.)
2. Explore the file tree to understand project structure (limit depth to 3 levels)
3. Check for existing plans in `.etch/plans/` to avoid overlap

## Step 2: Write the plan

Create the plan markdown file following this exact format:

### Single-feature plans (one logical grouping):

```markdown
# Plan: <Title>

## Overview
<1-3 paragraphs describing the overall goal>

### Task 1: <Title> [pending]
**Complexity:** small | medium | large
**Files:** src/auth.ts, src/auth.test.ts
**Depends on:** (none for first task)

<Description of what to implement and how>

**Acceptance Criteria:**
- [ ] Criterion 1
- [ ] Criterion 2
```

### Multi-feature plans (work spans distinct areas):

```markdown
# Plan: <Title>

## Overview
<1-3 paragraphs describing the overall goal>

---

## Feature 1: <Feature Title>

### Task 1.1: <Title> [pending]
**Complexity:** small | medium | large
**Files:** src/models/user.py, src/routes/auth.py
**Depends on:** (none for first task)

<Description>

**Acceptance Criteria:**
- [ ] Criterion 1

---

## Feature 2: <Feature Title>

### Task 2.1: <Title> [pending]
**Complexity:** small | medium | large
**Files:** tests/test_auth.py
**Depends on:** Task 1.1

<Description>

**Acceptance Criteria:**
- [ ] Criterion 1
```

## Plan format rules

### Sizing guidance
- Plans should have enough tasks to fully implement the feature. A typical plan has 5-15 tasks across 2-4 features. Don't under-scope — it's better to have more granular tasks than to combine too much into one.
- Each task should be completable in a single focused session (agent-sized)

### Task requirements
- Every task MUST have a status tag: `[pending]`
- Every task MUST have `**Complexity:**` (small, medium, or large)
- Every task MUST have `**Files:**` listing specific files it will create or modify
- Tasks MAY have `**Depends on:**` referencing other task IDs (e.g. "Task 1.1")
- Complexity ratings: **small** = isolated change in 1-2 files, **medium** = multiple files or moderate logic, **large** = cross-cutting or architecturally significant
- Task descriptions should be specific enough that an AI agent can implement them without ambiguity

### Acceptance criteria
- Each task should have 3-5 acceptance criteria
- Include at least one verification criterion per task (e.g., "Tests pass", "Feature works in the UI", "No regressions in existing tests")

### Validation tasks
- Every feature MUST end with a validation task that verifies the implementation works (e.g., writing tests, running the app, checking edge cases)
- Use single-feature format when there is only one logical grouping
- Use multi-feature format when work spans distinct areas

### Validation task example

```markdown
### Task 1.3: Validate authentication flow [pending]
**Complexity:** small
**Files:** tests/test_auth.py
**Depends on:** Task 1.2

Run the full auth test suite and manually verify the login flow works end-to-end.

**Acceptance Criteria:**
- [ ] All existing auth tests pass
- [ ] New tests cover the added login endpoint
- [ ] Manual test: can log in and receive a valid token
- [ ] No regressions in other test suites
```

## Step 3: Save the plan

1. **Use the slug provided via `--slug <slug>`** in the arguments. Do NOT generate your own slug — the CLI has already computed it. If no `--slug` is provided, generate one from the description: lowercase, hyphens for spaces, strip non-alphanumeric, max 50 chars.
2. Create the `.etch/plans/` directory if it doesn't exist
3. Write the file to `.etch/plans/<slug>.md`
4. Report the summary: title, feature count, task count, and file path

## Feature description

$ARGUMENTS
