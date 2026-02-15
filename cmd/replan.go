package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func replanCmd() *cli.Command {
	return &cli.Command{
		Name:  "replan",
		Usage: "Regenerate plan incorporating progress and feedback",
		Action: func(c *cli.Context) error {
			fmt.Println("not yet implemented")
			return nil
		},
	}
}
