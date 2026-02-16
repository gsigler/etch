package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	etcherr "github.com/gsigler/etch/internal/errors"
)

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
