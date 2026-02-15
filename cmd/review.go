package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func reviewCmd() *cli.Command {
	return &cli.Command{
		Name:      "review",
		Usage:     "Review a plan interactively",
		ArgsUsage: "<plan-name>",
		Action: func(c *cli.Context) error {
			fmt.Println("not yet implemented")
			return nil
		},
	}
}
