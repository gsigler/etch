package cmd

import (
	"fmt"

	etchcontext "github.com/gsigler/etch/internal/context"
	etcherr "github.com/gsigler/etch/internal/errors"
	"github.com/gsigler/etch/internal/models"
	"github.com/urfave/cli/v2"
)

func listCmd() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List available plans",
		Action: func(c *cli.Context) error {
			return runList()
		},
	}
}

func runList() error {
	rootDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	plans, err := etchcontext.DiscoverPlans(rootDir)
	if err != nil {
		return etcherr.Project("no plans found").
			WithHint("run 'etch plan <description>' to create one")
	}

	for _, plan := range plans {
		total, completed := countTasks(plan)
		pct := 0
		if total > 0 {
			pct = completed * 100 / total
		}
		fmt.Printf("%-30s  %d/%d tasks  %3d%%\n", plan.Title, completed, total, pct)
	}

	return nil
}

func countTasks(plan *models.Plan) (total, completed int) {
	for _, f := range plan.Features {
		for _, t := range f.Tasks {
			total++
			if t.Status == models.StatusCompleted {
				completed++
			}
		}
	}
	return
}
