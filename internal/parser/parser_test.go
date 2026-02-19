package parser

import (
	"strings"
	"testing"

	"github.com/gsigler/etch/internal/models"
)

func TestParse_MultiFeaturePlan(t *testing.T) {
	input := `# Plan: Auth System

## Overview

Build authentication for the API.

---

## Feature 1: Token Management

### Overview
JWT tokens and refresh flow.

### Task 1.1: Create token service [completed]
Build the token signing and verification service.

### Task 1.2: Token refresh endpoint [in_progress]
Implement POST /auth/refresh.

---

## Feature 2: Login Endpoints

### Task 2.1: Registration [pending]
Implement POST /auth/register.

### Task 2.2: Login [blocked]
Implement POST /auth/login.
`

	plan, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Title != "Auth System" {
		t.Errorf("title = %q, want %q", plan.Title, "Auth System")
	}
	if plan.Overview != "Build authentication for the API." {
		t.Errorf("overview = %q, want %q", plan.Overview, "Build authentication for the API.")
	}
	if len(plan.Features) != 2 {
		t.Fatalf("feature count = %d, want 2", len(plan.Features))
	}

	// Feature 1
	f1 := plan.Features[0]
	if f1.Number != 1 {
		t.Errorf("feature 1 number = %d, want 1", f1.Number)
	}
	if f1.Title != "Token Management" {
		t.Errorf("feature 1 title = %q, want %q", f1.Title, "Token Management")
	}
	if f1.Overview != "JWT tokens and refresh flow." {
		t.Errorf("feature 1 overview = %q, want %q", f1.Overview, "JWT tokens and refresh flow.")
	}
	if len(f1.Tasks) != 2 {
		t.Fatalf("feature 1 task count = %d, want 2", len(f1.Tasks))
	}

	// Task 1.1
	assertTask(t, f1.Tasks[0], 1, 1, "Create token service", models.StatusCompleted)
	// Task 1.2
	assertTask(t, f1.Tasks[1], 1, 2, "Token refresh endpoint", models.StatusInProgress)

	// Feature 2
	f2 := plan.Features[1]
	if f2.Number != 2 {
		t.Errorf("feature 2 number = %d, want 2", f2.Number)
	}
	if f2.Title != "Login Endpoints" {
		t.Errorf("feature 2 title = %q, want %q", f2.Title, "Login Endpoints")
	}
	if len(f2.Tasks) != 2 {
		t.Fatalf("feature 2 task count = %d, want 2", len(f2.Tasks))
	}

	assertTask(t, f2.Tasks[0], 2, 1, "Registration", models.StatusPending)
	assertTask(t, f2.Tasks[1], 2, 2, "Login", models.StatusBlocked)
}

func TestParse_SingleFeaturePlan(t *testing.T) {
	input := `# Plan: Add Rate Limiting

## Overview
Add rate limiting to all API endpoints.

### Task 1: Design rate limiter [pending]
Choose algorithm and storage.

### Task 2: Implement middleware [in_progress]
Wire up the rate limiter as HTTP middleware.

### Task 3: Add tests [pending]
Integration tests for rate limiting.
`

	plan, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Title != "Add Rate Limiting" {
		t.Errorf("title = %q, want %q", plan.Title, "Add Rate Limiting")
	}
	if len(plan.Features) != 1 {
		t.Fatalf("feature count = %d, want 1", len(plan.Features))
	}

	f := plan.Features[0]
	if f.Number != 1 {
		t.Errorf("implicit feature number = %d, want 1", f.Number)
	}
	if f.Title != "Add Rate Limiting" {
		t.Errorf("implicit feature title = %q, want %q", f.Title, "Add Rate Limiting")
	}
	if len(f.Tasks) != 3 {
		t.Fatalf("task count = %d, want 3", len(f.Tasks))
	}

	assertTask(t, f.Tasks[0], 1, 1, "Design rate limiter", models.StatusPending)
	assertTask(t, f.Tasks[1], 1, 2, "Implement middleware", models.StatusInProgress)
	assertTask(t, f.Tasks[2], 1, 3, "Add tests", models.StatusPending)
}

func TestParse_NoPlanHeading(t *testing.T) {
	input := `## Feature 1: Something

### Task 1.1: Do stuff [pending]
Some description.
`

	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for missing # Plan: heading, got nil")
	}
	if !strings.Contains(err.Error(), "no '# Plan:' heading") {
		t.Errorf("error = %q, want it to mention missing Plan heading", err.Error())
	}
}

func TestParse_EmptyInput(t *testing.T) {
	_, err := Parse(strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
}

func TestParse_EmptyFeature(t *testing.T) {
	input := `# Plan: Test Plan

## Feature 1: Has Tasks

### Task 1.1: Do something [pending]
Description here.

## Feature 2: Empty Feature

## Feature 3: Also Has Tasks

### Task 3.1: Another thing [completed]
More description.
`

	plan, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(plan.Features) != 3 {
		t.Fatalf("feature count = %d, want 3", len(plan.Features))
	}

	if len(plan.Features[0].Tasks) != 1 {
		t.Errorf("feature 1 task count = %d, want 1", len(plan.Features[0].Tasks))
	}
	if len(plan.Features[1].Tasks) != 0 {
		t.Errorf("feature 2 task count = %d, want 0 (empty feature)", len(plan.Features[1].Tasks))
	}
	if len(plan.Features[2].Tasks) != 1 {
		t.Errorf("feature 3 task count = %d, want 1", len(plan.Features[2].Tasks))
	}
}

func TestParse_MissingStatusTag(t *testing.T) {
	input := `# Plan: No Status Tags

## Feature 1: Stuff

### Task 1.1: No status tag
Some description.

### Task 1.2: Has status [completed]
Other description.
`

	plan, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Features[0].Tasks[0].Status != models.StatusPending {
		t.Errorf("task without status = %q, want %q", plan.Features[0].Tasks[0].Status, models.StatusPending)
	}
	if plan.Features[0].Tasks[1].Status != models.StatusCompleted {
		t.Errorf("task with status = %q, want %q", plan.Features[0].Tasks[1].Status, models.StatusCompleted)
	}
}

func TestParse_MissingOverview(t *testing.T) {
	input := `# Plan: No Overview

## Feature 1: Stuff

### Task 1.1: Do it [pending]
Description.
`

	plan, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Overview != "" {
		t.Errorf("overview = %q, want empty", plan.Overview)
	}
}

func TestParse_TaskMetadataExtraction(t *testing.T) {
	input := `# Plan: Desc Test

## Feature 1: Stuff

### Task 1.1: A task [pending]
**Complexity:** medium
**Files:** foo.go, bar.go
**Depends on:** Task 1.0

First paragraph of description.

Second paragraph with more detail.

- [ ] Criterion one
- [x] Criterion two
`

	plan, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	task := plan.Features[0].Tasks[0]

	// Metadata should be extracted, not in description.
	if task.Complexity != "medium" {
		t.Errorf("complexity = %q, want %q", task.Complexity, "medium")
	}
	if len(task.Files) != 2 || task.Files[0] != "foo.go" || task.Files[1] != "bar.go" {
		t.Errorf("files = %v, want [foo.go bar.go]", task.Files)
	}
	if len(task.DependsOn) != 1 || task.DependsOn[0] != "Task 1.0" {
		t.Errorf("depends_on = %v, want [Task 1.0]", task.DependsOn)
	}

	// Description should contain only the paragraphs.
	if !strings.Contains(task.Description, "First paragraph") {
		t.Error("description should contain first paragraph")
	}
	if !strings.Contains(task.Description, "Second paragraph") {
		t.Error("description should contain second paragraph")
	}
	if strings.Contains(task.Description, "**Complexity:**") {
		t.Error("description should NOT contain complexity metadata line")
	}

	// Acceptance criteria.
	if len(task.Criteria) != 2 {
		t.Fatalf("criteria count = %d, want 2", len(task.Criteria))
	}
	if task.Criteria[0].Description != "Criterion one" || task.Criteria[0].IsMet {
		t.Errorf("criterion 0 = %+v, want {Criterion one, false}", task.Criteria[0])
	}
	if task.Criteria[1].Description != "Criterion two" || !task.Criteria[1].IsMet {
		t.Errorf("criterion 1 = %+v, want {Criterion two, true}", task.Criteria[1])
	}
}

func TestParse_FeatureOverviewWithoutH3(t *testing.T) {
	input := `# Plan: Feature Overview Test

## Feature 1: My Feature
This is the feature overview without a ### Overview heading.

### Task 1.1: Do stuff [pending]
Task description.
`

	plan, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Features[0].Overview != "This is the feature overview without a ### Overview heading." {
		t.Errorf("feature overview = %q", plan.Features[0].Overview)
	}
}

func TestParse_UnknownStatusDefaultsPending(t *testing.T) {
	input := `# Plan: Bad Status

## Feature 1: Stuff

### Task 1.1: Bad status [bananas]
Description.
`

	plan, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Features[0].Tasks[0].Status != models.StatusPending {
		t.Errorf("unknown status = %q, want %q", plan.Features[0].Tasks[0].Status, models.StatusPending)
	}
}

func TestParseFile_SpecFile(t *testing.T) {
	// Parse the actual spec file used by this project.
	plan, err := ParseFile("../../.etch/plans/build-etch.md")
	if err != nil {
		t.Fatalf("failed to parse spec file: %v", err)
	}

	if plan.Title != "Build Etch – AI Implementation Planning CLI" {
		t.Errorf("title = %q", plan.Title)
	}
	if plan.Slug != "build-etch" {
		t.Errorf("slug = %q, want %q", plan.Slug, "build-etch")
	}
	if plan.Overview == "" {
		t.Error("overview should not be empty")
	}

	// Should have 5 features.
	if len(plan.Features) != 5 {
		t.Fatalf("feature count = %d, want 5", len(plan.Features))
	}

	// Verify feature titles and task counts.
	expectedFeatures := []struct {
		number    int
		title     string
		taskCount int
	}{
		{1, "Core Data Layer & Plan Parser", 7}, // 1.1, 1.2, 1.3, 1.3b, 1.4, 1.5, 1.6
		{2, "Plan Generation & AI Integration", 4},
		{3, "Context Generation & Status", 3},
		{4, "Interactive TUI Review Mode", 3},
		{5, "Developer Experience & Polish", 2},
	}

	for i, expected := range expectedFeatures {
		f := plan.Features[i]
		if f.Number != expected.number {
			t.Errorf("feature %d number = %d, want %d", i, f.Number, expected.number)
		}
		if f.Title != expected.title {
			t.Errorf("feature %d title = %q, want %q", i, f.Title, expected.title)
		}
		if len(f.Tasks) != expected.taskCount {
			t.Errorf("feature %d (%s) task count = %d, want %d", i, f.Title, len(f.Tasks), expected.taskCount)
		}
	}

	// Verify some specific task details.
	task11 := plan.TaskByID("1.1")
	if task11 == nil {
		t.Fatal("task 1.1 not found")
	}
	if task11.Title != "Project scaffold and CLI skeleton" {
		t.Errorf("task 1.1 title = %q", task11.Title)
	}
	if task11.Status != models.StatusCompleted {
		t.Errorf("task 1.1 status = %q, want completed", task11.Status)
	}

	task13 := plan.TaskByID("1.3")
	if task13 == nil {
		t.Fatal("task 1.3 not found")
	}
	if task13.Title != "Plan markdown parser — structure" {
		t.Errorf("task 1.3 title = %q", task13.Title)
	}

	// Verify Task 1.3b exists and parses correctly (letter-suffix fix).
	task13b := plan.TaskByID("1.3b")
	if task13b == nil {
		t.Fatal("task 1.3b not found — letter-suffix regex fix failed")
	}
	if task13b.Suffix != "b" {
		t.Errorf("task 1.3b suffix = %q, want %q", task13b.Suffix, "b")
	}
	if task13b.FeatureNumber != 1 || task13b.TaskNumber != 3 {
		t.Errorf("task 1.3b numbers = %d.%d, want 1.3", task13b.FeatureNumber, task13b.TaskNumber)
	}

	// Verify metadata extraction on a task with known metadata.
	if task13b.Complexity != "medium" {
		t.Errorf("task 1.3b complexity = %q, want %q", task13b.Complexity, "medium")
	}
	if len(task13b.Files) < 2 {
		t.Errorf("task 1.3b files = %v, want at least 2 files", task13b.Files)
	}
	if len(task13b.DependsOn) == 0 {
		t.Error("task 1.3b depends_on should not be empty")
	}
	if len(task13b.Criteria) == 0 {
		t.Error("task 1.3b criteria should not be empty")
	}
}

func TestParse_PlanOnlyTitle(t *testing.T) {
	input := `# Plan: Minimal Plan
`
	plan, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Title != "Minimal Plan" {
		t.Errorf("title = %q", plan.Title)
	}
	if len(plan.Features) != 0 {
		t.Errorf("feature count = %d, want 0", len(plan.Features))
	}
}

func TestParse_LetterSuffixTaskID(t *testing.T) {
	input := `# Plan: Suffix Test

## Feature 1: Core

### Task 1.1: First task [completed]
Do the first thing.

### Task 1.1b: Follow-up task [pending]
**Complexity:** small
Follow up on the first thing.

### Task 1.2: Second task [in_progress]
Do the second thing.
`

	plan, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	f := plan.Features[0]
	if len(f.Tasks) != 3 {
		t.Fatalf("task count = %d, want 3", len(f.Tasks))
	}

	// Task 1.1
	if f.Tasks[0].FullID() != "1.1" {
		t.Errorf("task 0 id = %q, want 1.1", f.Tasks[0].FullID())
	}
	if f.Tasks[0].Suffix != "" {
		t.Errorf("task 0 suffix = %q, want empty", f.Tasks[0].Suffix)
	}

	// Task 1.1b
	if f.Tasks[1].FullID() != "1.1b" {
		t.Errorf("task 1 id = %q, want 1.1b", f.Tasks[1].FullID())
	}
	if f.Tasks[1].Suffix != "b" {
		t.Errorf("task 1 suffix = %q, want b", f.Tasks[1].Suffix)
	}
	if f.Tasks[1].Complexity != "small" {
		t.Errorf("task 1.1b complexity = %q, want small", f.Tasks[1].Complexity)
	}

	// Task 1.2
	if f.Tasks[2].FullID() != "1.2" {
		t.Errorf("task 2 id = %q, want 1.2", f.Tasks[2].FullID())
	}
}

func TestParse_ReviewComments(t *testing.T) {
	input := `# Plan: Comment Test

## Feature 1: Core

### Task 1.1: A task [pending]
Description here.

> 💬 This is a single-line comment.

> 💬 This is a multi-line comment
> that spans two lines
> and even three.

Some more description.

> 💬 Another comment.
`

	plan, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	task := plan.Features[0].Tasks[0]
	if len(task.Comments) != 3 {
		t.Fatalf("comment count = %d, want 3", len(task.Comments))
	}

	if task.Comments[0] != "This is a single-line comment." {
		t.Errorf("comment 0 = %q", task.Comments[0])
	}
	expected := "This is a multi-line comment\nthat spans two lines\nand even three."
	if task.Comments[1] != expected {
		t.Errorf("comment 1 = %q, want %q", task.Comments[1], expected)
	}
	if task.Comments[2] != "Another comment." {
		t.Errorf("comment 2 = %q", task.Comments[2])
	}
}

func TestParse_TaskNoMetadata(t *testing.T) {
	input := `# Plan: Minimal Task Test

## Feature 1: Core

### Task 1.1: Bare task [pending]
Just a plain description with no metadata at all.
`

	plan, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	task := plan.Features[0].Tasks[0]
	if task.Complexity != "" {
		t.Errorf("complexity = %q, want empty", task.Complexity)
	}
	if len(task.Files) != 0 {
		t.Errorf("files = %v, want empty", task.Files)
	}
	if len(task.DependsOn) != 0 {
		t.Errorf("depends_on = %v, want empty", task.DependsOn)
	}
	if len(task.Criteria) != 0 {
		t.Errorf("criteria = %v, want empty", task.Criteria)
	}
	if len(task.Comments) != 0 {
		t.Errorf("comments = %v, want empty", task.Comments)
	}
	if !strings.Contains(task.Description, "Just a plain description") {
		t.Errorf("description = %q, should contain plain description", task.Description)
	}
}

func TestParse_TaskNoCriteria(t *testing.T) {
	input := `# Plan: No Criteria

## Feature 1: Core

### Task 1.1: Has metadata no criteria [pending]
**Complexity:** large
**Files:** main.go
**Depends on:** Task 0.1, Task 0.2

Description of the task.
`

	plan, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	task := plan.Features[0].Tasks[0]
	if task.Complexity != "large" {
		t.Errorf("complexity = %q, want large", task.Complexity)
	}
	if len(task.Files) != 1 || task.Files[0] != "main.go" {
		t.Errorf("files = %v, want [main.go]", task.Files)
	}
	if len(task.DependsOn) != 2 {
		t.Fatalf("depends_on count = %d, want 2", len(task.DependsOn))
	}
	if task.DependsOn[0] != "Task 0.1" || task.DependsOn[1] != "Task 0.2" {
		t.Errorf("depends_on = %v", task.DependsOn)
	}
	if len(task.Criteria) != 0 {
		t.Errorf("criteria = %v, want empty", task.Criteria)
	}
}

func TestParse_FilesInScope(t *testing.T) {
	input := `# Plan: Files In Scope Test

## Feature 1: Core

### Task 1.1: A task [pending]
**Files in Scope:** internal/parser/parser.go, internal/parser/parser_test.go
`

	plan, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	task := plan.Features[0].Tasks[0]
	if len(task.Files) != 2 {
		t.Fatalf("files count = %d, want 2", len(task.Files))
	}
	if task.Files[0] != "internal/parser/parser.go" {
		t.Errorf("file 0 = %q", task.Files[0])
	}
}

func TestParse_PriorityPresent(t *testing.T) {
	input := `# Plan: Priority Test
**Priority:** 3

## Overview
A plan with priority.

## Feature 1: Core

### Task 1.1: Do stuff [pending]
Description.
`

	plan, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Priority != 3 {
		t.Errorf("priority = %d, want 3", plan.Priority)
	}
}

func TestParse_PriorityAbsent(t *testing.T) {
	input := `# Plan: No Priority

## Feature 1: Core

### Task 1.1: Do stuff [pending]
Description.
`

	plan, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Priority != 0 {
		t.Errorf("priority = %d, want 0 (default)", plan.Priority)
	}
}

func TestParse_PriorityInsideTaskIgnored(t *testing.T) {
	input := `# Plan: Priority In Task

## Feature 1: Core

### Task 1.1: Do stuff [pending]
**Priority:** 5

Some description.
`

	plan, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Priority != 0 {
		t.Errorf("priority = %d, want 0 (should not parse priority from task section)", plan.Priority)
	}
}

func assertTask(t *testing.T, task models.Task, featureNum, taskNum int, title string, status models.Status) {
	t.Helper()
	if task.FeatureNumber != featureNum {
		t.Errorf("task %d.%d feature number = %d, want %d", featureNum, taskNum, task.FeatureNumber, featureNum)
	}
	if task.TaskNumber != taskNum {
		t.Errorf("task %d.%d task number = %d, want %d", featureNum, taskNum, task.TaskNumber, taskNum)
	}
	if task.Title != title {
		t.Errorf("task %d.%d title = %q, want %q", featureNum, taskNum, task.Title, title)
	}
	if task.Status != status {
		t.Errorf("task %d.%d status = %q, want %q", featureNum, taskNum, task.Status, status)
	}
}

func TestParsePlanStatus(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantTitle  string
		wantStatus models.Status
	}{
		{
			name: "no status tag",
			input: `# Plan: My Plan

### Task 1: Do something [pending]
Description.
`,
			wantTitle:  "My Plan",
			wantStatus: "",
		},
		{
			name: "completed status tag",
			input: `# Plan: My Plan [completed]

### Task 1: Do something [completed]
Description.
`,
			wantTitle:  "My Plan",
			wantStatus: models.StatusCompleted,
		},
		{
			name: "in_progress status tag",
			input: `# Plan: My Plan [in_progress]

### Task 1: Do something [in_progress]
Description.
`,
			wantTitle:  "My Plan",
			wantStatus: models.StatusInProgress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := Parse(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if plan.Title != tt.wantTitle {
				t.Errorf("title = %q, want %q", plan.Title, tt.wantTitle)
			}
			if plan.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", plan.Status, tt.wantStatus)
			}
		})
	}
}
