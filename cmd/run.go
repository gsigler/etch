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
		Name:  "run",
		Usage: "Launch Claude Code with assembled context for a task",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "plan",
				Aliases: []string{"p"},
				Usage:   "plan slug",
			},
			&cli.StringFlag{
				Name:    "task",
				Aliases: []string{"t"},
				Usage:   "task ID (e.g. 1.2)",
			},
			&cli.StringFlag{
				Name:    "feature",
				Aliases: []string{"f"},
				Usage:   "feature number (e.g. 2) — run all pending tasks in a feature",
			},
		},
		Action: func(c *cli.Context) error {
			if c.String("feature") != "" && c.String("task") != "" {
				return etcherr.Usage("--feature and --task are mutually exclusive").
					WithHint("use --feature to target an entire feature, or --task for a single task")
			}

			if c.String("feature") != "" {
				return runFeature(c)
			}

			rc, err := resolveContextArgs(c)
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

func runFeature(c *cli.Context) error {
	rf, err := resolveFeatureArgs(c)
	if err != nil {
		return err
	}

	feature := rf.Feature
	result := rf.Result
	rootDir := rf.RootDir

	relContext, _ := filepath.Rel(rootDir, result.ContextPath)

	fmt.Printf("Launching Claude for Feature %d — %s (%d tasks, session %03d)\n\n",
		feature.Number, feature.Title, len(result.ProgressPaths), result.SessionNum)
	fmt.Printf("  Context file:  %s\n", relContext)
	fmt.Printf("  Token estimate: ~%dk tokens\n\n", result.TokenEstimate/1000)

	if result.TokenEstimate > 80000 {
		fmt.Println("  ⚠ Warning: context exceeds 80K tokens — consider trimming the plan overview or splitting the feature.")
		fmt.Println()
	}

	content, err := os.ReadFile(result.ContextPath)
	if err != nil {
		return etcherr.WrapIO("reading context file", err).
			WithHint("context file may have been removed: " + result.ContextPath)
	}

	return claude.RunWithStdin(string(content), rootDir)
}
