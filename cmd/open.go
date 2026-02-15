package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func openCmd() *cli.Command {
	return &cli.Command{
		Name:      "open",
		Usage:     "Open a plan file in your editor",
		ArgsUsage: "<plan-name>",
		Action: func(c *cli.Context) error {
			fmt.Println("not yet implemented")
			return nil
		},
	}
}
