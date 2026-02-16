package claude

import (
	"os"
	"os/exec"
	"strconv"

	etcherr "github.com/gsigler/etch/internal/errors"
)

// Run launches the claude CLI interactively with the given prompt in the
// specified working directory. The user's terminal is connected directly
// so they can interact with Claude Code during the session.
func Run(prompt, workDir string) error {
	path, err := exec.LookPath("claude")
	if err != nil {
		return etcherr.New(etcherr.CatConfig, "claude CLI not found on PATH").
			WithHint("install Claude Code: https://docs.anthropic.com/en/docs/claude-code")
	}

	cmd := exec.Command(path, prompt)
	cmd.Dir = workDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return etcherr.New(etcherr.CatAPI, "claude session exited with non-zero status").
				WithHint("claude exited with code " + strconv.Itoa(exitErr.ExitCode()))
		}
		return etcherr.Wrap(etcherr.CatIO, "failed to run claude", err).
			WithHint("check that claude is installed and working")
	}

	return nil
}
