package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	etchcontext "github.com/gsigler/etch/internal/context"
	etcherr "github.com/gsigler/etch/internal/errors"
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
				return etcherr.Project("no plans found").
					WithHint("run 'etch plan <description>' to create one")
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
					return etcherr.Project(fmt.Sprintf("plan not found: %s", slug)).
						WithHint("run 'etch list' to see available plans")
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
				return etcherr.WrapIO("TUI error", err)
			}
			return nil
		},
	}
}
