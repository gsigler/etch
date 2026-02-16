package status

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gsigler/etch/internal/models"
)

// writePlanFile creates a minimal valid plan file in the temp dir.
func writePlanFile(t *testing.T, rootDir, slug, content string) string {
	t.Helper()
	dir := filepath.Join(rootDir, ".etch", "plans")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, slug+".md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// writeProgressFile creates a progress file for the given plan/task/session.
func writeProgressFile(t *testing.T, rootDir, slug, taskID string, session int, status string, criteria []string) {
	t.Helper()
	dir := filepath.Join(rootDir, ".etch", "progress")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	var b strings.Builder
	b.WriteString("# Session: Task " + taskID + "\n")
	b.WriteString("**Plan:** " + slug + "\n")
	b.WriteString("**Task:** " + taskID + "\n")
	b.WriteString("**Session:** " + pad3(session) + "\n")
	b.WriteString("**Started:** 2026-02-15\n")
	b.WriteString("**Status:** " + status + "\n")
	b.WriteString("\n## Changes Made\n- some-file.go\n")
	b.WriteString("\n## Acceptance Criteria Updates\n")
	for _, c := range criteria {
		b.WriteString(c + "\n")
	}
	b.WriteString("\n## Decisions & Notes\nSome decision\n")
	b.WriteString("\n## Blockers\nNone\n")
	b.WriteString("\n## Next\nContinue work\n")

	filename := slug + "--task-" + taskID + "--" + pad3(session) + ".md"
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func pad3(n int) string {
	return strings.Replace(strings.Replace(strings.Replace(
		"000", "000", func() string {
			s := ""
			if n < 100 {
				s += "0"
			}
			if n < 10 {
				s += "0"
			}
			s += itoa(n)
			return s
		}(), 1), "", "", 0), "", "", 0)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

const testPlan = `# Plan: Auth System

## Overview
Auth system for the app.

---

## Feature 1: Token Management

### Overview
JWT tokens.

### Task 1.1: Schema [pending]
**Complexity:** small
**Files:** db/schema.sql

Build the schema.

**Acceptance Criteria:**
- [ ] Migration file created
- [ ] Indexes added

### Task 1.2: Token gen [pending]
**Complexity:** medium
**Files:** auth/token.go

Generate tokens.

**Acceptance Criteria:**
- [ ] Tokens generated
- [ ] Expiry works

---

## Feature 2: Login Endpoints

### Overview
Login and registration.

### Task 2.1: Registration [pending]
**Complexity:** small
**Files:** api/register.go

Register endpoint.

**Acceptance Criteria:**
- [ ] Endpoint works
`

func TestReconcileCompletedStatus(t *testing.T) {
	root := t.TempDir()
	writePlanFile(t, root, "auth", testPlan)

	// Task 1.1 completed in session 1.
	writeProgressFile(t, root, "auth", "1.1", 1, "completed", []string{
		"- [x] Migration file created",
		"- [x] Indexes added",
	})

	plans, err := Run(root, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}

	task11 := findTask(plans[0], "1.1")
	if task11 == nil {
		t.Fatal("task 1.1 not found")
	}
	if task11.Status != models.StatusCompleted {
		t.Errorf("expected completed, got %s", task11.Status)
	}
	if task11.SessionCount != 1 {
		t.Errorf("expected 1 session, got %d", task11.SessionCount)
	}

	// Verify plan file was updated.
	planContent, _ := os.ReadFile(filepath.Join(root, ".etch", "plans", "auth.md"))
	if !strings.Contains(string(planContent), "[completed]") {
		t.Error("plan file not updated with [completed] status")
	}
	// Verify criteria were checked.
	if !strings.Contains(string(planContent), "- [x] Migration file created") {
		t.Error("criterion not checked in plan file")
	}
}

func TestReconcilePartialStatus(t *testing.T) {
	root := t.TempDir()
	writePlanFile(t, root, "auth", testPlan)

	writeProgressFile(t, root, "auth", "1.2", 1, "partial", []string{
		"- [x] Tokens generated",
		"- [ ] Expiry works",
	})

	plans, err := Run(root, "")
	if err != nil {
		t.Fatal(err)
	}

	task12 := findTask(plans[0], "1.2")
	if task12.Status != models.StatusInProgress {
		t.Errorf("expected in_progress, got %s", task12.Status)
	}
	if task12.LastOutcome != "partial" {
		t.Errorf("expected last outcome 'partial', got %s", task12.LastOutcome)
	}

	// Verify plan file has in_progress.
	planContent, _ := os.ReadFile(filepath.Join(root, ".etch", "plans", "auth.md"))
	if !strings.Contains(string(planContent), "[in_progress]") {
		t.Error("plan file not updated with [in_progress]")
	}
}

func TestReconcileFailedAndBlocked(t *testing.T) {
	root := t.TempDir()
	writePlanFile(t, root, "auth", testPlan)

	writeProgressFile(t, root, "auth", "1.1", 1, "failed", nil)
	writeProgressFile(t, root, "auth", "2.1", 1, "blocked", nil)

	plans, err := Run(root, "")
	if err != nil {
		t.Fatal(err)
	}

	task11 := findTask(plans[0], "1.1")
	if task11.Status != models.StatusFailed {
		t.Errorf("expected failed, got %s", task11.Status)
	}

	task21 := findTask(plans[0], "2.1")
	if task21.Status != models.StatusBlocked {
		t.Errorf("expected blocked, got %s", task21.Status)
	}
}

func TestMultipleSessionsUsesLatest(t *testing.T) {
	root := t.TempDir()
	writePlanFile(t, root, "auth", testPlan)

	// Session 1: partial, session 2: completed.
	writeProgressFile(t, root, "auth", "1.1", 1, "partial", []string{
		"- [x] Migration file created",
	})
	writeProgressFile(t, root, "auth", "1.1", 2, "completed", []string{
		"- [x] Migration file created",
		"- [x] Indexes added",
	})

	plans, err := Run(root, "")
	if err != nil {
		t.Fatal(err)
	}

	task11 := findTask(plans[0], "1.1")
	if task11.Status != models.StatusCompleted {
		t.Errorf("expected completed, got %s", task11.Status)
	}
	if task11.SessionCount != 2 {
		t.Errorf("expected 2 sessions, got %d", task11.SessionCount)
	}
}

func TestCriteriaMergingAcrossSessions(t *testing.T) {
	root := t.TempDir()
	writePlanFile(t, root, "auth", testPlan)

	// Session 1: first criterion met.
	writeProgressFile(t, root, "auth", "1.1", 1, "partial", []string{
		"- [x] Migration file created",
		"- [ ] Indexes added",
	})
	// Session 2: second criterion met but first not listed.
	writeProgressFile(t, root, "auth", "1.1", 2, "completed", []string{
		"- [ ] Migration file created",
		"- [x] Indexes added",
	})

	plans, err := Run(root, "")
	if err != nil {
		t.Fatal(err)
	}

	task11 := findTask(plans[0], "1.1")
	// Both criteria should be met (merge across sessions).
	for _, c := range task11.Criteria {
		if !c.IsMet {
			t.Errorf("criterion %q should be met", c.Description)
		}
	}

	// Verify plan file has both checked.
	planContent, _ := os.ReadFile(filepath.Join(root, ".etch", "plans", "auth.md"))
	if strings.Contains(string(planContent), "- [ ] Migration file created") {
		t.Error("first criterion should be checked in plan file")
	}
	if strings.Contains(string(planContent), "- [ ] Indexes added") {
		t.Error("second criterion should be checked in plan file")
	}
}

func TestIsActive(t *testing.T) {
	tests := []struct {
		name   string
		ps     PlanStatus
		active bool
	}{
		{
			name:   "fully pending is not active",
			ps:     PlanStatus{CompletedTasks: 0, TotalTasks: 3, Features: []FeatureStatus{{Tasks: []TaskStatus{{Status: models.StatusPending}}}}},
			active: false,
		},
		{
			name:   "fully completed is not active",
			ps:     PlanStatus{CompletedTasks: 3, TotalTasks: 3, Features: []FeatureStatus{{Tasks: []TaskStatus{{Status: models.StatusCompleted}}}}},
			active: false,
		},
		{
			name:   "in-progress task is active",
			ps:     PlanStatus{CompletedTasks: 0, TotalTasks: 3, Features: []FeatureStatus{{Tasks: []TaskStatus{{Status: models.StatusInProgress}}}}},
			active: true,
		},
		{
			name:   "failed task is active",
			ps:     PlanStatus{CompletedTasks: 0, TotalTasks: 3, Features: []FeatureStatus{{Tasks: []TaskStatus{{Status: models.StatusFailed}}}}},
			active: true,
		},
		{
			name:   "blocked task is active",
			ps:     PlanStatus{CompletedTasks: 0, TotalTasks: 3, Features: []FeatureStatus{{Tasks: []TaskStatus{{Status: models.StatusBlocked}}}}},
			active: true,
		},
		{
			name:   "partially completed is active",
			ps:     PlanStatus{CompletedTasks: 1, TotalTasks: 3, Features: []FeatureStatus{{Tasks: []TaskStatus{{Status: models.StatusPending}, {Status: models.StatusCompleted}}}}},
			active: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ps.IsActive(); got != tt.active {
				t.Errorf("IsActive() = %v, want %v", got, tt.active)
			}
		})
	}
}

func TestFilterActive(t *testing.T) {
	plans := []PlanStatus{
		{Title: "Active", CompletedTasks: 1, TotalTasks: 3, Features: []FeatureStatus{{Tasks: []TaskStatus{{Status: models.StatusInProgress}}}}},
		{Title: "Pending", CompletedTasks: 0, TotalTasks: 2, Features: []FeatureStatus{{Tasks: []TaskStatus{{Status: models.StatusPending}}}}},
		{Title: "Done", CompletedTasks: 2, TotalTasks: 2, Features: []FeatureStatus{{Tasks: []TaskStatus{{Status: models.StatusCompleted}}}}},
	}

	active := FilterActive(plans)
	if len(active) != 1 {
		t.Fatalf("expected 1 active plan, got %d", len(active))
	}
	if active[0].Title != "Active" {
		t.Errorf("expected Active plan, got %s", active[0].Title)
	}
}

func TestProgressBar(t *testing.T) {
	tests := []struct {
		pct  int
		want string
	}{
		{0, "[░░░░░░░░░░] 0%"},
		{50, "[█████░░░░░] 50%"},
		{100, "[██████████] 100%"},
		{33, "[███░░░░░░░] 33%"},
	}
	for _, tt := range tests {
		got := progressBar(tt.pct)
		if got != tt.want {
			t.Errorf("progressBar(%d) = %q, want %q", tt.pct, got, tt.want)
		}
	}
}

func TestFormatSummarySeparators(t *testing.T) {
	plans := []PlanStatus{
		{Title: "Plan A", Slug: "plan-a", CompletedTasks: 0, TotalTasks: 1, Features: []FeatureStatus{{Number: 1, Title: "F1", TotalTasks: 1, Tasks: []TaskStatus{{ID: "1", Title: "T", Status: models.StatusPending}}}}},
		{Title: "Plan B", Slug: "plan-b", CompletedTasks: 0, TotalTasks: 1, Features: []FeatureStatus{{Number: 1, Title: "F1", TotalTasks: 1, Tasks: []TaskStatus{{ID: "1", Title: "T", Status: models.StatusPending}}}}},
	}
	output := FormatSummary(plans)
	if !strings.Contains(output, "────") {
		t.Error("expected separator line between plans")
	}
}

func TestReconcilePopulatesPlanTotals(t *testing.T) {
	root := t.TempDir()
	writePlanFile(t, root, "auth", testPlan)

	writeProgressFile(t, root, "auth", "1.1", 1, "completed", []string{
		"- [x] Migration file created",
		"- [x] Indexes added",
	})

	plans, err := Run(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if plans[0].TotalTasks != 3 {
		t.Errorf("expected 3 total tasks, got %d", plans[0].TotalTasks)
	}
	if plans[0].CompletedTasks != 1 {
		t.Errorf("expected 1 completed task, got %d", plans[0].CompletedTasks)
	}
}

func TestNoPlansDirGraceful(t *testing.T) {
	root := t.TempDir()
	// No .etch/plans/ directory at all.

	plans, err := Run(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 0 {
		t.Errorf("expected 0 plans, got %d", len(plans))
	}

	output := FormatSummary(plans)
	if output != "No plans found." {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestPlanWithNoProgress(t *testing.T) {
	root := t.TempDir()
	writePlanFile(t, root, "auth", testPlan)
	// No progress files.

	plans, err := Run(root, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}

	// All tasks should retain their original pending status.
	for _, f := range plans[0].Features {
		for _, t2 := range f.Tasks {
			if t2.Status != models.StatusPending {
				t.Errorf("task %s expected pending, got %s", t2.ID, t2.Status)
			}
			if t2.SessionCount != 0 {
				t.Errorf("task %s expected 0 sessions, got %d", t2.ID, t2.SessionCount)
			}
		}
	}
}

func TestOrphanedProgressIgnored(t *testing.T) {
	root := t.TempDir()
	writePlanFile(t, root, "auth", testPlan)

	// Progress for a task that doesn't exist in the plan.
	writeProgressFile(t, root, "auth", "9.9", 1, "completed", nil)

	plans, err := Run(root, "")
	if err != nil {
		t.Fatal(err)
	}

	// Should not error and should not include orphaned task.
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}

	// Verify no task 9.9 exists in output.
	for _, f := range plans[0].Features {
		for _, t2 := range f.Tasks {
			if t2.ID == "9.9" {
				t.Error("orphaned task 9.9 should not be in output")
			}
		}
	}
}

func TestPlanFilter(t *testing.T) {
	root := t.TempDir()
	writePlanFile(t, root, "auth", testPlan)
	writePlanFile(t, root, "api-refactor", `# Plan: API Refactor

## Overview
Refactor the API.

### Task 1: Migrate [pending]
**Complexity:** small

Migrate stuff.

**Acceptance Criteria:**
- [ ] Done
`)

	plans, err := Run(root, "auth")
	if err != nil {
		t.Fatal(err)
	}

	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	if plans[0].Slug != "auth" {
		t.Errorf("expected auth plan, got %s", plans[0].Slug)
	}
}

func TestFormatSummary(t *testing.T) {
	plans := []PlanStatus{
		{
			Title:          "Auth System",
			Slug:           "auth-system",
			CompletedTasks: 3,
			TotalTasks:     5,
			Features: []FeatureStatus{
				{
					Number:         1,
					Title:          "Token Management",
					CompletedTasks: 2,
					TotalTasks:     2,
					Tasks: []TaskStatus{
						{ID: "1.1", Title: "Schema", Status: models.StatusCompleted},
						{ID: "1.2", Title: "Token gen", Status: models.StatusCompleted},
					},
				},
				{
					Number:         2,
					Title:          "Login Endpoints",
					CompletedTasks: 1,
					TotalTasks:     3,
					Tasks: []TaskStatus{
						{ID: "2.1", Title: "Registration", Status: models.StatusCompleted},
						{ID: "2.2", Title: "Login", Status: models.StatusInProgress, SessionCount: 2, LastOutcome: "partial"},
						{ID: "2.3", Title: "Password", Status: models.StatusPending},
					},
				},
			},
		},
	}

	output := FormatSummary(plans)

	// Check progress bar.
	if !strings.Contains(output, "[██████░░░░] 60%") {
		t.Error("expected progress bar with 60%")
	}
	// Check slug.
	if !strings.Contains(output, "slug: auth-system") {
		t.Error("expected slug line")
	}
	// Check icons.
	if !strings.Contains(output, "✓ Feature 1") {
		t.Error("expected completed icon for feature 1")
	}
	if !strings.Contains(output, "▶ Feature 2") {
		t.Error("expected in-progress icon for feature 2")
	}
	if !strings.Contains(output, "[2/2 tasks]") {
		t.Error("expected [2/2 tasks]")
	}
	if !strings.Contains(output, "[1/3 tasks]") {
		t.Error("expected [1/3 tasks]")
	}
	if !strings.Contains(output, "2.2") && !strings.Contains(output, "Login") {
		t.Error("expected task 2.2 Login in output")
	}
	if !strings.Contains(output, "○") {
		t.Error("expected pending icon for task 2.3")
	}
}

func TestFormatDetailed(t *testing.T) {
	ps := PlanStatus{
		Title:          "Auth System",
		Slug:           "auth-system",
		CompletedTasks: 1,
		TotalTasks:     1,
		Features: []FeatureStatus{
			{
				Number:         1,
				Title:          "Token Management",
				CompletedTasks: 1,
				TotalTasks:     1,
				Tasks: []TaskStatus{
					{
						ID:       "1.1",
						Title:    "Schema",
						Status:   models.StatusCompleted,
						Criteria: []models.Criterion{{Description: "Migration", IsMet: true}},
						SessionCount:  1,
						LastOutcome:   "completed",
						LastDecisions: "Used postgres",
						LastNext:      "All done",
					},
				},
			},
		},
	}

	output := FormatDetailed(ps)

	if !strings.Contains(output, "[██████████] 100%") {
		t.Error("expected progress bar with 100%")
	}
	if !strings.Contains(output, "slug: auth-system") {
		t.Error("expected slug line in detailed view")
	}
	if !strings.Contains(output, "[x] Migration") {
		t.Error("expected checked criterion in detailed view")
	}
	if !strings.Contains(output, "Notes: Used postgres") {
		t.Error("expected decisions in detailed view")
	}
	if !strings.Contains(output, "Next: All done") {
		t.Error("expected next in detailed view")
	}
}

func TestFormatJSON(t *testing.T) {
	plans := []PlanStatus{
		{
			Title: "Auth System",
			Slug:  "auth",
			Features: []FeatureStatus{
				{
					Number:         1,
					Title:          "Tokens",
					CompletedTasks: 1,
					TotalTasks:     2,
					Tasks: []TaskStatus{
						{ID: "1.1", Title: "Schema", Status: models.StatusCompleted, SessionCount: 1},
						{ID: "1.2", Title: "Token gen", Status: models.StatusPending},
					},
				},
			},
		},
	}

	out, err := FormatJSON(plans)
	if err != nil {
		t.Fatal(err)
	}

	// Validate JSON.
	var parsed []PlanStatus
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(parsed) != 1 {
		t.Fatalf("expected 1 plan in JSON, got %d", len(parsed))
	}
	if parsed[0].Title != "Auth System" {
		t.Errorf("expected Auth System, got %s", parsed[0].Title)
	}
	if parsed[0].Features[0].Tasks[0].Status != models.StatusCompleted {
		t.Error("expected completed status in JSON")
	}
}

func TestPlanFilePreservesOtherContent(t *testing.T) {
	root := t.TempDir()
	planPath := writePlanFile(t, root, "auth", testPlan)

	// Read original content.
	original, _ := os.ReadFile(planPath)

	// Run status with a completed task.
	writeProgressFile(t, root, "auth", "1.1", 1, "completed", []string{
		"- [x] Migration file created",
	})

	_, err := Run(root, "")
	if err != nil {
		t.Fatal(err)
	}

	updated, _ := os.ReadFile(planPath)

	// Verify overview is preserved.
	if !strings.Contains(string(updated), "Auth system for the app.") {
		t.Error("overview was modified")
	}
	// Verify other tasks are unchanged.
	if !strings.Contains(string(updated), "### Task 1.2: Token gen [pending]") {
		t.Error("task 1.2 was modified")
	}
	// Verify the changed task has new status.
	if !strings.Contains(string(updated), "### Task 1.1: Schema [completed]") {
		t.Error("task 1.1 not updated")
	}

	// Verify unchanged lines count roughly matches (allowing for status tag and criteria changes).
	origLines := strings.Split(string(original), "\n")
	updLines := strings.Split(string(updated), "\n")
	if len(origLines) != len(updLines) {
		t.Errorf("line count changed: %d -> %d", len(origLines), len(updLines))
	}
}

func TestFeatureIconLogic(t *testing.T) {
	tests := []struct {
		name     string
		feature  FeatureStatus
		wantIcon string
	}{
		{
			name:     "all completed",
			feature:  FeatureStatus{CompletedTasks: 3, TotalTasks: 3},
			wantIcon: "✓",
		},
		{
			name:     "some completed",
			feature:  FeatureStatus{CompletedTasks: 1, TotalTasks: 3},
			wantIcon: "▶",
		},
		{
			name: "none completed but one in progress",
			feature: FeatureStatus{
				CompletedTasks: 0, TotalTasks: 2,
				Tasks: []TaskStatus{{Status: models.StatusInProgress}, {Status: models.StatusPending}},
			},
			wantIcon: "▶",
		},
		{
			name: "all pending",
			feature: FeatureStatus{
				CompletedTasks: 0, TotalTasks: 2,
				Tasks: []TaskStatus{{Status: models.StatusPending}, {Status: models.StatusPending}},
			},
			wantIcon: "○",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := featureIcon(tt.feature)
			if got != tt.wantIcon {
				t.Errorf("expected icon %s, got %s", tt.wantIcon, got)
			}
		})
	}
}

func TestMapProgressStatus(t *testing.T) {
	tests := []struct {
		input string
		want  models.Status
	}{
		{"completed", models.StatusCompleted},
		{"partial", models.StatusInProgress},
		{"failed", models.StatusFailed},
		{"blocked", models.StatusBlocked},
		{"pending", models.StatusPending},
		{"unknown", models.StatusPending},
		{"", models.StatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapProgressStatus(tt.input)
			if got != tt.want {
				t.Errorf("mapProgressStatus(%q) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

// findTask finds a task by ID in a PlanStatus.
func findTask(ps PlanStatus, id string) *TaskStatus {
	for _, f := range ps.Features {
		for i, t := range f.Tasks {
			if t.ID == id {
				return &f.Tasks[i]
			}
		}
	}
	return nil
}

// --- Blocked task detection tests ---

const testPlanWithDeps = `# Plan: Dependency Test

## Overview
Plan with task dependencies.

---

## Feature 1: Core

### Overview
Core tasks.

### Task 1.1: Foundation [pending]
**Complexity:** small
**Files:** core/base.go

Build foundation.

**Acceptance Criteria:**
- [ ] Base done

### Task 1.2: Middleware [pending]
**Complexity:** medium
**Files:** core/mid.go
**Depends on:** Task 1.1

Build middleware on top of foundation.

**Acceptance Criteria:**
- [ ] Middleware done

### Task 1.3: API [pending]
**Complexity:** large
**Files:** core/api.go
**Depends on:** Task 1.1, Task 1.2

Build API on top of foundation and middleware.

**Acceptance Criteria:**
- [ ] API done
`

func TestResolveBlocked(t *testing.T) {
	tests := []struct {
		name         string
		tasks        []TaskStatus
		wantBlocked  map[string]bool
	}{
		{
			name: "no dependencies means not blocked",
			tasks: []TaskStatus{
				{ID: "1.1", Status: models.StatusPending},
				{ID: "1.2", Status: models.StatusPending},
			},
			wantBlocked: map[string]bool{"1.1": false, "1.2": false},
		},
		{
			name: "dependency not completed means blocked",
			tasks: []TaskStatus{
				{ID: "1.1", Status: models.StatusPending},
				{ID: "1.2", Status: models.StatusPending, DependsOn: []string{"Task 1.1"}},
			},
			wantBlocked: map[string]bool{"1.1": false, "1.2": true},
		},
		{
			name: "dependency completed means not blocked",
			tasks: []TaskStatus{
				{ID: "1.1", Status: models.StatusCompleted},
				{ID: "1.2", Status: models.StatusPending, DependsOn: []string{"Task 1.1"}},
			},
			wantBlocked: map[string]bool{"1.1": false, "1.2": false},
		},
		{
			name: "chain: A pending -> B blocked -> C blocked",
			tasks: []TaskStatus{
				{ID: "1.1", Status: models.StatusPending},
				{ID: "1.2", Status: models.StatusPending, DependsOn: []string{"Task 1.1"}},
				{ID: "1.3", Status: models.StatusPending, DependsOn: []string{"Task 1.1", "Task 1.2"}},
			},
			wantBlocked: map[string]bool{"1.1": false, "1.2": true, "1.3": true},
		},
		{
			name: "chain: A completed -> B ready -> C blocked (B not done)",
			tasks: []TaskStatus{
				{ID: "1.1", Status: models.StatusCompleted},
				{ID: "1.2", Status: models.StatusPending, DependsOn: []string{"Task 1.1"}},
				{ID: "1.3", Status: models.StatusPending, DependsOn: []string{"Task 1.1", "Task 1.2"}},
			},
			wantBlocked: map[string]bool{"1.1": false, "1.2": false, "1.3": true},
		},
		{
			name: "all deps completed means ready",
			tasks: []TaskStatus{
				{ID: "1.1", Status: models.StatusCompleted},
				{ID: "1.2", Status: models.StatusCompleted},
				{ID: "1.3", Status: models.StatusPending, DependsOn: []string{"Task 1.1", "Task 1.2"}},
			},
			wantBlocked: map[string]bool{"1.1": false, "1.2": false, "1.3": false},
		},
		{
			name: "in-progress task with deps is not marked blocked",
			tasks: []TaskStatus{
				{ID: "1.1", Status: models.StatusPending},
				{ID: "1.2", Status: models.StatusInProgress, DependsOn: []string{"Task 1.1"}},
			},
			wantBlocked: map[string]bool{"1.1": false, "1.2": false},
		},
		{
			name: "completed task with deps is not marked blocked",
			tasks: []TaskStatus{
				{ID: "1.1", Status: models.StatusPending},
				{ID: "1.2", Status: models.StatusCompleted, DependsOn: []string{"Task 1.1"}},
			},
			wantBlocked: map[string]bool{"1.1": false, "1.2": false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &PlanStatus{
				Features: []FeatureStatus{{Tasks: tt.tasks}},
			}
			resolveBlocked(ps)
			for _, task := range ps.Features[0].Tasks {
				want := tt.wantBlocked[task.ID]
				if task.IsBlocked != want {
					t.Errorf("task %s: IsBlocked = %v, want %v", task.ID, task.IsBlocked, want)
				}
			}
		})
	}
}

func TestExtractDepID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Task 1.1", "1.1"},
		{"Task 1.2", "1.2"},
		{"Task 1.3b", "1.3b"},
		{"Task 2.10", "2.10"},
		{"no match here", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := extractDepID(tt.input)
		if got != tt.want {
			t.Errorf("extractDepID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestReconcileWithDependencies(t *testing.T) {
	root := t.TempDir()
	writePlanFile(t, root, "dep-test", testPlanWithDeps)

	// Task 1.1 still pending, so 1.2 and 1.3 should be blocked.
	plans, err := Run(root, "dep-test")
	if err != nil {
		t.Fatal(err)
	}

	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}

	task11 := findTask(plans[0], "1.1")
	task12 := findTask(plans[0], "1.2")
	task13 := findTask(plans[0], "1.3")

	if task11.IsBlocked {
		t.Error("task 1.1 should not be blocked (no deps)")
	}
	if !task12.IsBlocked {
		t.Error("task 1.2 should be blocked (depends on pending 1.1)")
	}
	if !task13.IsBlocked {
		t.Error("task 1.3 should be blocked (depends on pending 1.1 and 1.2)")
	}
}

func TestReconcileWithDependenciesPartiallyMet(t *testing.T) {
	root := t.TempDir()
	writePlanFile(t, root, "dep-test", testPlanWithDeps)

	// Complete task 1.1.
	writeProgressFile(t, root, "dep-test", "1.1", 1, "completed", []string{
		"- [x] Base done",
	})

	plans, err := Run(root, "dep-test")
	if err != nil {
		t.Fatal(err)
	}

	task12 := findTask(plans[0], "1.2")
	task13 := findTask(plans[0], "1.3")

	if task12.IsBlocked {
		t.Error("task 1.2 should not be blocked (1.1 is completed)")
	}
	if !task13.IsBlocked {
		t.Error("task 1.3 should be blocked (1.2 is still pending)")
	}
}

func TestBlockedTaskIcon(t *testing.T) {
	tests := []struct {
		name     string
		task     TaskStatus
		wantIcon string
	}{
		{
			name:     "pending not blocked shows ready",
			task:     TaskStatus{Status: models.StatusPending, IsBlocked: false},
			wantIcon: "○",
		},
		{
			name:     "pending blocked shows blocked",
			task:     TaskStatus{Status: models.StatusPending, IsBlocked: true},
			wantIcon: "⊘",
		},
		{
			name:     "in-progress shows in-progress regardless",
			task:     TaskStatus{Status: models.StatusInProgress, IsBlocked: false},
			wantIcon: "▶",
		},
		{
			name:     "completed shows completed",
			task:     TaskStatus{Status: models.StatusCompleted, IsBlocked: false},
			wantIcon: "✓",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := taskIcon(tt.task)
			if got != tt.wantIcon {
				t.Errorf("taskIcon() = %q, want %q", got, tt.wantIcon)
			}
		})
	}
}

func TestFormatSummaryBlockedIcons(t *testing.T) {
	plans := []PlanStatus{
		{
			Title: "Dep Plan",
			Slug:  "dep-plan",
			Features: []FeatureStatus{
				{
					Number:     1,
					Title:      "Core",
					TotalTasks: 3,
					Tasks: []TaskStatus{
						{ID: "1.1", Title: "Foundation", Status: models.StatusPending},
						{ID: "1.2", Title: "Middleware", Status: models.StatusPending, IsBlocked: true, DependsOn: []string{"Task 1.1"}},
						{ID: "1.3", Title: "API", Status: models.StatusPending, IsBlocked: true, DependsOn: []string{"Task 1.1", "Task 1.2"}},
					},
				},
			},
		},
	}

	output := FormatSummary(plans)

	// Task 1.1 should show ○ (ready), 1.2 and 1.3 should show ⊘ (blocked).
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "1.1") && strings.Contains(line, "Foundation") {
			if !strings.Contains(line, "○") {
				t.Errorf("task 1.1 should show ○, got: %s", line)
			}
		}
		if strings.Contains(line, "1.2") && strings.Contains(line, "Middleware") {
			if !strings.Contains(line, "⊘") {
				t.Errorf("task 1.2 should show ⊘, got: %s", line)
			}
		}
		if strings.Contains(line, "1.3") && strings.Contains(line, "API") {
			if !strings.Contains(line, "⊘") {
				t.Errorf("task 1.3 should show ⊘, got: %s", line)
			}
		}
	}
}

func TestFormatDetailedBlockedShowsDeps(t *testing.T) {
	ps := PlanStatus{
		Title: "Dep Plan",
		Slug:  "dep-plan",
		Features: []FeatureStatus{
			{
				Number:     1,
				Title:      "Core",
				TotalTasks: 2,
				Tasks: []TaskStatus{
					{ID: "1.1", Title: "Foundation", Status: models.StatusPending},
					{ID: "1.2", Title: "Middleware", Status: models.StatusPending, IsBlocked: true, DependsOn: []string{"Task 1.1"}},
				},
			},
		},
	}

	output := FormatDetailed(ps)

	if !strings.Contains(output, "Waiting on: Task 1.1") {
		t.Error("detailed view should show 'Waiting on: Task 1.1' for blocked task")
	}
	// Non-blocked task should not show waiting on.
	lines := strings.Split(output, "\n")
	foundWaiting := 0
	for _, line := range lines {
		if strings.Contains(line, "Waiting on:") {
			foundWaiting++
		}
	}
	if foundWaiting != 1 {
		t.Errorf("expected exactly 1 'Waiting on:' line, got %d", foundWaiting)
	}
}

func TestPercentageEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		completed int
		total     int
		wantPct   int
	}{
		{"zero tasks", 0, 0, 0},
		{"all completed", 5, 5, 100},
		{"none completed", 0, 5, 0},
		{"one of three", 1, 3, 33},
		{"two of three", 2, 3, 66},
		{"one of one", 1, 1, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := PlanStatus{CompletedTasks: tt.completed, TotalTasks: tt.total}
			got := ps.Percentage()
			if got != tt.wantPct {
				t.Errorf("Percentage() = %d, want %d", got, tt.wantPct)
			}
		})
	}
}

func TestProgressBarEdgeCases(t *testing.T) {
	tests := []struct {
		pct  int
		want string
	}{
		{0, "[░░░░░░░░░░] 0%"},
		{5, "[░░░░░░░░░░] 5%"},
		{10, "[█░░░░░░░░░] 10%"},
		{99, "[█████████░] 99%"},
		{100, "[██████████] 100%"},
		{150, "[██████████] 150%"},  // clamped filled to 10
	}
	for _, tt := range tests {
		got := progressBar(tt.pct)
		if got != tt.want {
			t.Errorf("progressBar(%d) = %q, want %q", tt.pct, got, tt.want)
		}
	}
}

func TestFormatDetailedMultipleDeps(t *testing.T) {
	ps := PlanStatus{
		Title: "Multi Dep",
		Slug:  "multi-dep",
		Features: []FeatureStatus{
			{
				Number:     1,
				Title:      "Core",
				TotalTasks: 3,
				Tasks: []TaskStatus{
					{ID: "1.1", Title: "A", Status: models.StatusPending},
					{ID: "1.2", Title: "B", Status: models.StatusPending},
					{ID: "1.3", Title: "C", Status: models.StatusPending, IsBlocked: true, DependsOn: []string{"Task 1.1", "Task 1.2"}},
				},
			},
		},
	}

	output := FormatDetailed(ps)
	if !strings.Contains(output, "Waiting on: Task 1.1, Task 1.2") {
		t.Error("detailed view should show all dependencies for blocked task")
	}
}
