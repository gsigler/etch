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
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "all",
				Usage: "Show all plans including completed ones",
			},
		},
		Action: func(c *cli.Context) error {
			return runList(c.Bool("all"))
		},
	}
}

func runList(showAll bool) error {
	rootDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	plans, err := etchcontext.DiscoverPlans(rootDir)
	if err != nil {
		return etcherr.Project("no plans found").
			WithHint("run 'etch plan <description>' to create one")
	}

	sortPlansByPriority(plans)

	for _, plan := range plans {
		total, completed := countTasks(plan)
		pct := 0
		if total > 0 {
			pct = completed * 100 / total
		}
		if !showAll && total > 0 && completed == total {
			continue
		}
		prioTag := "[ ]"
		if plan.Priority > 0 {
			prioTag = fmt.Sprintf("[%d]", plan.Priority)
		}
		fmt.Printf("%s %-25s  %-30s  %d/%d tasks  %3d%%\n", prioTag, plan.Slug, plan.Title, completed, total, pct)
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
