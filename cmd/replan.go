package cmd

import (
	"fmt"
	"os"

	"github.com/gsigler/etch/internal/claude"
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
			var targetDesc string
			switch target.Type {
			case "task":
				targetDesc = fmt.Sprintf("Task %s: %s", target.TaskID, target.Task.Title)
				fmt.Printf("Replanning %s\n", targetDesc)
			case "feature":
				targetDesc = fmt.Sprintf("Feature %d: %s", target.FeatureNum, target.Feature.Title)
				fmt.Printf("Replanning %s\n", targetDesc)
			}

			// Read the current plan content.
			planContent, err := os.ReadFile(plan.FilePath)
			if err != nil {
				return etcherr.WrapIO("reading plan file", err).
					WithHint("check that the plan file exists and is readable: " + plan.FilePath)
			}

			// Backup the plan before launching the interactive session.
			backupPath, err := generator.BackupPlan(plan.FilePath, rootDir)
			if err != nil {
				return err
			}
			fmt.Printf("Backup saved to: %s\n\n", backupPath)

			// Build the prompt for Claude Code.
			prompt := fmt.Sprintf(
				"I need to replan part of an etch implementation plan.\n\n"+
					"**Target:** %s\n\n"+
					"**Plan file:** %s\n\n"+
					"**Current plan content:**\n```markdown\n%s\n```\n\n"+
					"Please modify the plan file at `%s` to replan the target above. "+
					"Preserve any completed tasks (marked with ✓) as-is. "+
					"Update the pending/in-progress tasks for the target to reflect a better approach. "+
					"Follow the etch plan format with proper markdown headings, task IDs, and acceptance criteria.",
				targetDesc,
				plan.FilePath,
				string(planContent),
				plan.FilePath,
			)

			fmt.Println("Launching Claude Code to replan...")
			fmt.Println()

			if err := claude.Run(prompt, rootDir); err != nil {
				return err
			}

			// Verify the plan file still parses correctly.
			if _, err := parser.ParseFile(plan.FilePath); err != nil {
				fmt.Printf("\nWarning: plan has parse issues after replanning: %v\n", err)
				fmt.Println("You may want to review the file manually or restore from backup.")
				fmt.Printf("Backup: %s\n", backupPath)
			} else {
				fmt.Println("\nPlan updated successfully.")
			}

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
