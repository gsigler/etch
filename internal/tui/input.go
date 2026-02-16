package tui

import (
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// commentInput wraps a bubbles textinput for inline comment entry.
type commentInput struct {
	ti textinput.Model
}

func newCommentInput() commentInput {
	ti := textinput.New()
	ti.Placeholder = "Type comment..."
	ti.CharLimit = 500
	ti.Width = 60
	return commentInput{ti: ti}
}

func (c *commentInput) Focus() tea.Cmd {
	return c.ti.Focus()
}

func (c *commentInput) Blur() {
	c.ti.Blur()
}

func (c *commentInput) Value() string {
	return c.ti.Value()
}

func (c *commentInput) Reset() {
	c.ti.Reset()
}

func (c *commentInput) Update(msg tea.Msg) (textinput.Model, tea.Cmd) {
	return c.ti.Update(msg)
}

func (c *commentInput) View() string {
	return c.ti.View()
}

// editorResultMsg is sent when $EDITOR closes with a result.
type editorResultMsg struct {
	content string
	err     error
}

// openEditor launches $EDITOR with a temp file and returns the content on close.
func openEditor() tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	tmpFile, err := os.CreateTemp("", "etch-comment-*.md")
	if err != nil {
		return func() tea.Msg {
			return editorResultMsg{err: err}
		}
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	c := exec.Command(editor, tmpPath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			os.Remove(tmpPath)
			return editorResultMsg{err: err}
		}
		data, readErr := os.ReadFile(tmpPath)
		os.Remove(tmpPath)
		if readErr != nil {
			return editorResultMsg{err: readErr}
		}
		return editorResultMsg{content: strings.TrimSpace(string(data))}
	})
}
