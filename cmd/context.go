package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	etchcontext "github.com/gsigler/etch/internal/context"
	etcherr "github.com/gsigler/etch/internal/errors"
	"github.com/gsigler/etch/internal/models"
	"github.com/urfave/cli/v2"
)

func contextCmd() *cli.Command {
	return &cli.Command{
		Name:  "context",
		Usage: "Generate context prompt for AI agent",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "plan",
				Aliases: []string{"p"},
				Usage:   "plan slug",
			},
			&cli.StringFlag{
				Name:    "task",
				Aliases: []string{"t"},
				Usage:   "task ID (e.g. 1.2)",
			},
			&cli.StringFlag{
				Name:    "feature",
				Aliases: []string{"f"},
				Usage:   "feature number (e.g. 2) — assemble context for an entire feature",
			},
		},
		Action: func(c *cli.Context) error {
			return runContext(c)
		},
	}
}

// resolvedContext holds the results of argument resolution and context assembly.
type resolvedContext struct {
	RootDir string
	Task    *models.Task
	Result  etchcontext.Result
}

// resolveContextArgs parses CLI arguments, resolves the plan and task, and
// assembles the context. Shared between `etch context` and `etch run`.
func resolveContextArgs(c *cli.Context) (*resolvedContext, error) {
	rootDir, err := findProjectRoot()
	if err != nil {
		return nil, err
	}

	plans, err := etchcontext.DiscoverPlans(rootDir)
	if err != nil {
		return nil, err
	}

	planSlug := c.String("plan")
	taskID := c.String("task")

	// Auto-select plan if not specified.
	if planSlug == "" && len(plans) > 1 {
		needsPicker, ambiguousPlans := etchcontext.NeedsPlanPicker(plans, rootDir)
		if needsPicker {
			slug, err := pickPlan(ambiguousPlans)
			if err != nil {
				return nil, err
			}
			planSlug = slug
		}
	}

	plan, task, err := etchcontext.ResolveTask(plans, planSlug, taskID, rootDir)
	if err != nil {
		return nil, err
	}

	result, err := etchcontext.Assemble(rootDir, plan, task)
	if err != nil {
		return nil, err
	}

	return &resolvedContext{RootDir: rootDir, Task: task, Result: result}, nil
}

// resolvedFeature holds the results of feature argument resolution and context assembly.
type resolvedFeature struct {
	RootDir string
	Feature *models.Feature
	Result  etchcontext.FeatureResult
}

// resolveFeatureArgs parses CLI arguments, resolves the plan and feature, and
// assembles the feature context. Shared between `etch context` and `etch run`.
func resolveFeatureArgs(c *cli.Context) (*resolvedFeature, error) {
	rootDir, err := findProjectRoot()
	if err != nil {
		return nil, err
	}

	plans, err := etchcontext.DiscoverPlans(rootDir)
	if err != nil {
		return nil, err
	}

	planSlug := c.String("plan")
	featureStr := c.String("feature")

	featureNum, err := strconv.Atoi(featureStr)
	if err != nil {
		return nil, etcherr.Usage(fmt.Sprintf("invalid feature number: %q", featureStr)).
			WithHint("feature must be a number, e.g. --feature 2")
	}

	// Auto-select plan if not specified.
	if planSlug == "" && len(plans) > 1 {
		needsPicker, ambiguousPlans := etchcontext.NeedsPlanPicker(plans, rootDir)
		if needsPicker {
			slug, err := pickPlan(ambiguousPlans)
			if err != nil {
				return nil, err
			}
			planSlug = slug
		}
	}

	plan, feature, err := etchcontext.ResolveFeature(plans, planSlug, featureNum)
	if err != nil {
		return nil, err
	}

	result, err := etchcontext.AssembleFeature(rootDir, plan, feature)
	if err != nil {
		return nil, err
	}

	return &resolvedFeature{RootDir: rootDir, Feature: feature, Result: result}, nil
}

func runContext(c *cli.Context) error {
	if c.String("feature") != "" && c.String("task") != "" {
		return etcherr.Usage("--feature and --task are mutually exclusive").
			WithHint("use --feature to target an entire feature, or --task for a single task")
	}

	if c.String("feature") != "" {
		return runFeatureContext(c)
	}

	rc, err := resolveContextArgs(c)
	if err != nil {
		return err
	}

	task := rc.Task
	result := rc.Result
	rootDir := rc.RootDir

	// Print confirmation.
	relContext, _ := filepath.Rel(rootDir, result.ContextPath)
	relProgress, _ := filepath.Rel(rootDir, result.ProgressPath)

	fmt.Printf("Context assembled for Task %s — %s (session %03d)\n\n", task.FullID(), task.Title, result.SessionNum)
	fmt.Printf("  Context file:  %s\n", relContext)
	fmt.Printf("  Progress file: %s\n", relProgress)
	fmt.Printf("  Token estimate: ~%dk tokens\n\n", result.TokenEstimate/1000)

	if result.TokenEstimate > 80000 {
		fmt.Println("  ⚠ Warning: context exceeds 80K tokens — consider trimming the plan overview or splitting the task.")
		fmt.Println()
	}

	fmt.Printf("  Ready to run:\n")
	fmt.Printf("    cat %s | claude\n", relContext)

	return nil
}

func runFeatureContext(c *cli.Context) error {
	rf, err := resolveFeatureArgs(c)
	if err != nil {
		return err
	}

	feature := rf.Feature
	result := rf.Result
	rootDir := rf.RootDir

	relContext, _ := filepath.Rel(rootDir, result.ContextPath)

	fmt.Printf("Context assembled for Feature %d — %s (%d tasks, session %03d)\n\n",
		feature.Number, feature.Title, len(result.ProgressPaths), result.SessionNum)
	fmt.Printf("  Context file:  %s\n", relContext)
	fmt.Printf("  Token estimate: ~%dk tokens\n\n", result.TokenEstimate/1000)

	if result.TokenEstimate > 80000 {
		fmt.Println("  ⚠ Warning: context exceeds 80K tokens — consider trimming the plan overview or splitting the feature.")
		fmt.Println()
	}

	fmt.Printf("  Ready to run:\n")
	fmt.Printf("    cat %s | claude\n", relContext)

	return nil
}

// findProjectRoot walks up from cwd looking for .etch directory.
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", etcherr.WrapIO("getting working directory", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".etch")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", etcherr.Project("not an etch project").
				WithHint("run 'etch init' to initialize a project in this directory")
		}
		dir = parent
	}
}


func pickPlan(plans []*models.Plan) (string, error) {
	fmt.Println("Which plan?")
	for i, p := range plans {
		fmt.Printf("  (%d) %s\n", i+1, p.Slug)
	}
	fmt.Print("Enter number: ")

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(answer)

	var choice int
	if _, err := fmt.Sscanf(answer, "%d", &choice); err != nil || choice < 1 || choice > len(plans) {
		return "", etcherr.Usage(fmt.Sprintf("invalid choice: %s", answer)).
			WithHint(fmt.Sprintf("enter a number between 1 and %d", len(plans)))
	}
	return plans[choice-1].Slug, nil
}
