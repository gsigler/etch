package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gsigler/etch/internal/api"
	"github.com/gsigler/etch/internal/config"
	etchcontext "github.com/gsigler/etch/internal/context"
	etcherr "github.com/gsigler/etch/internal/errors"
	"github.com/gsigler/etch/internal/generator"
	"github.com/gsigler/etch/internal/models"
	"github.com/gsigler/etch/internal/parser"
	"github.com/urfave/cli/v2"
)

func replanCmd() *cli.Command {
	return &cli.Command{
		Name:      "replan",
		Usage:     "Regenerate plan incorporating progress and feedback",
		ArgsUsage: "[plan-name] <target>",
		Description: `Replan a task or feature that needs rethinking.

Target resolution:
  etch replan 1.2            → replan Task 1.2
  etch replan feature:2      → replan Feature 2
  etch replan "Login System"  → replan feature by title
  etch replan my-plan 1.2    → replan task 1.2 in specific plan`,
		Action: func(c *cli.Context) error {
			args := c.Args().Slice()
			if len(args) == 0 {
				return etcherr.Usage("target is required").
					WithHint("usage: etch replan <target>  (e.g. 'etch replan 1.2' or 'etch replan feature:2')")
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

			// Discover plans.
			plans, err := etchcontext.DiscoverPlans(rootDir)
			if err != nil {
				return err
			}

			// Parse args: [plan-slug] <target>
			var planSlug, targetStr string
			switch len(args) {
			case 1:
				targetStr = args[0]
			case 2:
				planSlug = args[0]
				targetStr = args[1]
			default:
				return etcherr.Usage("too many arguments").
					WithHint("usage: etch replan [plan-name] <target>")
			}

			// Find the plan.
			var plan *models.Plan
			if planSlug != "" {
				plan = findReplanPlan(plans, planSlug)
				if plan == nil {
					return etcherr.Project(fmt.Sprintf("plan %q not found", planSlug)).
						WithHint("run 'etch list' to see available plans")
				}
			} else if len(plans) == 1 {
				plan = plans[0]
			} else {
				slug, pickErr := pickPlan(plans)
				if pickErr != nil {
					return pickErr
				}
				plan = findReplanPlan(plans, slug)
				if plan == nil {
					return etcherr.Project(fmt.Sprintf("plan %q not found", slug)).
						WithHint("run 'etch list' to see available plans")
				}
			}

			// Resolve the target.
			target, err := generator.ResolveTarget(plan, targetStr)
			if err != nil {
				return err
			}

			// Display what we're replanning.
			switch target.Type {
			case "task":
				fmt.Printf("Replanning Task %s: %s\n\n", target.TaskID, target.Task.Title)
			case "feature":
				fmt.Printf("Replanning Feature %d: %s\n\n", target.FeatureNum, target.Feature.Title)
			}

			client := api.NewClient(apiKey, cfg.API.Model)

			// Stream the replan.
			result, err := generator.Replan(client, plan.FilePath, rootDir, target, func(text string) {
				fmt.Print(text)
			})
			if err != nil {
				return err
			}
			fmt.Println()

			// Show diff.
			if result.Diff != "" {
				fmt.Println("\n--- Changes ---")
				fmt.Println(result.Diff)
			} else {
				fmt.Println("\nNo changes detected.")
				return nil
			}

			fmt.Printf("\nBackup saved to: %s\n", result.BackupPath)

			// Confirm.
			fmt.Print("\nApply changes? (y/n): ")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))

			if input != "y" && input != "yes" {
				fmt.Println("Changes discarded. Backup remains at:", result.BackupPath)
				return nil
			}

			if err := generator.ApplyReplan(plan.FilePath, result.NewMarkdown); err != nil {
				return etcherr.WrapIO("applying replan", err)
			}

			// Verify the written plan parses correctly.
			if _, err := parser.ParseFile(plan.FilePath); err != nil {
				fmt.Printf("Warning: written plan has parse issues: %v\n", err)
				fmt.Println("You may want to review the file manually or restore from backup.")
			}

			fmt.Println("Plan updated successfully.")
			return nil
		},
	}
}

func findReplanPlan(plans []*models.Plan, slug string) *models.Plan {
	for _, p := range plans {
		if p.Slug == slug {
			return p
		}
	}
	return nil
}
