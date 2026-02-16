package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gsigler/etch/internal/claude"
	"github.com/gsigler/etch/internal/config"
	etcherr "github.com/gsigler/etch/internal/errors"
	"github.com/gsigler/etch/internal/generator"
	"github.com/gsigler/etch/internal/parser"
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
			_ = cfg // loaded for slug generation; no API key needed

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

			// Build the prompt for Claude Code to create the plan.
			prompt := fmt.Sprintf(
				"/etch-plan --slug %s %s",
				slug,
				description,
			)

			fmt.Printf("Launching Claude Code to generate plan for: %s\n", description)
			fmt.Printf("Target: .etch/plans/%s.md\n\n", slug)

			if err := claude.Run(prompt, rootDir); err != nil {
				return err
			}

			// Verify the plan file was created.
			planPath := filepath.Join(rootDir, ".etch", "plans", slug+".md")
			if _, err := os.Stat(planPath); os.IsNotExist(err) {
				return etcherr.New(etcherr.CatIO, "plan file was not created").
					WithHint(fmt.Sprintf("expected file at .etch/plans/%s.md — the Claude Code session may have ended before completing the plan", slug))
			}

			// Validate the plan parses correctly.
			plan, err := parser.ParseFile(planPath)
			if err != nil {
				return etcherr.WrapIO("plan file exists but failed to parse", err).
					WithHint(fmt.Sprintf("check .etch/plans/%s.md for formatting issues", slug))
			}

			// Print summary.
			fmt.Println()
			fmt.Printf("Plan created: .etch/plans/%s.md\n", slug)
			fmt.Printf("  Title:    %s\n", plan.Title)
			fmt.Printf("  Features: %d\n", len(plan.Features))
			taskCount := 0
			for _, f := range plan.Features {
				taskCount += len(f.Tasks)
			}
			fmt.Printf("  Tasks:    %d\n", taskCount)

			return nil
		},
	}
}
