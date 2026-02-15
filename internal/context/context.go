package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// DiscoverPlans finds all plan files in the project root.
func DiscoverPlans(rootDir string) ([]*models.Plan, error) {
	dir := filepath.Join(rootDir, plansDir)
	pattern := filepath.Join(dir, "*.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("globbing plans: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no plan files found in %s", dir)
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
		return nil, fmt.Errorf("no valid plan files found in %s", dir)
	}
	return plans, nil
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
			return nil, nil, fmt.Errorf("no plan found with slug %q", planSlug)
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

	return nil, nil, fmt.Errorf("task %q not found", taskID)
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
		return nil, nil, fmt.Errorf("no pending tasks with satisfied dependencies")
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
		return Result{}, fmt.Errorf("creating progress file: %w", err)
	}

	// Determine session number from filename.
	sessionNum := sessionNumberFromPath(progressPath)

	// Build context content.
	content := buildTemplate(plan, task, allProgress, sessionNum, progressPath, rootDir)

	// Write context file.
	ctxDir := filepath.Join(rootDir, contextDir)
	if err := os.MkdirAll(ctxDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("creating context dir: %w", err)
	}

	ctxFilename := fmt.Sprintf("%s--task-%s--%03d.md", plan.Slug, task.FullID(), sessionNum)
	ctxPath := filepath.Join(ctxDir, ctxFilename)
	if err := os.WriteFile(ctxPath, []byte(content), 0o644); err != nil {
		return Result{}, fmt.Errorf("writing context file: %w", err)
	}

	tokenEstimate := len(content) * 10 / 35 // chars / 3.5

	return Result{
		ContextPath:   ctxPath,
		ProgressPath:  progressPath,
		SessionNum:    sessionNum,
		TokenEstimate: tokenEstimate,
	}, nil
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

	// Session progress file instructions.
	relProgress, _ := filepath.Rel(rootDir, progressPath)
	if relProgress == "" {
		relProgress = progressPath
	}
	b.WriteString("## Session Progress File\n\n")
	b.WriteString(fmt.Sprintf("Update your progress file as you work:\n`%s`\n\n", relProgress))
	b.WriteString("This file has been created for you. Fill in each section:\n")
	b.WriteString("- **Changes Made:** files created or modified\n")
	b.WriteString("- **Acceptance Criteria Updates:** check off what you completed\n")
	b.WriteString("- **Decisions & Notes:** design decisions, important context\n")
	b.WriteString("- **Blockers:** anything blocking progress\n")
	b.WriteString("- **Next:** what still needs to happen\n")
	b.WriteString("- **Status:** update to completed, partial, failed, or blocked\n\n")
	b.WriteString("Rules:\n")
	b.WriteString("- Stay within the files listed in scope. Ask before modifying others.\n")
	b.WriteString("- Do NOT modify the plan file. Only update your progress file.\n")
	b.WriteString("- Keep notes concise but useful — future sessions depend on them.\n")

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
