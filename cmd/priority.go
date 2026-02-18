package cmd

import (
	"fmt"
	"sort"

	etchcontext "github.com/gsigler/etch/internal/context"
	etcherr "github.com/gsigler/etch/internal/errors"
	"github.com/gsigler/etch/internal/models"
	"github.com/gsigler/etch/internal/serializer"
	"github.com/urfave/cli/v2"
)

func priorityCmd() *cli.Command {
	return &cli.Command{
		Name:  "priority",
		Usage: "View or change plan priorities",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "plan",
				Aliases: []string{"p"},
				Usage:   "plan slug",
			},
			&cli.IntFlag{
				Name:  "set",
				Usage: "set priority value (positive integer, lower = higher priority)",
			},
			&cli.BoolFlag{
				Name:  "unset",
				Usage: "remove priority from a plan",
			},
		},
		Action: func(c *cli.Context) error {
			rootDir, err := findProjectRoot()
			if err != nil {
				return err
			}

			slug := c.String("plan")
			setVal := c.Int("set")
			unset := c.Bool("unset")

			// No flags: list all plans by priority.
			if slug == "" && !unset && setVal == 0 {
				return listPriorities(rootDir)
			}

			// --set or --unset require --plan.
			if slug == "" {
				return etcherr.Usage("missing --plan flag").
					WithHint("usage: etch priority -p <plan-slug> --set <N>")
			}

			if unset {
				return setPriority(rootDir, slug, 0)
			}

			if setVal < 1 {
				return etcherr.Usage("missing or invalid --set value").
					WithHint("priority must be a positive integer (e.g. etch priority -p my-plan --set 1)")
			}

			return setPriority(rootDir, slug, setVal)
		},
	}
}

func listPriorities(rootDir string) error {
	plans, err := etchcontext.DiscoverPlans(rootDir)
	if err != nil {
		return err
	}

	sortPlansByPriority(plans)

	for _, plan := range plans {
		prioTag := "[ ]"
		if plan.Priority > 0 {
			prioTag = fmt.Sprintf("[%d]", plan.Priority)
		}
		fmt.Printf("%s %s\n", prioTag, plan.Title)
	}

	return nil
}

func setPriority(rootDir, slug string, priority int) error {
	plans, err := etchcontext.DiscoverPlans(rootDir)
	if err != nil {
		return err
	}

	var planPath string
	for _, p := range plans {
		if p.Slug == slug {
			planPath = p.FilePath
			break
		}
	}
	if planPath == "" {
		return etcherr.Project(fmt.Sprintf("plan %q not found", slug)).
			WithHint("run 'etch list' to see available plans")
	}

	if err := serializer.UpdatePlanPriority(planPath, priority); err != nil {
		return etcherr.WrapIO("updating plan priority", err).
			WithHint("check that the plan file is writable")
	}

	if priority == 0 {
		fmt.Printf("Removed priority from %q\n", slug)
	} else {
		fmt.Printf("Set priority of %q to %d\n", slug, priority)
	}

	// Re-read and display the new ordering.
	fmt.Println()
	return listPriorities(rootDir)
}

// sortPlansByPriority sorts plans with priority > 0 first (ascending), then
// plans with no priority alphabetically by title.
func sortPlansByPriority(plans []*models.Plan) {
	sort.Slice(plans, func(i, j int) bool {
		pi, pj := plans[i].Priority, plans[j].Priority
		if pi == pj {
			return plans[i].Title < plans[j].Title
		}
		if pi == 0 {
			return false
		}
		if pj == 0 {
			return true
		}
		return pi < pj
	})
}
