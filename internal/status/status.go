package status

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	etcherr "github.com/gsigler/etch/internal/errors"
	"github.com/gsigler/etch/internal/models"
	"github.com/gsigler/etch/internal/parser"
	"github.com/gsigler/etch/internal/progress"
	"github.com/gsigler/etch/internal/serializer"
)

// PlanStatus holds the reconciled status for a single plan.
type PlanStatus struct {
	Title          string          `json:"title"`
	Slug           string          `json:"slug"`
	FilePath       string          `json:"file_path"`
	Priority       int             `json:"priority"`
	Features       []FeatureStatus `json:"features"`
	CompletedTasks int             `json:"completed_tasks"`
	TotalTasks     int             `json:"total_tasks"`
}

// IsActive returns true if the plan has at least one in-progress, failed, or blocked task,
// or is partially completed (some but not all tasks done). Fully pending and fully completed
// plans are not considered active.
func (ps PlanStatus) IsActive() bool {
	for _, f := range ps.Features {
		for _, t := range f.Tasks {
			if t.Status == models.StatusInProgress || t.Status == models.StatusFailed || t.Status == models.StatusBlocked {
				return true
			}
		}
	}
	// Partially completed: some done but not all.
	return ps.CompletedTasks > 0 && ps.CompletedTasks < ps.TotalTasks
}

// Percentage returns the overall completion percentage for the plan.
func (ps PlanStatus) Percentage() int {
	if ps.TotalTasks == 0 {
		return 0
	}
	return ps.CompletedTasks * 100 / ps.TotalTasks
}

// FeatureStatus holds status summary for a feature.
type FeatureStatus struct {
	Number         int          `json:"number"`
	Title          string       `json:"title"`
	Tasks          []TaskStatus `json:"tasks"`
	CompletedTasks int          `json:"completed_tasks"`
	TotalTasks     int          `json:"total_tasks"`
}

// TaskStatus holds reconciled status for a single task.
type TaskStatus struct {
	ID            string             `json:"id"`
	Title         string             `json:"title"`
	Status        models.Status      `json:"status"`
	DependsOn     []string           `json:"depends_on,omitempty"`
	IsBlocked     bool               `json:"is_blocked,omitempty"`
	SessionCount  int                `json:"session_count"`
	LastOutcome   string             `json:"last_outcome,omitempty"`
	Criteria      []models.Criterion `json:"criteria,omitempty"`
	LastDecisions string             `json:"last_decisions,omitempty"`
	LastNext      string             `json:"last_next,omitempty"`
}

// Run reads all plans (or a specific one), reconciles progress, updates plan files, and returns status.
func Run(rootDir string, planFilter string) ([]PlanStatus, error) {
	plansDir := filepath.Join(rootDir, ".etch", "plans")

	entries, err := os.ReadDir(plansDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, etcherr.WrapIO("reading plans directory", err)
	}

	var results []PlanStatus

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		slug := strings.TrimSuffix(entry.Name(), ".md")
		if planFilter != "" && slug != planFilter {
			continue
		}

		planPath := filepath.Join(plansDir, entry.Name())
		plan, err := parser.ParseFile(planPath)
		if err != nil {
			return nil, etcherr.WrapParse(fmt.Sprintf("parsing plan %s", entry.Name()), err)
		}

		progressMap, err := progress.ReadAll(rootDir, plan.Slug)
		if err != nil {
			return nil, etcherr.WrapIO(fmt.Sprintf("reading progress for %s", plan.Slug), err)
		}

		ps, err := reconcile(plan, progressMap)
		if err != nil {
			return nil, etcherr.WrapIO(fmt.Sprintf("reconciling %s", plan.Slug), err)
		}

		results = append(results, ps)
	}

	return results, nil
}

// reconcile merges progress data into plan status and updates the plan file if needed.
func reconcile(plan *models.Plan, progressMap map[string][]models.SessionProgress) (PlanStatus, error) {
	changed := false

	ps := PlanStatus{
		Title:    plan.Title,
		Slug:     plan.Slug,
		FilePath: plan.FilePath,
		Priority: plan.Priority,
	}

	for i := range plan.Features {
		f := &plan.Features[i]
		fs := FeatureStatus{
			Number:     f.Number,
			Title:      f.Title,
			TotalTasks: len(f.Tasks),
		}

		for j := range f.Tasks {
			task := &f.Tasks[j]
			sessions := progressMap[task.FullID()]

			ts := TaskStatus{
				ID:           task.FullID(),
				Title:        task.Title,
				Status:       task.Status,
				DependsOn:    task.DependsOn,
				SessionCount: len(sessions),
				Criteria:     task.Criteria,
			}

			if len(sessions) > 0 {
				latest := sessions[len(sessions)-1]
				newStatus := mapProgressStatus(latest.Status)

				if newStatus != task.Status {
					ts.Status = newStatus
					if err := serializer.UpdateTaskStatus(plan.FilePath, task.FullID(), newStatus); err != nil {
						return ps, etcherr.WrapIO(fmt.Sprintf("updating task %s status", task.FullID()), err)
					}
					task.Status = newStatus
					changed = true
				}

				ts.LastOutcome = latest.Status

				// Merge criteria: if any session marks [x], plan gets [x].
				for _, sess := range sessions {
					for _, cu := range sess.CriteriaUpdates {
						if !cu.IsMet {
							continue
						}
						for k := range task.Criteria {
							if task.Criteria[k].Description == cu.Description && !task.Criteria[k].IsMet {
								task.Criteria[k].IsMet = true
								ts.Criteria = task.Criteria
								if err := serializer.UpdateCriterion(plan.FilePath, task.FullID(), cu.Description, true); err != nil {
									return ps, etcherr.WrapIO(fmt.Sprintf("updating criterion for task %s", task.FullID()), err)
								}
								changed = true
							}
						}
					}
				}

				ts.LastDecisions = latest.Decisions
				ts.LastNext = latest.Next
			}

			if ts.Status == models.StatusCompleted {
				fs.CompletedTasks++
			}
			fs.Tasks = append(fs.Tasks, ts)
		}

		ps.Features = append(ps.Features, fs)
	}

	// Aggregate plan-level totals from features.
	for _, f := range ps.Features {
		ps.CompletedTasks += f.CompletedTasks
		ps.TotalTasks += f.TotalTasks
	}

	// Resolve blocked status: a pending task is blocked if any dependency is not completed.
	resolveBlocked(&ps)

	_ = changed // tracking for potential future use
	return ps, nil
}

// depIDRegex extracts a task ID like "1.2" or "1.3b" from a dependency string like "Task 1.2".
var depIDRegex = regexp.MustCompile(`(\d+\.\d+[a-z]?)`)

// depBareIDRegex extracts a bare task number like "2" from a dependency string like "Task 2".
var depBareIDRegex = regexp.MustCompile(`(?:^|\D)(\d+[a-z]?)(?:\D|$)`)

// resolveBlocked marks pending tasks as blocked if any of their dependencies are not completed.
func resolveBlocked(ps *PlanStatus) {
	// Build a map of task ID -> status for quick lookup.
	statusMap := make(map[string]models.Status)
	for _, f := range ps.Features {
		for _, t := range f.Tasks {
			statusMap[t.ID] = t.Status
		}
	}

	// Detect single-feature plan: all tasks have feature number 1.
	singleFeature := len(ps.Features) == 1

	for i := range ps.Features {
		for j := range ps.Features[i].Tasks {
			t := &ps.Features[i].Tasks[j]
			if t.Status != models.StatusPending || len(t.DependsOn) == 0 {
				continue
			}
			for _, dep := range t.DependsOn {
				depID := extractDepID(dep)
				// For single-feature plans, deps may be bare numbers like "Task 2".
				if depID == "" && singleFeature {
					depID = extractBareDepID(dep)
				}
				if depID == "" {
					continue
				}
				if s, ok := statusMap[depID]; ok && s != models.StatusCompleted {
					t.IsBlocked = true
					break
				}
			}
		}
	}
}

// extractDepID pulls a task ID from a dependency string like "Task 1.2".
func extractDepID(dep string) string {
	m := depIDRegex.FindString(dep)
	return m
}

// extractBareDepID pulls a bare task number from a dependency string like "Task 2"
// and normalizes it to "1.N" format for single-feature plan lookup.
func extractBareDepID(dep string) string {
	m := depBareIDRegex.FindStringSubmatch(dep)
	if m == nil {
		return ""
	}
	return "1." + m[1]
}

// mapProgressStatus converts a progress file status string to a plan Status.
func mapProgressStatus(progressStatus string) models.Status {
	switch progressStatus {
	case "completed":
		return models.StatusCompleted
	case "partial":
		return models.StatusInProgress
	case "failed":
		return models.StatusFailed
	case "blocked":
		return models.StatusBlocked
	default:
		return models.StatusPending
	}
}

// FilterActive returns only plans that are active (have in-progress, failed,
// or blocked tasks, or are partially completed).
func FilterActive(plans []PlanStatus) []PlanStatus {
	var active []PlanStatus
	for _, p := range plans {
		if p.IsActive() {
			active = append(active, p)
		}
	}
	return active
}

// progressBar returns a 10-character progress bar string like [████░░░░░░] 45%.
func progressBar(pct int) string {
	filled := pct / 10
	if filled > 10 {
		filled = 10
	}
	empty := 10 - filled
	return fmt.Sprintf("[%s%s] %d%%",
		strings.Repeat("█", filled),
		strings.Repeat("░", empty),
		pct)
}

// FormatSummary renders all plan statuses as a summary view.
func FormatSummary(plans []PlanStatus) string {
	if len(plans) == 0 {
		return "No plans found."
	}

	var b strings.Builder
	for i, p := range plans {
		if i > 0 {
			b.WriteString("\n────────────────────────────────────────\n\n")
		}
		pct := p.Percentage()
		priorityTag := "[ ]"
		if p.Priority > 0 {
			priorityTag = fmt.Sprintf("[%d]", p.Priority)
		}
		b.WriteString(fmt.Sprintf("📋 %s %s  %s\n", priorityTag, p.Title, progressBar(pct)))
		b.WriteString(fmt.Sprintf("  slug: %s\n", p.Slug))

		for _, f := range p.Features {
			icon := featureIcon(f)
			b.WriteString(fmt.Sprintf("   %s Feature %d: %s [%d/%d tasks]\n",
				icon, f.Number, f.Title, f.CompletedTasks, f.TotalTasks))

			for _, t := range f.Tasks {
				icon := taskIcon(t)
				line := fmt.Sprintf("      %s %-6s %s", icon, t.ID, t.Title)
				if t.SessionCount > 0 && t.Status != models.StatusCompleted {
					line += fmt.Sprintf(" (%d sessions, last: %s)", t.SessionCount, t.LastOutcome)
				}
				b.WriteString(line + "\n")
			}
		}
	}
	return b.String()
}

// FormatDetailed renders a single plan with criteria and session notes.
func FormatDetailed(ps PlanStatus) string {
	var b strings.Builder
	pct := ps.Percentage()
	priorityTag := "[ ]"
	if ps.Priority > 0 {
		priorityTag = fmt.Sprintf("[%d]", ps.Priority)
	}
	b.WriteString(fmt.Sprintf("📋 %s %s  %s\n", priorityTag, ps.Title, progressBar(pct)))
	b.WriteString(fmt.Sprintf("  slug: %s\n\n", ps.Slug))

	for _, f := range ps.Features {
		icon := featureIcon(f)
		b.WriteString(fmt.Sprintf("%s Feature %d: %s [%d/%d tasks]\n",
			icon, f.Number, f.Title, f.CompletedTasks, f.TotalTasks))

		for _, t := range f.Tasks {
			icon := taskIcon(t)
			b.WriteString(fmt.Sprintf("\n  %s %-6s %s", icon, t.ID, t.Title))
			if t.SessionCount > 0 {
				b.WriteString(fmt.Sprintf(" (%d sessions, last: %s)", t.SessionCount, t.LastOutcome))
			}
			b.WriteString("\n")

			if t.IsBlocked && len(t.DependsOn) > 0 {
				b.WriteString(fmt.Sprintf("    Waiting on: %s\n", strings.Join(t.DependsOn, ", ")))
			}

			if len(t.Criteria) > 0 {
				for _, c := range t.Criteria {
					check := "[ ]"
					if c.IsMet {
						check = "[x]"
					}
					b.WriteString(fmt.Sprintf("    %s %s\n", check, c.Description))
				}
			}

			if t.LastDecisions != "" {
				b.WriteString(fmt.Sprintf("    Notes: %s\n", t.LastDecisions))
			}
			if t.LastNext != "" {
				b.WriteString(fmt.Sprintf("    Next: %s\n", t.LastNext))
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

// FormatJSON renders plan statuses as JSON.
func FormatJSON(plans []PlanStatus) (string, error) {
	data, err := json.MarshalIndent(plans, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// featureIcon returns a status icon based on task completion.
func featureIcon(f FeatureStatus) string {
	if f.CompletedTasks == f.TotalTasks && f.TotalTasks > 0 {
		return models.StatusCompleted.Icon()
	}
	if f.CompletedTasks > 0 {
		return models.StatusInProgress.Icon()
	}
	// Check if any task is in progress, failed, or blocked.
	for _, t := range f.Tasks {
		if t.Status == models.StatusInProgress || t.Status == models.StatusFailed || t.Status == models.StatusBlocked {
			return t.Status.Icon()
		}
	}
	return models.StatusPending.Icon()
}

// taskIcon returns the display icon for a task, showing ⊘ for blocked pending tasks.
func taskIcon(t TaskStatus) string {
	if t.IsBlocked {
		return models.StatusBlocked.Icon()
	}
	return t.Status.Icon()
}

// SortPlanStatuses sorts plans by priority first (ascending, unset/0 last), then alphabetically by title.
func SortPlanStatuses(plans []PlanStatus) {
	sort.Slice(plans, func(i, j int) bool {
		pi, pj := plans[i].Priority, plans[j].Priority
		if pi == pj {
			return plans[i].Title < plans[j].Title
		}
		if pi == 0 {
			return false
		}
		if pj == 0 {
			return true
		}
		return pi < pj
	})
}
