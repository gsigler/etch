package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/gsigler/etch/internal/models"
)

// Styles using lipgloss adaptive colors for terminal compatibility.
var (
	// Top and bottom bars.
	barStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	// Feature heading.
	featureStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	// Task heading by status.
	taskCompleted  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("34"))
	taskInProgress = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))
	taskPending    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245"))
	taskFailed     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	taskBlocked    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))

	// Criteria.
	criterionMet   = lipgloss.NewStyle().Foreground(lipgloss.Color("34"))
	criterionUnmet = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	// Comments.
	commentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("178")).
			Padding(0, 1)

	// HR rule.
	hrStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Bold text.
	boldStyle = lipgloss.NewStyle().Bold(true)

	// Plan title.
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))

	// Overview text.
	overviewStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))

	// Description text.
	descStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))

	// Metadata labels.
	metaLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	metaValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	// Search highlight.
	searchHighlight = lipgloss.NewStyle().
			Background(lipgloss.Color("220")).
			Foreground(lipgloss.Color("0"))

	// Comment prompt.
	commentPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("178")).
				Bold(true)

	// Delete confirmation prompt.
	deletePromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)

)

// lineEntry tracks which feature/task a rendered line belongs to.
type lineEntry struct {
	text         string
	featureIndex int // -1 if not in a feature
	taskIndex    int // -1 if not in a task
}

// renderPlan builds all the styled lines for the plan.
func renderPlan(plan *models.Plan, width int) []lineEntry {
	var lines []lineEntry
	fi, ti := -1, -1

	addLine := func(s string) {
		lines = append(lines, lineEntry{text: s, featureIndex: fi, taskIndex: ti})
	}
	addBlank := func() { addLine("") }

	// Plan title.
	addLine(titleStyle.Render("# " + plan.Title))
	addBlank()

	// Plan overview.
	if plan.Overview != "" {
		for _, line := range wrapText(plan.Overview, width-2) {
			addLine(overviewStyle.Render(line))
		}
		addBlank()
	}

	for i := range plan.Features {
		feat := &plan.Features[i]
		fi = i
		ti = -1

		// Feature heading.
		counts := featureCounts(feat)
		heading := fmt.Sprintf("## Feature %d: %s  %s", feat.Number, feat.Title, counts)
		addLine(featureStyle.Render(heading))

		if feat.Overview != "" {
			for _, line := range wrapText(feat.Overview, width-2) {
				addLine(overviewStyle.Render(line))
			}
		}
		addBlank()

		for j := range feat.Tasks {
			task := &feat.Tasks[j]
			ti = j

			// Task heading.
			icon := task.Status.Icon()
			style := taskStyleFor(task.Status)
			heading := fmt.Sprintf("### Task %s: %s  %s %s",
				task.FullID(), task.Title, icon, string(task.Status))
			addLine(style.Render(heading))

			// Metadata.
			if task.Complexity != "" {
				addLine(metaLabelStyle.Render("  Complexity: ") + metaValueStyle.Render(string(task.Complexity)))
			}
			if len(task.Files) > 0 {
				addLine(metaLabelStyle.Render("  Files: ") + metaValueStyle.Render(strings.Join(task.Files, ", ")))
			}
			if len(task.DependsOn) > 0 {
				addLine(metaLabelStyle.Render("  Depends on: ") + metaValueStyle.Render(strings.Join(task.DependsOn, ", ")))
			}

			// Description.
			if task.Description != "" {
				addBlank()
				for _, line := range wrapText(task.Description, width-4) {
					addLine(descStyle.Render("  " + line))
				}
			}

			// Criteria.
			if len(task.Criteria) > 0 {
				addBlank()
				addLine(boldStyle.Render("  Acceptance Criteria:"))
				for _, c := range task.Criteria {
					if c.IsMet {
						addLine(criterionMet.Render(fmt.Sprintf("  [x] %s", c.Description)))
					} else {
						addLine(criterionUnmet.Render(fmt.Sprintf("  [ ] %s", c.Description)))
					}
				}
			}

			// Comments.
			for _, comment := range task.Comments {
				addBlank()
				for _, line := range wrapText(comment, width-6) {
					addLine(commentStyle.Render("> " + line))
				}
			}

			addBlank()

			// HR between tasks.
			if j < len(feat.Tasks)-1 {
				addLine(hrStyle.Render(strings.Repeat("─", min(width-2, 60))))
				addBlank()
			}
		}

		// HR between features.
		if i < len(plan.Features)-1 {
			addLine(hrStyle.Render(strings.Repeat("━", min(width-2, 60))))
			addBlank()
		}
	}

	return lines
}

func taskStyleFor(s models.Status) lipgloss.Style {
	switch s {
	case models.StatusCompleted:
		return taskCompleted
	case models.StatusInProgress:
		return taskInProgress
	case models.StatusFailed:
		return taskFailed
	case models.StatusBlocked:
		return taskBlocked
	default:
		return taskPending
	}
}

func featureCounts(feat *models.Feature) string {
	done := 0
	for _, t := range feat.Tasks {
		if t.Status == models.StatusCompleted {
			done++
		}
	}
	return fmt.Sprintf("(%d/%d)", done, len(feat.Tasks))
}

// wrapText does simple word wrapping at the given width.
func wrapText(text string, width int) []string {
	if width <= 0 {
		width = 80
	}
	var result []string
	for _, paragraph := range strings.Split(text, "\n") {
		if paragraph == "" {
			result = append(result, "")
			continue
		}
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			result = append(result, "")
			continue
		}
		line := words[0]
		for _, word := range words[1:] {
			if len(line)+1+len(word) > width {
				result = append(result, line)
				line = word
			} else {
				line += " " + word
			}
		}
		result = append(result, line)
	}
	return result
}

// renderTopBar builds the top bar with plan title, task counts, and key hints.
func renderTopBar(plan *models.Plan, width int) string {
	done, total := 0, 0
	for _, f := range plan.Features {
		for _, t := range f.Tasks {
			total++
			if t.Status == models.StatusCompleted {
				done++
			}
		}
	}
	left := fmt.Sprintf(" %s  [%d/%d tasks]", plan.Title, done, total)
	right := hintStyle.Render("j/k:scroll  n/p:task  c:comment  a:apply  /:search  q:quit ")
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return barStyle.Width(width).Render(left + strings.Repeat(" ", gap) + right)
}

// renderBottomBar builds the bottom bar with position and mode indicator.
func renderBottomBar(m *Model, width int) string {
	pos := ""
	if m.curFeature >= 0 && m.curFeature < len(m.plan.Features) {
		feat := &m.plan.Features[m.curFeature]
		pos = fmt.Sprintf(" Feature %d: %s", feat.Number, feat.Title)
		if m.curTask >= 0 && m.curTask < len(feat.Tasks) {
			task := &feat.Tasks[m.curTask]
			pos += fmt.Sprintf("  >  Task %s: %s", task.FullID(), task.Title)
		}
	}

	mode := ""
	switch m.mode {
	case modeSearch:
		mode = fmt.Sprintf("  SEARCH: %s", m.searchQuery)
		if len(m.searchMatches) > 0 {
			mode += fmt.Sprintf("  [%d/%d]", m.searchIdx+1, len(m.searchMatches))
		}
	case modeComment:
		mode = "  COMMENT"
	case modeConfirm:
		mode = "  CONFIRM DELETE"
	case modeApplyConfirm:
		mode = "  APPLY REFINEMENT"
	}

	if m.statusMsg != "" {
		mode = "  " + m.statusMsg
	}

	content := pos + mode
	return barStyle.Width(width).Render(content)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
