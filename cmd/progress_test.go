package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cli "github.com/urfave/cli/v2"
)

// minimalPlanFile returns a minimal plan markdown file with one feature and one task.
func minimalPlanFile(status string) string {
	return `# Plan: Test Plan

## Overview

A test plan.

### Task 1: Do the thing [` + status + `]
**Complexity:** small
**Files:** foo.go

Implement the thing.

**Acceptance Criteria:**
- [ ] Thing works
- [ ] Thing is tested
`
}

// setupTestProject creates a temp directory with .etch/plans/ containing a plan file,
// changes to that directory, and returns a cleanup function.
func setupTestProject(t *testing.T, planContent string) string {
	t.Helper()
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })

	os.MkdirAll(filepath.Join(dir, ".etch", "plans"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".etch", "progress"), 0o755)

	os.WriteFile(
		filepath.Join(dir, ".etch", "plans", "test-plan.md"),
		[]byte(planContent),
		0o644,
	)

	os.Chdir(dir)
	return dir
}

func TestProgressStart_CreatesSessionFile(t *testing.T) {
	dir := setupTestProject(t, minimalPlanFile("pending"))

	app := &cli.App{
		Commands: []*cli.Command{progressCmd()},
	}

	err := app.Run([]string{"etch", "progress", "start", "-p", "test-plan", "-t", "1.1"})
	if err != nil {
		t.Fatalf("progress start error: %v", err)
	}

	// Check that a progress file was created.
	matches, _ := filepath.Glob(filepath.Join(dir, ".etch", "progress", "test-plan--task-1.1--*.md"))
	if len(matches) == 0 {
		t.Fatal("expected a progress file to be created")
	}

	// Check that the progress file status is in_progress.
	data, _ := os.ReadFile(matches[0])
	content := string(data)
	if !strings.Contains(content, "**Status:** in_progress") {
		t.Error("progress file should have status in_progress")
	}

	// Check that the plan file was updated.
	planData, _ := os.ReadFile(filepath.Join(dir, ".etch", "plans", "test-plan.md"))
	planContent := string(planData)
	if !strings.Contains(planContent, "[in_progress]") {
		t.Error("plan file should have task status updated to in_progress")
	}
}

func TestProgressStart_ReusesExistingSession(t *testing.T) {
	dir := setupTestProject(t, minimalPlanFile("pending"))

	app := &cli.App{
		Commands: []*cli.Command{progressCmd()},
	}

	// Start twice — should reuse the session file, not create a second one.
	app.Run([]string{"etch", "progress", "start", "-p", "test-plan", "-t", "1.1"})
	err := app.Run([]string{"etch", "progress", "start", "-p", "test-plan", "-t", "1.1"})
	if err != nil {
		t.Fatalf("second progress start error: %v", err)
	}

	matches, _ := filepath.Glob(filepath.Join(dir, ".etch", "progress", "test-plan--task-1.1--*.md"))
	if len(matches) != 1 {
		t.Errorf("expected 1 progress file, got %d", len(matches))
	}
}

func TestProgressUpdate_AppendsMessage(t *testing.T) {
	dir := setupTestProject(t, minimalPlanFile("pending"))

	app := &cli.App{
		Commands: []*cli.Command{progressCmd()},
	}

	// Start first to create the session file.
	app.Run([]string{"etch", "progress", "start", "-p", "test-plan", "-t", "1.1"})

	// Then update.
	err := app.Run([]string{"etch", "progress", "update", "-p", "test-plan", "-t", "1.1", "-m", "Added foo.go"})
	if err != nil {
		t.Fatalf("progress update error: %v", err)
	}

	matches, _ := filepath.Glob(filepath.Join(dir, ".etch", "progress", "test-plan--task-1.1--*.md"))
	data, _ := os.ReadFile(matches[0])
	content := string(data)
	if !strings.Contains(content, "Added foo.go") {
		t.Error("progress file should contain the update message")
	}
}

func TestProgressUpdate_ErrorsWithoutSession(t *testing.T) {
	setupTestProject(t, minimalPlanFile("pending"))

	app := &cli.App{
		Commands: []*cli.Command{progressCmd()},
	}

	err := app.Run([]string{"etch", "progress", "update", "-p", "test-plan", "-t", "1.1", "-m", "test"})
	if err == nil {
		t.Fatal("expected error when no session exists")
	}
}

func TestProgressDone_CompletesTask(t *testing.T) {
	dir := setupTestProject(t, minimalPlanFile("in_progress"))

	app := &cli.App{
		Commands: []*cli.Command{progressCmd()},
	}

	// Start first to create a session file.
	app.Run([]string{"etch", "progress", "start", "-p", "test-plan", "-t", "1.1"})

	err := app.Run([]string{"etch", "progress", "done", "-p", "test-plan", "-t", "1.1"})
	if err != nil {
		t.Fatalf("progress done error: %v", err)
	}

	// Plan file should be updated.
	planData, _ := os.ReadFile(filepath.Join(dir, ".etch", "plans", "test-plan.md"))
	if !strings.Contains(string(planData), "[completed]") {
		t.Error("plan file should have task status completed")
	}

	// Progress file should be updated.
	matches, _ := filepath.Glob(filepath.Join(dir, ".etch", "progress", "test-plan--task-1.1--*.md"))
	if len(matches) > 0 {
		data, _ := os.ReadFile(matches[0])
		if !strings.Contains(string(data), "**Status:** completed") {
			t.Error("progress file should have status completed")
		}
	}
}

func TestProgressCriteria_ChecksOff(t *testing.T) {
	dir := setupTestProject(t, minimalPlanFile("in_progress"))

	app := &cli.App{
		Commands: []*cli.Command{progressCmd()},
	}

	// Start first.
	app.Run([]string{"etch", "progress", "start", "-p", "test-plan", "-t", "1.1"})

	err := app.Run([]string{"etch", "progress", "criteria", "-p", "test-plan", "-t", "1.1", "--check", "Thing works"})
	if err != nil {
		t.Fatalf("progress criteria error: %v", err)
	}

	// Plan file should have the criterion checked.
	planData, _ := os.ReadFile(filepath.Join(dir, ".etch", "plans", "test-plan.md"))
	if !strings.Contains(string(planData), "- [x] Thing works") {
		t.Error("plan file should have criterion checked off")
	}

	// Progress file should also have it checked.
	matches, _ := filepath.Glob(filepath.Join(dir, ".etch", "progress", "test-plan--task-1.1--*.md"))
	if len(matches) > 0 {
		data, _ := os.ReadFile(matches[0])
		if !strings.Contains(string(data), "- [x] Thing works") {
			t.Error("progress file should have criterion checked off")
		}
	}
}

func TestProgressCriteria_SubstringMatch(t *testing.T) {
	dir := setupTestProject(t, minimalPlanFile("in_progress"))

	app := &cli.App{
		Commands: []*cli.Command{progressCmd()},
	}

	app.Run([]string{"etch", "progress", "start", "-p", "test-plan", "-t", "1.1"})

	// Use a substring that should match "Thing is tested".
	err := app.Run([]string{"etch", "progress", "criteria", "-p", "test-plan", "-t", "1.1", "--check", "is tested"})
	if err != nil {
		t.Fatalf("progress criteria error: %v", err)
	}

	planData, _ := os.ReadFile(filepath.Join(dir, ".etch", "plans", "test-plan.md"))
	if !strings.Contains(string(planData), "- [x] Thing is tested") {
		t.Error("plan file should have criterion checked via substring match")
	}
}

func TestProgressBlock_BlocksTask(t *testing.T) {
	dir := setupTestProject(t, minimalPlanFile("in_progress"))

	app := &cli.App{
		Commands: []*cli.Command{progressCmd()},
	}

	app.Run([]string{"etch", "progress", "start", "-p", "test-plan", "-t", "1.1"})

	err := app.Run([]string{"etch", "progress", "block", "-p", "test-plan", "-t", "1.1", "--reason", "Waiting on API"})
	if err != nil {
		t.Fatalf("progress block error: %v", err)
	}

	// Plan file should be blocked.
	planData, _ := os.ReadFile(filepath.Join(dir, ".etch", "plans", "test-plan.md"))
	if !strings.Contains(string(planData), "[blocked]") {
		t.Error("plan file should have task status blocked")
	}

	// Progress file should be blocked with reason.
	matches, _ := filepath.Glob(filepath.Join(dir, ".etch", "progress", "test-plan--task-1.1--*.md"))
	data, _ := os.ReadFile(matches[0])
	content := string(data)
	if !strings.Contains(content, "**Status:** blocked") {
		t.Error("progress file should have status blocked")
	}
	if !strings.Contains(content, "Waiting on API") {
		t.Error("progress file should contain block reason")
	}
}

func TestProgressFail_FailsTask(t *testing.T) {
	dir := setupTestProject(t, minimalPlanFile("in_progress"))

	app := &cli.App{
		Commands: []*cli.Command{progressCmd()},
	}

	app.Run([]string{"etch", "progress", "start", "-p", "test-plan", "-t", "1.1"})

	err := app.Run([]string{"etch", "progress", "fail", "-p", "test-plan", "-t", "1.1", "--reason", "Tests broken"})
	if err != nil {
		t.Fatalf("progress fail error: %v", err)
	}

	// Plan file should be failed.
	planData, _ := os.ReadFile(filepath.Join(dir, ".etch", "plans", "test-plan.md"))
	if !strings.Contains(string(planData), "[failed]") {
		t.Error("plan file should have task status failed")
	}

	// Progress file should be failed with reason.
	matches, _ := filepath.Glob(filepath.Join(dir, ".etch", "progress", "test-plan--task-1.1--*.md"))
	data, _ := os.ReadFile(matches[0])
	content := string(data)
	if !strings.Contains(content, "**Status:** failed") {
		t.Error("progress file should have status failed")
	}
	if !strings.Contains(content, "Tests broken") {
		t.Error("progress file should contain failure reason")
	}
}

func TestProgressStart_NoTask(t *testing.T) {
	setupTestProject(t, minimalPlanFile("pending"))

	app := &cli.App{
		Commands: []*cli.Command{progressCmd()},
	}

	err := app.Run([]string{"etch", "progress", "start"})
	if err == nil {
		t.Fatal("expected error with no --task flag")
	}
}
