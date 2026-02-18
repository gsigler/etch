package serializer

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	etcherr "github.com/gsigler/etch/internal/errors"
	"github.com/gsigler/etch/internal/models"
)

// Serialize converts a Plan struct into its markdown representation.
// For single-feature plans (one feature whose title matches the plan title),
// it omits the "## Feature N:" heading and uses "### Task N:" format.
func Serialize(plan *models.Plan) string {
	var b strings.Builder

	b.WriteString("# Plan: ")
	b.WriteString(plan.Title)
	b.WriteString("\n")

	if plan.Priority > 0 {
		b.WriteString(fmt.Sprintf("**Priority:** %d\n", plan.Priority))
	}

	if plan.Overview != "" {
		b.WriteString("\n## Overview\n\n")
		b.WriteString(plan.Overview)
		b.WriteString("\n")
	}

	singleFeature := len(plan.Features) == 1

	for i, f := range plan.Features {
		if !singleFeature {
			b.WriteString("\n---\n")
			b.WriteString("\n## Feature ")
			b.WriteString(fmt.Sprintf("%d", f.Number))
			b.WriteString(": ")
			b.WriteString(f.Title)
			b.WriteString("\n")

			if f.Overview != "" {
				b.WriteString("\n### Overview\n")
				b.WriteString(f.Overview)
				b.WriteString("\n")
			}
		}

		for _, task := range f.Tasks {
			b.WriteString("\n")
			b.WriteString("### Task ")
			if singleFeature {
				b.WriteString(fmt.Sprintf("%d%s", task.TaskNumber, task.Suffix))
			} else {
				b.WriteString(task.FullID())
			}
			b.WriteString(": ")
			b.WriteString(task.Title)
			if task.Status != "" {
				b.WriteString(" [")
				b.WriteString(string(task.Status))
				b.WriteString("]")
			}
			b.WriteString("\n")

			if task.Complexity != "" {
				b.WriteString("**Complexity:** ")
				b.WriteString(string(task.Complexity))
				b.WriteString("\n")
			}
			if len(task.Files) > 0 {
				b.WriteString("**Files:** ")
				b.WriteString(strings.Join(task.Files, ", "))
				b.WriteString("\n")
			}
			if len(task.DependsOn) > 0 {
				b.WriteString("**Depends on:** ")
				b.WriteString(strings.Join(task.DependsOn, ", "))
				b.WriteString("\n")
			}

			if task.Description != "" {
				b.WriteString("\n")
				b.WriteString(task.Description)
				b.WriteString("\n")
			}

			for _, comment := range task.Comments {
				b.WriteString("\n")
				lines := strings.Split(comment, "\n")
				for k, line := range lines {
					if k == 0 {
						b.WriteString("> 💬 ")
					} else {
						b.WriteString("> ")
					}
					b.WriteString(line)
					b.WriteString("\n")
				}
			}

			if len(task.Criteria) > 0 {
				b.WriteString("\n**Acceptance Criteria:**\n")
				for _, c := range task.Criteria {
					if c.IsMet {
						b.WriteString("- [x] ")
					} else {
						b.WriteString("- [ ] ")
					}
					b.WriteString(c.Description)
					b.WriteString("\n")
				}
			}
		}

		_ = i // used for loop indexing
	}

	return b.String()
}

var (
	taskLineRe      = regexp.MustCompile(`^(### Task \d+(?:\.\d+[a-z]?)?:\s*.+?)\s*\[(\w+)\]\s*$`)
	criterionLineRe = regexp.MustCompile(`^(- \[)([ x])(\] .+)$`)
)

// UpdateTaskStatus reads a plan file, changes the status tag on the specified
// task, and writes the file back. It preserves all other content exactly.
func UpdateTaskStatus(path string, taskID string, newStatus models.Status) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading plan file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	found := false

	taskPatterns := TaskIDPatterns(taskID)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		matched := false
		for _, pat := range taskPatterns {
			if strings.HasPrefix(trimmed, pat) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		if m := taskLineRe.FindStringSubmatchIndex(line); m != nil {
			lines[i] = line[:m[4]] + string(newStatus) + line[m[5]:]
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("task %s not found in %s", taskID, path)
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

// UpdateCriterion reads a plan file, finds the specified task's acceptance
// criteria by text match, and flips the checkbox. It preserves all other
// content exactly.
func UpdateCriterion(path string, taskID string, criterionText string, met bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading plan file: %w", err)
	}

	lines := strings.Split(string(data), "\n")

	taskPatterns := TaskIDPatterns(taskID)
	inTask := false
	found := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inTask {
			for _, pat := range taskPatterns {
				if strings.HasPrefix(trimmed, pat) {
					inTask = true
					break
				}
			}
			if inTask {
				continue
			}
		}

		if inTask && (strings.HasPrefix(trimmed, "### ") || strings.HasPrefix(trimmed, "## ")) {
			break
		}

		if !inTask {
			continue
		}

		if m := criterionLineRe.FindStringSubmatch(line); m != nil {
			desc := strings.TrimSpace(m[3][2:])
			if desc == criterionText {
				check := " "
				if met {
					check = "x"
				}
				lines[i] = m[1] + check + m[3]
				found = true
				break
			}
		}
	}

	if !found {
		return fmt.Errorf("criterion %q not found in task %s", criterionText, taskID)
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

var priorityLineRe = regexp.MustCompile(`^\*\*Priority:\*\*\s*\d+\s*$`)

// UpdatePlanPriority reads a plan file, updates/inserts/removes the priority
// metadata line, and writes the file back. It preserves all other content exactly.
func UpdatePlanPriority(path string, newPriority int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return etcherr.WrapIO("reading plan file", err).WithHint("Check that the plan file exists at " + path)
	}

	lines := strings.Split(string(data), "\n")

	// Find the # Plan: heading line.
	planIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "# Plan:") {
			planIdx = i
			break
		}
	}
	if planIdx < 0 {
		return etcherr.IO("no # Plan: heading found in " + path).WithHint("Ensure the file is a valid etch plan")
	}

	// Find existing priority line between plan heading and next ## heading.
	priorityIdx := -1
	nextSectionIdx := len(lines)
	for i := planIdx + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			nextSectionIdx = i
			break
		}
		if priorityLineRe.MatchString(lines[i]) {
			priorityIdx = i
		}
	}

	if priorityIdx >= 0 && newPriority > 0 {
		// Replace existing line.
		lines[priorityIdx] = fmt.Sprintf("**Priority:** %d", newPriority)
	} else if priorityIdx < 0 && newPriority > 0 {
		// Insert after plan heading.
		newLine := fmt.Sprintf("**Priority:** %d", newPriority)
		insertAt := planIdx + 1
		lines = append(lines[:insertAt], append([]string{newLine}, lines[insertAt:]...)...)
		_ = nextSectionIdx // not needed after insert
	} else if priorityIdx >= 0 && newPriority == 0 {
		// Remove the priority line.
		lines = append(lines[:priorityIdx], lines[priorityIdx+1:]...)
	}
	// If priorityIdx < 0 && newPriority == 0, nothing to do.

	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return etcherr.WrapIO("writing plan file", err).WithHint("Check file permissions for " + path)
	}
	return nil
}

// TaskIDPatterns returns the heading prefixes to match for a given task ID.
// For IDs like "1.2" (feature 1), it returns both "### Task 1.2:" (multi-feature)
// and "### Task 2:" (single-feature), since single-feature plans omit the feature number.
func TaskIDPatterns(taskID string) []string {
	patterns := []string{"### Task " + taskID + ":"}
	if strings.HasPrefix(taskID, "1.") {
		short := strings.TrimPrefix(taskID, "1.")
		patterns = append(patterns, "### Task "+short+":")
	}
	return patterns
}
