package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// diffLineKind categorizes a line in a unified diff.
type diffLineKind int

const (
	diffContext diffLineKind = iota
	diffAdded
	diffRemoved
)

// diffLine is a single line in the computed diff.
type diffLine struct {
	kind diffLineKind
	text string
}

// Diff styles.
var (
	diffAddedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("34"))

	diffRemovedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196"))

	diffContextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))
)

// computeDiff produces a line-by-line diff between old and new text using LCS.
func computeDiff(oldText, newText string) []diffLine {
	oldLines := splitTrimmed(oldText)
	newLines := splitTrimmed(newText)

	m, n := len(oldLines), len(newLines)

	// Build LCS table.
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if oldLines[i-1] == newLines[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Backtrack to build diff.
	var result []diffLine
	i, j := m, n
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			result = append(result, diffLine{kind: diffContext, text: oldLines[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			result = append(result, diffLine{kind: diffAdded, text: newLines[j-1]})
			j--
		} else {
			result = append(result, diffLine{kind: diffRemoved, text: oldLines[i-1]})
			i--
		}
	}

	// Reverse since we built it backwards.
	for l, r := 0, len(result)-1; l < r; l, r = l+1, r-1 {
		result[l], result[r] = result[r], result[l]
	}

	return result
}

// splitTrimmed splits text into lines, removing a trailing empty line from
// a final newline (so "a\nb\n" gives ["a","b"] not ["a","b",""]).
func splitTrimmed(text string) []string {
	lines := strings.Split(text, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// renderDiffLine styles a single diff line with its prefix.
func renderDiffLine(dl diffLine) string {
	switch dl.kind {
	case diffAdded:
		return diffAddedStyle.Render("+ " + dl.text)
	case diffRemoved:
		return diffRemovedStyle.Render("- " + dl.text)
	default:
		return diffContextStyle.Render("  " + dl.text)
	}
}

// diffStats returns counts of added and removed lines.
func diffStats(lines []diffLine) (added, removed int) {
	for _, l := range lines {
		switch l.kind {
		case diffAdded:
			added++
		case diffRemoved:
			removed++
		}
	}
	return
}
