package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	etcherr "github.com/gsigler/etch/internal/errors"
	"github.com/gsigler/etch/internal/models"
	"github.com/gsigler/etch/internal/parser"
	"github.com/gsigler/etch/internal/progress"
)

const (
	plansDir   = ".etch/plans"
	contextDir = ".etch/context"
)

// Result holds the output of assembling a context prompt.
type Result struct {
	ContextPath  string
	ProgressPath string
	SessionNum   int
	TokenEstimate int
}

// FeatureResult holds the output of assembling a feature-level context prompt.
type FeatureResult struct {
	ContextPath   string
	ProgressPaths map[string]string // task ID → progress file path
	SessionNum    int
	TokenEstimate int
}

// DiscoverPlans finds all plan files in the project root.
func DiscoverPlans(rootDir string) ([]*models.Plan, error) {
	dir := filepath.Join(rootDir, plansDir)
	pattern := filepath.Join(dir, "*.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, etcherr.WrapIO("globbing plans", err)
	}
	if len(matches) == 0 {
		return nil, etcherr.Project("no plan files found").
			WithHint("run 'etch plan <description>' to create one")
	}

	var plans []*models.Plan
	for _, path := range matches {
		plan, err := parser.ParseFile(path)
		if err != nil {
			continue // skip unparseable files
		}
		plans = append(plans, plan)
	}
	if len(plans) == 0 {
		return nil, etcherr.Project("no valid plan files found").
			WithHint("check that plan files in .etch/plans/ follow the expected markdown format")
	}
	return plans, nil
}

// ResolveFeature resolves a feature by number within a plan identified by slug.
func ResolveFeature(plans []*models.Plan, planSlug string, featureNumber int) (*models.Plan, *models.Feature, error) {
	// Filter to specific plan if slug given.
	var candidates []*models.Plan
	if planSlug != "" {
		for _, p := range plans {
			if p.Slug == planSlug {
				candidates = append(candidates, p)
			}
		}
		if len(candidates) == 0 {
			return nil, nil, etcherr.Project(fmt.Sprintf("no plan found with slug %q", planSlug)).
				WithHint("run 'etch list' to see available plans")
		}
	} else {
		candidates = plans
	}

	for _, plan := range candidates {
		for i := range plan.Features {
			if plan.Features[i].Number == featureNumber {
				return plan, &plan.Features[i], nil
			}
		}
	}

	return nil, nil, etcherr.Project(fmt.Sprintf("feature %d not found", featureNumber)).
		WithHint("run 'etch status' to see available features")
}

// featurePendingTasks returns the ordered list of pending tasks in a feature
// whose dependencies are satisfied or are within the feature itself, skipping
// completed tasks.
func featurePendingTasks(plan *models.Plan, feature *models.Feature, allProgress map[string][]models.SessionProgress) []*models.Task {
	// Build set of task IDs within this feature for intra-feature dep checking.
	featureTaskIDs := make(map[string]bool)
	for _, t := range feature.Tasks {
		featureTaskIDs[t.FullID()] = true
	}

	var result []*models.Task
	for i := range feature.Tasks {
		task := &feature.Tasks[i]
		status := effectiveStatus(task, allProgress)
		if status != models.StatusPending {
			continue
		}
		if featureDepsReady(plan, task, featureTaskIDs, allProgress) {
			result = append(result, task)
		}
	}
	return result
}

// featureDepsReady checks if a task's dependencies are satisfied for feature-level
// execution. A dependency is satisfied if it's completed, or if it belongs to the
// same feature (intra-feature deps are allowed since the feature runs as a group).
func featureDepsReady(plan *models.Plan, task *models.Task, featureTaskIDs map[string]bool, allProgress map[string][]models.SessionProgress) bool {
	for _, dep := range task.DependsOn {
		depID := extractTaskID(dep)
		// Intra-feature dependencies are considered satisfied.
		if featureTaskIDs[depID] {
			continue
		}
		depTask := plan.TaskByID(depID)
		if depTask == nil {
			continue
		}
		status := effectiveStatus(depTask, allProgress)
		if status != models.StatusCompleted {
			return false
		}
	}
	return true
}

// ResolveTask resolves a task from the given arguments.
// Forms:
//   - "" (empty) → auto-select next pending task
//   - "1.2" → Feature 1, Task 2
//   - "2" → Task 2 in single-feature plan
//   - planSlug + "1.2" → specific plan, specific task
func ResolveTask(plans []*models.Plan, planSlug, taskID string, rootDir string) (*models.Plan, *models.Task, error) {
	// Filter to specific plan if slug given.
	var candidates []*models.Plan
	if planSlug != "" {
		for _, p := range plans {
			if p.Slug == planSlug {
				candidates = append(candidates, p)
			}
		}
		if len(candidates) == 0 {
			return nil, nil, etcherr.Project(fmt.Sprintf("no plan found with slug %q", planSlug)).
				WithHint("run 'etch list' to see available plans")
		}
	} else {
		candidates = plans
	}

	// Auto-select if no task ID given.
	if taskID == "" {
		return autoSelectTask(candidates, rootDir)
	}

	// Resolve task ID within candidates.
	for _, plan := range candidates {
		task := resolveTaskID(plan, taskID)
		if task != nil {
			return plan, task, nil
		}
	}

	return nil, nil, etcherr.Project(fmt.Sprintf("task %q not found", taskID)).
		WithHint("run 'etch status' to see available tasks")
}

// resolveTaskID resolves a task ID within a plan.
// Supports "1.2" (full ID) and "2" (task number in single-feature plan).
func resolveTaskID(plan *models.Plan, id string) *models.Task {
	// Try full ID first (e.g. "1.2", "1.3b").
	if task := plan.TaskByID(id); task != nil {
		return task
	}

	// Try as bare task number in single-feature plan (e.g. "2" → "1.2").
	if len(plan.Features) == 1 {
		fullID := fmt.Sprintf("1.%s", id)
		if task := plan.TaskByID(fullID); task != nil {
			return task
		}
	}

	return nil
}

// autoSelectTask finds the next pending task where all dependencies are satisfied.
func autoSelectTask(plans []*models.Plan, rootDir string) (*models.Plan, *models.Task, error) {
	type candidate struct {
		plan *models.Plan
		task *models.Task
	}
	var candidates []candidate

	for _, plan := range plans {
		allProgress, _ := progress.ReadAll(rootDir, plan.Slug)
		for i := range plan.Features {
			for j := range plan.Features[i].Tasks {
				task := &plan.Features[i].Tasks[j]
				status := effectiveStatus(task, allProgress)
				if status != models.StatusPending {
					continue
				}
				if allDepsCompleted(plan, task, allProgress) {
					candidates = append(candidates, candidate{plan, task})
				}
			}
		}
	}

	if len(candidates) == 0 {
		return nil, nil, etcherr.Project("no pending tasks with satisfied dependencies").
			WithHint("all tasks may be completed or blocked — run 'etch status' to check")
	}

	// If only one candidate or all from same plan, return first.
	return candidates[0].plan, candidates[0].task, nil
}

// NeedsPlanPicker returns true if auto-select would be ambiguous across plans.
func NeedsPlanPicker(plans []*models.Plan, rootDir string) (bool, []*models.Plan) {
	planSet := make(map[string]*models.Plan)
	for _, plan := range plans {
		allProgress, _ := progress.ReadAll(rootDir, plan.Slug)
		for i := range plan.Features {
			for j := range plan.Features[i].Tasks {
				task := &plan.Features[i].Tasks[j]
				status := effectiveStatus(task, allProgress)
				if status != models.StatusPending {
					continue
				}
				if allDepsCompleted(plan, task, allProgress) {
					planSet[plan.Slug] = plan
				}
			}
		}
	}

	if len(planSet) <= 1 {
		return false, nil
	}
	var result []*models.Plan
	for _, p := range planSet {
		result = append(result, p)
	}
	return true, result
}

// effectiveStatus returns a task's status, considering progress files.
func effectiveStatus(task *models.Task, allProgress map[string][]models.SessionProgress) models.Status {
	sessions := allProgress[task.FullID()]
	if len(sessions) == 0 {
		return task.Status
	}
	// Use the latest session's status if filled in.
	latest := sessions[len(sessions)-1]
	if latest.Status != "" && latest.Status != "pending" {
		return models.ParseStatus(latest.Status)
	}
	return task.Status
}

// allDepsCompleted checks if all of a task's dependencies are completed.
func allDepsCompleted(plan *models.Plan, task *models.Task, allProgress map[string][]models.SessionProgress) bool {
	for _, dep := range task.DependsOn {
		depID := extractTaskID(dep)
		depTask := plan.TaskByID(depID)
		if depTask == nil {
			continue // unknown dependency, skip
		}
		status := effectiveStatus(depTask, allProgress)
		if status != models.StatusCompleted {
			return false
		}
	}
	return true
}

// extractTaskID extracts a task ID from dependency strings like "Task 1.2" or "1.2".
func extractTaskID(dep string) string {
	dep = strings.TrimSpace(dep)
	// Strip "Task " prefix if present.
	if strings.HasPrefix(dep, "Task ") {
		dep = strings.TrimPrefix(dep, "Task ")
	}
	// Strip any trailing description after the ID (e.g. "1.2 (completed)").
	parts := strings.Fields(dep)
	if len(parts) > 0 {
		return parts[0]
	}
	return dep
}

// Assemble builds the context prompt for the given plan and task.
func Assemble(rootDir string, plan *models.Plan, task *models.Task) (Result, error) {
	allProgress, err := progress.ReadAll(rootDir, plan.Slug)
	if err != nil {
		allProgress = make(map[string][]models.SessionProgress)
	}

	// Create session progress file.
	progressPath, err := progress.WriteSession(rootDir, plan, task)
	if err != nil {
		return Result{}, etcherr.WrapIO("creating progress file", err)
	}

	// Determine session number from filename.
	sessionNum := sessionNumberFromPath(progressPath)

	// Build context content.
	content := buildTemplate(plan, task, allProgress, sessionNum, progressPath, rootDir)

	// Write context file.
	ctxDir := filepath.Join(rootDir, contextDir)
	if err := os.MkdirAll(ctxDir, 0o755); err != nil {
		return Result{}, etcherr.WrapIO("creating context dir", err)
	}

	ctxFilename := fmt.Sprintf("%s--task-%s--%03d.md", plan.Slug, task.FullID(), sessionNum)
	ctxPath := filepath.Join(ctxDir, ctxFilename)
	if err := os.WriteFile(ctxPath, []byte(content), 0o644); err != nil {
		return Result{}, etcherr.WrapIO("writing context file", err)
	}

	tokenEstimate := len(content) * 10 / 35 // chars / 3.5

	return Result{
		ContextPath:   ctxPath,
		ProgressPath:  progressPath,
		SessionNum:    sessionNum,
		TokenEstimate: tokenEstimate,
	}, nil
}

// AssembleFeature builds a combined context prompt for all actionable tasks in a feature.
func AssembleFeature(rootDir string, plan *models.Plan, feature *models.Feature) (FeatureResult, error) {
	allProgress, err := progress.ReadAll(rootDir, plan.Slug)
	if err != nil {
		allProgress = make(map[string][]models.SessionProgress)
	}

	tasks := featurePendingTasks(plan, feature, allProgress)
	if len(tasks) == 0 {
		return FeatureResult{}, etcherr.Project("no actionable pending tasks in feature").
			WithHint("all tasks may be completed or have unsatisfied external dependencies — run 'etch status' to check")
	}

	// Create progress files for each pending task.
	progressPaths := make(map[string]string, len(tasks))
	var sessionNum int
	for _, task := range tasks {
		pPath, err := progress.WriteSession(rootDir, plan, task)
		if err != nil {
			return FeatureResult{}, etcherr.WrapIO(fmt.Sprintf("creating progress file for task %s", task.FullID()), err)
		}
		progressPaths[task.FullID()] = pPath
		n := sessionNumberFromPath(pPath)
		if n > sessionNum {
			sessionNum = n
		}
	}

	// Build combined context content.
	content := buildFeatureTemplate(plan, feature, tasks, allProgress, sessionNum, progressPaths, rootDir)

	// Write context file.
	ctxDir := filepath.Join(rootDir, contextDir)
	if err := os.MkdirAll(ctxDir, 0o755); err != nil {
		return FeatureResult{}, etcherr.WrapIO("creating context dir", err)
	}

	ctxFilename := fmt.Sprintf("%s--feature-%d--%03d.md", plan.Slug, feature.Number, sessionNum)
	ctxPath := filepath.Join(ctxDir, ctxFilename)
	if err := os.WriteFile(ctxPath, []byte(content), 0o644); err != nil {
		return FeatureResult{}, etcherr.WrapIO("writing context file", err)
	}

	tokenEstimate := len(content) * 10 / 35

	return FeatureResult{
		ContextPath:   ctxPath,
		ProgressPaths: progressPaths,
		SessionNum:    sessionNum,
		TokenEstimate: tokenEstimate,
	}, nil
}

func buildFeatureTemplate(plan *models.Plan, feature *models.Feature, tasks []*models.Task, allProgress map[string][]models.SessionProgress, sessionNum int, progressPaths map[string]string, rootDir string) string {
	var b strings.Builder

	// Header.
	b.WriteString("# Etch Context — Feature Implementation\n\n")
	b.WriteString("You are working on an entire feature as part of an implementation plan managed by Etch.\n")
	b.WriteString("Work through the tasks in order, completing each one before moving to the next.\n\n")

	// Plan section.
	b.WriteString(fmt.Sprintf("## Plan: %s\n", plan.Title))
	overview := condenseOverview(plan.Overview)
	if overview != "" {
		b.WriteString(overview + "\n")
	}
	b.WriteString("\n")

	// Current plan state.
	b.WriteString("## Current Plan State\n")
	for _, f := range plan.Features {
		b.WriteString(fmt.Sprintf("Feature %d: %s\n", f.Number, f.Title))
		for _, t := range f.Tasks {
			status := effectiveStatus(&t, allProgress)
			icon := status.Icon()
			annotation := formatFeatureTaskAnnotation(&t, feature, tasks, status)
			b.WriteString(fmt.Sprintf("  %s Task %s: %s %s\n", icon, t.FullID(), t.Title, annotation))
		}
	}
	b.WriteString("\n")

	// Your Feature section — summary of all tasks being worked on.
	b.WriteString(fmt.Sprintf("## Your Feature: Feature %d — %s\n\n", feature.Number, feature.Title))
	b.WriteString(fmt.Sprintf("You are working on **%d tasks** in this feature. Complete them in order.\n\n", len(tasks)))
	for i, task := range tasks {
		b.WriteString(fmt.Sprintf("  %d. **Task %s** — %s", i+1, task.FullID(), task.Title))
		if len(task.Files) > 0 {
			b.WriteString(fmt.Sprintf(" (`%s`)", strings.Join(task.Files, "`, `")))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n---\n\n")

	// Full details for each task.
	for i, task := range tasks {
		b.WriteString(fmt.Sprintf("## Task %d of %d: Task %s — %s\n", i+1, len(tasks), task.FullID(), task.Title))
		if task.Complexity != "" {
			b.WriteString(fmt.Sprintf("**Complexity:** %s\n", task.Complexity))
		}
		if len(task.Files) > 0 {
			b.WriteString(fmt.Sprintf("**Files in Scope:** %s\n", strings.Join(task.Files, ", ")))
		}
		if len(task.DependsOn) > 0 {
			depParts := make([]string, len(task.DependsOn))
			for j, dep := range task.DependsOn {
				depID := extractTaskID(dep)
				depTask := plan.TaskByID(depID)
				if depTask != nil {
					status := effectiveStatus(depTask, allProgress)
					depParts[j] = fmt.Sprintf("%s (%s)", dep, status)
				} else {
					depParts[j] = dep
				}
			}
			b.WriteString(fmt.Sprintf("**Depends on:** %s\n", strings.Join(depParts, ", ")))
		}
		b.WriteString("\n")

		if task.Description != "" {
			b.WriteString(task.Description + "\n\n")
		}

		// Acceptance criteria.
		if len(task.Criteria) > 0 {
			b.WriteString("### Acceptance Criteria\n")
			criteriaMap := make(map[string]bool)
			sessions := allProgress[task.FullID()]
			for _, s := range sessions {
				for _, c := range s.CriteriaUpdates {
					if c.IsMet {
						criteriaMap[c.Description] = true
					}
				}
			}
			for _, c := range task.Criteria {
				check := " "
				if c.IsMet || criteriaMap[c.Description] {
					check = "x"
				}
				b.WriteString(fmt.Sprintf("- [%s] %s\n", check, c.Description))
			}
			b.WriteString("\n")
		}

		// Comments.
		if len(task.Comments) > 0 {
			b.WriteString("### Review Comments\n")
			for _, c := range task.Comments {
				b.WriteString(fmt.Sprintf("> 💬 %s\n\n", c))
			}
		}

		// Previous sessions for this task.
		sessions := allProgress[task.FullID()]
		var priorSessions []models.SessionProgress
		for _, s := range sessions {
			if s.SessionNumber < sessionNum {
				priorSessions = append(priorSessions, s)
			}
		}
		if len(priorSessions) > 0 {
			b.WriteString("### Previous Sessions\n")
			for _, s := range priorSessions {
				b.WriteString(fmt.Sprintf("\n**Session %03d (%s, %s):**\n", s.SessionNumber, s.Started, s.Status))
				if len(s.ChangesMade) > 0 {
					b.WriteString(fmt.Sprintf("Changes: %s\n", strings.Join(s.ChangesMade, ", ")))
				}
				if s.Decisions != "" {
					b.WriteString(fmt.Sprintf("Decisions: %s\n", s.Decisions))
				}
				if s.Blockers != "" {
					b.WriteString(fmt.Sprintf("Blockers: %s\n", s.Blockers))
				}
				if s.Next != "" {
					b.WriteString(fmt.Sprintf("Next: %s\n", s.Next))
				}
			}
			b.WriteString("\n")
		}

		// Per-task progress reporting.
		relProgress, _ := filepath.Rel(rootDir, progressPaths[task.FullID()])
		if relProgress == "" {
			relProgress = progressPaths[task.FullID()]
		}
		b.WriteString(fmt.Sprintf("### Progress for Task %s\n\n", task.FullID()))
		b.WriteString(fmt.Sprintf("Progress file: `%s`\n\n", relProgress))
		b.WriteString(fmt.Sprintf("```bash\n"))
		b.WriteString(fmt.Sprintf("etch progress start -p %s -t %s\n", plan.Slug, task.FullID()))
		b.WriteString(fmt.Sprintf("etch progress update -p %s -t %s -m \"description\"\n", plan.Slug, task.FullID()))
		b.WriteString(fmt.Sprintf("etch progress criteria -p %s -t %s --check \"criterion text\"\n", plan.Slug, task.FullID()))
		b.WriteString(fmt.Sprintf("etch progress done -p %s -t %s\n", plan.Slug, task.FullID()))
		b.WriteString("```\n\n")

		if i < len(tasks)-1 {
			b.WriteString("---\n\n")
		}
	}

	// General workflow instructions.
	b.WriteString("\n## Workflow\n\n")
	b.WriteString("Work through the tasks **in order** (Task 1 first, then Task 2, etc.).\n")
	b.WriteString("For each task:\n\n")
	b.WriteString("1. Run `etch progress start` for the task\n")
	b.WriteString("2. Implement the changes described\n")
	b.WriteString("3. Log updates with `etch progress update` as you work\n")
	b.WriteString("4. Check off criteria with `etch progress criteria --check`\n")
	b.WriteString("5. Mark the task done with `etch progress done`\n")
	b.WriteString("6. Move to the next task\n\n")
	b.WriteString("### Rules\n")
	b.WriteString("- Stay within the files listed in scope for each task. Ask before modifying others.\n")
	b.WriteString("- Do NOT modify the plan file directly — use `etch progress` commands instead.\n")
	b.WriteString("- Log updates frequently so future sessions have context.\n")
	b.WriteString("- Complete each task fully before starting the next one.\n")

	return b.String()
}

// formatFeatureTaskAnnotation formats the annotation for a task in the plan state
// section of a feature context.
func formatFeatureTaskAnnotation(t *models.Task, feature *models.Feature, activeTasks []*models.Task, status models.Status) string {
	// Check if this task is one of the active tasks in the feature.
	for _, at := range activeTasks {
		if t.FullID() == at.FullID() {
			return "(in_progress — included in this feature run)"
		}
	}
	switch status {
	case models.StatusCompleted:
		return "(completed)"
	case models.StatusInProgress:
		return "(in_progress)"
	case models.StatusBlocked:
		return "(blocked)"
	case models.StatusFailed:
		return "(failed)"
	case models.StatusPending:
		if len(t.DependsOn) > 0 {
			depIDs := make([]string, len(t.DependsOn))
			for i, dep := range t.DependsOn {
				depIDs[i] = extractTaskID(dep)
			}
			return fmt.Sprintf("(pending, depends on %s)", strings.Join(depIDs, ", "))
		}
		return "(pending)"
	default:
		return fmt.Sprintf("(%s)", status)
	}
}

func sessionNumberFromPath(path string) int {
	base := filepath.Base(path)
	ext := strings.TrimSuffix(base, ".md")
	parts := strings.Split(ext, "--")
	if len(parts) < 3 {
		return 1
	}
	var n int
	fmt.Sscanf(parts[len(parts)-1], "%d", &n)
	if n == 0 {
		return 1
	}
	return n
}

func buildTemplate(plan *models.Plan, task *models.Task, allProgress map[string][]models.SessionProgress, sessionNum int, progressPath, rootDir string) string {
	var b strings.Builder

	// Header.
	b.WriteString("# Etch Context — Implementation Task\n\n")
	b.WriteString("You are working on a task as part of an implementation plan managed by Etch.\n\n")

	// Plan section.
	b.WriteString(fmt.Sprintf("## Plan: %s\n", plan.Title))
	overview := condenseOverview(plan.Overview)
	if overview != "" {
		b.WriteString(overview + "\n")
	}
	b.WriteString("\n")

	// Current plan state.
	b.WriteString("## Current Plan State\n")
	for _, f := range plan.Features {
		b.WriteString(fmt.Sprintf("Feature %d: %s\n", f.Number, f.Title))
		for _, t := range f.Tasks {
			status := effectiveStatus(&t, allProgress)
			icon := status.Icon()
			annotation := formatTaskAnnotation(&t, task, status, allProgress)
			b.WriteString(fmt.Sprintf("  %s Task %s: %s %s\n", icon, t.FullID(), t.Title, annotation))
		}
	}
	b.WriteString("\n")

	// Your task section.
	b.WriteString(fmt.Sprintf("## Your Task: Task %s — %s\n", task.FullID(), task.Title))
	if task.Complexity != "" {
		b.WriteString(fmt.Sprintf("**Complexity:** %s\n", task.Complexity))
	}
	if len(task.Files) > 0 {
		b.WriteString(fmt.Sprintf("**Files in Scope:** %s\n", strings.Join(task.Files, ", ")))
	}
	if len(task.DependsOn) > 0 {
		depParts := make([]string, len(task.DependsOn))
		for i, dep := range task.DependsOn {
			depID := extractTaskID(dep)
			depTask := plan.TaskByID(depID)
			if depTask != nil {
				status := effectiveStatus(depTask, allProgress)
				depParts[i] = fmt.Sprintf("%s (%s)", dep, status)
			} else {
				depParts[i] = dep
			}
		}
		b.WriteString(fmt.Sprintf("**Depends on:** %s\n", strings.Join(depParts, ", ")))
	}
	b.WriteString("\n")

	// Full task description.
	if task.Description != "" {
		b.WriteString(task.Description + "\n\n")
	}

	// Acceptance criteria.
	if len(task.Criteria) > 0 {
		b.WriteString("### Acceptance Criteria\n")
		// Merge with progress updates if available.
		criteriaMap := make(map[string]bool)
		sessions := allProgress[task.FullID()]
		for _, s := range sessions {
			for _, c := range s.CriteriaUpdates {
				if c.IsMet {
					criteriaMap[c.Description] = true
				}
			}
		}
		for _, c := range task.Criteria {
			check := " "
			if c.IsMet || criteriaMap[c.Description] {
				check = "x"
			}
			b.WriteString(fmt.Sprintf("- [%s] %s\n", check, c.Description))
		}
		b.WriteString("\n")
	}

	// Comments.
	if len(task.Comments) > 0 {
		b.WriteString("### Review Comments\n")
		for _, c := range task.Comments {
			b.WriteString(fmt.Sprintf("> 💬 %s\n\n", c))
		}
	}

	// Previous sessions.
	sessions := allProgress[task.FullID()]
	// Exclude the session we just created.
	var priorSessions []models.SessionProgress
	for _, s := range sessions {
		if s.SessionNumber < sessionNum {
			priorSessions = append(priorSessions, s)
		}
	}
	if len(priorSessions) > 0 {
		b.WriteString("### Previous Sessions\n")
		for _, s := range priorSessions {
			b.WriteString(fmt.Sprintf("\n**Session %03d (%s, %s):**\n", s.SessionNumber, s.Started, s.Status))
			if len(s.ChangesMade) > 0 {
				b.WriteString(fmt.Sprintf("Changes: %s\n", strings.Join(s.ChangesMade, ", ")))
			}
			if s.Decisions != "" {
				b.WriteString(fmt.Sprintf("Decisions: %s\n", s.Decisions))
			}
			if s.Blockers != "" {
				b.WriteString(fmt.Sprintf("Blockers: %s\n", s.Blockers))
			}
			if s.Next != "" {
				b.WriteString(fmt.Sprintf("Next: %s\n", s.Next))
			}
		}
		b.WriteString("\n")
	} else {
		b.WriteString("### Previous Sessions\nNone — this is session 001.\n\n")
	}

	// Completed prerequisites.
	completedDeps := getCompletedPrereqs(plan, task, allProgress)
	if len(completedDeps) > 0 {
		b.WriteString("### Completed Prerequisites\n")
		for _, dep := range completedDeps {
			b.WriteString(fmt.Sprintf("\n**Task %s (%s):**\n", dep.task.FullID(), dep.task.Title))
			if dep.summary != "" {
				b.WriteString(dep.summary + "\n")
			}
		}
		b.WriteString("\n")
	}

	// Progress reporting instructions.
	relProgress, _ := filepath.Rel(rootDir, progressPath)
	if relProgress == "" {
		relProgress = progressPath
	}
	b.WriteString("## Reporting Progress\n\n")
	b.WriteString("Use `etch progress` commands to report your work. These update both the plan file and your session progress file.\n\n")
	b.WriteString(fmt.Sprintf("Your progress file: `%s`\n\n", relProgress))
	b.WriteString("### Workflow\n\n")
	b.WriteString(fmt.Sprintf("1. **Start** (already done if launched via `etch run`):\n"))
	b.WriteString(fmt.Sprintf("   ```bash\n   etch progress start -p %s -t %s\n   ```\n\n", plan.Slug, task.FullID()))
	b.WriteString(fmt.Sprintf("2. **Log updates** as you make changes:\n"))
	b.WriteString(fmt.Sprintf("   ```bash\n   etch progress update -p %s -t %s -m \"description of what you changed\"\n   ```\n\n", plan.Slug, task.FullID()))
	b.WriteString(fmt.Sprintf("3. **Check off criteria** as you complete them:\n"))
	b.WriteString(fmt.Sprintf("   ```bash\n   etch progress criteria -p %s -t %s --check \"criterion text or substring\"\n   ```\n\n", plan.Slug, task.FullID()))
	b.WriteString(fmt.Sprintf("4. **When finished**, mark the task done:\n"))
	b.WriteString(fmt.Sprintf("   ```bash\n   etch progress done -p %s -t %s\n   ```\n\n", plan.Slug, task.FullID()))
	b.WriteString("If you get blocked or the task fails:\n")
	b.WriteString(fmt.Sprintf("```bash\netch progress block -p %s -t %s --reason \"why it's blocked\"\netch progress fail -p %s -t %s --reason \"why it failed\"\n```\n\n", plan.Slug, task.FullID(), plan.Slug, task.FullID()))
	b.WriteString("### Rules\n")
	b.WriteString("- Stay within the files listed in scope. Ask before modifying others.\n")
	b.WriteString("- Do NOT modify the plan file directly — use `etch progress` commands instead.\n")
	b.WriteString("- Log updates frequently so future sessions have context.\n")

	return b.String()
}

func formatTaskAnnotation(t, currentTask *models.Task, status models.Status, allProgress map[string][]models.SessionProgress) string {
	if t.FullID() == currentTask.FullID() {
		return "(in_progress — this is your task)"
	}
	switch status {
	case models.StatusCompleted:
		return "(completed)"
	case models.StatusInProgress:
		return "(in_progress)"
	case models.StatusBlocked:
		return "(blocked)"
	case models.StatusFailed:
		return "(failed)"
	case models.StatusPending:
		if len(t.DependsOn) > 0 {
			depIDs := make([]string, len(t.DependsOn))
			for i, dep := range t.DependsOn {
				depIDs[i] = extractTaskID(dep)
			}
			return fmt.Sprintf("(pending, depends on %s)", strings.Join(depIDs, ", "))
		}
		return "(pending)"
	default:
		return fmt.Sprintf("(%s)", status)
	}
}

func condenseOverview(overview string) string {
	if overview == "" {
		return ""
	}
	// Take first 3 sentences or the whole thing if shorter.
	sentences := splitSentences(overview)
	if len(sentences) > 3 {
		sentences = sentences[:3]
	}
	return strings.Join(sentences, " ")
}

func splitSentences(text string) []string {
	// Simple sentence splitter.
	var sentences []string
	current := strings.Builder{}
	for _, r := range text {
		current.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			s := strings.TrimSpace(current.String())
			if s != "" {
				sentences = append(sentences, s)
			}
			current.Reset()
		}
	}
	// Add remaining text if any.
	remaining := strings.TrimSpace(current.String())
	if remaining != "" {
		sentences = append(sentences, remaining)
	}
	return sentences
}

type completedDep struct {
	task    *models.Task
	summary string
}

func getCompletedPrereqs(plan *models.Plan, task *models.Task, allProgress map[string][]models.SessionProgress) []completedDep {
	var deps []completedDep
	for _, dep := range task.DependsOn {
		depID := extractTaskID(dep)
		depTask := plan.TaskByID(depID)
		if depTask == nil {
			continue
		}
		status := effectiveStatus(depTask, allProgress)
		if status != models.StatusCompleted {
			continue
		}
		summary := summarizeTask(depTask, allProgress)
		deps = append(deps, completedDep{task: depTask, summary: summary})
	}
	return deps
}

func summarizeTask(task *models.Task, allProgress map[string][]models.SessionProgress) string {
	sessions := allProgress[task.FullID()]
	if len(sessions) == 0 {
		return ""
	}
	// Use the latest session for summary.
	latest := sessions[len(sessions)-1]
	var parts []string
	if len(latest.ChangesMade) > 0 {
		parts = append(parts, strings.Join(latest.ChangesMade, ", "))
	}
	if latest.Decisions != "" {
		parts = append(parts, latest.Decisions)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ". ")
}
