package generator

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gsigler/etch/internal/api"
	etcherr "github.com/gsigler/etch/internal/errors"
	"github.com/gsigler/etch/internal/models"
	"github.com/gsigler/etch/internal/parser"
	"github.com/gsigler/etch/internal/progress"
)

// ReplanTarget identifies what to replan: a single task or an entire feature.
type ReplanTarget struct {
	Type      string // "task" or "feature"
	TaskID    string // e.g. "1.2" (for task targets)
	FeatureNum int   // e.g. 2 (for feature targets)
	Task      *models.Task
	Feature   *models.Feature
}

// ReplanResult holds the output of a replan operation.
type ReplanResult struct {
	OldMarkdown string
	NewMarkdown string
	NewPlan     *models.Plan
	BackupPath  string
	Diff        string
	Target      ReplanTarget
}

// ResolveTarget parses a target string and resolves it against the plan.
// Supported forms:
//   - "1.2" or "1.2b" → task by ID
//   - "feature:2" → feature by number
//   - "Feature Title" → feature by title (case-insensitive)
//   - "2" → ambiguous: prefer task interpretation (e.g. Task 2 in single-feature plan)
func ResolveTarget(plan *models.Plan, target string) (ReplanTarget, error) {
	if target == "" {
		return ReplanTarget{}, etcherr.Usage("target is required").
			WithHint("use a task ID (e.g. 1.2), feature reference (e.g. feature:2), or feature title")
	}

	// Check for "feature:N" prefix.
	if strings.HasPrefix(strings.ToLower(target), "feature:") {
		numStr := strings.TrimPrefix(strings.ToLower(target), "feature:")
		num, err := strconv.Atoi(strings.TrimSpace(numStr))
		if err != nil {
			return ReplanTarget{}, etcherr.Usage(fmt.Sprintf("invalid feature number: %q", numStr)).
			WithHint("use feature:N where N is the feature number (e.g. feature:2)")
		}
		return resolveFeatureByNumber(plan, num)
	}

	// Check for task ID pattern: N.M or N.Mb (with optional suffix).
	taskIDRe := regexp.MustCompile(`^(\d+)\.(\d+)([a-z]?)$`)
	if m := taskIDRe.FindStringSubmatch(target); m != nil {
		return resolveTaskByID(plan, target)
	}

	// Check for plain number — prefer task interpretation.
	if num, err := strconv.Atoi(target); err == nil {
		// Try as task first (single-feature plan: Task N).
		taskID := fmt.Sprintf("1.%d", num)
		if t := plan.TaskByID(taskID); t != nil {
			return ReplanTarget{
				Type:   "task",
				TaskID: taskID,
				Task:   t,
			}, nil
		}
		// Try as feature number.
		rt, err := resolveFeatureByNumber(plan, num)
		if err == nil {
			return rt, nil
		}
		return ReplanTarget{}, etcherr.Project(fmt.Sprintf("no task %q or feature %d found in plan", taskID, num)).
			WithHint("run 'etch status' to see available tasks and features")
	}

	// Try matching as feature title (case-insensitive).
	lower := strings.ToLower(target)
	for i := range plan.Features {
		if strings.ToLower(plan.Features[i].Title) == lower {
			return ReplanTarget{
				Type:       "feature",
				FeatureNum: plan.Features[i].Number,
				Feature:    &plan.Features[i],
			}, nil
		}
	}

	// Try substring match on feature title.
	for i := range plan.Features {
		if strings.Contains(strings.ToLower(plan.Features[i].Title), lower) {
			return ReplanTarget{
				Type:       "feature",
				FeatureNum: plan.Features[i].Number,
				Feature:    &plan.Features[i],
			}, nil
		}
	}

	return ReplanTarget{}, etcherr.Project(fmt.Sprintf("could not resolve target %q", target)).
		WithHint("use a task ID (e.g. 1.2), feature reference (e.g. feature:2), or feature title")
}

func resolveTaskByID(plan *models.Plan, id string) (ReplanTarget, error) {
	t := plan.TaskByID(id)
	if t == nil {
		return ReplanTarget{}, etcherr.Project(fmt.Sprintf("task %q not found in plan", id)).
			WithHint("run 'etch status' to see available tasks")
	}
	return ReplanTarget{
		Type:   "task",
		TaskID: id,
		Task:   t,
	}, nil
}

func resolveFeatureByNumber(plan *models.Plan, num int) (ReplanTarget, error) {
	for i := range plan.Features {
		if plan.Features[i].Number == num {
			return ReplanTarget{
				Type:       "feature",
				FeatureNum: num,
				Feature:    &plan.Features[i],
			}, nil
		}
	}
	return ReplanTarget{}, etcherr.Project(fmt.Sprintf("feature %d not found in plan", num)).
		WithHint("run 'etch status' to see available features")
}

// BuildReplanScope constructs the scope description for the replan prompt.
// It adapts the prompt based on whether this is a planning issue (no sessions)
// or an approach issue (has failed/blocked sessions).
func BuildReplanScope(target ReplanTarget, sessions map[string][]models.SessionProgress) string {
	var b strings.Builder

	switch target.Type {
	case "task":
		b.WriteString(fmt.Sprintf("Replan Task %s: %s\n\n", target.TaskID, target.Task.Title))

		taskSessions := sessions[target.TaskID]
		if len(taskSessions) == 0 {
			// Planning issue — no attempts yet.
			b.WriteString("This task has not been attempted yet. This is a planning issue.\n\n")
			b.WriteString("Consider:\n")
			b.WriteString("- Is the scope right? Is it too broad or too narrow?\n")
			b.WriteString("- Are the acceptance criteria clear and testable?\n")
			b.WriteString("- Should it be split into smaller tasks?\n")
			b.WriteString("- Are the file assignments correct?\n")
			b.WriteString("- Are dependencies properly specified?\n")
		} else {
			// Approach issue — has session history.
			b.WriteString(fmt.Sprintf("This task has been attempted %d time(s). Here's what happened:\n\n", len(taskSessions)))
			b.WriteString(formatSessionHistory(taskSessions))
			b.WriteString("\nSuggest an alternative approach, break the task down differently, or restructure it.\n")
		}

	case "feature":
		b.WriteString(fmt.Sprintf("Replan Feature %d: %s\n\n", target.FeatureNum, target.Feature.Title))
		b.WriteString("Replan all tasks within this feature.\n\n")

		// Note completed tasks that should be preserved.
		var completed, pending []string
		for _, t := range target.Feature.Tasks {
			if t.Status == models.StatusCompleted {
				completed = append(completed, fmt.Sprintf("- Task %s: %s [completed]", t.FullID(), t.Title))
			} else {
				pending = append(pending, fmt.Sprintf("- Task %s: %s [%s]", t.FullID(), t.Title, t.Status))
			}
		}

		if len(completed) > 0 {
			b.WriteString("Completed tasks (MUST be preserved as-is):\n")
			b.WriteString(strings.Join(completed, "\n"))
			b.WriteString("\n\n")
		}
		if len(pending) > 0 {
			b.WriteString("Tasks to replan:\n")
			b.WriteString(strings.Join(pending, "\n"))
			b.WriteString("\n\n")
		}

		// Include session history for all tasks in the feature.
		var historyParts []string
		for _, t := range target.Feature.Tasks {
			taskSessions := sessions[t.FullID()]
			if len(taskSessions) > 0 {
				historyParts = append(historyParts, fmt.Sprintf("### Task %s: %s\n%s", t.FullID(), t.Title, formatSessionHistory(taskSessions)))
			}
		}
		if len(historyParts) > 0 {
			b.WriteString("Session history for tasks in this feature:\n\n")
			b.WriteString(strings.Join(historyParts, "\n"))
		}
	}

	return b.String()
}

// formatSessionHistory formats session progress entries for inclusion in the prompt.
func formatSessionHistory(sessions []models.SessionProgress) string {
	var b strings.Builder
	for _, s := range sessions {
		b.WriteString(fmt.Sprintf("**Session %03d** (Status: %s)\n", s.SessionNumber, s.Status))
		if len(s.ChangesMade) > 0 {
			b.WriteString("Changes: " + strings.Join(s.ChangesMade, ", ") + "\n")
		}
		if s.Decisions != "" {
			b.WriteString("Decisions: " + s.Decisions + "\n")
		}
		if s.Blockers != "" {
			b.WriteString("Blockers: " + s.Blockers + "\n")
		}
		if s.Next != "" {
			b.WriteString("Next: " + s.Next + "\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

// Replan sends a plan with replan context to the AI for replanning.
// It validates the response and returns a ReplanResult with the old/new markdown and diff.
func Replan(client APIClient, planPath, rootDir string, target ReplanTarget, streamCb api.StreamCallback) (ReplanResult, error) {
	// 1. Read the plan.
	oldMarkdown, err := os.ReadFile(planPath)
	if err != nil {
		return ReplanResult{}, etcherr.WrapIO("reading plan", err)
	}

	plan, err := parser.Parse(strings.NewReader(string(oldMarkdown)))
	if err != nil {
		return ReplanResult{}, etcherr.WrapParse("parsing plan", err)
	}

	// 2. Read session history.
	sessions, err := progress.ReadAll(rootDir, plan.Slug)
	if err != nil {
		// Non-fatal: proceed without session history.
		sessions = make(map[string][]models.SessionProgress)
	}

	// 3. Build scope and session history.
	scope := BuildReplanScope(target, sessions)
	sessionHistory := buildAllSessionHistory(sessions)

	// 4. Backup the plan.
	backupPath, err := BackupPlan(planPath, rootDir)
	if err != nil {
		return ReplanResult{}, err
	}

	// 5. Build prompts and call the API.
	systemPrompt := buildReplanSystemPrompt()
	userMessage := buildReplanUserMessage(string(oldMarkdown), scope, sessionHistory)

	fullText, err := client.SendStream(systemPrompt, userMessage, streamCb)
	if err != nil {
		return ReplanResult{}, etcherr.WrapAPI("replanning", err)
	}

	// 6. Extract and validate the new plan.
	newMarkdown := extractMarkdown(fullText)
	newPlan, err := parser.Parse(strings.NewReader(newMarkdown))
	if err != nil {
		return ReplanResult{}, etcherr.WrapParse("replanned plan failed validation", err).
			WithHint("the AI response may not follow the expected format — try again")
	}

	// 7. Generate diff.
	diff := GenerateDiff(string(oldMarkdown), newMarkdown)

	return ReplanResult{
		OldMarkdown: string(oldMarkdown),
		NewMarkdown: newMarkdown,
		NewPlan:     newPlan,
		BackupPath:  backupPath,
		Diff:        diff,
		Target:      target,
	}, nil
}

// buildAllSessionHistory formats all session history for the user message.
func buildAllSessionHistory(sessions map[string][]models.SessionProgress) string {
	if len(sessions) == 0 {
		return ""
	}

	var b strings.Builder
	for taskID, taskSessions := range sessions {
		if len(taskSessions) == 0 {
			continue
		}
		b.WriteString(fmt.Sprintf("### Task %s\n", taskID))
		b.WriteString(formatSessionHistory(taskSessions))
	}
	return b.String()
}

// ApplyReplan writes the replanned plan to disk, overwriting the original.
func ApplyReplan(planPath, newMarkdown string) error {
	return ApplyRefinement(planPath, newMarkdown)
}
