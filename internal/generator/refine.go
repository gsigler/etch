package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	etcherr "github.com/gsigler/etch/internal/errors"
	"github.com/gsigler/etch/internal/models"
)

// ExtractComments collects all review comments from a plan, grouped by task.
// Returns a formatted string with task context, and the total comment count.
func ExtractComments(plan *models.Plan) (string, int) {
	var b strings.Builder
	count := 0

	for _, f := range plan.Features {
		for _, t := range f.Tasks {
			if len(t.Comments) == 0 {
				continue
			}
			b.WriteString(fmt.Sprintf("### Task %s: %s\n", t.FullID(), t.Title))
			for _, c := range t.Comments {
				b.WriteString(fmt.Sprintf("> 💬 %s\n", c))
				count++
			}
			b.WriteString("\n")
		}
	}

	return b.String(), count
}

// BackupPlan copies the plan file to .etch/backups/<name>-<timestamp>.md.
// Returns the backup file path.
func BackupPlan(planPath, rootDir string) (string, error) {
	backupsDir := filepath.Join(rootDir, ".etch", "backups")
	if err := os.MkdirAll(backupsDir, 0o755); err != nil {
		return "", etcherr.WrapIO("creating backups directory", err)
	}

	data, err := os.ReadFile(planPath)
	if err != nil {
		return "", etcherr.WrapIO("reading plan for backup", err)
	}

	base := strings.TrimSuffix(filepath.Base(planPath), ".md")
	ts := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("%s-%s.md", base, ts)
	backupPath := filepath.Join(backupsDir, backupName)

	if err := os.WriteFile(backupPath, data, 0o644); err != nil {
		return "", etcherr.WrapIO("writing backup", err)
	}

	return backupPath, nil
}

// ApplyRefinement writes the refined plan to disk, overwriting the original.
func ApplyRefinement(planPath, newMarkdown string) error {
	if err := os.WriteFile(planPath, []byte(newMarkdown), 0o644); err != nil {
		return etcherr.WrapIO("writing refined plan", err)
	}
	return nil
}

// GenerateDiff produces a unified-style colored diff between old and new text.
// Red (prefixed with -) for removed lines, green (prefixed with +) for added lines.
func GenerateDiff(old, new string) string {
	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(new, "\n")

	// Simple line-by-line diff using longest common subsequence.
	lcs := computeLCS(oldLines, newLines)

	var b strings.Builder
	oi, ni, li := 0, 0, 0

	for li < len(lcs) {
		// Lines removed from old (not in LCS).
		for oi < len(oldLines) && oldLines[oi] != lcs[li] {
			b.WriteString("\033[31m- " + oldLines[oi] + "\033[0m\n")
			oi++
		}
		// Lines added in new (not in LCS).
		for ni < len(newLines) && newLines[ni] != lcs[li] {
			b.WriteString("\033[32m+ " + newLines[ni] + "\033[0m\n")
			ni++
		}
		// Common line — skip it in diff output.
		oi++
		ni++
		li++
	}
	// Remaining lines after LCS is exhausted.
	for oi < len(oldLines) {
		b.WriteString("\033[31m- " + oldLines[oi] + "\033[0m\n")
		oi++
	}
	for ni < len(newLines) {
		b.WriteString("\033[32m+ " + newLines[ni] + "\033[0m\n")
		ni++
	}

	return b.String()
}

// computeLCS returns the longest common subsequence of two string slices.
func computeLCS(a, b []string) []string {
	m, n := len(a), len(b)
	// Build DP table.
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Backtrack to find LCS.
	lcs := make([]string, dp[m][n])
	i, j := m, n
	k := len(lcs) - 1
	for i > 0 && j > 0 {
		if a[i-1] == b[j-1] {
			lcs[k] = a[i-1]
			k--
			i--
			j--
		} else if dp[i-1][j] >= dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	return lcs
}
