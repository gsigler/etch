package cmd

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	etchcontext "github.com/gsigler/etch/internal/context"
	etcherr "github.com/gsigler/etch/internal/errors"
	"github.com/gsigler/etch/internal/models"
	"github.com/gsigler/etch/internal/progress"
	"github.com/gsigler/etch/internal/serializer"
	"github.com/urfave/cli/v2"
)

func progressCmd() *cli.Command {
	return &cli.Command{
		Name:  "progress",
		Usage: "Report progress on tasks",
		Subcommands: []*cli.Command{
			progressStartCmd(),
			progressUpdateCmd(),
			progressDoneCmd(),
			progressCriteriaCmd(),
			progressBlockCmd(),
			progressFailCmd(),
		},
	}
}

func progressStartCmd() *cli.Command {
	return &cli.Command{
		Name:      "start",
		Usage:     "Mark a task as in progress",
		ArgsUsage: "[plan-name] <task-id>",
		Action: func(c *cli.Context) error {
			return runProgressStart(c)
		},
	}
}

func runProgressStart(c *cli.Context) error {
	rootDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	plans, err := etchcontext.DiscoverPlans(rootDir)
	if err != nil {
		return err
	}

	var planSlug, taskID string

	args := c.Args().Slice()
	switch len(args) {
	case 0:
		return etcherr.Usage("task ID is required").
			WithHint("usage: etch progress start [plan-name] <task-id>")
	case 1:
		taskID = args[0]
	case 2:
		planSlug = args[0]
		taskID = args[1]
	default:
		return etcherr.Usage("too many arguments").
			WithHint("usage: etch progress start [plan-name] <task-id>")
	}

	plan, task, err := etchcontext.ResolveTask(plans, planSlug, taskID, rootDir)
	if err != nil {
		return err
	}

	// Update task status in the plan file.
	if err := serializer.UpdateTaskStatus(plan.FilePath, task.FullID(), models.StatusInProgress); err != nil {
		return etcherr.WrapIO("updating task status", err).
			WithHint(fmt.Sprintf("could not update task %s in plan file", task.FullID()))
	}

	// Find or create session progress file.
	allProgress, err := progress.ReadAll(rootDir, plan.Slug)
	if err != nil {
		return etcherr.WrapIO("reading progress files", err)
	}

	var sessionNum int
	var progressPath string

	if sessions, ok := allProgress[task.FullID()]; ok && len(sessions) > 0 {
		// Reuse the latest session file.
		latest := sessions[len(sessions)-1]
		sessionNum = latest.SessionNumber
		progressPath = progressFilePath(rootDir, plan.Slug, task.FullID(), sessionNum)

		// Update the status line in the existing progress file.
		if err := progress.UpdateStatus(progressPath, "in_progress"); err != nil {
			return etcherr.WrapIO("updating progress file status", err)
		}
	} else {
		// Create a new session file.
		progressPath, err = progress.WriteSession(rootDir, plan, task)
		if err != nil {
			return etcherr.WrapIO("creating progress file", err)
		}
		// Extract session number from filename.
		sessionNum = extractSessionNumber(progressPath)

		// Update the status line from pending to in_progress.
		if err := progress.UpdateStatus(progressPath, "in_progress"); err != nil {
			return etcherr.WrapIO("updating progress file status", err)
		}
	}

	fmt.Printf("Task %s started (session %03d)\n", task.FullID(), sessionNum)
	return nil
}

func progressFilePath(rootDir, planSlug, taskID string, session int) string {
	return fmt.Sprintf("%s/.etch/progress/%s--task-%s--%03d.md", rootDir, planSlug, taskID, session)
}

var sessionNumRe = regexp.MustCompile(`--(\d{3})\.md$`)

func extractSessionNumber(path string) int {
	m := sessionNumRe.FindStringSubmatch(path)
	if m == nil {
		return 1
	}
	var n int
	fmt.Sscanf(m[1], "%d", &n)
	return n
}

func progressUpdateCmd() *cli.Command {
	return &cli.Command{
		Name:      "update",
		Usage:     "Log a progress update for a task",
		ArgsUsage: "[plan-name] <task-id>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "message",
				Aliases:  []string{"m"},
				Usage:    "update message (required)",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			return runProgressUpdate(c)
		},
	}
}

func runProgressUpdate(c *cli.Context) error {
	rootDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	plans, err := etchcontext.DiscoverPlans(rootDir)
	if err != nil {
		return err
	}

	var planSlug, taskID string
	args := c.Args().Slice()
	switch len(args) {
	case 0:
		return etcherr.Usage("task ID is required").
			WithHint("usage: etch progress update [plan-name] <task-id> --message \"text\"")
	case 1:
		taskID = args[0]
	case 2:
		planSlug = args[0]
		taskID = args[1]
	default:
		return etcherr.Usage("too many arguments").
			WithHint("usage: etch progress update [plan-name] <task-id> --message \"text\"")
	}

	plan, task, err := etchcontext.ResolveTask(plans, planSlug, taskID, rootDir)
	if err != nil {
		return err
	}

	sessionPath, _, err := progress.FindLatestSessionPath(rootDir, plan.Slug, task.FullID())
	if err != nil {
		return etcherr.WrapIO("finding session file", err).
			WithHint(fmt.Sprintf("run 'etch progress start %s' first to create a session", task.FullID()))
	}

	message := c.String("message")
	timestamp := time.Now().Format("15:04")
	entry := fmt.Sprintf("- [%s] %s", timestamp, message)

	if err := progress.AppendToSection(sessionPath, "Changes Made", entry); err != nil {
		return etcherr.WrapIO("appending to progress file", err)
	}

	fmt.Printf("Logged update for Task %s\n", task.FullID())
	return nil
}

func progressDoneCmd() *cli.Command {
	return &cli.Command{
		Name:      "done",
		Usage:     "Mark a task as completed",
		ArgsUsage: "[plan-name] <task-id>",
		Action: func(c *cli.Context) error {
			return runProgressDone(c)
		},
	}
}

func runProgressDone(c *cli.Context) error {
	rootDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	plans, err := etchcontext.DiscoverPlans(rootDir)
	if err != nil {
		return err
	}

	var planSlug, taskID string
	args := c.Args().Slice()
	switch len(args) {
	case 0:
		return etcherr.Usage("task ID is required").
			WithHint("usage: etch progress done [plan-name] <task-id>")
	case 1:
		taskID = args[0]
	case 2:
		planSlug = args[0]
		taskID = args[1]
	default:
		return etcherr.Usage("too many arguments").
			WithHint("usage: etch progress done [plan-name] <task-id>")
	}

	plan, task, err := etchcontext.ResolveTask(plans, planSlug, taskID, rootDir)
	if err != nil {
		return err
	}

	// Update plan file status to completed.
	if err := serializer.UpdateTaskStatus(plan.FilePath, task.FullID(), models.StatusCompleted); err != nil {
		return etcherr.WrapIO("updating task status", err).
			WithHint(fmt.Sprintf("could not update task %s in plan file", task.FullID()))
	}

	// Update progress file status to completed.
	sessionPath, _, err := progress.FindLatestSessionPath(rootDir, plan.Slug, task.FullID())
	if err == nil {
		if err := progress.UpdateStatus(sessionPath, "completed"); err != nil {
			return etcherr.WrapIO("updating progress file status", err)
		}
	}

	// Check for unchecked acceptance criteria and warn.
	var unchecked []string
	for _, c := range task.Criteria {
		if !c.IsMet {
			unchecked = append(unchecked, c.Description)
		}
	}

	fmt.Printf("Task %s completed\n", task.FullID())
	if len(unchecked) > 0 {
		fmt.Printf("Warning: %d unchecked acceptance criteria:\n", len(unchecked))
		for _, desc := range unchecked {
			fmt.Printf("  - [ ] %s\n", desc)
		}
	}

	return nil
}

func progressCriteriaCmd() *cli.Command {
	return &cli.Command{
		Name:      "criteria",
		Usage:     "Check off acceptance criteria for a task",
		ArgsUsage: "[plan-name] <task-id>",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:     "check",
				Usage:    "criterion text to check off (can be specified multiple times)",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			return runProgressCriteria(c)
		},
	}
}

func runProgressCriteria(c *cli.Context) error {
	rootDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	plans, err := etchcontext.DiscoverPlans(rootDir)
	if err != nil {
		return err
	}

	var planSlug, taskID string
	args := c.Args().Slice()
	switch len(args) {
	case 0:
		return etcherr.Usage("task ID is required").
			WithHint("usage: etch progress criteria [plan-name] <task-id> --check \"text\"")
	case 1:
		taskID = args[0]
	case 2:
		planSlug = args[0]
		taskID = args[1]
	default:
		return etcherr.Usage("too many arguments").
			WithHint("usage: etch progress criteria [plan-name] <task-id> --check \"text\"")
	}

	plan, task, err := etchcontext.ResolveTask(plans, planSlug, taskID, rootDir)
	if err != nil {
		return err
	}

	checks := c.StringSlice("check")
	matched := 0
	var unmatched []string

	for _, checkText := range checks {
		// Try exact match first.
		err := serializer.UpdateCriterion(plan.FilePath, task.FullID(), checkText, true)
		if err == nil {
			// Also update progress file.
			updateProgressCriterion(rootDir, plan.Slug, task.FullID(), checkText)
			fmt.Printf("  ✓ %s\n", checkText)
			matched++
			continue
		}

		// Try substring match (case-insensitive).
		found := false
		for _, criterion := range task.Criteria {
			if criterion.IsMet {
				continue
			}
			if strings.Contains(strings.ToLower(criterion.Description), strings.ToLower(checkText)) {
				err := serializer.UpdateCriterion(plan.FilePath, task.FullID(), criterion.Description, true)
				if err == nil {
					updateProgressCriterion(rootDir, plan.Slug, task.FullID(), criterion.Description)
					fmt.Printf("  ✓ %s (matched: %s)\n", checkText, criterion.Description)
					matched++
					found = true
					break
				}
			}
		}

		if !found {
			fmt.Printf("  ✗ %s (no match)\n", checkText)
			unmatched = append(unmatched, checkText)
		}
	}

	fmt.Printf("Checked %d/%d criteria for Task %s\n", matched, len(checks), task.FullID())

	if len(unmatched) > 0 {
		return etcherr.Usage(fmt.Sprintf("%d criteria did not match", len(unmatched))).
			WithHint("check the criterion text matches what's in the plan")
	}

	return nil
}

// updateProgressCriterion updates the criterion in the progress file, if a session exists.
func updateProgressCriterion(rootDir, planSlug, taskID, criterionText string) {
	sessionPath, _, err := progress.FindLatestSessionPath(rootDir, planSlug, taskID)
	if err != nil {
		return
	}
	// Best-effort update — don't fail the command if progress file update fails.
	progress.UpdateCriterion(sessionPath, criterionText)
}

func progressBlockCmd() *cli.Command {
	return &cli.Command{
		Name:      "block",
		Usage:     "Mark a task as blocked",
		ArgsUsage: "[plan-name] <task-id>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "reason",
				Usage:    "reason the task is blocked (required)",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			return runProgressBlock(c)
		},
	}
}

func runProgressBlock(c *cli.Context) error {
	rootDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	plans, err := etchcontext.DiscoverPlans(rootDir)
	if err != nil {
		return err
	}

	var planSlug, taskID string
	args := c.Args().Slice()
	switch len(args) {
	case 0:
		return etcherr.Usage("task ID is required").
			WithHint("usage: etch progress block [plan-name] <task-id> --reason \"text\"")
	case 1:
		taskID = args[0]
	case 2:
		planSlug = args[0]
		taskID = args[1]
	default:
		return etcherr.Usage("too many arguments").
			WithHint("usage: etch progress block [plan-name] <task-id> --reason \"text\"")
	}

	plan, task, err := etchcontext.ResolveTask(plans, planSlug, taskID, rootDir)
	if err != nil {
		return err
	}

	// Update plan file status to blocked.
	if err := serializer.UpdateTaskStatus(plan.FilePath, task.FullID(), models.StatusBlocked); err != nil {
		return etcherr.WrapIO("updating task status", err).
			WithHint(fmt.Sprintf("could not update task %s in plan file", task.FullID()))
	}

	// Find latest session file.
	sessionPath, _, err := progress.FindLatestSessionPath(rootDir, plan.Slug, task.FullID())
	if err != nil {
		return etcherr.WrapIO("finding session file", err).
			WithHint(fmt.Sprintf("run 'etch progress start %s' first to create a session", task.FullID()))
	}

	// Update progress file status to blocked.
	if err := progress.UpdateStatus(sessionPath, "blocked"); err != nil {
		return etcherr.WrapIO("updating progress file status", err)
	}

	// Append reason to Blockers section.
	reason := c.String("reason")
	entry := fmt.Sprintf("- %s", reason)
	if err := progress.AppendToSection(sessionPath, "Blockers", entry); err != nil {
		return etcherr.WrapIO("appending to blockers section", err)
	}

	fmt.Printf("Task %s blocked: %s\n", task.FullID(), reason)
	return nil
}

func progressFailCmd() *cli.Command {
	return &cli.Command{
		Name:      "fail",
		Usage:     "Mark a task as failed",
		ArgsUsage: "[plan-name] <task-id>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "reason",
				Usage:    "reason the task failed (required)",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			return runProgressFail(c)
		},
	}
}

func runProgressFail(c *cli.Context) error {
	rootDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	plans, err := etchcontext.DiscoverPlans(rootDir)
	if err != nil {
		return err
	}

	var planSlug, taskID string
	args := c.Args().Slice()
	switch len(args) {
	case 0:
		return etcherr.Usage("task ID is required").
			WithHint("usage: etch progress fail [plan-name] <task-id> --reason \"text\"")
	case 1:
		taskID = args[0]
	case 2:
		planSlug = args[0]
		taskID = args[1]
	default:
		return etcherr.Usage("too many arguments").
			WithHint("usage: etch progress fail [plan-name] <task-id> --reason \"text\"")
	}

	plan, task, err := etchcontext.ResolveTask(plans, planSlug, taskID, rootDir)
	if err != nil {
		return err
	}

	// Update plan file status to failed.
	if err := serializer.UpdateTaskStatus(plan.FilePath, task.FullID(), models.StatusFailed); err != nil {
		return etcherr.WrapIO("updating task status", err).
			WithHint(fmt.Sprintf("could not update task %s in plan file", task.FullID()))
	}

	// Find latest session file.
	sessionPath, _, err := progress.FindLatestSessionPath(rootDir, plan.Slug, task.FullID())
	if err != nil {
		return etcherr.WrapIO("finding session file", err).
			WithHint(fmt.Sprintf("run 'etch progress start %s' first to create a session", task.FullID()))
	}

	// Update progress file status to failed.
	if err := progress.UpdateStatus(sessionPath, "failed"); err != nil {
		return etcherr.WrapIO("updating progress file status", err)
	}

	// Append reason to Blockers section.
	reason := c.String("reason")
	entry := fmt.Sprintf("- %s", reason)
	if err := progress.AppendToSection(sessionPath, "Blockers", entry); err != nil {
		return etcherr.WrapIO("appending to blockers section", err)
	}

	fmt.Printf("Task %s failed: %s\n", task.FullID(), reason)
	return nil
}

