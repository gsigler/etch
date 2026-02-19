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
	"github.com/gsigler/etch/internal/serializer"
	"github.com/urfave/cli/v2"
)

func replanCmd() *cli.Command {
	return &cli.Command{
		Name:  "replan",
		Usage: "Regenerate plan incorporating progress and feedback",
		Description: `Replan an entire plan, a feature, or a single task.

Examples:
  etch replan                             → replan the only plan (or pick from list)
  etch replan -p my-plan                  → replan entire plan by name
  etch replan --target 1.2               → replan Task 1.2
  etch replan --target feature:2         → replan Feature 2
  etch replan --target "Login System"    → replan feature by title
  etch replan -p my-plan --target 1.2    → replan task 1.2 in specific plan
  etch replan -r "tasks are too granular" → replan with a reason for context`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "plan",
				Aliases: []string{"p"},
				Usage:   "plan slug",
			},
			&cli.StringFlag{
				Name:  "target",
				Usage: "replan target: task ID (1.2), feature (feature:2), or feature title",
			},
			&cli.StringFlag{
				Name:    "reason",
				Aliases: []string{"r"},
				Usage:   "reason for replanning (included in prompt for context)",
			},
			&cli.IntFlag{
				Name:  "priority",
				Usage: "set plan priority (lower = higher priority)",
			},
		},
		Action: func(c *cli.Context) error {
			rootDir, err := findProjectRoot()
			if err != nil {
				return err
			}

			// Discover plans.
			plans, err := etchcontext.DiscoverPlans(rootDir)
			if err != nil {
				return err
			}

			planSlug := c.String("plan")
			targetStr := c.String("target")

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

			var prompt string
			if targetStr == "" {
				// Whole-plan replan.
				fmt.Printf("Replanning entire plan: %s\n", plan.Title)

				prompt = fmt.Sprintf(
					"I need to replan an entire etch implementation plan.\n\n"+
						"**Plan:** %s\n\n"+
						"**Plan file:** %s\n\n"+
						"**Current plan content:**\n```markdown\n%s\n```\n\n"+
						"Please modify the plan file at `%s` to replan it. "+
						"Preserve any completed tasks (marked with ✓) as-is. "+
						"Restructure, reorder, add, remove, or revise any pending/in-progress tasks and features as needed. "+
						"Follow the etch plan format with proper markdown headings, task IDs, and acceptance criteria.",
					plan.Title,
					plan.FilePath,
					string(planContent),
					plan.FilePath,
				)
			} else {
				// Targeted replan (task or feature).
				target, resolveErr := generator.ResolveTarget(plan, targetStr)
				if resolveErr != nil {
					return resolveErr
				}

				var targetDesc string
				switch target.Type {
				case "task":
					targetDesc = fmt.Sprintf("Task %s: %s", target.TaskID, target.Task.Title)
				case "feature":
					targetDesc = fmt.Sprintf("Feature %d: %s", target.FeatureNum, target.Feature.Title)
				}
				fmt.Printf("Replanning %s\n", targetDesc)

				prompt = fmt.Sprintf(
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
			}

			if reason := c.String("reason"); reason != "" {
				prompt += fmt.Sprintf("\n\n**Reason for replanning:** %s", reason)
			}

			fmt.Println("Launching Claude Code to replan...")
			fmt.Println()

			if err := claude.RunWithStdin(prompt, rootDir); err != nil {
				return err
			}

			// Apply priority surgically if flag was set.
			if priority := c.Int("priority"); priority > 0 {
				if err := serializer.UpdatePlanPriority(plan.FilePath, priority); err != nil {
					return etcherr.WrapIO("setting plan priority", err).
						WithHint("the plan was replanned but priority could not be set — edit the file manually")
				}
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
