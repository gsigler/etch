package errors

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Category classifies an error for display and handling.
type Category string

const (
	CatConfig  Category = "config"
	CatAPI     Category = "api"
	CatParse   Category = "parse"
	CatProject Category = "project"
	CatUsage   Category = "usage"
	CatIO      Category = "io"
)

// Error is the standard error type for etch. It carries a category,
// a user-facing message, an actionable hint, and optionally the
// underlying cause.
type Error struct {
	Category Category
	Message  string
	Hint     string
	Cause    error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// New creates an Error with the given category and message.
func New(cat Category, msg string) *Error {
	return &Error{Category: cat, Message: msg}
}

// Wrap creates an Error wrapping an existing error.
func Wrap(cat Category, msg string, cause error) *Error {
	return &Error{Category: cat, Message: msg, Cause: cause}
}

// WithHint returns a copy of the error with the given hint.
func (e *Error) WithHint(hint string) *Error {
	e.Hint = hint
	return e
}

// --- Convenience constructors ---

func Config(msg string) *Error  { return New(CatConfig, msg) }
func API(msg string) *Error     { return New(CatAPI, msg) }
func Parse(msg string) *Error   { return New(CatParse, msg) }
func Project(msg string) *Error { return New(CatProject, msg) }
func Usage(msg string) *Error   { return New(CatUsage, msg) }
func IO(msg string) *Error      { return New(CatIO, msg) }

func WrapConfig(msg string, cause error) *Error  { return Wrap(CatConfig, msg, cause) }
func WrapAPI(msg string, cause error) *Error      { return Wrap(CatAPI, msg, cause) }
func WrapParse(msg string, cause error) *Error    { return Wrap(CatParse, msg, cause) }
func WrapProject(msg string, cause error) *Error  { return Wrap(CatProject, msg, cause) }
func WrapIO(msg string, cause error) *Error       { return Wrap(CatIO, msg, cause) }

// --- Styled output ---

var (
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)  // red bold
	hintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // dim
	catStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))            // yellow
)

// Format renders an error for terminal display with colors.
// If verbose is true, the full causal chain is printed.
func Format(err error, verbose bool) string {
	var b strings.Builder

	var etchErr *Error
	if errors.As(err, &etchErr) {
		// Category label.
		b.WriteString(errorStyle.Render("error"))
		b.WriteString(catStyle.Render(fmt.Sprintf("[%s]", etchErr.Category)))
		b.WriteString(": ")
		b.WriteString(etchErr.Message)
		b.WriteString("\n")

		if etchErr.Hint != "" {
			b.WriteString(hintStyle.Render("  hint: "+etchErr.Hint) + "\n")
		}

		if verbose && etchErr.Cause != nil {
			b.WriteString(hintStyle.Render(fmt.Sprintf("  cause: %v", etchErr.Cause)) + "\n")
		}
	} else {
		// Fallback for non-etch errors.
		b.WriteString(errorStyle.Render("error") + ": " + err.Error() + "\n")
	}

	return b.String()
}

// Render writes a formatted error to stderr.
func Render(err error, verbose bool) {
	fmt.Fprint(os.Stderr, Format(err, verbose))
}
