package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func statusCmd() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "Show current plan progress",
		Action: func(c *cli.Context) error {
			fmt.Println("not yet implemented")
			return nil
		},
	}
}
