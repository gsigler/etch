package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func listCmd() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List available plans",
		Action: func(c *cli.Context) error {
			fmt.Println("not yet implemented")
			return nil
		},
	}
}
