package serializer

import (
	"fmt"
	"os"
	"regexp"
	"strings"

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

	taskPattern := "### Task " + taskID + ":"

	for i, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), taskPattern) {
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

	taskPattern := "### Task " + taskID + ":"
	inTask := false
	found := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, taskPattern) {
			inTask = true
			continue
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
