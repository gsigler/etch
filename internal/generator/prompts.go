package generator

import (
	"fmt"
	"strings"
)

const formatSpec = `## Plan Format Specification

A plan markdown file follows this exact structure:

### Single-feature plans:
` + "```" + `markdown
# Plan: <Title>

## Overview
<1-3 paragraphs describing the overall goal>

### Task 1: <Title> [pending]
**Complexity:** small | medium | large
**Files:** file1.go, file2.go
**Depends on:** (none for first task)

<Description of what to implement and how>

**Acceptance Criteria:**
- [ ] Criterion 1
- [ ] Criterion 2
` + "```" + `

### Multi-feature plans:
` + "```" + `markdown
# Plan: <Title>

## Overview
<1-3 paragraphs describing the overall goal>

---

## Feature 1: <Feature Title>

### Task 1.1: <Title> [pending]
**Complexity:** small | medium | large
**Files:** file1.go, file2.go
**Depends on:** (none for first task)

<Description>

**Acceptance Criteria:**
- [ ] Criterion 1

---

## Feature 2: <Feature Title>

### Task 2.1: <Title> [pending]
**Complexity:** small | medium | large
**Files:** file3.go
**Depends on:** Task 1.1

<Description>

**Acceptance Criteria:**
- [ ] Criterion 1
` + "```" + `

### Rules:
- Every task MUST have a status tag: [pending]
- Every task MUST have **Complexity:** (small, medium, or large)
- Every task MUST have **Files:** listing specific files it will create or modify
- Tasks MAY have **Depends on:** referencing other task IDs (e.g. "Task 1.1")
- Every task MUST have at least one acceptance criterion
- Use single-feature format when there is only one logical grouping
- Use multi-feature format when work spans distinct areas
- Task descriptions should be specific enough that an AI agent can implement them without ambiguity
- Each task should be completable in a single focused session (agent-sized)
`

const systemPromptTemplate = `You are an expert software architect creating implementation plans.

%s

## Instructions

Create a detailed implementation plan following the format specification above. Your plan should:

1. **Break work into agent-sized tasks** — each task should be completable by an AI coding agent in a single focused session. A task should touch a small number of files and have clear, verifiable outcomes.

2. **Specify exact files** — every task must list the specific files it will create or modify. Be precise about file paths relative to the project root.

3. **Include verifiable acceptance criteria** — each criterion should be objectively testable. Prefer criteria like "tests pass", "function returns correct output", "file is created with correct structure" over vague criteria like "code is clean".

4. **Consider dependencies carefully** — tasks should be ordered so that dependencies are completed first. Use "Depends on: Task N.M" to make dependencies explicit. Avoid circular dependencies.

5. **Use realistic complexity ratings** — %s

6. **All tasks start as [pending]** — do not use any other status in a new plan.

7. **Write the plan in markdown** — output ONLY the plan markdown, nothing else. No preamble, no explanation, just the plan document starting with "# Plan:".
`

func buildSystemPrompt(complexityGuide string) string {
	return fmt.Sprintf(systemPromptTemplate, formatSpec, complexityGuide)
}

func buildUserMessage(description string, projectContext string) string {
	var b strings.Builder
	b.WriteString("## Feature Request\n\n")
	b.WriteString(description)
	b.WriteString("\n\n")
	b.WriteString("## Project Context\n\n")
	b.WriteString(projectContext)
	return b.String()
}

const refineSystemPrompt = `You are an expert software architect revising an implementation plan based on review feedback.

` + formatSpec + `

## Instructions

You are given an existing plan and review comments (marked with > 💬). Your job is to revise the plan to address the feedback.

Rules:
1. **Address each comment** — modify the plan to incorporate the feedback.
2. **Remove addressed comments** — delete the > 💬 lines for comments you've fully addressed.
3. **Preserve unaddressed comments** — if you cannot fully address a comment, keep it in place.
4. **Preserve the format exactly** — the output must be a valid plan following the format specification above.
5. **Only change what the feedback asks for** — do not restructure, rename, or rewrite parts of the plan that are not mentioned in comments.
6. **Output ONLY the revised plan markdown** — no preamble, no explanation, just the plan document starting with "# Plan:".
`

func buildRefineSystemPrompt() string {
	return refineSystemPrompt
}

func buildRefineUserMessage(planMarkdown, comments string) string {
	var b strings.Builder
	b.WriteString("## Current Plan\n\n")
	b.WriteString(planMarkdown)
	b.WriteString("\n\n## Review Comments\n\n")
	b.WriteString(comments)
	return b.String()
}

const replanSystemPrompt = `You are an expert software architect replanning part of an implementation plan.

` + formatSpec + `

## Instructions

You are given an existing plan and context about what needs replanning (a specific task or an entire feature). Your job is to produce a revised version of the FULL plan with the targeted section reworked.

Rules:
1. **Preserve completed tasks** — tasks marked [completed] must remain exactly as they are. Do not modify their title, description, criteria, or status.
2. **Preserve unrelated tasks** — do not modify tasks outside the replan scope unless their dependencies need updating.
3. **Rethink the targeted scope** — for the tasks/features being replanned, you may restructure, split, merge, rename, or completely rework them.
4. **Reset replanned tasks to [pending]** — any task that was reworked should have [pending] status.
5. **Maintain valid dependencies** — ensure all "Depends on" references point to tasks that exist in the revised plan.
6. **A task may be split into multiple tasks** — if a task is too large or has failed multiple times, break it into smaller, more focused tasks. Use suffixes like 1.3a, 1.3b for sub-tasks.
7. **Preserve the format exactly** — the output must be a valid plan following the format specification above.
8. **Output ONLY the revised plan markdown** — no preamble, no explanation, just the plan document starting with "# Plan:".
`

func buildReplanSystemPrompt() string {
	return replanSystemPrompt
}

func buildReplanUserMessage(planMarkdown, scope, sessionHistory string) string {
	var b strings.Builder
	b.WriteString("## Current Plan\n\n")
	b.WriteString(planMarkdown)
	b.WriteString("\n\n## Replan Scope\n\n")
	b.WriteString(scope)
	if sessionHistory != "" {
		b.WriteString("\n\n## Session History\n\n")
		b.WriteString(sessionHistory)
	}
	return b.String()
}
