package cmd

import (
	"fmt"

	etcherr "github.com/gsigler/etch/internal/errors"
	"github.com/gsigler/etch/internal/status"
	"github.com/urfave/cli/v2"
)

func statusCmd() *cli.Command {
	return &cli.Command{
		Name:      "status",
		Usage:     "Show current plan progress",
		ArgsUsage: "[plan-slug]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "output in JSON format",
			},
			&cli.BoolFlag{
				Name:  "all",
				Usage: "show all plans including fully pending and completed",
			},
		},
		Action: func(c *cli.Context) error {
			rootDir, err := findProjectRoot()
			if err != nil {
				return err
			}

			planFilter := c.Args().First()

			plans, err := status.Run(rootDir, planFilter)
			if err != nil {
				return err
			}

			status.SortPlanStatuses(plans)

			// Filter to active plans unless --all is passed or a specific plan is requested.
			showAll := c.Bool("all") || planFilter != ""
			if !showAll {
				plans = status.FilterActive(plans)
			}

			if c.Bool("json") {
				out, err := status.FormatJSON(plans)
				if err != nil {
					return etcherr.WrapIO("formatting JSON output", err)
				}
				fmt.Println(out)
				return nil
			}

			if planFilter != "" && len(plans) == 1 {
				fmt.Print(status.FormatDetailed(plans[0]))
			} else if len(plans) == 0 && !showAll {
				fmt.Println("No active plans. Use --all to see all plans.")
			} else {
				fmt.Print(status.FormatSummary(plans))
			}

			return nil
		},
	}
}
