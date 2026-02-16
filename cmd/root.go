package cmd

import (
	"os"

	"github.com/urfave/cli/v2"
)

// verbose tracks the global --verbose flag for use by main.
var verbose bool

// Execute runs the etch CLI. It returns the verbose flag value and any error.
func Execute() (bool, error) {
	app := &cli.App{
		Name:    "etch",
		Usage:   "AI implementation planning CLI",
		Version: "0.1.1",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "enable verbose output",
			},
		},
		Before: func(c *cli.Context) error {
			verbose = c.Bool("verbose")
			return nil
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
			skillCmd(),
		},
	}

	err := app.Run(os.Args)
	return verbose, err
}
