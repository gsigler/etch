package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func planCmd() *cli.Command {
	return &cli.Command{
		Name:      "plan",
		Usage:     "Generate an implementation plan",
		ArgsUsage: "<description>",
		Action: func(c *cli.Context) error {
			fmt.Println("not yet implemented")
			return nil
		},
	}
}
