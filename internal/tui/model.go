package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsigler/etch/internal/models"
	"github.com/gsigler/etch/internal/parser"
)

// tuiMode tracks which input mode we're in.
type tuiMode int

const (
	modeNormal       tuiMode = iota
	modeSearch                // typing a search query
	modeComment               // typing an inline comment
	modeConfirm               // confirming comment deletion
	modeApplyConfirm          // "Send N comments for refinement?" confirmation
	modeLoading               // spinner while waiting for API
	modeDiff                  // diff view with accept/reject
)

// Model is the Bubbletea model for the plan review TUI.
type Model struct {
	plan     *models.Plan
	planPath string      // path to plan file on disk
	lines    []lineEntry // rendered plan lines
	width    int
	height   int

	// Scroll state.
	offset int // first visible line index

	// Current position tracking (derived from offset).
	curFeature int
	curTask    int

	// Mode.
	mode tuiMode

	// Search mode.
	searchQuery   string
	searchMatches []int // line indices matching the query
	searchIdx     int   // current match index

	// Comment input mode.
	commentInput commentInput

	// Delete confirmation mode.
	deleteCommentIdx  int    // index into current task's Comments slice
	deleteCommentText string // text of the comment to delete

	// gg detection: was previous key 'g'?
	lastKeyG bool

	// Status message shown temporarily.
	statusMsg string

	// Refinement state.
	refineFn         RefineFunc
	spinner          spinner.Model
	diffLines        []diffLine
	diffOffset       int
	oldPlanContent   string
	newPlanContent   string
	backupPath       string
	applyCommentCount int
}

// New creates a new TUI model for the given plan.
func New(plan *models.Plan, planPath string, opts ...Option) Model {
	m := Model{
		plan:         plan,
		planPath:     planPath,
		curFeature:   -1,
		curTask:      -1,
		commentInput: newCommentInput(),
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.lines = renderPlan(m.plan, m.width)
		m.clampOffset()
		m.updatePosition()
		return m, nil

	case editorResultMsg:
		return m.handleEditorResult(msg)

	case refinementResultMsg:
		return m.handleRefinementResult(msg)

	case spinner.TickMsg:
		if m.mode == modeLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		// Clear status message on any keypress.
		m.statusMsg = ""

		switch m.mode {
		case modeSearch:
			return m.updateSearch(msg)
		case modeComment:
			return m.updateComment(msg)
		case modeConfirm:
			return m.updateConfirm(msg)
		case modeApplyConfirm:
			return m.updateApplyConfirm(msg)
		case modeLoading:
			return m.updateLoadingKey(msg)
		case modeDiff:
			return m.updateDiff(msg)
		default:
			return m.updateNormal(msg)
		}
	}
	return m, nil
}

func (m Model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	viewH := m.viewHeight()

	// Handle 'g' for gg (go to top).
	if msg.String() == "g" {
		if m.lastKeyG {
			m.offset = 0
			m.lastKeyG = false
			m.updatePosition()
			return m, nil
		}
		m.lastKeyG = true
		return m, nil
	}
	m.lastKeyG = false

	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Down):
		m.offset++

	case key.Matches(msg, keys.Up):
		m.offset--

	case key.Matches(msg, keys.HalfDown):
		m.offset += viewH / 2

	case key.Matches(msg, keys.HalfUp):
		m.offset -= viewH / 2

	case msg.String() == "G":
		m.offset = len(m.lines) - viewH

	case key.Matches(msg, keys.Top):
		m.offset = 0

	case key.Matches(msg, keys.Bottom):
		m.offset = len(m.lines) - viewH

	case key.Matches(msg, keys.NextTask):
		m.jumpToNextTask()

	case key.Matches(msg, keys.PrevTask):
		m.jumpToPrevTask()

	case key.Matches(msg, keys.NextFeat):
		m.jumpToNextFeature()

	case msg.String() == "F":
		m.jumpToPrevFeature()

	case key.Matches(msg, keys.Search):
		m.mode = modeSearch
		m.searchQuery = ""
		m.searchMatches = nil
		m.searchIdx = 0
		return m, nil

	case msg.String() == "c":
		return m.enterCommentMode()

	case msg.String() == "C":
		return m.enterEditorComment()

	case msg.String() == "x":
		return m.enterDeleteMode()

	case msg.String() == "a":
		return m.enterApplyMode()
	}

	m.clampOffset()
	m.updatePosition()
	return m, nil
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape):
		m.mode = modeNormal
		m.searchMatches = nil
		return m, nil

	case key.Matches(msg, keys.Confirm):
		// Execute search, stay in results mode until Escape.
		m.executeSearch()
		if len(m.searchMatches) > 0 {
			m.offset = m.searchMatches[0]
			m.searchIdx = 0
		}
		m.mode = modeNormal
		m.clampOffset()
		m.updatePosition()
		return m, nil

	case msg.Type == tea.KeyBackspace:
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
		return m, nil

	default:
		if msg.Type == tea.KeyRunes {
			m.searchQuery += string(msg.Runes)
		}
		return m, nil
	}
}

func (m *Model) executeSearch() {
	m.searchMatches = nil
	if m.searchQuery == "" {
		return
	}
	q := strings.ToLower(m.searchQuery)
	for i, line := range m.lines {
		if strings.Contains(strings.ToLower(line.text), q) {
			m.searchMatches = append(m.searchMatches, i)
		}
	}
}

func (m *Model) jumpToNextSearchMatch() {
	if len(m.searchMatches) == 0 {
		return
	}
	// Find next match after current offset.
	for i, idx := range m.searchMatches {
		if idx > m.offset {
			m.searchIdx = i
			m.offset = idx
			m.clampOffset()
			m.updatePosition()
			return
		}
	}
	// Wrap around.
	m.searchIdx = 0
	m.offset = m.searchMatches[0]
	m.clampOffset()
	m.updatePosition()
}

// updateComment handles keypresses in comment input mode.
func (m Model) updateComment(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape):
		m.mode = modeNormal
		m.commentInput.Blur()
		m.commentInput.Reset()
		return m, nil

	case key.Matches(msg, keys.Confirm):
		text := strings.TrimSpace(m.commentInput.Value())
		if text != "" {
			m.saveComment(text)
		}
		m.mode = modeNormal
		m.commentInput.Blur()
		m.commentInput.Reset()
		return m, nil

	default:
		ti, cmd := m.commentInput.Update(msg)
		m.commentInput.ti = ti
		return m, cmd
	}
}

// updateConfirm handles keypresses in delete confirmation mode.
func (m Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.doDeleteComment()
		m.mode = modeNormal
		return m, nil
	default:
		// Any other key cancels.
		m.mode = modeNormal
		m.statusMsg = "Delete cancelled"
		return m, nil
	}
}

// enterCommentMode starts inline comment entry for the current task.
func (m Model) enterCommentMode() (tea.Model, tea.Cmd) {
	if m.curFeature < 0 || m.curTask < 0 {
		m.statusMsg = "Navigate to a task first"
		return m, nil
	}
	m.mode = modeComment
	m.commentInput.Reset()
	cmd := m.commentInput.Focus()
	return m, cmd
}

// enterEditorComment opens $EDITOR for multi-line comment entry.
func (m Model) enterEditorComment() (tea.Model, tea.Cmd) {
	if m.curFeature < 0 || m.curTask < 0 {
		m.statusMsg = "Navigate to a task first"
		return m, nil
	}
	return m, openEditor()
}

// handleEditorResult processes the result from $EDITOR.
func (m Model) handleEditorResult(msg editorResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.statusMsg = "Editor error: " + msg.err.Error()
		return m, nil
	}
	if msg.content != "" {
		m.saveComment(msg.content)
	}
	return m, nil
}

// enterDeleteMode checks if the current line is a comment and asks for confirmation.
func (m Model) enterDeleteMode() (tea.Model, tea.Cmd) {
	if m.curFeature < 0 || m.curTask < 0 {
		m.statusMsg = "Navigate to a task first"
		return m, nil
	}
	if m.offset >= len(m.lines) {
		return m, nil
	}

	// Check if the current line is a comment line.
	lineText := m.lines[m.offset].text
	if !isCommentLine(lineText) {
		m.statusMsg = "Not a comment line"
		return m, nil
	}

	// Find which comment this corresponds to.
	feat := &m.plan.Features[m.curFeature]
	if m.curTask >= len(feat.Tasks) {
		return m, nil
	}
	task := &feat.Tasks[m.curTask]

	commentIdx := findCommentForLine(lineText, task.Comments)
	if commentIdx < 0 {
		m.statusMsg = "Comment not found"
		return m, nil
	}

	m.mode = modeConfirm
	m.deleteCommentIdx = commentIdx
	m.deleteCommentText = task.Comments[commentIdx]
	return m, nil
}

// saveComment adds a comment to the current task and reloads.
func (m *Model) saveComment(text string) {
	if m.curFeature < 0 || m.curTask < 0 {
		return
	}
	feat := &m.plan.Features[m.curFeature]
	if m.curTask >= len(feat.Tasks) {
		return
	}
	task := &feat.Tasks[m.curTask]
	taskID := task.FullID()

	if err := AddComment(m.planPath, taskID, text); err != nil {
		m.statusMsg = "Save error: " + err.Error()
		return
	}

	m.reloadPlan()
	m.statusMsg = "Comment added"
}

// doDeleteComment removes the confirmed comment and reloads.
func (m *Model) doDeleteComment() {
	if m.curFeature < 0 || m.curTask < 0 {
		return
	}
	feat := &m.plan.Features[m.curFeature]
	if m.curTask >= len(feat.Tasks) {
		return
	}
	task := &feat.Tasks[m.curTask]
	taskID := task.FullID()

	if err := DeleteComment(m.planPath, taskID, m.deleteCommentText); err != nil {
		m.statusMsg = "Delete error: " + err.Error()
		return
	}

	m.reloadPlan()
	m.statusMsg = "Comment deleted"
}

// reloadPlan re-reads the plan file from disk and refreshes the rendered lines.
func (m *Model) reloadPlan() {
	plan, err := parser.ParseFile(m.planPath)
	if err != nil {
		m.statusMsg = "Reload error: " + err.Error()
		return
	}
	m.plan = plan
	m.lines = renderPlan(m.plan, m.width)
	m.clampOffset()
	m.updatePosition()
}

// isCommentLine checks whether a rendered line looks like a comment.
func isCommentLine(text string) bool {
	// The rendered text goes through lipgloss styling, so we need to check
	// the underlying content. Comment lines are rendered as "> 💬 ..." or "> ..."
	// with commentStyle. We check for the ">" prefix in the unstyled text.
	stripped := stripAnsi(text)
	trimmed := strings.TrimSpace(stripped)
	return strings.HasPrefix(trimmed, "> ")
}

// findCommentForLine finds which comment index matches the given rendered line.
func findCommentForLine(lineText string, comments []string) int {
	stripped := stripAnsi(lineText)
	trimmed := strings.TrimSpace(stripped)
	// Remove "> " or "> 💬 " prefix.
	content := strings.TrimPrefix(trimmed, "> 💬 ")
	if content == trimmed {
		content = strings.TrimPrefix(trimmed, "> ")
	}
	content = strings.TrimSpace(content)

	for i, c := range comments {
		// Match against first line of multi-line comment.
		firstLine := strings.Split(c, "\n")[0]
		if strings.TrimSpace(firstLine) == content {
			return i
		}
	}
	return -1
}

// stripAnsi removes ANSI escape sequences from a string.
func stripAnsi(s string) string {
	var result strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

// viewHeight returns the number of lines available for content (minus top and bottom bars).
func (m *Model) viewHeight() int {
	h := m.height - 2 // top bar + bottom bar
	if m.mode == modeComment || m.mode == modeConfirm || m.mode == modeApplyConfirm {
		h-- // prompt line takes one row
	}
	if h < 1 {
		h = 1
	}
	return h
}

func (m *Model) clampOffset() {
	maxOff := len(m.lines) - m.viewHeight()
	if maxOff < 0 {
		maxOff = 0
	}
	if m.offset > maxOff {
		m.offset = maxOff
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

// updatePosition sets curFeature and curTask based on the visible line at offset.
func (m *Model) updatePosition() {
	if m.offset < len(m.lines) {
		entry := m.lines[m.offset]
		m.curFeature = entry.featureIndex
		m.curTask = entry.taskIndex
	}
}

// jumpToNextTask scrolls to the next task heading.
func (m *Model) jumpToNextTask() {
	// If we have search matches, navigate those instead when 'n' is pressed after search.
	if len(m.searchMatches) > 0 {
		m.jumpToNextSearchMatch()
		return
	}
	for i := m.offset + 1; i < len(m.lines); i++ {
		entry := m.lines[i]
		if entry.taskIndex >= 0 && (i == 0 || m.lines[i-1].taskIndex != entry.taskIndex || m.lines[i-1].featureIndex != entry.featureIndex) {
			m.offset = i
			m.clampOffset()
			m.updatePosition()
			return
		}
	}
}

// jumpToPrevTask scrolls to the previous task heading.
func (m *Model) jumpToPrevTask() {
	curFeat := m.curFeature
	curTask := m.curTask
	for i := m.offset - 1; i >= 0; i-- {
		entry := m.lines[i]
		if entry.taskIndex >= 0 && (entry.taskIndex != curTask || entry.featureIndex != curFeat) {
			// Found a different task. Jump to its first line.
			for i > 0 && m.lines[i-1].taskIndex == entry.taskIndex && m.lines[i-1].featureIndex == entry.featureIndex {
				i--
			}
			m.offset = i
			m.clampOffset()
			m.updatePosition()
			return
		}
	}
}

// jumpToNextFeature scrolls to the next feature heading.
func (m *Model) jumpToNextFeature() {
	curFeat := m.curFeature
	for i := m.offset + 1; i < len(m.lines); i++ {
		entry := m.lines[i]
		if entry.featureIndex >= 0 && entry.featureIndex != curFeat && entry.taskIndex == -1 {
			m.offset = i
			m.clampOffset()
			m.updatePosition()
			return
		}
	}
}

// jumpToPrevFeature scrolls to the previous feature heading.
func (m *Model) jumpToPrevFeature() {
	curFeat := m.curFeature
	for i := m.offset - 1; i >= 0; i-- {
		entry := m.lines[i]
		if entry.featureIndex >= 0 && entry.featureIndex != curFeat && entry.taskIndex == -1 {
			m.offset = i
			m.clampOffset()
			m.updatePosition()
			return
		}
	}
}

// View renders the full TUI.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Loading and diff modes use entirely different layouts.
	switch m.mode {
	case modeLoading:
		return m.viewLoading()
	case modeDiff:
		return m.viewDiff()
	}

	var b strings.Builder

	// Top bar.
	b.WriteString(renderTopBar(m.plan, m.width))
	b.WriteByte('\n')

	// Content area.
	viewH := m.viewHeight()
	end := m.offset + viewH
	if end > len(m.lines) {
		end = len(m.lines)
	}

	for i := m.offset; i < end; i++ {
		line := m.lines[i].text
		// Apply search highlighting if there are matches.
		if len(m.searchMatches) > 0 && m.searchQuery != "" {
			line = highlightSearch(line, m.searchQuery)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}

	// Pad remaining lines if content is shorter than viewport.
	for i := end - m.offset; i < viewH; i++ {
		b.WriteByte('\n')
	}

	// Mode-specific prompts above the bottom bar.
	switch m.mode {
	case modeComment:
		b.WriteString(commentPromptStyle.Render(" 💬 Comment: ") + m.commentInput.View())
		b.WriteByte('\n')
	case modeConfirm:
		preview := m.deleteCommentText
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		b.WriteString(deletePromptStyle.Render(" Delete comment: \"" + preview + "\"? (y/N) "))
		b.WriteByte('\n')
	case modeApplyConfirm:
		b.WriteString(applyPromptStyle.Render(
			fmt.Sprintf(" Send %d comment(s) for refinement? (y/N) ", m.applyCommentCount)))
		b.WriteByte('\n')
	}

	// Bottom bar.
	b.WriteString(renderBottomBar(&m, m.width))

	return b.String()
}

// highlightSearch applies search highlighting to a line.
func highlightSearch(line, query string) string {
	lower := strings.ToLower(line)
	q := strings.ToLower(query)
	idx := strings.Index(lower, q)
	if idx < 0 {
		return line
	}
	// Highlight all occurrences.
	var result strings.Builder
	for idx >= 0 {
		result.WriteString(line[:idx])
		result.WriteString(searchHighlight.Render(line[idx : idx+len(query)]))
		line = line[idx+len(query):]
		lower = strings.ToLower(line)
		idx = strings.Index(lower, q)
	}
	result.WriteString(line)
	return result.String()
}
