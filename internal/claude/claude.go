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
		return handleExecError(err)
	}

	return nil
}

// RunWithStdin launches the claude CLI interactively, piping the prompt via
// stdin instead of passing it as a CLI argument. This avoids OS argument
// length limits for large context prompts. stdout and stderr remain connected
// to the user's terminal for interactive use.
func RunWithStdin(prompt, workDir string) error {
	path, err := exec.LookPath("claude")
	if err != nil {
		return etcherr.New(etcherr.CatConfig, "claude CLI not found on PATH").
			WithHint("install Claude Code: https://docs.anthropic.com/en/docs/claude-code")
	}

	cmd := exec.Command(path, "--prompt", "-")
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return etcherr.Wrap(etcherr.CatIO, "failed to create stdin pipe", err).
			WithHint("check system resources")
	}

	if err := cmd.Start(); err != nil {
		return etcherr.Wrap(etcherr.CatIO, "failed to start claude", err).
			WithHint("check that claude is installed and working")
	}

	if _, err := stdin.Write([]byte(prompt)); err != nil {
		return etcherr.Wrap(etcherr.CatIO, "failed to write prompt to stdin", err).
			WithHint("check system resources")
	}
	stdin.Close()

	if err := cmd.Wait(); err != nil {
		return handleExecError(err)
	}

	return nil
}

func handleExecError(err error) error {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return etcherr.New(etcherr.CatAPI, "claude session exited with non-zero status").
			WithHint("claude exited with code " + strconv.Itoa(exitErr.ExitCode()))
	}
	return etcherr.Wrap(etcherr.CatIO, "failed to run claude", err).
		WithHint("check that claude is installed and working")
}
