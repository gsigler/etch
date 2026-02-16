package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gsigler/etch/internal/api"
	etcherr "github.com/gsigler/etch/internal/errors"
	"github.com/gsigler/etch/internal/models"
	"github.com/gsigler/etch/internal/parser"
)

const (
	maxTreeLines   = 2000
	maxTreeDepth   = 3
	configLineLimit = 100
)

// excludeDirs are directories to skip when building the file tree.
var excludeDirs = map[string]bool{
	".git":        true,
	"node_modules": true,
	"target":       true,
	"__pycache__":  true,
	".etch":        true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
}

// configFiles are project config files to include (first 100 lines each).
var configFiles = []string{
	"package.json",
	"Cargo.toml",
	"go.mod",
	"pyproject.toml",
	"tsconfig.json",
	"Makefile",
	"docker-compose.yml",
}

// APIClient is the interface for sending messages to the AI API.
// This allows mocking in tests.
type APIClient interface {
	SendStream(system, userMessage string, cb api.StreamCallback) (string, error)
}

// Result holds the output of plan generation.
type Result struct {
	Plan     *models.Plan
	FilePath string
	Slug     string
	Markdown string
}

// Summary returns a human-readable summary of the generated plan.
func (r Result) Summary() string {
	featureCount := len(r.Plan.Features)
	taskCount := 0
	for _, f := range r.Plan.Features {
		taskCount += len(f.Tasks)
	}
	return fmt.Sprintf("Created plan: %s\n  %d feature(s), %d task(s)\n  Written to: %s",
		r.Plan.Title, featureCount, taskCount, r.FilePath)
}

// Generate creates an implementation plan from a description.
// It gathers project context, calls the AI API with streaming, validates
// the response, and writes the plan file.
func Generate(client APIClient, rootDir, description, complexityGuide string, streamCb api.StreamCallback) (Result, error) {
	// 1. Gather project context.
	ctx := gatherProjectContext(rootDir)

	// 2. Build prompts.
	systemPrompt := buildSystemPrompt(complexityGuide)
	userMessage := buildUserMessage(description, ctx)

	// 3. Stream to get the response.
	fullText, err := client.SendStream(systemPrompt, userMessage, streamCb)
	if err != nil {
		return Result{}, etcherr.WrapAPI("generating plan", err)
	}

	// 4. Extract markdown from response.
	markdown := extractMarkdown(fullText)

	// 5. Validate by parsing.
	plan, err := parser.Parse(strings.NewReader(markdown))
	if err != nil {
		return Result{}, etcherr.WrapParse("generated plan failed validation", err).
			WithHint("the AI response may not follow the expected format — try again")
	}

	return Result{
		Plan:     plan,
		Markdown: markdown,
	}, nil
}

// WritePlan writes the plan markdown to a file in the plans directory.
func WritePlan(rootDir, slug, markdown string) (string, error) {
	plansDir := filepath.Join(rootDir, ".etch", "plans")
	if err := os.MkdirAll(plansDir, 0o755); err != nil {
		return "", etcherr.WrapIO("creating plans directory", err)
	}

	path := filepath.Join(plansDir, slug+".md")
	if err := os.WriteFile(path, []byte(markdown), 0o644); err != nil {
		return "", etcherr.WrapIO("writing plan file", err)
	}
	return path, nil
}

// SlugExists checks if a plan file already exists for the given slug.
func SlugExists(rootDir, slug string) bool {
	path := filepath.Join(rootDir, ".etch", "plans", slug+".md")
	_, err := os.Stat(path)
	return err == nil
}

// Slugify converts a description into a URL-friendly slug.
// Lowercase, hyphens for spaces, strip non-alphanumeric, max 50 chars.
func Slugify(s string) string {
	s = strings.ToLower(s)
	// Replace spaces and underscores with hyphens.
	s = strings.Map(func(r rune) rune {
		if r == ' ' || r == '_' {
			return '-'
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1 // strip
	}, s)
	// Collapse multiple hyphens.
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	// Truncate to 50 chars, but don't cut in the middle of a word.
	if len(s) > 50 {
		s = s[:50]
		if idx := strings.LastIndex(s, "-"); idx > 30 {
			s = s[:idx]
		}
	}
	return s
}

// ResolveSlug returns a unique slug, appending -2, -3, etc. if needed.
// Returns the slug and whether it already existed (collision).
func ResolveSlug(rootDir, slug string) (string, bool) {
	if !SlugExists(rootDir, slug) {
		return slug, false
	}
	for i := 2; i <= 100; i++ {
		candidate := fmt.Sprintf("%s-%d", slug, i)
		if !SlugExists(rootDir, candidate) {
			return candidate, true
		}
	}
	return slug, true
}

// ExistingPlanTitles returns the titles of all existing plans in .etch/plans/.
func ExistingPlanTitles(rootDir string) []string {
	plansDir := filepath.Join(rootDir, ".etch", "plans")
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		return nil
	}
	var titles []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(plansDir, e.Name())
		plan, err := parser.ParseFile(path)
		if err != nil {
			continue
		}
		titles = append(titles, plan.Title)
	}
	return titles
}

// gatherProjectContext builds the project context string for the AI prompt.
func gatherProjectContext(rootDir string) string {
	var b strings.Builder

	// File tree.
	tree := buildFileTree(rootDir, maxTreeDepth)
	if len(tree) > 0 {
		b.WriteString("### File Tree\n```\n")
		if len(tree) >= maxTreeLines {
			tree = tree[:maxTreeLines]
			b.WriteString(strings.Join(tree, "\n"))
			b.WriteString("\n... (truncated)\n")
		} else {
			b.WriteString(strings.Join(tree, "\n"))
			b.WriteString("\n")
		}
		b.WriteString("```\n\n")
	}

	// Key config files.
	for _, name := range configFiles {
		path := filepath.Join(rootDir, name)
		content := readFileHead(path, configLineLimit)
		if content != "" {
			b.WriteString(fmt.Sprintf("### %s\n```\n%s\n```\n\n", name, content))
		}
	}

	// CLAUDE.md if present.
	claudePath := filepath.Join(rootDir, "CLAUDE.md")
	claudeContent := readFileHead(claudePath, 200)
	if claudeContent != "" {
		b.WriteString("### CLAUDE.md\n```\n")
		b.WriteString(claudeContent)
		b.WriteString("\n```\n\n")
	}

	// Existing plan titles.
	titles := ExistingPlanTitles(rootDir)
	if len(titles) > 0 {
		b.WriteString("### Existing Plans\n")
		for _, t := range titles {
			b.WriteString(fmt.Sprintf("- %s\n", t))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// buildFileTree returns a flat list of indented file/dir entries up to maxDepth levels.
func buildFileTree(root string, maxDepth int) []string {
	var lines []string
	walkTree(root, "", 0, maxDepth, &lines)
	return lines
}

func walkTree(dir, prefix string, depth, maxDepth int, lines *[]string) {
	if depth >= maxDepth || len(*lines) >= maxTreeLines {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, e := range entries {
		if len(*lines) >= maxTreeLines {
			return
		}
		name := e.Name()
		if excludeDirs[name] && e.IsDir() {
			continue
		}
		// Skip hidden files/dirs (except CLAUDE.md style files).
		if strings.HasPrefix(name, ".") && name != ".env.example" {
			continue
		}

		if e.IsDir() {
			*lines = append(*lines, prefix+name+"/")
			walkTree(filepath.Join(dir, name), prefix+"  ", depth+1, maxDepth, lines)
		} else {
			*lines = append(*lines, prefix+name)
		}
	}
}

// readFileHead reads up to n lines from a file. Returns empty string if file doesn't exist.
func readFileHead(path string, n int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

// extractMarkdown extracts the plan markdown from the AI response.
// If the response contains a fenced code block, extract its content.
// Otherwise, look for the first "# Plan:" line and take everything from there.
func extractMarkdown(text string) string {
	// Try to find a fenced markdown block.
	fenceRe := regexp.MustCompile("(?s)```(?:markdown)?\\s*\\n(.*?)```")
	if m := fenceRe.FindStringSubmatch(text); m != nil {
		return strings.TrimSpace(m[1])
	}

	// Otherwise, find the first "# Plan:" line and take everything from there.
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "# Plan:") {
			return strings.TrimSpace(strings.Join(lines[i:], "\n"))
		}
	}

	// Fall back to the full text.
	return strings.TrimSpace(text)
}
