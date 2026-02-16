package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gsigler/etch/internal/models"
)

// --- Comment extraction tests ---

func TestExtractComments_WithComments(t *testing.T) {
	plan := &models.Plan{
		Title: "Test Plan",
		Features: []models.Feature{
			{
				Number: 1,
				Title:  "Feature 1",
				Tasks: []models.Task{
					{
						FeatureNumber: 1,
						TaskNumber:    1,
						Title:         "Task One",
						Comments:      []string{"This should use a different approach"},
					},
					{
						FeatureNumber: 1,
						TaskNumber:    2,
						Title:         "Task Two",
						Comments:      nil,
					},
				},
			},
			{
				Number: 2,
				Title:  "Feature 2",
				Tasks: []models.Task{
					{
						FeatureNumber: 2,
						TaskNumber:    1,
						Title:         "Task Three",
						Comments:      []string{"Add error handling", "Consider edge cases"},
					},
				},
			},
		},
	}

	text, count := ExtractComments(plan)

	if count != 3 {
		t.Errorf("comment count = %d, want 3", count)
	}
	if !strings.Contains(text, "Task 1.1") {
		t.Error("expected Task 1.1 reference")
	}
	if !strings.Contains(text, "This should use a different approach") {
		t.Error("expected first comment")
	}
	if !strings.Contains(text, "Task 2.1") {
		t.Error("expected Task 2.1 reference")
	}
	if !strings.Contains(text, "Add error handling") {
		t.Error("expected second task comment")
	}
	// Task 1.2 has no comments — should not appear.
	if strings.Contains(text, "Task 1.2") {
		t.Error("task with no comments should not appear")
	}
}

func TestExtractComments_NoComments(t *testing.T) {
	plan := &models.Plan{
		Title: "Test Plan",
		Features: []models.Feature{
			{
				Number: 1,
				Title:  "Feature 1",
				Tasks: []models.Task{
					{FeatureNumber: 1, TaskNumber: 1, Title: "Task"},
				},
			},
		},
	}

	_, count := ExtractComments(plan)
	if count != 0 {
		t.Errorf("expected 0 comments, got %d", count)
	}
}

// --- Backup tests ---

func TestBackupPlan_CreatesBackup(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, ".etch", "plans")
	os.MkdirAll(plansDir, 0o755)

	planContent := "# Plan: Test\n\n### Task 1: Do thing [pending]\n**Complexity:** small\n**Files:** test.go\n\nDo it.\n\n**Acceptance Criteria:**\n- [ ] Done\n"
	planPath := filepath.Join(plansDir, "test-plan.md")
	os.WriteFile(planPath, []byte(planContent), 0o644)

	backupPath, err := BackupPlan(planPath, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify backup was created.
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatal("backup file was not created")
	}

	// Verify backup content matches original.
	data, _ := os.ReadFile(backupPath)
	if string(data) != planContent {
		t.Error("backup content does not match original")
	}

	// Verify backup is in the backups directory.
	if !strings.Contains(backupPath, ".etch/backups/") {
		t.Error("backup should be in .etch/backups/")
	}

	// Verify backup filename contains the plan name.
	if !strings.Contains(filepath.Base(backupPath), "test-plan-") {
		t.Errorf("backup filename should contain plan name, got %s", filepath.Base(backupPath))
	}
}

func TestBackupPlan_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, ".etch", "plans")
	os.MkdirAll(plansDir, 0o755)

	planPath := filepath.Join(plansDir, "test.md")
	os.WriteFile(planPath, []byte("# Plan: Test\n"), 0o644)

	backupPath, err := BackupPlan(planPath, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Backups dir should have been created.
	info, err := os.Stat(filepath.Dir(backupPath))
	if err != nil {
		t.Fatal("backups directory was not created")
	}
	if !info.IsDir() {
		t.Fatal("backups path is not a directory")
	}
}

func TestBackupPlan_MissingPlan(t *testing.T) {
	dir := t.TempDir()
	_, err := BackupPlan(filepath.Join(dir, "nonexistent.md"), dir)
	if err == nil {
		t.Error("expected error for missing plan file")
	}
}

// --- Diff tests ---

func TestGenerateDiff_ShowsChanges(t *testing.T) {
	old := "line 1\nline 2\nline 3\n"
	new := "line 1\nline 2 modified\nline 3\n"

	diff := GenerateDiff(old, new)

	// Should contain removed line in red.
	if !strings.Contains(diff, "- line 2") {
		t.Error("diff should show removed line")
	}
	// Should contain added line in green.
	if !strings.Contains(diff, "+ line 2 modified") {
		t.Error("diff should show added line")
	}
	// ANSI red escape.
	if !strings.Contains(diff, "\033[31m") {
		t.Error("diff should contain red ANSI escape for removals")
	}
	// ANSI green escape.
	if !strings.Contains(diff, "\033[32m") {
		t.Error("diff should contain green ANSI escape for additions")
	}
}

func TestGenerateDiff_IdenticalContent(t *testing.T) {
	text := "line 1\nline 2\n"
	diff := GenerateDiff(text, text)

	if diff != "" {
		t.Errorf("expected empty diff for identical content, got %q", diff)
	}
}

func TestGenerateDiff_AddedLines(t *testing.T) {
	old := "line 1\n"
	new := "line 1\nline 2\n"

	diff := GenerateDiff(old, new)

	if !strings.Contains(diff, "+ line 2") {
		t.Error("diff should show added line")
	}
}

func TestGenerateDiff_RemovedLines(t *testing.T) {
	old := "line 1\nline 2\n"
	new := "line 1\n"

	diff := GenerateDiff(old, new)

	if !strings.Contains(diff, "- line 2") {
		t.Error("diff should show removed line")
	}
}

// --- Refine integration tests (mock API) ---

var samplePlanWithComments = `# Plan: Test Feature

## Overview
This is a test plan.

### Task 1: Implement feature [pending]
**Complexity:** small
**Files:** main.go

Implement the feature.

> 💬 Should also add error handling

**Acceptance Criteria:**
- [ ] Feature works
- [ ] Tests pass
`

var refinedPlan = `# Plan: Test Feature

## Overview
This is a test plan.

### Task 1: Implement feature [pending]
**Complexity:** small
**Files:** main.go

Implement the feature with proper error handling.

**Acceptance Criteria:**
- [ ] Feature works
- [ ] Tests pass
- [ ] Error handling implemented
`

func TestRefine_Success(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, ".etch", "plans")
	os.MkdirAll(plansDir, 0o755)

	planPath := filepath.Join(plansDir, "test-plan.md")
	os.WriteFile(planPath, []byte(samplePlanWithComments), 0o644)

	client := &mockClient{response: refinedPlan}

	result, err := Refine(client, planPath, dir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify backup was created.
	if result.BackupPath == "" {
		t.Error("expected backup path")
	}
	if _, err := os.Stat(result.BackupPath); os.IsNotExist(err) {
		t.Error("backup file was not created")
	}

	// Verify new plan is valid.
	if result.NewPlan == nil {
		t.Fatal("expected parsed new plan")
	}
	if result.NewPlan.Title != "Test Feature" {
		t.Errorf("plan title = %q, want %q", result.NewPlan.Title, "Test Feature")
	}

	// Verify diff is non-empty (content changed).
	if result.Diff == "" {
		t.Error("expected non-empty diff")
	}

	// Verify old markdown preserved.
	if result.OldMarkdown != samplePlanWithComments {
		t.Error("old markdown should match original file content")
	}
}

func TestRefine_NoComments(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, ".etch", "plans")
	os.MkdirAll(plansDir, 0o755)

	planNoComments := `# Plan: Test Feature

## Overview
This is a test plan.

### Task 1: Implement feature [pending]
**Complexity:** small
**Files:** main.go

Implement the feature.

**Acceptance Criteria:**
- [ ] Feature works
`

	planPath := filepath.Join(plansDir, "test-plan.md")
	os.WriteFile(planPath, []byte(planNoComments), 0o644)

	client := &mockClient{response: "should not be called"}

	_, err := Refine(client, planPath, dir, nil)
	if err == nil {
		t.Fatal("expected error for plan with no comments")
	}
	if !strings.Contains(err.Error(), "no") || !strings.Contains(err.Error(), "comments") {
		t.Errorf("expected 'no comments' error, got: %v", err)
	}

	// Verify no backup was created (we fail before backup for no-comments case... actually we fail before backup).
	backupsDir := filepath.Join(dir, ".etch", "backups")
	if _, err := os.Stat(backupsDir); err == nil {
		entries, _ := os.ReadDir(backupsDir)
		if len(entries) > 0 {
			t.Error("no backup should be created when there are no comments")
		}
	}
}

func TestRefine_InvalidResponse(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, ".etch", "plans")
	os.MkdirAll(plansDir, 0o755)

	planPath := filepath.Join(plansDir, "test-plan.md")
	os.WriteFile(planPath, []byte(samplePlanWithComments), 0o644)

	client := &mockClient{response: "This is not a valid plan at all."}

	_, err := Refine(client, planPath, dir, nil)
	if err == nil {
		t.Fatal("expected error for invalid API response")
	}
	if !strings.Contains(err.Error(), "validation") {
		t.Errorf("expected validation error, got: %v", err)
	}
}

func TestRefine_APIError(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, ".etch", "plans")
	os.MkdirAll(plansDir, 0o755)

	planPath := filepath.Join(plansDir, "test-plan.md")
	os.WriteFile(planPath, []byte(samplePlanWithComments), 0o644)

	client := &mockClient{err: fmt.Errorf("connection refused")}

	_, err := Refine(client, planPath, dir, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRefine_StreamCallback(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, ".etch", "plans")
	os.MkdirAll(plansDir, 0o755)

	planPath := filepath.Join(plansDir, "test-plan.md")
	os.WriteFile(planPath, []byte(samplePlanWithComments), 0o644)

	client := &mockClient{response: refinedPlan}

	var streamed strings.Builder
	_, err := Refine(client, planPath, dir, func(text string) {
		streamed.WriteString(text)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if streamed.Len() == 0 {
		t.Error("expected stream callback to be called")
	}
}

func TestApplyRefinement(t *testing.T) {
	dir := t.TempDir()
	planPath := filepath.Join(dir, "test.md")
	os.WriteFile(planPath, []byte("old content"), 0o644)

	err := ApplyRefinement(planPath, "new content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(planPath)
	if string(data) != "new content" {
		t.Errorf("file content = %q, want %q", string(data), "new content")
	}
}

// --- Prompt construction tests ---

func TestBuildRefineSystemPrompt(t *testing.T) {
	prompt := buildRefineSystemPrompt()

	if !strings.Contains(prompt, "review") {
		t.Error("refine system prompt should mention review")
	}
	if !strings.Contains(prompt, "Plan Format Specification") {
		t.Error("refine system prompt should contain format specification")
	}
	if !strings.Contains(prompt, "Remove addressed comments") {
		t.Error("refine system prompt should instruct to remove addressed comments")
	}
}

func TestBuildRefineUserMessage(t *testing.T) {
	msg := buildRefineUserMessage("# Plan: Test\n", "### Task 1.1\n> 💬 Fix this\n")

	if !strings.Contains(msg, "# Plan: Test") {
		t.Error("user message should contain plan markdown")
	}
	if !strings.Contains(msg, "Fix this") {
		t.Error("user message should contain comments")
	}
	if !strings.Contains(msg, "Current Plan") {
		t.Error("user message should have Current Plan section")
	}
	if !strings.Contains(msg, "Review Comments") {
		t.Error("user message should have Review Comments section")
	}
}
