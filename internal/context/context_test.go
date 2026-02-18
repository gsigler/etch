package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gsigler/etch/internal/models"
)

// writePlanFile writes a plan markdown file into the temp dir's .etch/plans/.
func writePlanFile(t *testing.T, dir, slug, content string) string {
	t.Helper()
	planDir := filepath.Join(dir, ".etch", "plans")
	os.MkdirAll(planDir, 0o755)
	path := filepath.Join(planDir, slug+".md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing plan file: %v", err)
	}
	return path
}

// writeProgressFile creates a progress file in the temp dir.
func writeProgressFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	progDir := filepath.Join(dir, ".etch", "progress")
	os.MkdirAll(progDir, 0o755)
	path := filepath.Join(progDir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing progress file: %v", err)
	}
}

const multiFeaturePlan = `# Plan: Auth System

## Overview

Build authentication for the API. Uses JWT tokens with refresh flow. Supports OAuth providers.

---

## Feature 1: Token Management

### Task 1.1: Create token service [completed]
**Complexity:** small
**Files:** internal/token/token.go
Build the token signing and verification service.

- [x] Token signing works
- [x] Token verification works

### Task 1.2: Token refresh endpoint [pending]
**Complexity:** medium
**Files:** internal/api/refresh.go
**Depends on:** Task 1.1
Implement POST /auth/refresh.

- [ ] Endpoint returns new token
- [ ] Old token invalidated

---

## Feature 2: Login Endpoints

### Task 2.1: Registration [pending]
**Complexity:** medium
**Files:** internal/api/register.go
**Depends on:** Task 1.1
Implement POST /auth/register.

- [ ] User can register
- [ ] Duplicate email rejected

### Task 2.2: Login [pending]
**Complexity:** large
**Files:** internal/api/login.go
**Depends on:** Task 2.1
Implement POST /auth/login.

- [ ] User can login
- [ ] Invalid password rejected
`

const singleFeaturePlan = `# Plan: Add Rate Limiting

## Overview

Add rate limiting to all API endpoints.

### Task 1: Design rate limiter [completed]
**Complexity:** small
Choose algorithm and storage.

- [x] Algorithm chosen

### Task 2: Implement middleware [pending]
**Complexity:** medium
**Files:** internal/middleware/ratelimit.go
**Depends on:** Task 1
Wire up the rate limiter as HTTP middleware.

- [ ] Middleware works
- [ ] Rate limits configurable

### Task 3: Add tests [pending]
**Complexity:** small
**Depends on:** Task 2
Integration tests for rate limiting.

- [ ] Tests pass
`

func TestAssemble_TemplateStructure(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, "auth-system", multiFeaturePlan)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	plan := plans[0]
	task := plan.TaskByID("1.2")
	if task == nil {
		t.Fatal("task 1.2 not found")
	}

	result, err := Assemble(dir, plan, task)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	content, err := os.ReadFile(result.ContextPath)
	if err != nil {
		t.Fatalf("reading context file: %v", err)
	}
	ctx := string(content)

	// Check required template sections.
	requiredStrings := []string{
		"# Etch Context — Implementation Task",
		"You are working on a task as part of an implementation plan managed by Etch.",
		"## Plan: Auth System",
		"## Current Plan State",
		"Feature 1: Token Management",
		"Feature 2: Login Endpoints",
		"## Your Task: Task 1.2 — Token refresh endpoint",
		"**Complexity:** medium",
		"**Files in Scope:** internal/api/refresh.go",
		"**Depends on:**",
		"### Acceptance Criteria",
		"- [ ] Endpoint returns new token",
		"- [ ] Old token invalidated",
		"### Previous Sessions",
		"None — this is session 001.",
		"## Reporting Progress",
		"etch progress",
		"Do NOT modify the plan file directly",
	}

	for _, s := range requiredStrings {
		if !strings.Contains(ctx, s) {
			t.Errorf("context missing required string: %q", s)
		}
	}

	// Check task annotation for current task.
	if !strings.Contains(ctx, "(in_progress — this is your task)") {
		t.Error("current task should be annotated as 'this is your task'")
	}

	// Check completed task icon.
	if !strings.Contains(ctx, "✓ Task 1.1: Create token service (completed)") {
		t.Error("completed task should have ✓ icon and (completed) annotation")
	}

	// Check pending task with deps.
	if !strings.Contains(ctx, "○ Task 2.2: Login (pending, depends on 2.1)") {
		t.Error("pending task with deps should show dependency IDs")
	}
}

func TestAssemble_WritesFiles(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, "auth-system", multiFeaturePlan)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	plan := plans[0]
	task := plan.TaskByID("1.2")

	result, err := Assemble(dir, plan, task)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	// Context file should exist.
	if _, err := os.Stat(result.ContextPath); os.IsNotExist(err) {
		t.Error("context file was not created")
	}

	// Progress file should exist.
	if _, err := os.Stat(result.ProgressPath); os.IsNotExist(err) {
		t.Error("progress file was not created")
	}

	// Context file should be in .etch/context/.
	if !strings.Contains(result.ContextPath, ".etch/context/") {
		t.Errorf("context path = %q, want it under .etch/context/", result.ContextPath)
	}

	// Filename format.
	base := filepath.Base(result.ContextPath)
	if base != "auth-system--task-1.2--001.md" {
		t.Errorf("context filename = %q, want auth-system--task-1.2--001.md", base)
	}
}

func TestAssemble_TokenEstimate(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, "auth-system", multiFeaturePlan)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	plan := plans[0]
	task := plan.TaskByID("1.2")

	result, err := Assemble(dir, plan, task)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	// Token estimate should be approximately chars / 3.5.
	content, _ := os.ReadFile(result.ContextPath)
	expectedEstimate := len(content) * 10 / 35
	if result.TokenEstimate != expectedEstimate {
		t.Errorf("token estimate = %d, want %d", result.TokenEstimate, expectedEstimate)
	}
	// Should be a reasonable number (not zero).
	if result.TokenEstimate < 100 {
		t.Errorf("token estimate = %d, seems too low", result.TokenEstimate)
	}
}

func TestAutoSelect_PicksNextPending(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, "auth-system", multiFeaturePlan)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	// Task 1.1 is completed. Tasks 1.2 and 2.1 depend on 1.1 (completed).
	// Both should be eligible. Auto-select should pick the first one: 1.2.
	plan, task, err := ResolveTask(plans, "", "", dir)
	if err != nil {
		t.Fatalf("ResolveTask: %v", err)
	}

	if plan.Slug != "auth-system" {
		t.Errorf("plan = %q, want auth-system", plan.Slug)
	}
	if task.FullID() != "1.2" {
		t.Errorf("task = %q, want 1.2 (first pending with satisfied deps)", task.FullID())
	}
}

func TestAutoSelect_RespectsDepOrder(t *testing.T) {
	// Plan where task 2.1 depends on 1.2, which is still pending.
	dir := t.TempDir()
	planContent := `# Plan: Dep Test

## Feature 1: Core

### Task 1.1: First [completed]
Done.

### Task 1.2: Second [pending]
**Depends on:** Task 1.1
Do second.

### Task 1.3: Third [pending]
**Depends on:** Task 1.2
Do third.
`
	writePlanFile(t, dir, "dep-test", planContent)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	_, task, err := ResolveTask(plans, "", "", dir)
	if err != nil {
		t.Fatalf("ResolveTask: %v", err)
	}

	if task.FullID() != "1.2" {
		t.Errorf("task = %q, want 1.2 (1.3 has unsatisfied dep)", task.FullID())
	}
}

func TestAutoSelect_AllCompleted(t *testing.T) {
	dir := t.TempDir()
	planContent := `# Plan: All Done

## Feature 1: Core

### Task 1.1: Only task [completed]
Done.
`
	writePlanFile(t, dir, "all-done", planContent)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	_, _, err = ResolveTask(plans, "", "", dir)
	if err == nil {
		t.Error("expected error when all tasks completed, got nil")
	}
}

func TestResolveTask_FullID(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, "auth-system", multiFeaturePlan)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	plan, task, err := ResolveTask(plans, "", "2.1", dir)
	if err != nil {
		t.Fatalf("ResolveTask: %v", err)
	}
	if plan.Slug != "auth-system" {
		t.Errorf("plan = %q", plan.Slug)
	}
	if task.FullID() != "2.1" {
		t.Errorf("task = %q, want 2.1", task.FullID())
	}
}

func TestResolveTask_BareNumber_SingleFeature(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, "rate-limiting", singleFeaturePlan)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	_, task, err := ResolveTask(plans, "", "2", dir)
	if err != nil {
		t.Fatalf("ResolveTask: %v", err)
	}
	if task.FullID() != "1.2" {
		t.Errorf("task = %q, want 1.2", task.FullID())
	}
}

func TestResolveTask_PlanSlugAndTaskID(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, "auth-system", multiFeaturePlan)
	writePlanFile(t, dir, "rate-limiting", singleFeaturePlan)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	plan, task, err := ResolveTask(plans, "auth-system", "1.2", dir)
	if err != nil {
		t.Fatalf("ResolveTask: %v", err)
	}
	if plan.Slug != "auth-system" {
		t.Errorf("plan = %q, want auth-system", plan.Slug)
	}
	if task.FullID() != "1.2" {
		t.Errorf("task = %q", task.FullID())
	}
}

func TestResolveTask_InvalidTaskID(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, "auth-system", multiFeaturePlan)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	_, _, err = ResolveTask(plans, "", "9.9", dir)
	if err == nil {
		t.Error("expected error for invalid task ID, got nil")
	}
}

func TestResolveTask_InvalidPlanSlug(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, "auth-system", multiFeaturePlan)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	_, _, err = ResolveTask(plans, "nonexistent", "1.1", dir)
	if err == nil {
		t.Error("expected error for invalid plan slug, got nil")
	}
}

func TestAssemble_PreviousSessions(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, "auth-system", multiFeaturePlan)

	// Create a prior progress file for task 1.2.
	writeProgressFile(t, dir, "auth-system--task-1.2--001.md", `# Session: Task 1.2 – Token refresh endpoint
**Plan:** auth-system
**Task:** 1.2
**Session:** 001
**Started:** 2026-02-15 10:00
**Status:** partial

## Changes Made
- internal/api/refresh.go

## Acceptance Criteria Updates
- [x] Endpoint returns new token
- [ ] Old token invalidated

## Decisions & Notes
Using rotating refresh tokens.

## Blockers
Need to decide on token expiry.

## Next
Implement token invalidation.
`)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	plan := plans[0]
	task := plan.TaskByID("1.2")

	result, err := Assemble(dir, plan, task)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	content, _ := os.ReadFile(result.ContextPath)
	ctx := string(content)

	// Should be session 002.
	if result.SessionNum != 2 {
		t.Errorf("session = %d, want 2", result.SessionNum)
	}

	// Should include previous session summary.
	if !strings.Contains(ctx, "**Session 001") {
		t.Error("context should include previous session header")
	}
	if !strings.Contains(ctx, "Changes: internal/api/refresh.go") {
		t.Error("context should include previous session changes")
	}
	if !strings.Contains(ctx, "Decisions: Using rotating refresh tokens.") {
		t.Error("context should include previous session decisions")
	}
	if !strings.Contains(ctx, "Blockers: Need to decide on token expiry.") {
		t.Error("context should include previous session blockers")
	}
	if !strings.Contains(ctx, "Next: Implement token invalidation.") {
		t.Error("context should include previous session next")
	}

	// Criteria should reflect progress (endpoint returns new token should be checked).
	if !strings.Contains(ctx, "- [x] Endpoint returns new token") {
		t.Error("criteria should merge progress updates")
	}

	// Should NOT say "this is session 001".
	if strings.Contains(ctx, "None — this is session 001.") {
		t.Error("should not say session 001 when prior sessions exist")
	}
}

func TestAssemble_PrerequisiteSummaries(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, "auth-system", multiFeaturePlan)

	// Create a completed progress file for task 1.1 (which is a dep of 1.2).
	writeProgressFile(t, dir, "auth-system--task-1.1--001.md", `# Session: Task 1.1 – Create token service
**Plan:** auth-system
**Task:** 1.1
**Session:** 001
**Started:** 2026-02-14 09:00
**Status:** completed

## Changes Made
- internal/token/token.go
- internal/token/token_test.go

## Acceptance Criteria Updates
- [x] Token signing works
- [x] Token verification works

## Decisions & Notes
Using Ed25519 for signing.

## Blockers

## Next
`)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	plan := plans[0]
	task := plan.TaskByID("1.2")

	result, err := Assemble(dir, plan, task)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	content, _ := os.ReadFile(result.ContextPath)
	ctx := string(content)

	// Should include completed prerequisite section.
	if !strings.Contains(ctx, "### Completed Prerequisites") {
		t.Error("context should include completed prerequisites section")
	}
	if !strings.Contains(ctx, "**Task 1.1 (Create token service):**") {
		t.Error("context should include prerequisite task details")
	}
	if !strings.Contains(ctx, "Using Ed25519 for signing") {
		t.Error("context should include prerequisite decisions")
	}
}

func TestAssemble_FirstTask_NoPrereqs(t *testing.T) {
	dir := t.TempDir()
	planContent := `# Plan: Simple Plan

## Overview

A simple plan.

## Feature 1: Core

### Task 1.1: First task [pending]
**Complexity:** small
Do the first thing.

- [ ] Thing done
`
	writePlanFile(t, dir, "simple-plan", planContent)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	plan := plans[0]
	task := plan.TaskByID("1.1")

	result, err := Assemble(dir, plan, task)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	content, _ := os.ReadFile(result.ContextPath)
	ctx := string(content)

	// No prerequisites section for first task.
	if strings.Contains(ctx, "### Completed Prerequisites") {
		t.Error("first task should not have completed prerequisites section")
	}

	// Should say session 001.
	if !strings.Contains(ctx, "None — this is session 001.") {
		t.Error("first task should say session 001")
	}
}

func TestAssemble_MultiplePriorSessions(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, "auth-system", multiFeaturePlan)

	// Two prior sessions for task 1.2.
	writeProgressFile(t, dir, "auth-system--task-1.2--001.md", `# Session: Task 1.2 – Token refresh endpoint
**Plan:** auth-system
**Task:** 1.2
**Session:** 001
**Started:** 2026-02-15 10:00
**Status:** partial

## Changes Made
- internal/api/refresh.go

## Acceptance Criteria Updates
- [ ] Endpoint returns new token
- [ ] Old token invalidated

## Decisions & Notes
Started scaffolding.

## Blockers

## Next
Continue implementation.
`)
	writeProgressFile(t, dir, "auth-system--task-1.2--002.md", `# Session: Task 1.2 – Token refresh endpoint
**Plan:** auth-system
**Task:** 1.2
**Session:** 002
**Started:** 2026-02-15 14:00
**Status:** partial

## Changes Made
- internal/api/refresh.go
- internal/api/refresh_test.go

## Acceptance Criteria Updates
- [x] Endpoint returns new token
- [ ] Old token invalidated

## Decisions & Notes
Endpoint working but invalidation not done.

## Blockers

## Next
Add old token invalidation.
`)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	plan := plans[0]
	task := plan.TaskByID("1.2")

	result, err := Assemble(dir, plan, task)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	if result.SessionNum != 3 {
		t.Errorf("session = %d, want 3", result.SessionNum)
	}

	content, _ := os.ReadFile(result.ContextPath)
	ctx := string(content)

	// Both prior sessions should be included.
	if !strings.Contains(ctx, "**Session 001") {
		t.Error("should include session 001")
	}
	if !strings.Contains(ctx, "**Session 002") {
		t.Error("should include session 002")
	}
}

func TestNeedsPlanPicker(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, "auth-system", multiFeaturePlan)
	writePlanFile(t, dir, "rate-limiting", singleFeaturePlan)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	needsPicker, ambiguousPlans := NeedsPlanPicker(plans, dir)
	if !needsPicker {
		t.Error("should need picker with multiple plans having pending tasks")
	}
	if len(ambiguousPlans) != 2 {
		t.Errorf("ambiguous plans = %d, want 2", len(ambiguousPlans))
	}
}

func TestNeedsPlanPicker_SinglePlan(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, "auth-system", multiFeaturePlan)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	needsPicker, _ := NeedsPlanPicker(plans, dir)
	if needsPicker {
		t.Error("should not need picker with single plan")
	}
}

func TestDiscoverPlans_NoPlanDir(t *testing.T) {
	dir := t.TempDir()
	_, err := DiscoverPlans(dir)
	if err == nil {
		t.Error("expected error when no plan dir exists")
	}
}

func TestExtractTaskID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Task 1.2", "1.2"},
		{"Task 1.3b", "1.3b"},
		{"1.2", "1.2"},
		{"Task 1.2 (completed)", "1.2"},
	}
	for _, tt := range tests {
		got := extractTaskID(tt.input)
		if got != tt.want {
			t.Errorf("extractTaskID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCondenseOverview(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"Single sentence.", "Single sentence."},
		{"One. Two. Three.", "One. Two. Three."},
		{"One. Two. Three. Four. Five.", "One. Two. Three."},
	}
	for _, tt := range tests {
		got := condenseOverview(tt.input)
		if got != tt.want {
			t.Errorf("condenseOverview(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSessionNumberFromPath(t *testing.T) {
	tests := []struct {
		path string
		want int
	}{
		{"/foo/auth-system--task-1.2--001.md", 1},
		{"/foo/auth-system--task-1.2--003.md", 3},
		{"bad-path.md", 1},
	}
	for _, tt := range tests {
		got := sessionNumberFromPath(tt.path)
		if got != tt.want {
			t.Errorf("sessionNumberFromPath(%q) = %d, want %d", tt.path, got, tt.want)
		}
	}
}

func TestAssemble_ContextFilePath(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, "auth-system", multiFeaturePlan)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	plan := plans[0]
	task := plan.TaskByID("2.1")

	result, err := Assemble(dir, plan, task)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	expectedName := "auth-system--task-2.1--001.md"
	if filepath.Base(result.ContextPath) != expectedName {
		t.Errorf("context filename = %q, want %q", filepath.Base(result.ContextPath), expectedName)
	}

	// Should be under .etch/context/.
	expectedDir := filepath.Join(dir, ".etch", "context")
	if filepath.Dir(result.ContextPath) != expectedDir {
		t.Errorf("context dir = %q, want %q", filepath.Dir(result.ContextPath), expectedDir)
	}
}

func TestEffectiveStatus_UsesProgress(t *testing.T) {
	task := &models.Task{
		FeatureNumber: 1,
		TaskNumber:    1,
		Status:        models.StatusPending,
	}

	allProgress := map[string][]models.SessionProgress{
		"1.1": {
			{TaskID: "1.1", SessionNumber: 1, Status: "completed"},
		},
	}

	status := effectiveStatus(task, allProgress)
	if status != models.StatusCompleted {
		t.Errorf("effectiveStatus = %q, want completed", status)
	}
}

func TestEffectiveStatus_FallsBackToPlanStatus(t *testing.T) {
	task := &models.Task{
		FeatureNumber: 1,
		TaskNumber:    1,
		Status:        models.StatusCompleted,
	}

	allProgress := map[string][]models.SessionProgress{}

	status := effectiveStatus(task, allProgress)
	if status != models.StatusCompleted {
		t.Errorf("effectiveStatus = %q, want completed", status)
	}
}

func TestAssemble_OverviewCondensed(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, "auth-system", multiFeaturePlan)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	plan := plans[0]
	task := plan.TaskByID("1.2")

	result, err := Assemble(dir, plan, task)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	content, _ := os.ReadFile(result.ContextPath)
	ctx := string(content)

	// The overview is 3 sentences, so should be fully included.
	if !strings.Contains(ctx, "Build authentication for the API.") {
		t.Error("overview should contain first sentence")
	}
}

func TestResolveTask_LetterSuffix(t *testing.T) {
	dir := t.TempDir()
	planContent := `# Plan: Suffix Test

## Feature 1: Core

### Task 1.3: Base task [completed]
Done.

### Task 1.3b: Follow-up [pending]
**Depends on:** Task 1.3
Follow up.
`
	writePlanFile(t, dir, "suffix-test", planContent)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	_, task, err := ResolveTask(plans, "", "1.3b", dir)
	if err != nil {
		t.Fatalf("ResolveTask: %v", err)
	}
	if task.FullID() != "1.3b" {
		t.Errorf("task = %q, want 1.3b", task.FullID())
	}
}

func TestFormatTaskAnnotation(t *testing.T) {
	current := &models.Task{FeatureNumber: 1, TaskNumber: 2}
	allProgress := map[string][]models.SessionProgress{}

	tests := []struct {
		task   *models.Task
		status models.Status
		want   string
	}{
		{
			&models.Task{FeatureNumber: 1, TaskNumber: 2},
			models.StatusPending,
			"(in_progress — this is your task)",
		},
		{
			&models.Task{FeatureNumber: 1, TaskNumber: 1},
			models.StatusCompleted,
			"(completed)",
		},
		{
			&models.Task{FeatureNumber: 2, TaskNumber: 1, DependsOn: []string{"Task 1.2"}},
			models.StatusPending,
			"(pending, depends on 1.2)",
		},
		{
			&models.Task{FeatureNumber: 2, TaskNumber: 2},
			models.StatusPending,
			"(pending)",
		},
	}

	for i, tt := range tests {
		got := formatTaskAnnotation(tt.task, current, tt.status, allProgress)
		if got != tt.want {
			t.Errorf("test %d: formatTaskAnnotation = %q, want %q", i, got, tt.want)
		}
	}
}

func TestAssemble_TaskDescription(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, "auth-system", multiFeaturePlan)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	plan := plans[0]
	task := plan.TaskByID("1.2")

	result, err := Assemble(dir, plan, task)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	content, _ := os.ReadFile(result.ContextPath)
	ctx := string(content)

	if !strings.Contains(ctx, "Implement POST /auth/refresh.") {
		t.Error("context should include full task description")
	}
}

func TestAssemble_InstructionRules(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, dir, "auth-system", multiFeaturePlan)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	plan := plans[0]
	task := plan.TaskByID("1.2")

	result, err := Assemble(dir, plan, task)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	content, _ := os.ReadFile(result.ContextPath)
	ctx := string(content)

	rules := []string{
		"Stay within the files listed in scope.",
		"Do NOT modify the plan file directly",
		"Log updates frequently so future sessions have context.",
	}
	for _, rule := range rules {
		if !strings.Contains(ctx, rule) {
			t.Errorf("context missing rule: %q", rule)
		}
	}
}

func TestAssemble_ReviewComments(t *testing.T) {
	dir := t.TempDir()
	planContent := `# Plan: Comment Test

## Feature 1: Core

### Task 1.1: A task [pending]
Description here.

> 💬 Consider using middleware pattern.

> 💬 Watch out for race conditions.

- [ ] Task done
`
	writePlanFile(t, dir, "comment-test", planContent)

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	plan := plans[0]
	task := plan.TaskByID("1.1")

	result, err := Assemble(dir, plan, task)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	content, _ := os.ReadFile(result.ContextPath)
	ctx := string(content)

	if !strings.Contains(ctx, "### Review Comments") {
		t.Error("context should include review comments section")
	}
	if !strings.Contains(ctx, "Consider using middleware pattern.") {
		t.Error("context should include first comment")
	}
	if !strings.Contains(ctx, "Watch out for race conditions.") {
		t.Error("context should include second comment")
	}
}

func TestAutoSelect_WithProgressOverride(t *testing.T) {
	dir := t.TempDir()
	// Plan where 1.1 is marked pending in plan, but completed via progress.
	planContent := `# Plan: Progress Override

## Feature 1: Core

### Task 1.1: First [pending]
Do first.

### Task 1.2: Second [pending]
**Depends on:** Task 1.1
Do second.
`
	writePlanFile(t, dir, "progress-override", planContent)

	// Mark 1.1 as completed via progress file.
	writeProgressFile(t, dir, "progress-override--task-1.1--001.md", fmt.Sprintf(`# Session: Task 1.1 – First
**Plan:** progress-override
**Task:** 1.1
**Session:** 001
**Started:** 2026-02-15
**Status:** completed

## Changes Made
- done.go

## Acceptance Criteria Updates

## Decisions & Notes

## Blockers

## Next
`))

	plans, err := DiscoverPlans(dir)
	if err != nil {
		t.Fatalf("DiscoverPlans: %v", err)
	}

	_, task, err := ResolveTask(plans, "", "", dir)
	if err != nil {
		t.Fatalf("ResolveTask: %v", err)
	}

	// 1.1 is completed via progress, so 1.2 should be auto-selected.
	if task.FullID() != "1.2" {
		t.Errorf("task = %q, want 1.2 (1.1 completed via progress)", task.FullID())
	}
}
