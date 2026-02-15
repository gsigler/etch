package cmd

import (
	"fmt"
	"os"

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
		},
		Action: func(c *cli.Context) error {
			rootDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
			}

			planFilter := c.Args().First()

			plans, err := status.Run(rootDir, planFilter)
			if err != nil {
				return err
			}

			status.SortPlanStatuses(plans)

			if c.Bool("json") {
				out, err := status.FormatJSON(plans)
				if err != nil {
					return fmt.Errorf("formatting JSON: %w", err)
				}
				fmt.Println(out)
				return nil
			}

			if planFilter != "" && len(plans) == 1 {
				fmt.Print(status.FormatDetailed(plans[0]))
			} else {
				fmt.Print(status.FormatSummary(plans))
			}

			return nil
		},
	}
}
