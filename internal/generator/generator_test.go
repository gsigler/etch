package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gsigler/etch/internal/api"
)

// --- Slug tests ---

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Add user authentication", "add-user-authentication"},
		{"Hello, World!", "hello-world"},
		{"  spaces   everywhere  ", "spaces-everywhere"},
		{"UPPERCASE", "uppercase"},
		{"special@#$chars", "specialchars"},
		{"a--b--c", "a-b-c"},
		{"", ""},
		{strings.Repeat("a", 100), strings.Repeat("a", 50)},
		{"this-is-a-very-long-slug-that-exceeds-the-maximum-allowed-characters-for-a-slug", "this-is-a-very-long-slug-that-exceeds-the-maximum"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Slugify(tt.input)
			if got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSlugify_MaxLength(t *testing.T) {
	slug := Slugify("word " + strings.Repeat("a", 100))
	if len(slug) > 50 {
		t.Errorf("slug length %d exceeds 50", len(slug))
	}
}

func TestResolveSlug_NoCollision(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".etch", "plans"), 0o755)

	slug, collision := ResolveSlug(dir, "my-plan")
	if collision {
		t.Error("expected no collision")
	}
	if slug != "my-plan" {
		t.Errorf("got %q, want %q", slug, "my-plan")
	}
}

func TestResolveSlug_Collision(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, ".etch", "plans")
	os.MkdirAll(plansDir, 0o755)
	os.WriteFile(filepath.Join(plansDir, "my-plan.md"), []byte("# Plan: Test\n"), 0o644)

	slug, collision := ResolveSlug(dir, "my-plan")
	if !collision {
		t.Error("expected collision")
	}
	if slug != "my-plan-2" {
		t.Errorf("got %q, want %q", slug, "my-plan-2")
	}
}

func TestResolveSlug_MultipleCollisions(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, ".etch", "plans")
	os.MkdirAll(plansDir, 0o755)
	os.WriteFile(filepath.Join(plansDir, "my-plan.md"), []byte("# Plan: Test\n"), 0o644)
	os.WriteFile(filepath.Join(plansDir, "my-plan-2.md"), []byte("# Plan: Test 2\n"), 0o644)

	slug, collision := ResolveSlug(dir, "my-plan")
	if !collision {
		t.Error("expected collision")
	}
	if slug != "my-plan-3" {
		t.Errorf("got %q, want %q", slug, "my-plan-3")
	}
}

// --- Project context tests ---

func TestGatherProjectContext_FileTree(t *testing.T) {
	dir := t.TempDir()
	// Create some project structure.
	os.MkdirAll(filepath.Join(dir, "cmd"), 0o755)
	os.MkdirAll(filepath.Join(dir, "internal", "api"), 0o755)
	os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0o755) // should be excluded
	os.MkdirAll(filepath.Join(dir, ".git", "objects"), 0o755)     // should be excluded
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "cmd", "root.go"), []byte("package cmd\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "internal", "api", "client.go"), []byte("package api\n"), 0o644)

	ctx := gatherProjectContext(dir)

	if !strings.Contains(ctx, "main.go") {
		t.Error("expected main.go in file tree")
	}
	if !strings.Contains(ctx, "cmd/") {
		t.Error("expected cmd/ in file tree")
	}
	if strings.Contains(ctx, "node_modules") {
		t.Error("node_modules should be excluded")
	}
	if strings.Contains(ctx, ".git") {
		t.Error(".git should be excluded")
	}
}

func TestGatherProjectContext_ConfigDetection(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.21\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name": "test"}`+"\n"), 0o644)

	ctx := gatherProjectContext(dir)

	if !strings.Contains(ctx, "go.mod") {
		t.Error("expected go.mod section")
	}
	if !strings.Contains(ctx, "module example.com/test") {
		t.Error("expected go.mod content")
	}
	if !strings.Contains(ctx, "package.json") {
		t.Error("expected package.json section")
	}
}

func TestGatherProjectContext_EmptyProject(t *testing.T) {
	dir := t.TempDir()
	ctx := gatherProjectContext(dir)
	// Should not panic, just return minimal or empty context.
	if ctx == "" {
		// Empty is fine — no files means no context.
		return
	}
}

func TestGatherProjectContext_CLAUDEmd(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Project Notes\nUse bun instead of npm\n"), 0o644)

	ctx := gatherProjectContext(dir)

	if !strings.Contains(ctx, "CLAUDE.md") {
		t.Error("expected CLAUDE.md section")
	}
	if !strings.Contains(ctx, "Use bun instead of npm") {
		t.Error("expected CLAUDE.md content")
	}
}

func TestGatherProjectContext_ExistingPlans(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, ".etch", "plans")
	os.MkdirAll(plansDir, 0o755)
	os.WriteFile(filepath.Join(plansDir, "auth.md"), []byte("# Plan: Add Authentication\n\n### Task 1: Setup [pending]\n**Complexity:** small\n**Files:** auth.go\n\nSetup auth.\n\n**Acceptance Criteria:**\n- [ ] Done\n"), 0o644)

	ctx := gatherProjectContext(dir)

	if !strings.Contains(ctx, "Add Authentication") {
		t.Error("expected existing plan title in context")
	}
}

func TestGatherProjectContext_LargeTree(t *testing.T) {
	dir := t.TempDir()
	// Create many files to trigger truncation.
	for i := 0; i < 2500; i++ {
		os.WriteFile(filepath.Join(dir, fmt.Sprintf("file-%04d.txt", i)), []byte(""), 0o644)
	}

	ctx := gatherProjectContext(dir)

	if !strings.Contains(ctx, "truncated") {
		t.Error("expected truncation notice for large tree")
	}
}

// itoa is a simple int-to-string for test use.
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

// --- Prompt construction tests ---

func TestBuildSystemPrompt_ContainsFormatSpec(t *testing.T) {
	prompt := buildSystemPrompt("small = easy, large = hard")
	if !strings.Contains(prompt, "## Plan Format Specification") {
		t.Error("system prompt should contain format specification")
	}
	if !strings.Contains(prompt, "# Plan:") {
		t.Error("system prompt should reference plan heading format")
	}
	if !strings.Contains(prompt, "small = easy, large = hard") {
		t.Error("system prompt should contain complexity guide")
	}
}

func TestBuildUserMessage(t *testing.T) {
	msg := buildUserMessage("Add auth", "### File Tree\nmain.go\n")
	if !strings.Contains(msg, "Add auth") {
		t.Error("user message should contain description")
	}
	if !strings.Contains(msg, "main.go") {
		t.Error("user message should contain project context")
	}
}

// --- Extract markdown tests ---

func TestExtractMarkdown_FencedBlock(t *testing.T) {
	input := "Here is the plan:\n```markdown\n# Plan: Test\n\n### Task 1: Do thing [pending]\n```\nDone!"
	got := extractMarkdown(input)
	if !strings.HasPrefix(got, "# Plan: Test") {
		t.Errorf("expected plan content, got %q", got)
	}
}

func TestExtractMarkdown_DirectPlan(t *testing.T) {
	input := "# Plan: Test\n\n### Task 1: Do thing [pending]"
	got := extractMarkdown(input)
	if got != input {
		t.Errorf("expected %q, got %q", input, got)
	}
}

func TestExtractMarkdown_PreambleBeforePlan(t *testing.T) {
	input := "Sure, here's your plan:\n\n# Plan: Test\n\n### Task 1: Do thing [pending]"
	got := extractMarkdown(input)
	if !strings.HasPrefix(got, "# Plan: Test") {
		t.Errorf("expected plan to start with heading, got %q", got)
	}
}

// --- Mock API client for generation tests ---

type mockClient struct {
	response string
	err      error
}

func (m *mockClient) SendStream(system, userMessage string, cb api.StreamCallback) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if cb != nil {
		cb(m.response)
	}
	return m.response, nil
}

func TestGenerate_ValidPlan(t *testing.T) {
	dir := t.TempDir()
	planMarkdown := `# Plan: Test Feature

## Overview
This is a test plan.

### Task 1: Implement feature [pending]
**Complexity:** small
**Files:** main.go

Implement the feature.

**Acceptance Criteria:**
- [ ] Feature works
- [ ] Tests pass
`

	client := &mockClient{response: planMarkdown}
	result, err := Generate(client, dir, "test feature", "small = easy", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Plan.Title != "Test Feature" {
		t.Errorf("plan title = %q, want %q", result.Plan.Title, "Test Feature")
	}

	taskCount := 0
	for _, f := range result.Plan.Features {
		taskCount += len(f.Tasks)
	}
	if taskCount != 1 {
		t.Errorf("task count = %d, want 1", taskCount)
	}
}

func TestGenerate_InvalidPlan(t *testing.T) {
	dir := t.TempDir()
	client := &mockClient{response: "This is not a valid plan at all."}

	_, err := Generate(client, dir, "test", "small = easy", nil)
	if err == nil {
		t.Error("expected error for invalid plan")
	}
	if !strings.Contains(err.Error(), "validation") {
		t.Errorf("expected validation error, got: %v", err)
	}
}

func TestGenerate_APIError(t *testing.T) {
	dir := t.TempDir()
	client := &mockClient{err: &api.APIError{StatusCode: 401, Message: "bad key"}}

	_, err := Generate(client, dir, "test", "small = easy", nil)
	if err == nil {
		t.Error("expected error")
	}
}

func TestGenerate_StreamCallback(t *testing.T) {
	dir := t.TempDir()
	planMarkdown := `# Plan: Streamed

### Task 1: Stream test [pending]
**Complexity:** small
**Files:** test.go

Do the thing.

**Acceptance Criteria:**
- [ ] Works
`
	client := &mockClient{response: planMarkdown}

	var streamed strings.Builder
	_, err := Generate(client, dir, "stream test", "small = easy", func(text string) {
		streamed.WriteString(text)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if streamed.Len() == 0 {
		t.Error("expected stream callback to be called")
	}
}

func TestWritePlan(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".etch", "plans"), 0o755)

	path, err := WritePlan(dir, "my-plan", "# Plan: Test\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(dir, ".etch", "plans", "my-plan.md")
	if path != expected {
		t.Errorf("path = %q, want %q", path, expected)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "# Plan: Test\n" {
		t.Errorf("file content = %q", string(data))
	}
}

func TestGenerate_MultiFeaturePlan(t *testing.T) {
	dir := t.TempDir()
	planMarkdown := `# Plan: Multi Feature

## Overview
A plan with multiple features.

---

## Feature 1: Backend

### Task 1.1: Setup API [pending]
**Complexity:** medium
**Files:** api/main.go

Setup the API server.

**Acceptance Criteria:**
- [ ] Server starts

---

## Feature 2: Frontend

### Task 2.1: Create UI [pending]
**Complexity:** small
**Files:** ui/app.tsx
**Depends on:** Task 1.1

Build the UI.

**Acceptance Criteria:**
- [ ] UI renders
`
	client := &mockClient{response: planMarkdown}
	result, err := Generate(client, dir, "multi feature", "small = easy", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Plan.Features) != 2 {
		t.Errorf("feature count = %d, want 2", len(result.Plan.Features))
	}

	summary := result.Summary()
	if !strings.Contains(summary, "2 feature(s)") {
		t.Errorf("summary should mention 2 features: %s", summary)
	}
	if !strings.Contains(summary, "2 task(s)") {
		t.Errorf("summary should mention 2 tasks: %s", summary)
	}
}
