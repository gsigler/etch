package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gsigler/etch/internal/claude"
	etcherr "github.com/gsigler/etch/internal/errors"
	"github.com/urfave/cli/v2"
)

func runCmd() *cli.Command {
	return &cli.Command{
		Name:      "run",
		Usage:     "Launch Claude Code with assembled context for a task",
		ArgsUsage: "[plan-name] [task-id]",
		Action: func(c *cli.Context) error {
			rc, err := resolveContextArgs(c, "run")
			if err != nil {
				return err
			}

			task := rc.Task
			result := rc.Result
			rootDir := rc.RootDir

			relContext, _ := filepath.Rel(rootDir, result.ContextPath)
			relProgress, _ := filepath.Rel(rootDir, result.ProgressPath)

			fmt.Printf("Launching Claude for Task %s — %s (session %03d)\n\n", task.FullID(), task.Title, result.SessionNum)
			fmt.Printf("  Context file:  %s\n", relContext)
			fmt.Printf("  Progress file: %s\n", relProgress)
			fmt.Printf("  Token estimate: ~%dk tokens\n\n", result.TokenEstimate/1000)

			if result.TokenEstimate > 80000 {
				fmt.Println("  ⚠ Warning: context exceeds 80K tokens — consider trimming the plan overview or splitting the task.")
				fmt.Println()
			}

			content, err := os.ReadFile(result.ContextPath)
			if err != nil {
				return etcherr.WrapIO("reading context file", err).
					WithHint("context file may have been removed: " + result.ContextPath)
			}

			return claude.RunWithStdin(string(content), rootDir)
		},
	}
}
