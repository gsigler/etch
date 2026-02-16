package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	etchcontext "github.com/gsigler/etch/internal/context"
	"github.com/gsigler/etch/internal/tui"
	"github.com/urfave/cli/v2"
)

func reviewCmd() *cli.Command {
	return &cli.Command{
		Name:      "review",
		Usage:     "Review a plan interactively",
		ArgsUsage: "<plan-name>",
		Action: func(c *cli.Context) error {
			rootDir, err := findProjectRoot()
			if err != nil {
				return err
			}

			plans, err := etchcontext.DiscoverPlans(rootDir)
			if err != nil {
				return err
			}

			if len(plans) == 0 {
				return fmt.Errorf("no plans found in .etch/plans/")
			}

			// Resolve which plan to review.
			var plan = plans[0]
			if slug := c.Args().First(); slug != "" {
				found := false
				for _, p := range plans {
					if p.Slug == slug {
						plan = p
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("plan not found: %s", slug)
				}
			} else if len(plans) > 1 {
				slug, err := pickPlan(plans)
				if err != nil {
					return err
				}
				for _, p := range plans {
					if p.Slug == slug {
						plan = p
						break
					}
				}
			}

			m := tui.New(plan, plan.FilePath)
			p := tea.NewProgram(m, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("TUI error: %w", err)
			}
			return nil
		},
	}
}
