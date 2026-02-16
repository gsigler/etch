package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	etchcontext "github.com/gsigler/etch/internal/context"
	etcherr "github.com/gsigler/etch/internal/errors"
	"github.com/gsigler/etch/internal/models"
	"github.com/urfave/cli/v2"
)

func contextCmd() *cli.Command {
	return &cli.Command{
		Name:      "context",
		Usage:     "Generate context prompt for AI agent",
		ArgsUsage: "[plan-name] [task-id]",
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
func resolveContextArgs(c *cli.Context, cmdName string) (*resolvedContext, error) {
	rootDir, err := findProjectRoot()
	if err != nil {
		return nil, err
	}

	plans, err := etchcontext.DiscoverPlans(rootDir)
	if err != nil {
		return nil, err
	}

	var planSlug, taskID string

	args := c.Args().Slice()
	switch len(args) {
	case 0:
		// Auto-select: check if picker needed.
		if len(plans) > 1 {
			needsPicker, ambiguousPlans := etchcontext.NeedsPlanPicker(plans, rootDir)
			if needsPicker {
				slug, err := pickPlan(ambiguousPlans)
				if err != nil {
					return nil, err
				}
				planSlug = slug
			}
		}
	case 1:
		// Could be a task ID or a plan slug.
		arg := args[0]
		if looksLikeTaskID(arg) {
			taskID = arg
		} else {
			planSlug = arg
		}
	case 2:
		planSlug = args[0]
		taskID = args[1]
	default:
		return nil, etcherr.Usage("too many arguments").
			WithHint(fmt.Sprintf("usage: etch %s [plan-name] [task-id]", cmdName))
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

func runContext(c *cli.Context) error {
	rc, err := resolveContextArgs(c, "context")
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

// looksLikeTaskID returns true if the string looks like a task ID (e.g. "1.2", "3", "1.3b").
func looksLikeTaskID(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Must start with a digit.
	if s[0] < '0' || s[0] > '9' {
		return false
	}
	// Allow digits, dots, and trailing letters.
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' || (r >= 'a' && r <= 'z') {
			continue
		}
		return false
	}
	return true
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
