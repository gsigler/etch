package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gsigler/etch/internal/api"
	"github.com/gsigler/etch/internal/config"
	etcherr "github.com/gsigler/etch/internal/errors"
	"github.com/gsigler/etch/internal/generator"
	"github.com/urfave/cli/v2"
)

func planCmd() *cli.Command {
	return &cli.Command{
		Name:      "plan",
		Usage:     "Generate an implementation plan",
		ArgsUsage: "<description>",
		Action: func(c *cli.Context) error {
			description := strings.Join(c.Args().Slice(), " ")
			if description == "" {
				return etcherr.Usage("missing feature description").
					WithHint("usage: etch plan \"add user authentication\"")
			}

			rootDir, err := findProjectRoot()
			if err != nil {
				return err
			}

			cfg, err := config.Load(rootDir)
			if err != nil {
				return etcherr.WrapConfig("loading config", err)
			}

			apiKey, err := cfg.ResolveAPIKey()
			if err != nil {
				return err
			}

			// Determine slug and handle collisions.
			slug := generator.Slugify(description)
			if generator.SlugExists(rootDir, slug) {
				newSlug, _ := generator.ResolveSlug(rootDir, slug)
				fmt.Printf("Plan '%s' already exists. Create '%s'? Or did you mean `etch replan %s`?\n", slug, newSlug, slug)
				fmt.Print("(c)reate / (r)eplan / (q)uit: ")
				reader := bufio.NewReader(os.Stdin)
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(strings.ToLower(input))
				switch input {
				case "c", "create":
					slug = newSlug
				case "r", "replan":
					return etcherr.Usage(fmt.Sprintf("plan '%s' already exists", slug)).
						WithHint(fmt.Sprintf("run 'etch replan %s' to modify the existing plan", slug))
				default:
					return nil
				}
			}

			client := api.NewClient(apiKey, cfg.API.Model)

			fmt.Printf("Generating plan for: %s\n\n", description)

			// Stream tokens to terminal.
			result, err := generator.Generate(client, rootDir, description, cfg.Defaults.ComplexityGuide, func(text string) {
				fmt.Print(text)
			})
			if err != nil {
				return err
			}
			fmt.Println() // newline after streaming

			// Write to file.
			path, err := generator.WritePlan(rootDir, slug, result.Markdown)
			if err != nil {
				return err
			}
			result.FilePath = path
			result.Plan.Slug = slug

			fmt.Println()
			fmt.Println(result.Summary())

			return nil
		},
	}
}
