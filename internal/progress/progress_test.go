package progress

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gsigler/etch/internal/models"
)

func testPlan() *models.Plan {
	return &models.Plan{
		Title: "Auth System",
		Slug:  "auth-system",
	}
}

func testTask(featureNum, taskNum int, suffix, title string, criteria []models.Criterion) *models.Task {
	return &models.Task{
		FeatureNumber: featureNum,
		TaskNumber:    taskNum,
		Suffix:        suffix,
		Title:         title,
		Criteria:      criteria,
	}
}

func TestWriteSession_CorrectFilename(t *testing.T) {
	dir := t.TempDir()
	plan := testPlan()
	task := testTask(1, 1, "", "Database Schema", nil)

	path, err := WriteSession(dir, plan, task)
	if err != nil {
		t.Fatalf("WriteSession() error: %v", err)
	}

	base := filepath.Base(path)
	if base != "auth-system--task-1.1--001.md" {
		t.Errorf("filename = %q, want %q", base, "auth-system--task-1.1--001.md")
	}
}

func TestWriteSession_LetterSuffixTaskID(t *testing.T) {
	dir := t.TempDir()
	plan := testPlan()
	task := testTask(1, 3, "b", "Parser metadata", nil)

	path, err := WriteSession(dir, plan, task)
	if err != nil {
		t.Fatalf("WriteSession() error: %v", err)
	}

	base := filepath.Base(path)
	if base != "auth-system--task-1.3b--001.md" {
		t.Errorf("filename = %q, want %q", base, "auth-system--task-1.3b--001.md")
	}
}

func TestWriteSession_AutoIncrement(t *testing.T) {
	dir := t.TempDir()
	plan := testPlan()
	task := testTask(1, 1, "", "Database Schema", nil)

	path1, err := WriteSession(dir, plan, task)
	if err != nil {
		t.Fatalf("first WriteSession() error: %v", err)
	}
	path2, err := WriteSession(dir, plan, task)
	if err != nil {
		t.Fatalf("second WriteSession() error: %v", err)
	}

	base1 := filepath.Base(path1)
	base2 := filepath.Base(path2)
	if base1 != "auth-system--task-1.1--001.md" {
		t.Errorf("first filename = %q, want 001", base1)
	}
	if base2 != "auth-system--task-1.1--002.md" {
		t.Errorf("second filename = %q, want 002", base2)
	}
}

func TestWriteSession_AtomicCreation(t *testing.T) {
	dir := t.TempDir()
	plan := testPlan()
	task := testTask(2, 1, "", "API Client", nil)

	// Pre-create session 001 to simulate a race condition.
	progDir := filepath.Join(dir, progressDir)
	os.MkdirAll(progDir, 0o755)
	os.WriteFile(filepath.Join(progDir, "auth-system--task-2.1--001.md"), []byte("existing"), 0o644)

	path, err := WriteSession(dir, plan, task)
	if err != nil {
		t.Fatalf("WriteSession() error: %v", err)
	}

	base := filepath.Base(path)
	if base != "auth-system--task-2.1--002.md" {
		t.Errorf("filename = %q, want 002 (should skip existing 001)", base)
	}
}

func TestWriteSession_TemplateContent(t *testing.T) {
	dir := t.TempDir()
	plan := testPlan()
	task := testTask(1, 1, "", "Database Schema", []models.Criterion{
		{Description: "Migration file creates users table", IsMet: false},
		{Description: "User model struct matches schema", IsMet: true},
	})

	path, err := WriteSession(dir, plan, task)
	if err != nil {
		t.Fatalf("WriteSession() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	content := string(data)

	checks := []string{
		"# Session: Task 1.1 – Database Schema",
		"**Plan:** auth-system",
		"**Task:** 1.1",
		"**Session:** 001",
		"**Status:** pending",
		"## Changes Made",
		"## Acceptance Criteria Updates",
		"- [ ] Migration file creates users table",
		"- [x] User model struct matches schema",
		"## Decisions & Notes",
		"## Blockers",
		"## Next",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("template missing %q", check)
		}
	}
}

func TestReadAll_FullyFilledFile(t *testing.T) {
	dir := t.TempDir()
	progDir := filepath.Join(dir, progressDir)
	os.MkdirAll(progDir, 0o755)

	content := `# Session: Task 1.1 – Database Schema
**Plan:** auth-system
**Task:** 1.1
**Session:** 001
**Started:** 2026-02-16 09:30
**Status:** completed

## Changes Made
- internal/db/migration.go
- internal/models/user.go

## Acceptance Criteria Updates
- [x] Migration file creates users table
- [x] User model struct matches schema
- [ ] Migration runs successfully on empty database

## Decisions & Notes
Used pgx for database driver.

## Blockers
None.

## Next
Run integration tests.
`
	os.WriteFile(filepath.Join(progDir, "auth-system--task-1.1--001.md"), []byte(content), 0o644)

	result, err := ReadAll(dir, "auth-system")
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}

	sessions, ok := result["1.1"]
	if !ok {
		t.Fatal("expected task 1.1 in results")
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	sp := sessions[0]
	if sp.TaskID != "1.1" {
		t.Errorf("TaskID = %q, want %q", sp.TaskID, "1.1")
	}
	if sp.SessionNumber != 1 {
		t.Errorf("SessionNumber = %d, want 1", sp.SessionNumber)
	}
	if sp.Status != "completed" {
		t.Errorf("Status = %q, want %q", sp.Status, "completed")
	}
	if sp.Started != "2026-02-16 09:30" {
		t.Errorf("Started = %q, want %q", sp.Started, "2026-02-16 09:30")
	}
	if len(sp.ChangesMade) != 2 {
		t.Errorf("ChangesMade len = %d, want 2", len(sp.ChangesMade))
	}
	if len(sp.CriteriaUpdates) != 3 {
		t.Errorf("CriteriaUpdates len = %d, want 3", len(sp.CriteriaUpdates))
	}
	if sp.CriteriaUpdates[0].IsMet != true {
		t.Error("first criterion should be met")
	}
	if sp.CriteriaUpdates[2].IsMet != false {
		t.Error("third criterion should not be met")
	}
	if sp.Decisions != "Used pgx for database driver." {
		t.Errorf("Decisions = %q", sp.Decisions)
	}
	if sp.Blockers != "None." {
		t.Errorf("Blockers = %q", sp.Blockers)
	}
	if sp.Next != "Run integration tests." {
		t.Errorf("Next = %q", sp.Next)
	}
}

func TestReadAll_PartiallyFilledFile(t *testing.T) {
	dir := t.TempDir()
	progDir := filepath.Join(dir, progressDir)
	os.MkdirAll(progDir, 0o755)

	// A file where the agent only filled in some sections.
	content := `# Session: Task 2.1 – API Client
**Plan:** auth-system
**Task:** 2.1
**Session:** 001
**Started:** 2026-02-16 10:00
**Status:** pending

## Changes Made
<!-- List files created or modified -->

## Acceptance Criteria Updates
- [ ] API client connects successfully

## Decisions & Notes
<!-- Design decisions, important context for future sessions -->

## Blockers
<!-- Anything blocking progress -->

## Next
<!-- What still needs to happen -->
`
	os.WriteFile(filepath.Join(progDir, "auth-system--task-2.1--001.md"), []byte(content), 0o644)

	result, err := ReadAll(dir, "auth-system")
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}

	sp := result["2.1"][0]
	if len(sp.ChangesMade) != 0 {
		t.Errorf("ChangesMade should be empty, got %v", sp.ChangesMade)
	}
	if sp.Decisions != "" {
		t.Errorf("Decisions should be empty, got %q", sp.Decisions)
	}
	if sp.Blockers != "" {
		t.Errorf("Blockers should be empty, got %q", sp.Blockers)
	}
	if len(sp.CriteriaUpdates) != 1 {
		t.Errorf("CriteriaUpdates len = %d, want 1", len(sp.CriteriaUpdates))
	}
}

func TestReadAll_ExtraContent(t *testing.T) {
	dir := t.TempDir()
	progDir := filepath.Join(dir, progressDir)
	os.MkdirAll(progDir, 0o755)

	content := `# Session: Task 1.2 – Models
**Plan:** auth-system
**Task:** 1.2
**Session:** 001
**Started:** 2026-02-16
**Status:** completed

## Changes Made
- internal/models/models.go

## Acceptance Criteria Updates
- [x] All models defined

## Extra Section Added By Agent
This section is not in the standard template.

## Decisions & Notes
Decided to use structs.

## Blockers

## Next
`
	os.WriteFile(filepath.Join(progDir, "auth-system--task-1.2--001.md"), []byte(content), 0o644)

	result, err := ReadAll(dir, "auth-system")
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}

	sp := result["1.2"][0]
	if sp.Status != "completed" {
		t.Errorf("Status = %q, want %q", sp.Status, "completed")
	}
	if sp.Decisions != "Decided to use structs." {
		t.Errorf("Decisions = %q", sp.Decisions)
	}
}

func TestReadAll_Grouping(t *testing.T) {
	dir := t.TempDir()
	progDir := filepath.Join(dir, progressDir)
	os.MkdirAll(progDir, 0o755)

	makeFile := func(taskID string, session int, status string) {
		content := strings.Join([]string{
			"# Session: Task " + taskID,
			"**Plan:** myplan",
			"**Task:** " + taskID,
			"**Session:** " + strings.Repeat("0", 3-len(strings.TrimLeft(string(rune('0'+session)), "0"))) + string(rune('0'+session)),
			"**Status:** " + status,
			"",
			"## Changes Made",
			"",
			"## Acceptance Criteria Updates",
			"",
			"## Decisions & Notes",
			"",
			"## Blockers",
			"",
			"## Next",
		}, "\n")
		// Use simpler approach for session number formatting.
		content = strings.Join([]string{
			"# Session: Task " + taskID,
			"**Plan:** myplan",
			"**Task:** " + taskID,
			fmt.Sprintf("**Session:** %03d", session),
			"**Status:** " + status,
			"",
			"## Changes Made",
			"",
			"## Acceptance Criteria Updates",
			"",
			"## Decisions & Notes",
			"",
			"## Blockers",
			"",
			"## Next",
		}, "\n")
		filename := fmt.Sprintf("myplan--task-%s--%03d.md", taskID, session)
		os.WriteFile(filepath.Join(progDir, filename), []byte(content), 0o644)
	}

	makeFile("1.1", 1, "completed")
	makeFile("1.1", 2, "completed")
	makeFile("1.2", 1, "pending")
	makeFile("1.3b", 1, "in_progress")

	result, err := ReadAll(dir, "myplan")
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(result))
	}
	if len(result["1.1"]) != 2 {
		t.Errorf("task 1.1 should have 2 sessions, got %d", len(result["1.1"]))
	}
	if result["1.1"][0].SessionNumber != 1 || result["1.1"][1].SessionNumber != 2 {
		t.Error("task 1.1 sessions not sorted by session number")
	}
	if len(result["1.2"]) != 1 {
		t.Errorf("task 1.2 should have 1 session, got %d", len(result["1.2"]))
	}
	if len(result["1.3b"]) != 1 {
		t.Errorf("task 1.3b should have 1 session, got %d", len(result["1.3b"]))
	}
}

func TestReadAll_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	// Don't create any progress directory or files.

	result, err := ReadAll(dir, "nonexistent")
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d entries", len(result))
	}
}

func TestReadAll_MalformedFile(t *testing.T) {
	dir := t.TempDir()
	progDir := filepath.Join(dir, progressDir)
	os.MkdirAll(progDir, 0o755)

	// File with no Task ID should be skipped with a warning, not crash.
	content := `# Some random file
**Status:** pending
No task ID here.
`
	os.WriteFile(filepath.Join(progDir, "myplan--task-bad--001.md"), []byte(content), 0o644)

	result, err := ReadAll(dir, "myplan")
	if err != nil {
		t.Fatalf("ReadAll() should not error on malformed files: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("malformed file should be skipped, got %d entries", len(result))
	}
}

// --- FindLatestSessionPath tests ---

func TestFindLatestSessionPath_SingleFile(t *testing.T) {
	dir := t.TempDir()
	progDir := filepath.Join(dir, progressDir)
	os.MkdirAll(progDir, 0o755)

	os.WriteFile(filepath.Join(progDir, "myplan--task-1.1--001.md"), []byte("**Task:** 1.1\n"), 0o644)

	path, num, err := FindLatestSessionPath(dir, "myplan", "1.1")
	if err != nil {
		t.Fatalf("FindLatestSessionPath() error: %v", err)
	}
	if num != 1 {
		t.Errorf("session number = %d, want 1", num)
	}
	if filepath.Base(path) != "myplan--task-1.1--001.md" {
		t.Errorf("path = %q, want myplan--task-1.1--001.md", filepath.Base(path))
	}
}

func TestFindLatestSessionPath_MultipleSessions(t *testing.T) {
	dir := t.TempDir()
	progDir := filepath.Join(dir, progressDir)
	os.MkdirAll(progDir, 0o755)

	os.WriteFile(filepath.Join(progDir, "myplan--task-1.1--001.md"), []byte("**Task:** 1.1\n"), 0o644)
	os.WriteFile(filepath.Join(progDir, "myplan--task-1.1--002.md"), []byte("**Task:** 1.1\n"), 0o644)
	os.WriteFile(filepath.Join(progDir, "myplan--task-1.1--003.md"), []byte("**Task:** 1.1\n"), 0o644)

	path, num, err := FindLatestSessionPath(dir, "myplan", "1.1")
	if err != nil {
		t.Fatalf("FindLatestSessionPath() error: %v", err)
	}
	if num != 3 {
		t.Errorf("session number = %d, want 3", num)
	}
	if filepath.Base(path) != "myplan--task-1.1--003.md" {
		t.Errorf("path = %q, want myplan--task-1.1--003.md", filepath.Base(path))
	}
}

func TestFindLatestSessionPath_NoFiles(t *testing.T) {
	dir := t.TempDir()
	_, _, err := FindLatestSessionPath(dir, "myplan", "1.1")
	if err == nil {
		t.Fatal("expected error for missing session files")
	}
}

func TestFindLatestSessionPath_LetterSuffix(t *testing.T) {
	dir := t.TempDir()
	progDir := filepath.Join(dir, progressDir)
	os.MkdirAll(progDir, 0o755)

	os.WriteFile(filepath.Join(progDir, "myplan--task-1.3b--001.md"), []byte("**Task:** 1.3b\n"), 0o644)

	path, num, err := FindLatestSessionPath(dir, "myplan", "1.3b")
	if err != nil {
		t.Fatalf("FindLatestSessionPath() error: %v", err)
	}
	if num != 1 {
		t.Errorf("session number = %d, want 1", num)
	}
	if filepath.Base(path) != "myplan--task-1.3b--001.md" {
		t.Errorf("path = %q", filepath.Base(path))
	}
}

// --- AppendToSection tests ---

func TestAppendToSection_AddsContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	initial := "# Header\n**Status:** pending\n\n## Changes Made\n<!-- placeholder -->\n\n## Next\nDo stuff\n"
	os.WriteFile(path, []byte(initial), 0o644)

	err := AppendToSection(path, "Changes Made", "- internal/foo.go")
	if err != nil {
		t.Fatalf("AppendToSection() error: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)
	if !strings.Contains(content, "- internal/foo.go") {
		t.Error("expected appended content in file")
	}
}

func TestAppendToSection_AppendsMultiple(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	initial := "# Header\n\n## Changes Made\n<!-- placeholder -->\n\n## Next\n"
	os.WriteFile(path, []byte(initial), 0o644)

	AppendToSection(path, "Changes Made", "- file1.go")
	AppendToSection(path, "Changes Made", "- file2.go")

	data, _ := os.ReadFile(path)
	content := string(data)
	if !strings.Contains(content, "- file1.go") || !strings.Contains(content, "- file2.go") {
		t.Error("expected both appended lines")
	}
}

func TestAppendToSection_MissingSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("# Header\n## Other\n"), 0o644)

	err := AppendToSection(path, "Nonexistent", "content")
	if err == nil {
		t.Fatal("expected error for missing section")
	}
}

// --- UpdateStatus tests ---

func TestUpdateStatus_ChangesStatus(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	initial := "# Session\n**Plan:** myplan\n**Task:** 1.1\n**Session:** 001\n**Status:** pending\n\n## Changes Made\n"
	os.WriteFile(path, []byte(initial), 0o644)

	err := UpdateStatus(path, "in_progress")
	if err != nil {
		t.Fatalf("UpdateStatus() error: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)
	if !strings.Contains(content, "**Status:** in_progress") {
		t.Error("expected status to be updated to in_progress")
	}
	if strings.Contains(content, "**Status:** pending") {
		t.Error("old status should be replaced")
	}
}

func TestUpdateStatus_AllStatuses(t *testing.T) {
	statuses := []string{"in_progress", "completed", "blocked", "failed"}
	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "test.md")
			os.WriteFile(path, []byte("**Status:** pending\n"), 0o644)

			err := UpdateStatus(path, status)
			if err != nil {
				t.Fatalf("UpdateStatus(%q) error: %v", status, err)
			}

			data, _ := os.ReadFile(path)
			expected := "**Status:** " + status
			if !strings.Contains(string(data), expected) {
				t.Errorf("expected %q in file", expected)
			}
		})
	}
}

func TestUpdateStatus_NoStatusLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("# No status here\n"), 0o644)

	err := UpdateStatus(path, "completed")
	if err == nil {
		t.Fatal("expected error when no **Status:** line exists")
	}
}

func TestUpdateStatus_FileNotFound(t *testing.T) {
	err := UpdateStatus("/nonexistent/path.md", "completed")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// --- UpdateCriterion tests ---

func TestUpdateCriterion_ChecksOff(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	initial := "## Acceptance Criteria Updates\n- [ ] First criterion\n- [ ] Second criterion\n"
	os.WriteFile(path, []byte(initial), 0o644)

	err := UpdateCriterion(path, "First criterion")
	if err != nil {
		t.Fatalf("UpdateCriterion() error: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)
	if !strings.Contains(content, "- [x] First criterion") {
		t.Error("expected first criterion to be checked")
	}
	if !strings.Contains(content, "- [ ] Second criterion") {
		t.Error("second criterion should remain unchecked")
	}
}

func TestUpdateCriterion_NotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("- [ ] Some criterion\n"), 0o644)

	err := UpdateCriterion(path, "Nonexistent criterion")
	if err == nil {
		t.Fatal("expected error for unmatched criterion")
	}
}

func TestUpdateCriterion_AlreadyChecked(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("- [x] Already done\n"), 0o644)

	err := UpdateCriterion(path, "Already done")
	if err == nil {
		t.Fatal("expected error since criterion is already checked (no unchecked match)")
	}
}

func TestUpdateCriterion_FileNotFound(t *testing.T) {
	err := UpdateCriterion("/nonexistent/path.md", "anything")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
