package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	etchcontext "github.com/gsigler/etch/internal/context"
	etcherr "github.com/gsigler/etch/internal/errors"
	"github.com/urfave/cli/v2"
)

func openCmd() *cli.Command {
	return &cli.Command{
		Name:      "open",
		Usage:     "Open a plan file in your editor",
		ArgsUsage: "<plan-name>",
		Action: func(c *cli.Context) error {
			slug := c.Args().First()
			if slug == "" {
				return etcherr.Usage("missing plan name").
					WithHint("usage: etch open <plan-name>")
			}
			return runOpen(slug)
		},
	}
}

func runOpen(slug string) error {
	rootDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	plan, err := findPlanBySlug(rootDir, slug)
	if err != nil {
		return err
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, plan.FilePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func findPlanBySlug(rootDir, slug string) (*planMatch, error) {
	plans, err := etchcontext.DiscoverPlans(rootDir)
	if err != nil {
		return nil, etcherr.Project("no plans found").
			WithHint("run 'etch plan <description>' to create one")
	}

	for _, p := range plans {
		if p.Slug == slug {
			return &planMatch{FilePath: p.FilePath, Slug: p.Slug}, nil
		}
	}

	// Try direct file path as fallback.
	path := filepath.Join(rootDir, ".etch", "plans", slug+".md")
	if _, err := os.Stat(path); err == nil {
		return &planMatch{FilePath: path, Slug: slug}, nil
	}

	return nil, etcherr.Project(fmt.Sprintf("plan not found: %s", slug)).
		WithHint("run 'etch list' to see available plans")
}

type planMatch struct {
	FilePath string
	Slug     string
}
