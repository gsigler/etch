package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gsigler/etch/internal/skill"
)

// setupSkillTest creates a temp dir, cds into it, and creates a .claude
// directory so resolveSkillDir finds it without prompting.
func setupSkillTest(t *testing.T) (dir string, cleanup func()) {
	t.Helper()
	dir = t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	os.Mkdir(filepath.Join(dir, ".claude"), 0o755)
	return dir, func() { os.Chdir(origDir) }
}

func TestSkillInstallCreatesFile(t *testing.T) {
	dir, cleanup := setupSkillTest(t)
	defer cleanup()

	err := runSkillInstall()
	if err != nil {
		t.Fatalf("runSkillInstall() returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".claude", skillSubPath))
	if err != nil {
		t.Fatalf("expected skill file to exist: %v", err)
	}

	if string(data) != skill.Content {
		t.Error("skill file content does not match embedded content")
	}
}

func TestSkillInstallOverwritesExisting(t *testing.T) {
	dir, cleanup := setupSkillTest(t)
	defer cleanup()

	// Create the file with old content
	skillPath := filepath.Join(dir, ".claude", skillSubPath)
	os.MkdirAll(filepath.Dir(skillPath), 0o755)
	os.WriteFile(skillPath, []byte("old content"), 0o644)

	err := runSkillInstall()
	if err != nil {
		t.Fatalf("runSkillInstall() returned error: %v", err)
	}

	data, _ := os.ReadFile(skillPath)
	if string(data) != skill.Content {
		t.Error("install should overwrite with embedded content")
	}
}

func TestSkillInstallCreatesDirectories(t *testing.T) {
	dir, cleanup := setupSkillTest(t)
	defer cleanup()

	err := runSkillInstall()
	if err != nil {
		t.Fatalf("runSkillInstall() returned error: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, ".claude", "skills", "etch-plan"))
	if err != nil {
		t.Fatalf("expected directory to exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected .claude/skills/etch-plan to be a directory")
	}
}

func TestSkillFindsClaudeDirInParent(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	// Create .claude in the parent, cd into a subdirectory
	os.Mkdir(filepath.Join(dir, ".claude"), 0o755)
	sub := filepath.Join(dir, "subdir")
	os.Mkdir(sub, 0o755)
	os.Chdir(sub)

	err := runSkillInstall()
	if err != nil {
		t.Fatalf("runSkillInstall() returned error: %v", err)
	}

	// Should have installed in parent's .claude, not created a new one
	data, err := os.ReadFile(filepath.Join(dir, ".claude", skillSubPath))
	if err != nil {
		t.Fatalf("expected skill file in parent .claude dir: %v", err)
	}
	if string(data) != skill.Content {
		t.Error("skill file content does not match embedded content")
	}

	// Should NOT have created .claude in the subdirectory
	if _, err := os.Stat(filepath.Join(sub, ".claude")); err == nil {
		t.Error("should not have created .claude in subdirectory")
	}
}

func TestSkillNoClaudeDirPromptsUser(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// No .claude dir exists — provide "y" via stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString("y\n")
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	err := runSkillInstall()
	if err != nil {
		t.Fatalf("runSkillInstall() returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".claude", skillSubPath))
	if err != nil {
		t.Fatalf("expected skill file to exist: %v", err)
	}
	if string(data) != skill.Content {
		t.Error("skill file content does not match embedded content")
	}
}

func TestSkillNoClaudeDirUserDeclines(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// No .claude dir — user says no
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString("n\n")
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	err := runSkillInstall()
	if err == nil {
		t.Fatal("expected error when user declines, got nil")
	}

	// Should not have created anything
	if _, statErr := os.Stat(filepath.Join(dir, ".claude")); statErr == nil {
		t.Error("should not have created .claude directory when user declined")
	}
}
