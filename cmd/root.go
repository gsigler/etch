package cmd

import (
	"os"

	"github.com/urfave/cli/v2"
)

func Execute() error {
	app := &cli.App{
		Name:    "etch",
		Usage:   "AI implementation planning CLI",
		Version: "0.1.0",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "enable verbose output",
			},
		},
		Commands: []*cli.Command{
			initCmd(),
			planCmd(),
			reviewCmd(),
			statusCmd(),
			contextCmd(),
			replanCmd(),
			listCmd(),
			openCmd(),
			deleteCmd(),
		},
	}

	return app.Run(os.Args)
}
