package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInitCreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Provide "n" to stdin for the git tracking question
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString("n\n")
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	err := runInit()
	if err != nil {
		t.Fatalf("runInit() returned error: %v", err)
	}

	expectedDirs := []string{
		".etch/plans",
		".etch/progress",
		".etch/context",
		".etch/backups",
	}

	for _, d := range expectedDirs {
		path := filepath.Join(dir, d)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected directory %s to exist, got error: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", d)
		}
	}
}

func TestRunInitCreatesConfig(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	r, w, _ := os.Pipe()
	w.WriteString("n\n")
	w.Close()
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	err := runInit()
	if err != nil {
		t.Fatalf("runInit() returned error: %v", err)
	}

	configPath := filepath.Join(dir, ".etch", "config.toml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config.toml to exist: %v", err)
	}
	if !strings.Contains(string(data), "[api]") {
		t.Error("config.toml missing [api] section")
	}
}

func TestRunInitGitignoreNoTrackProgress(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	r, w, _ := os.Pipe()
	w.WriteString("n\n")
	w.Close()
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	err := runInit()
	if err != nil {
		t.Fatalf("runInit() returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("expected .gitignore to exist: %v", err)
	}

	content := string(data)
	expectedEntries := []string{
		".etch/progress/",
		".etch/backups/",
		".etch/context/",
		".etch/config.toml",
	}
	for _, entry := range expectedEntries {
		if !strings.Contains(content, entry) {
			t.Errorf(".gitignore missing entry: %s", entry)
		}
	}
}

func TestRunInitGitignoreTrackProgress(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	r, w, _ := os.Pipe()
	w.WriteString("y\n")
	w.Close()
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	err := runInit()
	if err != nil {
		t.Fatalf("runInit() returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("expected .gitignore to exist: %v", err)
	}

	content := string(data)

	// Should NOT contain progress entry when tracking
	if strings.Contains(content, ".etch/progress/") {
		t.Error(".gitignore should NOT contain .etch/progress/ when tracking is enabled")
	}

	// Should still contain other entries
	for _, entry := range []string{".etch/backups/", ".etch/context/", ".etch/config.toml"} {
		if !strings.Contains(content, entry) {
			t.Errorf(".gitignore missing entry: %s", entry)
		}
	}
}

func TestRunInitIdempotent(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Run init twice
	for i := 0; i < 2; i++ {
		r, w, _ := os.Pipe()
		w.WriteString("n\n")
		w.Close()
		oldStdin := os.Stdin
		os.Stdin = r

		err := runInit()
		os.Stdin = oldStdin
		if err != nil {
			t.Fatalf("runInit() iteration %d returned error: %v", i, err)
		}
	}

	// .gitignore should not have duplicate entries
	data, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	content := string(data)
	count := strings.Count(content, ".etch/backups/")
	if count != 1 {
		t.Errorf("expected 1 occurrence of .etch/backups/ in .gitignore, got %d", count)
	}
}
