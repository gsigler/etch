package status

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gsigler/etch/internal/models"
	"github.com/gsigler/etch/internal/parser"
	"github.com/gsigler/etch/internal/progress"
	"github.com/gsigler/etch/internal/serializer"
)

// PlanStatus holds the reconciled status for a single plan.
type PlanStatus struct {
	Title    string          `json:"title"`
	Slug     string          `json:"slug"`
	FilePath string          `json:"file_path"`
	Features []FeatureStatus `json:"features"`
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
		return nil, fmt.Errorf("reading plans directory: %w", err)
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
			return nil, fmt.Errorf("parsing plan %s: %w", entry.Name(), err)
		}

		progressMap, err := progress.ReadAll(rootDir, plan.Slug)
		if err != nil {
			return nil, fmt.Errorf("reading progress for %s: %w", plan.Slug, err)
		}

		ps, err := reconcile(plan, progressMap)
		if err != nil {
			return nil, fmt.Errorf("reconciling %s: %w", plan.Slug, err)
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
				SessionCount: len(sessions),
				Criteria:     task.Criteria,
			}

			if len(sessions) > 0 {
				latest := sessions[len(sessions)-1]
				newStatus := mapProgressStatus(latest.Status)

				if newStatus != task.Status {
					ts.Status = newStatus
					if err := serializer.UpdateTaskStatus(plan.FilePath, task.FullID(), newStatus); err != nil {
						return ps, fmt.Errorf("updating task %s status: %w", task.FullID(), err)
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
									return ps, fmt.Errorf("updating criterion for task %s: %w", task.FullID(), err)
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

	_ = changed // tracking for potential future use
	return ps, nil
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

// FormatSummary renders all plan statuses as a summary view.
func FormatSummary(plans []PlanStatus) string {
	if len(plans) == 0 {
		return "No plans found."
	}

	var b strings.Builder
	for i, p := range plans {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("📋 %s\n", p.Title))
		for _, f := range p.Features {
			icon := featureIcon(f)
			b.WriteString(fmt.Sprintf("   %s Feature %d: %s [%d/%d tasks]\n",
				icon, f.Number, f.Title, f.CompletedTasks, f.TotalTasks))

			for _, t := range f.Tasks {
				line := fmt.Sprintf("     %s %s: %s", t.Status.Icon(), t.ID, t.Title)
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
	b.WriteString(fmt.Sprintf("📋 %s\n\n", ps.Title))

	for _, f := range ps.Features {
		icon := featureIcon(f)
		b.WriteString(fmt.Sprintf("%s Feature %d: %s [%d/%d tasks]\n",
			icon, f.Number, f.Title, f.CompletedTasks, f.TotalTasks))

		for _, t := range f.Tasks {
			b.WriteString(fmt.Sprintf("\n  %s %s: %s", t.Status.Icon(), t.ID, t.Title))
			if t.SessionCount > 0 {
				b.WriteString(fmt.Sprintf(" (%d sessions, last: %s)", t.SessionCount, t.LastOutcome))
			}
			b.WriteString("\n")

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

// SortPlanStatuses sorts plans alphabetically by title for consistent output.
func SortPlanStatuses(plans []PlanStatus) {
	sort.Slice(plans, func(i, j int) bool {
		return plans[i].Title < plans[j].Title
	})
}
