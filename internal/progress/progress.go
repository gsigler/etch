package progress

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gsigler/etch/internal/models"
)

const progressDir = ".etch/progress"

// WriteSession creates a new progress file for the given plan and task.
// It auto-increments the session number by globbing existing files, and uses
// atomic file creation (O_CREATE|O_EXCL) to prevent race conditions.
func WriteSession(rootDir string, plan *models.Plan, task *models.Task) (string, error) {
	dir := filepath.Join(rootDir, progressDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating progress dir: %w", err)
	}

	nextNum := nextSessionNumber(dir, plan.Slug, task.FullID())

	for attempts := 0; attempts < 100; attempts++ {
		filename := formatFilename(plan.Slug, task.FullID(), nextNum)
		path := filepath.Join(dir, filename)

		content := renderTemplate(plan, task, nextNum)

		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			if os.IsExist(err) {
				nextNum++
				continue
			}
			return "", fmt.Errorf("creating progress file: %w", err)
		}
		if _, err := f.WriteString(content); err != nil {
			f.Close()
			return "", fmt.Errorf("writing progress file: %w", err)
		}
		if err := f.Close(); err != nil {
			return "", fmt.Errorf("closing progress file: %w", err)
		}
		return path, nil
	}
	return "", fmt.Errorf("failed to create progress file after 100 attempts")
}

// nextSessionNumber determines the next session number by globbing existing files.
func nextSessionNumber(dir, planSlug, taskID string) int {
	pattern := filepath.Join(dir, fmt.Sprintf("%s--task-%s--*.md", planSlug, taskID))
	matches, _ := filepath.Glob(pattern)
	if len(matches) == 0 {
		return 1
	}

	max := 0
	for _, m := range matches {
		base := filepath.Base(m)
		// Extract session number from end: <plan>--task-<id>--<NNN>.md
		ext := strings.TrimSuffix(base, ".md")
		parts := strings.Split(ext, "--")
		if len(parts) < 3 {
			continue
		}
		numStr := parts[len(parts)-1]
		var n int
		if _, err := fmt.Sscanf(numStr, "%d", &n); err == nil && n > max {
			max = n
		}
	}
	return max + 1
}

func formatFilename(planSlug, taskID string, session int) string {
	return fmt.Sprintf("%s--task-%s--%03d.md", planSlug, taskID, session)
}

func renderTemplate(plan *models.Plan, task *models.Task, session int) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Session: Task %s – %s\n", task.FullID(), task.Title))
	b.WriteString(fmt.Sprintf("**Plan:** %s\n", plan.Slug))
	b.WriteString(fmt.Sprintf("**Task:** %s\n", task.FullID()))
	b.WriteString(fmt.Sprintf("**Session:** %03d\n", session))
	b.WriteString(fmt.Sprintf("**Started:** %s\n", time.Now().Format("2006-01-02 15:04")))
	b.WriteString("**Status:** pending\n")
	b.WriteString("\n## Changes Made\n<!-- List files created or modified -->\n")
	b.WriteString("\n## Acceptance Criteria Updates\n")
	for _, c := range task.Criteria {
		check := " "
		if c.IsMet {
			check = "x"
		}
		b.WriteString(fmt.Sprintf("- [%s] %s\n", check, c.Description))
	}
	b.WriteString("\n## Decisions & Notes\n<!-- Design decisions, important context for future sessions -->\n")
	b.WriteString("\n## Blockers\n<!-- Anything blocking progress -->\n")
	b.WriteString("\n## Next\n<!-- What still needs to happen -->\n")
	return b.String()
}

// ReadAll reads all progress files for a plan and returns them grouped by task ID,
// sorted by session number within each group.
func ReadAll(rootDir, planSlug string) (map[string][]models.SessionProgress, error) {
	dir := filepath.Join(rootDir, progressDir)
	pattern := filepath.Join(dir, fmt.Sprintf("%s--*.md", planSlug))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("globbing progress files: %w", err)
	}

	result := make(map[string][]models.SessionProgress)
	for _, path := range matches {
		sp, err := parseProgressFile(path, planSlug)
		if err != nil {
			log.Printf("warning: skipping progress file %s: %v", filepath.Base(path), err)
			continue
		}
		result[sp.TaskID] = append(result[sp.TaskID], sp)
	}

	// Sort each task's sessions by session number.
	for taskID := range result {
		sort.Slice(result[taskID], func(i, j int) bool {
			return result[taskID][i].SessionNumber < result[taskID][j].SessionNumber
		})
	}

	return result, nil
}

func parseProgressFile(path, planSlug string) (models.SessionProgress, error) {
	f, err := os.Open(path)
	if err != nil {
		return models.SessionProgress{}, err
	}
	defer f.Close()

	sp := models.SessionProgress{PlanSlug: planSlug}

	scanner := bufio.NewScanner(f)
	var currentSection string

	var sectionText strings.Builder

	flushSection := func() {
		text := strings.TrimSpace(sectionText.String())
		// Strip HTML comments that are just placeholders.
		text = stripComments(text)
		switch currentSection {
		case "changes made":
			sp.ChangesMade = parseListItems(sectionText.String())
		case "acceptance criteria updates":
			sp.CriteriaUpdates = parseCriteria(sectionText.String())
		case "decisions & notes":
			sp.Decisions = text
		case "blockers":
			sp.Blockers = text
		case "next":
			sp.Next = text
		}
		sectionText.Reset()
	}

	for scanner.Scan() {
		line := scanner.Text()

		// Parse metadata lines.
		if strings.HasPrefix(line, "**Task:**") {
			sp.TaskID = strings.TrimSpace(strings.TrimPrefix(line, "**Task:**"))
			continue
		}
		if strings.HasPrefix(line, "**Session:**") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "**Session:**"))
			fmt.Sscanf(val, "%d", &sp.SessionNumber)
			continue
		}
		if strings.HasPrefix(line, "**Status:**") {
			sp.Status = strings.TrimSpace(strings.TrimPrefix(line, "**Status:**"))
			continue
		}
		if strings.HasPrefix(line, "**Started:**") {
			sp.Started = strings.TrimSpace(strings.TrimPrefix(line, "**Started:**"))
			continue
		}

		// Detect section headers.
		if strings.HasPrefix(line, "## ") {
			flushSection()
			currentSection = strings.ToLower(strings.TrimPrefix(line, "## "))
			continue
		}

		if currentSection != "" {
			sectionText.WriteString(line)
			sectionText.WriteString("\n")
		}
	}
	flushSection()

	if sp.TaskID == "" {
		return sp, fmt.Errorf("missing task ID")
	}

	return sp, scanner.Err()
}

func parseListItems(text string) []string {
	var items []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			item := strings.TrimPrefix(line, "- ")
			// Skip checkbox items and HTML comments.
			if strings.HasPrefix(item, "[") || strings.HasPrefix(item, "<!--") {
				continue
			}
			items = append(items, item)
		}
	}
	return items
}

func parseCriteria(text string) []models.Criterion {
	var criteria []models.Criterion
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- [x] ") {
			criteria = append(criteria, models.Criterion{
				Description: strings.TrimPrefix(line, "- [x] "),
				IsMet:       true,
			})
		} else if strings.HasPrefix(line, "- [ ] ") {
			criteria = append(criteria, models.Criterion{
				Description: strings.TrimPrefix(line, "- [ ] "),
				IsMet:       false,
			})
		}
	}
	return criteria
}

func stripComments(text string) string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<!--") && strings.HasSuffix(trimmed, "-->") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
