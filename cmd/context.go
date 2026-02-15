package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func contextCmd() *cli.Command {
	return &cli.Command{
		Name:  "context",
		Usage: "Generate context prompt for AI agent",
		Action: func(c *cli.Context) error {
			fmt.Println("not yet implemented")
			return nil
		},
	}
}
