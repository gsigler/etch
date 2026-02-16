package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsigler/etch/internal/models"
)

// RefineFunc is the function signature for plan refinement.
// It takes the current plan content and review comments, and returns the
// refined plan content.
type RefineFunc func(planContent string, comments []string) (string, error)

// Option configures the TUI model.
type Option func(*Model)

// WithRefineFunc sets the refinement function used by the apply flow.
func WithRefineFunc(fn RefineFunc) Option {
	return func(m *Model) { m.refineFn = fn }
}

// refinementResultMsg carries the result of an async refinement call.
type refinementResultMsg struct {
	newContent string
	err        error
}

// collectComments gathers all review comments from the plan, prefixed with
// their task ID for context.
func collectComments(plan *models.Plan) []string {
	var comments []string
	for _, feat := range plan.Features {
		for _, task := range feat.Tasks {
			for _, c := range task.Comments {
				comments = append(comments, fmt.Sprintf("[Task %s] %s", task.FullID(), c))
			}
		}
	}
	return comments
}

// backupPlan creates a timestamped backup of the plan file.
func backupPlan(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading plan for backup: %w", err)
	}
	backupPath := path + ".bak." + time.Now().Format("20060102-150405")
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return "", fmt.Errorf("writing backup: %w", err)
	}
	return backupPath, nil
}

// restoreBackup copies the backup file back to the original path.
func restoreBackup(backupPath, origPath string) error {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("reading backup: %w", err)
	}
	if err := os.WriteFile(origPath, data, 0644); err != nil {
		return fmt.Errorf("restoring backup: %w", err)
	}
	return nil
}

// startRefinement fires an async command that calls the refine function.
func startRefinement(refineFn RefineFunc, planContent string, comments []string) tea.Cmd {
	return func() tea.Msg {
		newContent, err := refineFn(planContent, comments)
		return refinementResultMsg{newContent: newContent, err: err}
	}
}

// newSpinner creates a configured spinner for the loading state.
func newSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return s
}

// --- Mode handlers ---

// enterApplyMode starts the refinement flow.
func (m Model) enterApplyMode() (tea.Model, tea.Cmd) {
	if m.refineFn == nil {
		m.statusMsg = "Refinement not configured"
		return m, nil
	}

	comments := collectComments(m.plan)
	if len(comments) == 0 {
		m.statusMsg = "No comments to send"
		return m, nil
	}

	m.mode = modeApplyConfirm
	m.applyCommentCount = len(comments)
	return m, nil
}

// updateApplyConfirm handles keypresses in the apply confirmation mode.
func (m Model) updateApplyConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// Read current plan content for diffing later.
		data, err := os.ReadFile(m.planPath)
		if err != nil {
			m.statusMsg = "Error reading plan: " + err.Error()
			m.mode = modeNormal
			return m, nil
		}
		m.oldPlanContent = string(data)

		// Create backup.
		bp, err := backupPlan(m.planPath)
		if err != nil {
			m.statusMsg = "Backup error: " + err.Error()
			m.mode = modeNormal
			return m, nil
		}
		m.backupPath = bp

		// Switch to loading mode with spinner.
		m.mode = modeLoading
		m.spinner = newSpinner()

		comments := collectComments(m.plan)
		return m, tea.Batch(
			m.spinner.Tick,
			startRefinement(m.refineFn, m.oldPlanContent, comments),
		)
	default:
		m.mode = modeNormal
		m.statusMsg = "Refinement cancelled"
		return m, nil
	}
}

// handleRefinementResult processes the async refinement response.
func (m Model) handleRefinementResult(msg refinementResultMsg) (tea.Model, tea.Cmd) {
	if m.mode != modeLoading {
		return m, nil
	}
	if msg.err != nil {
		m.mode = modeNormal
		m.statusMsg = "Refinement error: " + msg.err.Error()
		if m.backupPath != "" {
			os.Remove(m.backupPath)
			m.backupPath = ""
		}
		return m, nil
	}
	m.diffLines = computeDiff(m.oldPlanContent, msg.newContent)
	m.newPlanContent = msg.newContent
	m.diffOffset = 0
	m.mode = modeDiff
	return m, nil
}

// updateLoadingKey handles keypresses while the spinner is showing.
func (m Model) updateLoadingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.mode = modeNormal
		m.statusMsg = "Refinement cancelled"
		if m.backupPath != "" {
			os.Remove(m.backupPath)
			m.backupPath = ""
		}
		return m, nil
	}
	return m, nil
}

// updateDiff handles keypresses in the diff view mode.
func (m Model) updateDiff(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	viewH := m.diffViewHeight()

	switch msg.String() {
	case "y", "Y":
		// Accept: write new content to plan file.
		if err := os.WriteFile(m.planPath, []byte(m.newPlanContent), 0644); err != nil {
			m.statusMsg = "Error writing plan: " + err.Error()
			m.mode = modeNormal
			return m, nil
		}
		m.reloadPlan()
		m.mode = modeNormal
		added, removed := diffStats(m.diffLines)
		m.statusMsg = fmt.Sprintf("Plan updated (+%d/-%d lines)", added, removed)
		m.cleanupRefinement()
		return m, nil

	case "n", "N":
		// Reject: restore backup.
		if m.backupPath != "" {
			if err := restoreBackup(m.backupPath, m.planPath); err != nil {
				m.statusMsg = "Restore error: " + err.Error()
			} else {
				m.statusMsg = "Changes rejected, plan restored"
			}
		} else {
			m.statusMsg = "Changes rejected"
		}
		m.reloadPlan()
		m.mode = modeNormal
		m.cleanupRefinement()
		return m, nil

	case "j", "down":
		m.diffOffset++
	case "k", "up":
		m.diffOffset--
	case "d":
		m.diffOffset += viewH / 2
	case "u":
		m.diffOffset -= viewH / 2
	case "G":
		m.diffOffset = len(m.diffLines) - viewH
	case "g":
		m.diffOffset = 0
	}

	m.clampDiffOffset()
	return m, nil
}

// cleanupRefinement resets refinement state and removes the backup file.
func (m *Model) cleanupRefinement() {
	if m.backupPath != "" {
		os.Remove(m.backupPath)
	}
	m.backupPath = ""
	m.oldPlanContent = ""
	m.newPlanContent = ""
	m.diffLines = nil
	m.diffOffset = 0
}

// diffViewHeight returns the number of content lines available in diff mode.
func (m *Model) diffViewHeight() int {
	h := m.height - 2 // top bar + bottom bar
	if h < 1 {
		h = 1
	}
	return h
}

// clampDiffOffset keeps diffOffset within valid bounds.
func (m *Model) clampDiffOffset() {
	maxOff := len(m.diffLines) - m.diffViewHeight()
	if maxOff < 0 {
		maxOff = 0
	}
	if m.diffOffset > maxOff {
		m.diffOffset = maxOff
	}
	if m.diffOffset < 0 {
		m.diffOffset = 0
	}
}

// --- View helpers ---

// Apply prompt style.
var applyPromptStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("39")).
	Bold(true)

// viewLoading renders the loading/spinner screen.
func (m Model) viewLoading() string {
	var b strings.Builder

	b.WriteString(barStyle.Width(m.width).Render(" Refining plan..."))
	b.WriteByte('\n')

	viewH := m.height - 2
	topPad := viewH / 2
	for i := 0; i < topPad; i++ {
		b.WriteByte('\n')
	}

	spinnerText := m.spinner.View() + " Sending comments for AI refinement..."
	pad := (m.width - lipgloss.Width(spinnerText)) / 2
	if pad < 0 {
		pad = 0
	}
	b.WriteString(strings.Repeat(" ", pad) + spinnerText + "\n")

	for i := topPad + 1; i < viewH; i++ {
		b.WriteByte('\n')
	}

	b.WriteString(barStyle.Width(m.width).Render(" ESC to cancel"))

	return b.String()
}

// viewDiff renders the diff view with accept/reject controls.
func (m Model) viewDiff() string {
	var b strings.Builder

	// Top bar with diff stats.
	added, removed := diffStats(m.diffLines)
	left := fmt.Sprintf(" Refinement Diff  +%d -%d", added, removed)
	right := hintStyle.Render("j/k:scroll  y:accept  n:reject ")
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	b.WriteString(barStyle.Width(m.width).Render(left + strings.Repeat(" ", gap) + right))
	b.WriteByte('\n')

	// Diff lines.
	viewH := m.diffViewHeight()
	end := m.diffOffset + viewH
	if end > len(m.diffLines) {
		end = len(m.diffLines)
	}

	for i := m.diffOffset; i < end; i++ {
		b.WriteString(renderDiffLine(m.diffLines[i]))
		b.WriteByte('\n')
	}

	for i := end - m.diffOffset; i < viewH; i++ {
		b.WriteByte('\n')
	}

	// Bottom bar.
	pos := fmt.Sprintf(" Line %d/%d", m.diffOffset+1, len(m.diffLines))
	b.WriteString(barStyle.Width(m.width).Render(pos))

	return b.String()
}
