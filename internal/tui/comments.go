package tui

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/gsigler/etch/internal/serializer"
)

var planCommentRe = regexp.MustCompile(`^>\s*💬\s*(.+)$`)
var planCommentContRe = regexp.MustCompile(`^>\s*(.+)$`)

// AddComment inserts a `> 💬 <text>` comment into the plan file, appended
// after the task heading section identified by taskID. Multi-line comments
// have continuation lines prefixed with `> `.
func AddComment(path string, taskID string, comment string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading plan file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	taskPatterns := serializer.TaskIDPatterns(taskID)

	// Find the task, then find the insertion point: just before the next
	// heading or at end of task section.
	inTask := false
	insertIdx := -1

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

		if inTask {
			// Next heading means end of task section.
			if strings.HasPrefix(trimmed, "### ") || strings.HasPrefix(trimmed, "## ") {
				insertIdx = i
				break
			}
			insertIdx = i + 1
		}
	}

	if !inTask {
		return fmt.Errorf("task %s not found in %s", taskID, path)
	}

	// Build the comment lines.
	commentLines := buildCommentLines(comment)

	// Insert a blank line + comment lines at insertIdx.
	var insertion []string
	insertion = append(insertion, "")
	insertion = append(insertion, commentLines...)

	result := make([]string, 0, len(lines)+len(insertion))
	result = append(result, lines[:insertIdx]...)
	result = append(result, insertion...)
	result = append(result, lines[insertIdx:]...)

	return os.WriteFile(path, []byte(strings.Join(result, "\n")), 0644)
}

// DeleteComment removes a comment from the plan file. It matches by the
// comment text content within the specified task section.
func DeleteComment(path string, taskID string, commentText string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading plan file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	taskPatterns := serializer.TaskIDPatterns(taskID)

	inTask := false
	deleteStart := -1
	deleteEnd := -1

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

		// Look for comment start.
		if m := planCommentRe.FindStringSubmatch(trimmed); m != nil {
			// Collect the full comment text.
			fullComment := strings.TrimSpace(m[1])
			endIdx := i + 1
			for endIdx < len(lines) {
				nextTrimmed := strings.TrimSpace(lines[endIdx])
				if cm := planCommentContRe.FindStringSubmatch(nextTrimmed); cm != nil {
					if planCommentRe.MatchString(nextTrimmed) {
						break // new comment, stop
					}
					fullComment += "\n" + strings.TrimSpace(cm[1])
					endIdx++
				} else {
					break
				}
			}

			if fullComment == commentText {
				deleteStart = i
				deleteEnd = endIdx
				// Also remove a leading blank line if present.
				if deleteStart > 0 && strings.TrimSpace(lines[deleteStart-1]) == "" {
					deleteStart--
				}
				break
			}
		}
	}

	if deleteStart < 0 {
		return fmt.Errorf("comment not found in task %s", taskID)
	}

	result := make([]string, 0, len(lines)-(deleteEnd-deleteStart))
	result = append(result, lines[:deleteStart]...)
	result = append(result, lines[deleteEnd:]...)

	return os.WriteFile(path, []byte(strings.Join(result, "\n")), 0644)
}

// buildCommentLines formats a comment string as markdown blockquote lines.
func buildCommentLines(comment string) []string {
	parts := strings.Split(comment, "\n")
	var result []string
	for i, part := range parts {
		if i == 0 {
			result = append(result, "> 💬 "+part)
		} else {
			result = append(result, "> "+part)
		}
	}
	return result
}
