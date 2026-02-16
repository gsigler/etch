package claude

import (
	"errors"
	"os"
	"os/exec"
	"testing"

	etcherr "github.com/gsigler/etch/internal/errors"
)

func TestRun_ClaudeNotOnPath(t *testing.T) {
	// Save original PATH and set to empty so claude can't be found.
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", "")
	defer os.Setenv("PATH", origPath)

	err := Run("test prompt", t.TempDir())
	if err == nil {
		t.Fatal("expected error when claude is not on PATH")
	}

	var etchErr *etcherr.Error
	if !errors.As(err, &etchErr) {
		t.Fatalf("expected etcherr.Error, got %T: %v", err, err)
	}

	if etchErr.Category != etcherr.CatConfig {
		t.Errorf("expected category %q, got %q", etcherr.CatConfig, etchErr.Category)
	}

	if etchErr.Hint == "" {
		t.Error("expected non-empty hint")
	}
}

func TestRun_WorkDirIsSet(t *testing.T) {
	// Verify that exec.LookPath works for a known binary.
	_, err := exec.LookPath("echo")
	if err != nil {
		t.Skip("echo not on PATH")
	}

	// We can't easily test the full interactive flow, but we can verify
	// the function constructs the command correctly by checking that
	// a missing workDir produces an error.
	err = Run("hello", "/nonexistent-dir-for-test")
	if err == nil {
		t.Fatal("expected error for nonexistent workDir")
	}
}
