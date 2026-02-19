package generator

import (
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
- Each task should have 3-5 acceptance criteria
- Include at least one verification criterion per task (e.g., "Tests pass", "No regressions in existing tests")
- Every feature MUST end with a validation task that verifies the implementation works (e.g., writing tests, running the app, checking edge cases)
- Use single-feature format when there is only one logical grouping
- Use multi-feature format when work spans distinct areas
- Task descriptions should be specific enough that an AI agent can implement them without ambiguity
- Each task should be completable in a single focused session (agent-sized)
`

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
