package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupEtchProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create .etch directory structure.
	for _, d := range []string{"plans", "progress", "context"} {
		if err := os.MkdirAll(filepath.Join(dir, ".etch", d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}

func writePlan(t *testing.T, dir, slug, content string) {
	t.Helper()
	path := filepath.Join(dir, ".etch", "plans", slug+".md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeProgress(t *testing.T, dir, filename string) {
	t.Helper()
	path := filepath.Join(dir, ".etch", "progress", filename)
	if err := os.WriteFile(path, []byte("# Session\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func chdirTo(t *testing.T, dir string) {
	t.Helper()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(origDir) })
}

const planAlpha = `# Plan: Alpha Project

## Feature 1: Core
### Task 1.1: Setup [completed]
- Set up the project

### Task 1.2: Build [pending]
- Build things
`

const planBeta = `# Plan: Beta Project

## Feature 1: First
### Task 1.1: One [completed]
- First task

### Task 1.2: Two [completed]
- Second task

## Feature 2: Second
### Task 2.1: Three [pending]
- Third task
`

func TestRunList_MultiplePlans(t *testing.T) {
	dir := setupEtchProject(t)
	chdirTo(t, dir)

	writePlan(t, dir, "alpha", planAlpha)
	writePlan(t, dir, "beta", planBeta)

	output := captureStdout(t, func() {
		err := runList(false)
		if err != nil {
			t.Fatalf("runList() error: %v", err)
		}
	})

	// Should show both plans.
	if !strings.Contains(output, "Alpha Project") {
		t.Errorf("expected Alpha Project in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Beta Project") {
		t.Errorf("expected Beta Project in output, got:\n%s", output)
	}

	// Alpha: 1/2 tasks, 50%
	if !strings.Contains(output, "1/2 tasks") {
		t.Errorf("expected '1/2 tasks' for Alpha, got:\n%s", output)
	}
	if !strings.Contains(output, "50%") {
		t.Errorf("expected '50%%' for Alpha, got:\n%s", output)
	}

	// Beta: 2/3 tasks, 66%
	if !strings.Contains(output, "2/3 tasks") {
		t.Errorf("expected '2/3 tasks' for Beta, got:\n%s", output)
	}
	if !strings.Contains(output, "66%") {
		t.Errorf("expected '66%%' for Beta, got:\n%s", output)
	}
}

func TestRunList_NoPlans(t *testing.T) {
	dir := setupEtchProject(t)
	chdirTo(t, dir)

	err := runList(false)
	if err == nil {
		t.Fatal("expected error when no plans exist")
	}
	if !strings.Contains(err.Error(), "no plans") {
		t.Errorf("expected 'no plans' error, got: %v", err)
	}
}

const planDone = `# Plan: Done Project

## Feature 1: Core
### Task 1.1: Setup [completed]
- Set up the project
`

func TestRunList_ShowsSlugs(t *testing.T) {
	dir := setupEtchProject(t)
	chdirTo(t, dir)

	writePlan(t, dir, "alpha", planAlpha)

	output := captureStdout(t, func() {
		if err := runList(false); err != nil {
			t.Fatalf("runList() error: %v", err)
		}
	})

	if !strings.Contains(output, "alpha") {
		t.Errorf("expected slug 'alpha' in output, got:\n%s", output)
	}
}

func TestRunList_HidesCompleted(t *testing.T) {
	dir := setupEtchProject(t)
	chdirTo(t, dir)

	writePlan(t, dir, "alpha", planAlpha)
	writePlan(t, dir, "done-project", planDone)

	// Default: hide completed plans.
	output := captureStdout(t, func() {
		if err := runList(false); err != nil {
			t.Fatalf("runList() error: %v", err)
		}
	})
	if strings.Contains(output, "Done Project") {
		t.Errorf("expected completed plan to be hidden, got:\n%s", output)
	}
	if !strings.Contains(output, "Alpha Project") {
		t.Errorf("expected incomplete plan to appear, got:\n%s", output)
	}

	// --all: show completed plans.
	output = captureStdout(t, func() {
		if err := runList(true); err != nil {
			t.Fatalf("runList() error: %v", err)
		}
	})
	if !strings.Contains(output, "Done Project") {
		t.Errorf("expected completed plan with --all, got:\n%s", output)
	}
}

func TestRunDelete_RemovesPlanAndProgress(t *testing.T) {
	dir := setupEtchProject(t)
	chdirTo(t, dir)

	writePlan(t, dir, "alpha", planAlpha)
	writeProgress(t, dir, "alpha--task-1.1--001.md")
	writeProgress(t, dir, "alpha--task-1.2--001.md")

	// Write a context file too.
	os.MkdirAll(filepath.Join(dir, ".etch", "context"), 0o755)
	os.WriteFile(filepath.Join(dir, ".etch", "context", "alpha--task-1.1--001.md"), []byte("ctx"), 0o644)

	err := runDelete("alpha", true) // skip confirmation
	if err != nil {
		t.Fatalf("runDelete() error: %v", err)
	}

	// Plan file should be gone.
	if _, err := os.Stat(filepath.Join(dir, ".etch", "plans", "alpha.md")); !os.IsNotExist(err) {
		t.Error("expected plan file to be deleted")
	}

	// Progress files should be gone.
	matches, _ := filepath.Glob(filepath.Join(dir, ".etch", "progress", "alpha--*.md"))
	if len(matches) != 0 {
		t.Errorf("expected progress files to be deleted, found %d", len(matches))
	}

	// Context files should be gone.
	matches, _ = filepath.Glob(filepath.Join(dir, ".etch", "context", "alpha--*.md"))
	if len(matches) != 0 {
		t.Errorf("expected context files to be deleted, found %d", len(matches))
	}
}

func TestRunDelete_MissingPlan(t *testing.T) {
	dir := setupEtchProject(t)
	chdirTo(t, dir)

	err := runDelete("nonexistent", true)
	if err == nil {
		t.Fatal("expected error for missing plan")
	}
	if !strings.Contains(err.Error(), "plan not found") {
		t.Errorf("expected 'plan not found' error, got: %v", err)
	}
}

func TestRunDelete_Cancelled(t *testing.T) {
	dir := setupEtchProject(t)
	chdirTo(t, dir)

	writePlan(t, dir, "alpha", planAlpha)

	// Provide "n" to stdin.
	r, w, _ := os.Pipe()
	w.WriteString("n\n")
	w.Close()
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	captureStdout(t, func() {
		err := runDelete("alpha", false)
		if err != nil {
			t.Fatalf("runDelete() error: %v", err)
		}
	})

	// Plan should still exist.
	if _, err := os.Stat(filepath.Join(dir, ".etch", "plans", "alpha.md")); err != nil {
		t.Error("expected plan file to still exist after cancellation")
	}
}

func TestFindPlanBySlug_NotFound(t *testing.T) {
	dir := setupEtchProject(t)

	// Write a different plan so DiscoverPlans succeeds.
	writePlan(t, dir, "other", planAlpha)

	_, err := findPlanBySlug(dir, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent plan")
	}
	if !strings.Contains(err.Error(), "plan not found") {
		t.Errorf("expected 'plan not found', got: %v", err)
	}
}
