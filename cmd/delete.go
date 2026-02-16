package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	etcherr "github.com/gsigler/etch/internal/errors"
	"github.com/urfave/cli/v2"
)

func deleteCmd() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a plan and its progress files",
		ArgsUsage: "<plan-name>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "skip confirmation prompt",
			},
		},
		Action: func(c *cli.Context) error {
			slug := c.Args().First()
			if slug == "" {
				return etcherr.Usage("missing plan name").
					WithHint("usage: etch delete <plan-name>")
			}
			return runDelete(slug, c.Bool("yes"))
		},
	}
}

func runDelete(slug string, skipConfirm bool) error {
	rootDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	planPath := filepath.Join(rootDir, ".etch", "plans", slug+".md")
	if _, err := os.Stat(planPath); os.IsNotExist(err) {
		return etcherr.Project(fmt.Sprintf("plan not found: %s", slug)).
			WithHint("run 'etch list' to see available plans")
	}

	// Find matching progress files.
	progressDir := filepath.Join(rootDir, ".etch", "progress")
	progressPattern := filepath.Join(progressDir, slug+"--*.md")
	progressFiles, _ := filepath.Glob(progressPattern)

	// Find matching context files.
	contextDir := filepath.Join(rootDir, ".etch", "context")
	contextPattern := filepath.Join(contextDir, slug+"--*.md")
	contextFiles, _ := filepath.Glob(contextPattern)

	if !skipConfirm {
		fmt.Printf("Delete plan '%s'?\n", slug)
		fmt.Printf("  Plan file: %s\n", planPath)
		if len(progressFiles) > 0 {
			fmt.Printf("  Progress files: %d\n", len(progressFiles))
		}
		if len(contextFiles) > 0 {
			fmt.Printf("  Context files: %d\n", len(contextFiles))
		}
		fmt.Println()
		if !askYesNo("Are you sure? (y/N)") {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Remove plan file.
	if err := os.Remove(planPath); err != nil {
		return etcherr.WrapIO("removing plan file", err)
	}

	// Remove progress files.
	for _, f := range progressFiles {
		os.Remove(f)
	}

	// Remove context files.
	for _, f := range contextFiles {
		os.Remove(f)
	}

	fmt.Printf("Deleted plan '%s' (%d progress files, %d context files removed).\n",
		slug, len(progressFiles), len(contextFiles))
	return nil
}
