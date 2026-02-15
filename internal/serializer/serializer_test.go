package serializer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gsigler/etch/internal/models"
	"github.com/gsigler/etch/internal/parser"
)

func TestSerialize_MultiFeaturePlan(t *testing.T) {
	plan := &models.Plan{
		Title:    "Auth System",
		Overview: "Build authentication for the API.",
		Features: []models.Feature{
			{
				Number:   1,
				Title:    "Token Management",
				Overview: "JWT tokens and refresh flow.",
				Tasks: []models.Task{
					{
						FeatureNumber: 1,
						TaskNumber:    1,
						Title:         "Create token service",
						Status:        models.StatusCompleted,
						Complexity:    models.ComplexityMedium,
						Files:         []string{"token.go", "token_test.go"},
						DependsOn:     []string{"none"},
						Description:   "Build the token signing and verification service.",
						Criteria: []models.Criterion{
							{Description: "Tokens can be signed", IsMet: true},
							{Description: "Tokens can be verified", IsMet: false},
						},
					},
					{
						FeatureNumber: 1,
						TaskNumber:    2,
						Title:         "Token refresh endpoint",
						Status:        models.StatusInProgress,
						Description:   "Implement POST /auth/refresh.",
					},
				},
			},
			{
				Number: 2,
				Title:  "Login Endpoints",
				Tasks: []models.Task{
					{
						FeatureNumber: 2,
						TaskNumber:    1,
						Title:         "Registration",
						Status:        models.StatusPending,
					},
				},
			},
		},
	}

	output := Serialize(plan)

	// Verify key structural elements.
	assertContains(t, output, "# Plan: Auth System")
	assertContains(t, output, "## Overview")
	assertContains(t, output, "Build authentication for the API.")
	assertContains(t, output, "## Feature 1: Token Management")
	assertContains(t, output, "### Overview\nJWT tokens and refresh flow.")
	assertContains(t, output, "### Task 1.1: Create token service [completed]")
	assertContains(t, output, "**Complexity:** medium")
	assertContains(t, output, "**Files:** token.go, token_test.go")
	assertContains(t, output, "**Depends on:** none")
	assertContains(t, output, "- [x] Tokens can be signed")
	assertContains(t, output, "- [ ] Tokens can be verified")
	assertContains(t, output, "### Task 1.2: Token refresh endpoint [in_progress]")
	assertContains(t, output, "## Feature 2: Login Endpoints")
	assertContains(t, output, "### Task 2.1: Registration [pending]")
	assertContains(t, output, "---")
}

func TestSerialize_SingleFeaturePlan(t *testing.T) {
	plan := &models.Plan{
		Title:    "Add Rate Limiting",
		Overview: "Add rate limiting to all API endpoints.",
		Features: []models.Feature{
			{
				Number: 1,
				Title:  "Add Rate Limiting",
				Tasks: []models.Task{
					{
						FeatureNumber: 1,
						TaskNumber:    1,
						Title:         "Design rate limiter",
						Status:        models.StatusPending,
						Description:   "Choose algorithm and storage.",
					},
					{
						FeatureNumber: 1,
						TaskNumber:    2,
						Title:         "Implement middleware",
						Status:        models.StatusInProgress,
					},
				},
			},
		},
	}

	output := Serialize(plan)

	// Single-feature: no ## Feature heading, uses ### Task N: format.
	assertNotContains(t, output, "## Feature")
	assertContains(t, output, "### Task 1: Design rate limiter [pending]")
	assertContains(t, output, "### Task 2: Implement middleware [in_progress]")
}

func TestSerialize_WithComments(t *testing.T) {
	plan := &models.Plan{
		Title: "Comment Test",
		Features: []models.Feature{
			{
				Number: 1,
				Title:  "Core",
				Tasks: []models.Task{
					{
						FeatureNumber: 1,
						TaskNumber:    1,
						Title:         "A task",
						Status:        models.StatusPending,
						Description:   "Description here.",
						Comments: []string{
							"Single-line comment.",
							"Multi-line comment\nthat spans two lines.",
						},
					},
				},
			},
		},
	}

	output := Serialize(plan)

	assertContains(t, output, "> 💬 Single-line comment.")
	assertContains(t, output, "> 💬 Multi-line comment\n> that spans two lines.")
}

func TestSerialize_WithLetterSuffix(t *testing.T) {
	plan := &models.Plan{
		Title: "Suffix Test",
		Features: []models.Feature{
			{
				Number: 1,
				Title:  "Core",
				Tasks: []models.Task{
					{FeatureNumber: 1, TaskNumber: 1, Title: "First", Status: models.StatusCompleted},
					{FeatureNumber: 1, TaskNumber: 1, Suffix: "b", Title: "Follow-up", Status: models.StatusPending},
					{FeatureNumber: 1, TaskNumber: 2, Title: "Second", Status: models.StatusInProgress},
				},
			},
			{
				Number: 2,
				Title:  "Extra",
				Tasks: []models.Task{
					{FeatureNumber: 2, TaskNumber: 1, Title: "Other", Status: models.StatusPending},
				},
			},
		},
	}

	output := Serialize(plan)

	assertContains(t, output, "### Task 1.1: First [completed]")
	assertContains(t, output, "### Task 1.1b: Follow-up [pending]")
	assertContains(t, output, "### Task 1.2: Second [in_progress]")
}

func TestSerialize_MixedCriteriaStates(t *testing.T) {
	plan := &models.Plan{
		Title: "Criteria Test",
		Features: []models.Feature{
			{
				Number: 1,
				Title:  "Core",
				Tasks: []models.Task{
					{
						FeatureNumber: 1,
						TaskNumber:    1,
						Title:         "Task with criteria",
						Status:        models.StatusInProgress,
						Criteria: []models.Criterion{
							{Description: "First done", IsMet: true},
							{Description: "Second not done", IsMet: false},
							{Description: "Third done", IsMet: true},
							{Description: "Fourth not done", IsMet: false},
						},
					},
				},
			},
		},
	}

	output := Serialize(plan)

	assertContains(t, output, "- [x] First done")
	assertContains(t, output, "- [ ] Second not done")
	assertContains(t, output, "- [x] Third done")
	assertContains(t, output, "- [ ] Fourth not done")
}

// Round-trip test: build Plan → serialize → parse → compare.
func TestRoundTrip_MultiFeature(t *testing.T) {
	original := &models.Plan{
		Title:    "Round Trip Test",
		Overview: "Testing round-trip serialization.",
		Features: []models.Feature{
			{
				Number:   1,
				Title:    "Feature One",
				Overview: "First feature overview.",
				Tasks: []models.Task{
					{
						FeatureNumber: 1,
						TaskNumber:    1,
						Title:         "Task one",
						Status:        models.StatusCompleted,
						Complexity:    models.ComplexitySmall,
						Files:         []string{"a.go"},
						DependsOn:     []string{"none"},
						Description:   "Do the first thing.",
						Criteria: []models.Criterion{
							{Description: "It works", IsMet: true},
						},
						Comments: []string{"Looks good."},
					},
					{
						FeatureNumber: 1,
						TaskNumber:    2,
						Title:         "Task two",
						Status:        models.StatusPending,
						Description:   "Do the second thing.",
					},
				},
			},
			{
				Number: 2,
				Title:  "Feature Two",
				Tasks: []models.Task{
					{
						FeatureNumber: 2,
						TaskNumber:    1,
						Title:         "Task three",
						Status:        models.StatusInProgress,
						Complexity:    models.ComplexityLarge,
						Files:         []string{"b.go", "c.go"},
						DependsOn:     []string{"Task 1.1", "Task 1.2"},
						Description:   "Do the third thing.",
						Criteria: []models.Criterion{
							{Description: "Criterion A", IsMet: false},
							{Description: "Criterion B", IsMet: true},
						},
					},
				},
			},
		},
	}

	md := Serialize(original)
	parsed, err := parser.Parse(strings.NewReader(md))
	if err != nil {
		t.Fatalf("failed to parse serialized output: %v", err)
	}

	// Compare key fields.
	if parsed.Title != original.Title {
		t.Errorf("title = %q, want %q", parsed.Title, original.Title)
	}
	if parsed.Overview != original.Overview {
		t.Errorf("overview = %q, want %q", parsed.Overview, original.Overview)
	}
	if len(parsed.Features) != len(original.Features) {
		t.Fatalf("feature count = %d, want %d", len(parsed.Features), len(original.Features))
	}

	for fi, of := range original.Features {
		pf := parsed.Features[fi]
		if pf.Number != of.Number {
			t.Errorf("feature %d number = %d, want %d", fi, pf.Number, of.Number)
		}
		if pf.Title != of.Title {
			t.Errorf("feature %d title = %q, want %q", fi, pf.Title, of.Title)
		}
		if pf.Overview != of.Overview {
			t.Errorf("feature %d overview = %q, want %q", fi, pf.Overview, of.Overview)
		}
		if len(pf.Tasks) != len(of.Tasks) {
			t.Fatalf("feature %d task count = %d, want %d", fi, len(pf.Tasks), len(of.Tasks))
		}
		for ti, ot := range of.Tasks {
			pt := pf.Tasks[ti]
			if pt.FullID() != ot.FullID() {
				t.Errorf("task %d.%d id = %q, want %q", fi, ti, pt.FullID(), ot.FullID())
			}
			if pt.Title != ot.Title {
				t.Errorf("task %s title = %q, want %q", ot.FullID(), pt.Title, ot.Title)
			}
			if pt.Status != ot.Status {
				t.Errorf("task %s status = %q, want %q", ot.FullID(), pt.Status, ot.Status)
			}
			if pt.Complexity != ot.Complexity {
				t.Errorf("task %s complexity = %q, want %q", ot.FullID(), pt.Complexity, ot.Complexity)
			}
			if pt.Description != ot.Description {
				t.Errorf("task %s description = %q, want %q", ot.FullID(), pt.Description, ot.Description)
			}
			if len(pt.Files) != len(ot.Files) {
				t.Errorf("task %s files count = %d, want %d", ot.FullID(), len(pt.Files), len(ot.Files))
			}
			if len(pt.DependsOn) != len(ot.DependsOn) {
				t.Errorf("task %s depends_on count = %d, want %d", ot.FullID(), len(pt.DependsOn), len(ot.DependsOn))
			}
			if len(pt.Criteria) != len(ot.Criteria) {
				t.Errorf("task %s criteria count = %d, want %d", ot.FullID(), len(pt.Criteria), len(ot.Criteria))
			} else {
				for ci, oc := range ot.Criteria {
					pc := pt.Criteria[ci]
					if pc.Description != oc.Description || pc.IsMet != oc.IsMet {
						t.Errorf("task %s criterion %d = %+v, want %+v", ot.FullID(), ci, pc, oc)
					}
				}
			}
			if len(pt.Comments) != len(ot.Comments) {
				t.Errorf("task %s comments count = %d, want %d", ot.FullID(), len(pt.Comments), len(ot.Comments))
			} else {
				for ci, oc := range ot.Comments {
					if pt.Comments[ci] != oc {
						t.Errorf("task %s comment %d = %q, want %q", ot.FullID(), ci, pt.Comments[ci], oc)
					}
				}
			}
		}
	}
}

func TestRoundTrip_SingleFeature(t *testing.T) {
	original := &models.Plan{
		Title:    "Simple Plan",
		Overview: "A simple single-feature plan.",
		Features: []models.Feature{
			{
				Number: 1,
				Title:  "Simple Plan",
				Tasks: []models.Task{
					{
						FeatureNumber: 1,
						TaskNumber:    1,
						Title:         "First task",
						Status:        models.StatusPending,
						Description:   "Do it.",
					},
					{
						FeatureNumber: 1,
						TaskNumber:    2,
						Title:         "Second task",
						Status:        models.StatusCompleted,
						Complexity:    models.ComplexitySmall,
					},
				},
			},
		},
	}

	md := Serialize(original)
	parsed, err := parser.Parse(strings.NewReader(md))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if parsed.Title != original.Title {
		t.Errorf("title = %q, want %q", parsed.Title, original.Title)
	}
	if len(parsed.Features) != 1 {
		t.Fatalf("feature count = %d, want 1", len(parsed.Features))
	}
	if len(parsed.Features[0].Tasks) != 2 {
		t.Fatalf("task count = %d, want 2", len(parsed.Features[0].Tasks))
	}
	for i, ot := range original.Features[0].Tasks {
		pt := parsed.Features[0].Tasks[i]
		if pt.Title != ot.Title {
			t.Errorf("task %d title = %q, want %q", i, pt.Title, ot.Title)
		}
		if pt.Status != ot.Status {
			t.Errorf("task %d status = %q, want %q", i, pt.Status, ot.Status)
		}
	}
}

func TestUpdateTaskStatus(t *testing.T) {
	content := `# Plan: Status Update Test

## Feature 1: Core

### Task 1.1: First task [pending]
Description of first task.

### Task 1.2: Second task [in_progress]
Description of second task.

### Task 1.2b: Follow-up [pending]
Follow-up description.
`

	dir := t.TempDir()
	path := filepath.Join(dir, "plan.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Update task 1.1 status.
	if err := UpdateTaskStatus(path, "1.1", models.StatusCompleted); err != nil {
		t.Fatalf("UpdateTaskStatus: %v", err)
	}

	data, _ := os.ReadFile(path)
	result := string(data)

	// Task 1.1 should be updated.
	assertContains(t, result, "### Task 1.1: First task [completed]")
	// Task 1.2 should be unchanged.
	assertContains(t, result, "### Task 1.2: Second task [in_progress]")
	// Task 1.2b should be unchanged.
	assertContains(t, result, "### Task 1.2b: Follow-up [pending]")
	// Descriptions preserved.
	assertContains(t, result, "Description of first task.")
	assertContains(t, result, "Description of second task.")
}

func TestUpdateTaskStatus_LetterSuffix(t *testing.T) {
	content := `# Plan: Suffix Status Test

## Feature 1: Core

### Task 1.3: Main task [completed]
Main description.

### Task 1.3b: Follow-up task [pending]
Follow-up description.
`

	dir := t.TempDir()
	path := filepath.Join(dir, "plan.md")
	os.WriteFile(path, []byte(content), 0644)

	if err := UpdateTaskStatus(path, "1.3b", models.StatusInProgress); err != nil {
		t.Fatalf("UpdateTaskStatus: %v", err)
	}

	data, _ := os.ReadFile(path)
	result := string(data)

	assertContains(t, result, "### Task 1.3: Main task [completed]")
	assertContains(t, result, "### Task 1.3b: Follow-up task [in_progress]")
}

func TestUpdateTaskStatus_NotFound(t *testing.T) {
	content := `# Plan: Test

## Feature 1: Core

### Task 1.1: Only task [pending]
Description.
`
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.md")
	os.WriteFile(path, []byte(content), 0644)

	err := UpdateTaskStatus(path, "9.9", models.StatusCompleted)
	if err == nil {
		t.Fatal("expected error for non-existent task")
	}
}

func TestUpdateCriterion(t *testing.T) {
	content := `# Plan: Criterion Test

## Feature 1: Core

### Task 1.1: A task [in_progress]
Description here.

**Acceptance Criteria:**
- [ ] First criterion
- [ ] Second criterion
- [x] Third criterion

### Task 1.2: Another task [pending]
Other description.

**Acceptance Criteria:**
- [ ] Unrelated criterion
`

	dir := t.TempDir()
	path := filepath.Join(dir, "plan.md")
	os.WriteFile(path, []byte(content), 0644)

	// Flip "Second criterion" to checked.
	if err := UpdateCriterion(path, "1.1", "Second criterion", true); err != nil {
		t.Fatalf("UpdateCriterion: %v", err)
	}

	data, _ := os.ReadFile(path)
	result := string(data)

	assertContains(t, result, "- [ ] First criterion")
	assertContains(t, result, "- [x] Second criterion")
	assertContains(t, result, "- [x] Third criterion")
	// Task 1.2's criterion should be untouched.
	assertContains(t, result, "- [ ] Unrelated criterion")
	// Descriptions preserved.
	assertContains(t, result, "Description here.")
	assertContains(t, result, "Other description.")
}

func TestUpdateCriterion_Uncheck(t *testing.T) {
	content := `# Plan: Uncheck Test

## Feature 1: Core

### Task 1.1: A task [in_progress]

- [x] Done criterion
- [ ] Undone criterion
`

	dir := t.TempDir()
	path := filepath.Join(dir, "plan.md")
	os.WriteFile(path, []byte(content), 0644)

	if err := UpdateCriterion(path, "1.1", "Done criterion", false); err != nil {
		t.Fatalf("UpdateCriterion: %v", err)
	}

	data, _ := os.ReadFile(path)
	result := string(data)

	assertContains(t, result, "- [ ] Done criterion")
	assertContains(t, result, "- [ ] Undone criterion")
}

func TestUpdateCriterion_NotFound(t *testing.T) {
	content := `# Plan: Test

## Feature 1: Core

### Task 1.1: A task [pending]

- [ ] Real criterion
`
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.md")
	os.WriteFile(path, []byte(content), 0644)

	err := UpdateCriterion(path, "1.1", "Nonexistent criterion", true)
	if err == nil {
		t.Fatal("expected error for non-existent criterion")
	}
}

func TestTargetedUpdate_PreservesFormatting(t *testing.T) {
	// Verify that targeted updates don't introduce any formatting drift.
	content := `# Plan: Formatting Test

## Overview

This plan has careful formatting.

---

## Feature 1: Core

### Overview
Feature overview text.

### Task 1.1: First task [pending]
**Complexity:** medium
**Files:** a.go, b.go
**Depends on:** none

Description with multiple paragraphs.

Second paragraph here.

> 💬 A review comment.

**Acceptance Criteria:**
- [ ] Criterion one
- [ ] Criterion two
- [x] Criterion three

---

## Feature 2: Extensions

### Task 2.1: Extension task [pending]
Extension description.
`

	dir := t.TempDir()
	path := filepath.Join(dir, "plan.md")
	os.WriteFile(path, []byte(content), 0644)

	// Update status on task 1.1.
	if err := UpdateTaskStatus(path, "1.1", models.StatusInProgress); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	result := string(data)

	// Only the status tag should differ.
	expected := strings.Replace(content, "[pending]\n**Complexity:** medium", "[in_progress]\n**Complexity:** medium", 1)
	if result != expected {
		t.Errorf("targeted status update introduced formatting drift.\ngot:\n%s\nwant:\n%s", result, expected)
	}

	// Now flip a criterion.
	if err := UpdateCriterion(path, "1.1", "Criterion two", true); err != nil {
		t.Fatal(err)
	}

	data, _ = os.ReadFile(path)
	result = string(data)

	expected = strings.Replace(expected, "- [ ] Criterion two", "- [x] Criterion two", 1)
	if result != expected {
		t.Errorf("targeted criterion update introduced formatting drift.\ngot:\n%s\nwant:\n%s", result, expected)
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("output does not contain %q\nfull output:\n%s", substr, s)
	}
}

func assertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("output should not contain %q\nfull output:\n%s", substr, s)
	}
}
