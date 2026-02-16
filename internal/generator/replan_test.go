package generator

import (
	"strings"
	"testing"

	"github.com/gsigler/etch/internal/models"
)

// --- Target resolution tests ---

func testPlan() *models.Plan {
	return &models.Plan{
		Title: "Test Plan",
		Slug:  "test-plan",
		Features: []models.Feature{
			{
				Number: 1,
				Title:  "Backend API",
				Tasks: []models.Task{
					{FeatureNumber: 1, TaskNumber: 1, Title: "Setup server", Status: models.StatusCompleted},
					{FeatureNumber: 1, TaskNumber: 2, Title: "Add routes", Status: models.StatusPending},
					{FeatureNumber: 1, TaskNumber: 3, Title: "Auth middleware", Suffix: "b", Status: models.StatusPending},
				},
			},
			{
				Number: 2,
				Title:  "Frontend UI",
				Tasks: []models.Task{
					{FeatureNumber: 2, TaskNumber: 1, Title: "Create components", Status: models.StatusPending},
					{FeatureNumber: 2, TaskNumber: 2, Title: "Add styling", Status: models.StatusFailed},
				},
			},
		},
	}
}

func TestResolveTarget_TaskID(t *testing.T) {
	plan := testPlan()
	target, err := ResolveTarget(plan, "1.2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.Type != "task" {
		t.Errorf("type = %q, want %q", target.Type, "task")
	}
	if target.TaskID != "1.2" {
		t.Errorf("task ID = %q, want %q", target.TaskID, "1.2")
	}
	if target.Task.Title != "Add routes" {
		t.Errorf("task title = %q, want %q", target.Task.Title, "Add routes")
	}
}

func TestResolveTarget_TaskIDWithSuffix(t *testing.T) {
	plan := testPlan()
	target, err := ResolveTarget(plan, "1.3b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.Type != "task" {
		t.Errorf("type = %q, want %q", target.Type, "task")
	}
	if target.TaskID != "1.3b" {
		t.Errorf("task ID = %q, want %q", target.TaskID, "1.3b")
	}
}

func TestResolveTarget_FeatureRef(t *testing.T) {
	plan := testPlan()
	target, err := ResolveTarget(plan, "feature:2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.Type != "feature" {
		t.Errorf("type = %q, want %q", target.Type, "feature")
	}
	if target.FeatureNum != 2 {
		t.Errorf("feature num = %d, want %d", target.FeatureNum, 2)
	}
	if target.Feature.Title != "Frontend UI" {
		t.Errorf("feature title = %q, want %q", target.Feature.Title, "Frontend UI")
	}
}

func TestResolveTarget_FeatureRefCaseInsensitive(t *testing.T) {
	plan := testPlan()
	target, err := ResolveTarget(plan, "Feature:1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.Type != "feature" {
		t.Errorf("type = %q, want %q", target.Type, "feature")
	}
	if target.FeatureNum != 1 {
		t.Errorf("feature num = %d, want %d", target.FeatureNum, 1)
	}
}

func TestResolveTarget_FeatureByTitle(t *testing.T) {
	plan := testPlan()
	target, err := ResolveTarget(plan, "Frontend UI")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.Type != "feature" {
		t.Errorf("type = %q, want %q", target.Type, "feature")
	}
	if target.FeatureNum != 2 {
		t.Errorf("feature num = %d, want %d", target.FeatureNum, 2)
	}
}

func TestResolveTarget_FeatureByPartialTitle(t *testing.T) {
	plan := testPlan()
	target, err := ResolveTarget(plan, "frontend")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.Type != "feature" {
		t.Errorf("type = %q, want %q", target.Type, "feature")
	}
	if target.FeatureNum != 2 {
		t.Errorf("feature num = %d, want %d", target.FeatureNum, 2)
	}
}

func TestResolveTarget_AmbiguousNumber_PrefersTask(t *testing.T) {
	// Single-feature plan where "2" could be task 1.2 or feature 2.
	plan := &models.Plan{
		Title: "Test",
		Features: []models.Feature{
			{
				Number: 1,
				Title:  "Only Feature",
				Tasks: []models.Task{
					{FeatureNumber: 1, TaskNumber: 1, Title: "First"},
					{FeatureNumber: 1, TaskNumber: 2, Title: "Second"},
				},
			},
		},
	}
	target, err := ResolveTarget(plan, "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should prefer task interpretation.
	if target.Type != "task" {
		t.Errorf("type = %q, want %q (should prefer task for bare number)", target.Type, "task")
	}
	if target.TaskID != "1.2" {
		t.Errorf("task ID = %q, want %q", target.TaskID, "1.2")
	}
}

func TestResolveTarget_NumberFallsBackToFeature(t *testing.T) {
	plan := testPlan()
	// "2" in a multi-feature plan where task 1.2 exists — still prefers task.
	target, err := ResolveTarget(plan, "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.Type != "task" {
		t.Errorf("type = %q, want %q", target.Type, "task")
	}
}

func TestResolveTarget_NumberAsFeatureFallback(t *testing.T) {
	// Plan where "3" doesn't match any task but matches feature 3.
	plan := &models.Plan{
		Title: "Test",
		Features: []models.Feature{
			{Number: 1, Title: "F1", Tasks: []models.Task{{FeatureNumber: 1, TaskNumber: 1, Title: "T1"}}},
			{Number: 2, Title: "F2", Tasks: []models.Task{{FeatureNumber: 2, TaskNumber: 1, Title: "T2"}}},
			{Number: 3, Title: "F3", Tasks: []models.Task{{FeatureNumber: 3, TaskNumber: 1, Title: "T3"}}},
		},
	}
	target, err := ResolveTarget(plan, "3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// "3" doesn't match task "1.3", so should fall back to feature 3.
	if target.Type != "feature" {
		t.Errorf("type = %q, want %q", target.Type, "feature")
	}
	if target.FeatureNum != 3 {
		t.Errorf("feature num = %d, want %d", target.FeatureNum, 3)
	}
}

func TestResolveTarget_Empty(t *testing.T) {
	plan := testPlan()
	_, err := ResolveTarget(plan, "")
	if err == nil {
		t.Error("expected error for empty target")
	}
}

func TestResolveTarget_NotFound(t *testing.T) {
	plan := testPlan()
	_, err := ResolveTarget(plan, "9.9")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestResolveTarget_InvalidFeatureRef(t *testing.T) {
	plan := testPlan()
	_, err := ResolveTarget(plan, "feature:abc")
	if err == nil {
		t.Error("expected error for invalid feature ref")
	}
}

func TestResolveTarget_FeatureNotFound(t *testing.T) {
	plan := testPlan()
	_, err := ResolveTarget(plan, "feature:99")
	if err == nil {
		t.Error("expected error for nonexistent feature")
	}
}

// --- BuildReplanScope tests ---

func TestBuildReplanScope_TaskNoSessions(t *testing.T) {
	target := ReplanTarget{
		Type:   "task",
		TaskID: "1.2",
		Task:   &models.Task{FeatureNumber: 1, TaskNumber: 2, Title: "Add routes"},
	}
	sessions := make(map[string][]models.SessionProgress)

	scope := BuildReplanScope(target, sessions)

	if !strings.Contains(scope, "Task 1.2") {
		t.Error("scope should reference the task")
	}
	if !strings.Contains(scope, "not been attempted") {
		t.Error("scope should indicate planning issue for no sessions")
	}
	if !strings.Contains(scope, "scope right") {
		t.Error("scope should include planning prompts")
	}
}

func TestBuildReplanScope_TaskWithSessions(t *testing.T) {
	target := ReplanTarget{
		Type:   "task",
		TaskID: "1.2",
		Task:   &models.Task{FeatureNumber: 1, TaskNumber: 2, Title: "Add routes"},
	}
	sessions := map[string][]models.SessionProgress{
		"1.2": {
			{SessionNumber: 1, Status: "failed", Blockers: "API not available", Decisions: "Tried REST approach"},
			{SessionNumber: 2, Status: "blocked", Blockers: "Still blocked on API"},
		},
	}

	scope := BuildReplanScope(target, sessions)

	if !strings.Contains(scope, "attempted 2 time(s)") {
		t.Error("scope should indicate number of attempts")
	}
	if !strings.Contains(scope, "API not available") {
		t.Error("scope should include blocker info")
	}
	if !strings.Contains(scope, "Tried REST approach") {
		t.Error("scope should include decision info")
	}
	if !strings.Contains(scope, "alternative approach") {
		t.Error("scope should suggest alternative approach")
	}
}

func TestBuildReplanScope_Feature(t *testing.T) {
	feature := &models.Feature{
		Number: 1,
		Title:  "Backend API",
		Tasks: []models.Task{
			{FeatureNumber: 1, TaskNumber: 1, Title: "Setup server", Status: models.StatusCompleted},
			{FeatureNumber: 1, TaskNumber: 2, Title: "Add routes", Status: models.StatusPending},
			{FeatureNumber: 1, TaskNumber: 3, Title: "Auth", Status: models.StatusFailed},
		},
	}
	target := ReplanTarget{
		Type:       "feature",
		FeatureNum: 1,
		Feature:    feature,
	}
	sessions := map[string][]models.SessionProgress{
		"1.3": {{SessionNumber: 1, Status: "failed", Blockers: "Wrong auth library"}},
	}

	scope := BuildReplanScope(target, sessions)

	if !strings.Contains(scope, "Feature 1") {
		t.Error("scope should reference the feature")
	}
	if !strings.Contains(scope, "Setup server [completed]") {
		t.Error("scope should list completed tasks")
	}
	if !strings.Contains(scope, "MUST be preserved") {
		t.Error("scope should instruct to preserve completed tasks")
	}
	if !strings.Contains(scope, "Add routes") {
		t.Error("scope should list pending tasks")
	}
	if !strings.Contains(scope, "Wrong auth library") {
		t.Error("scope should include session history for feature tasks")
	}
}

func TestBuildReplanScope_FeatureAllCompleted(t *testing.T) {
	feature := &models.Feature{
		Number: 1,
		Title:  "Done Feature",
		Tasks: []models.Task{
			{FeatureNumber: 1, TaskNumber: 1, Title: "Task A", Status: models.StatusCompleted},
		},
	}
	target := ReplanTarget{Type: "feature", FeatureNum: 1, Feature: feature}
	sessions := make(map[string][]models.SessionProgress)

	scope := BuildReplanScope(target, sessions)

	if !strings.Contains(scope, "completed") {
		t.Error("scope should mention completed tasks")
	}
}

// --- Session history formatting tests ---

func TestFormatSessionHistory(t *testing.T) {
	sessions := []models.SessionProgress{
		{
			SessionNumber: 1,
			Status:        "failed",
			ChangesMade:   []string{"main.go", "routes.go"},
			Decisions:     "Used REST approach",
			Blockers:      "API timeout issues",
			Next:          "Try gRPC instead",
		},
	}

	result := formatSessionHistory(sessions)

	if !strings.Contains(result, "Session 001") {
		t.Error("should contain session number")
	}
	if !strings.Contains(result, "failed") {
		t.Error("should contain status")
	}
	if !strings.Contains(result, "main.go") {
		t.Error("should contain changes")
	}
	if !strings.Contains(result, "Used REST approach") {
		t.Error("should contain decisions")
	}
	if !strings.Contains(result, "API timeout issues") {
		t.Error("should contain blockers")
	}
	if !strings.Contains(result, "Try gRPC instead") {
		t.Error("should contain next steps")
	}
}

func TestFormatSessionHistory_Empty(t *testing.T) {
	result := formatSessionHistory(nil)
	if result != "" {
		t.Errorf("expected empty string for nil sessions, got %q", result)
	}
}
