package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
