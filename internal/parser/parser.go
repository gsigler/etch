package parser

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gsigler/etch/internal/models"
)

// state tracks which section the parser is currently accumulating content into.
type state int

const (
	stateInit        state = iota // before seeing # Plan:
	statePlanLevel                // after # Plan:, before any ## heading
	stateOverview                 // inside ## Overview
	stateFeature                  // inside ## Feature N: (feature-level content)
	stateFeatureOver              // inside ### Overview (feature overview)
	stateTask                     // inside ### Task N.M:
	stateOther                    // inside an unrecognized ## heading (e.g. ## Architecture Decisions)
)

var (
	planHeadingRe    = regexp.MustCompile(`^#\s+Plan:\s*(.+)$`)
	featureHeadingRe = regexp.MustCompile(`^##\s+Feature\s+(\d+):\s*(.+)$`)
	overviewH2Re     = regexp.MustCompile(`^##\s+Overview\s*$`)
	overviewH3Re     = regexp.MustCompile(`^###\s+Overview\s*$`)
	taskHeadingRe    = regexp.MustCompile(`^###\s+Task\s+(\d+)(?:\.(\d+)([a-z])?)?:\s*(.+)$`)
	statusTagRe      = regexp.MustCompile(`\[(\w+)\]\s*$`)
	separatorRe      = regexp.MustCompile(`^---+\s*$`)
	h2Re             = regexp.MustCompile(`^##\s+`)
	h3Re             = regexp.MustCompile(`^###\s+`)

	// Plan metadata patterns.
	priorityRe = regexp.MustCompile(`^\*\*Priority:\*\*\s*(\d+)\s*$`)

	// Task metadata patterns.
	complexityRe = regexp.MustCompile(`^\*\*Complexity:\*\*\s*(.+)$`)
	filesRe      = regexp.MustCompile(`^\*\*Files(?:\s+in\s+Scope)?:\*\*\s*(.+)$`)
	dependsOnRe  = regexp.MustCompile(`^\*\*Depends\s+on:\*\*\s*(.+)$`)
	criteriaHeadingRe = regexp.MustCompile(`^\*\*Acceptance\s+Criteria:\*\*\s*$`)
	criterionRe       = regexp.MustCompile(`^-\s+\[([ x])\]\s+(.+)$`)
	commentRe         = regexp.MustCompile(`^>\s*💬\s*(.+)$`)
	commentContRe = regexp.MustCompile(`^>\s*(.+)$`)
)

// ParseFile reads a plan markdown file from disk and parses it into a Plan.
func ParseFile(path string) (*models.Plan, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening plan file: %w", err)
	}
	defer f.Close()

	plan, err := Parse(f)
	if err != nil {
		return nil, err
	}
	plan.FilePath = path

	// Derive slug from filename.
	base := strings.TrimSuffix(filepath(path), ".md")
	plan.Slug = base

	return plan, nil
}

// filepath returns just the filename from a path (no directory).
func filepath(path string) string {
	i := strings.LastIndexAny(path, "/\\")
	if i < 0 {
		return path
	}
	return path[i+1:]
}

// Parse reads plan markdown from r and returns a Plan struct.
// It returns an error if no "# Plan:" heading is found.
func Parse(r io.Reader) (*models.Plan, error) {
	scanner := bufio.NewScanner(r)
	plan := &models.Plan{}

	cur := stateInit
	var descBuf strings.Builder

	var currentFeature *models.Feature
	var currentTask *models.Task
	sawFeatureHeading := false

	inComment := false // tracking multi-line > 💬 comments

	// flush saves accumulated description text into the current section.
	flush := func() {
		text := strings.TrimSpace(descBuf.String())
		descBuf.Reset()
		inComment = false
		if text == "" {
			return
		}
		switch cur {
		case stateOverview:
			plan.Overview = text
		case stateFeature:
			// Content between feature heading and first task/overview — treat as feature overview.
			if currentFeature != nil && currentFeature.Overview == "" {
				currentFeature.Overview = text
			}
		case stateFeatureOver:
			if currentFeature != nil {
				currentFeature.Overview = text
			}
		case stateTask:
			if currentTask != nil {
				currentTask.Description = text
			}
		}
	}

	inCodeFence := false

	for scanner.Scan() {
		line := scanner.Text()

		// Track fenced code blocks — skip everything inside them.
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeFence = !inCodeFence
			// Still accumulate code fence content into current section description.
			if cur == stateTask || cur == stateOverview || cur == stateFeature || cur == stateFeatureOver {
				descBuf.WriteString(line)
				descBuf.WriteString("\n")
			}
			continue
		}
		if inCodeFence {
			if cur == stateTask || cur == stateOverview || cur == stateFeature || cur == stateFeatureOver {
				descBuf.WriteString(line)
				descBuf.WriteString("\n")
			}
			continue
		}

		// Skip separators.
		if separatorRe.MatchString(line) {
			continue
		}

		// Check for # Plan: heading.
		if m := planHeadingRe.FindStringSubmatch(line); m != nil {
			flush()
			plan.Title = strings.TrimSpace(m[1])
			cur = statePlanLevel
			continue
		}

		// Check for ## Overview (plan-level overview).
		if overviewH2Re.MatchString(line) {
			flush()
			// Only treat as plan overview if we haven't entered a feature yet.
			if !sawFeatureHeading {
				cur = stateOverview
			} else {
				// Feature-level content that happens to be "## Overview" — unlikely but handle.
				cur = stateOther
			}
			continue
		}

		// Check for ## Feature N: heading.
		if m := featureHeadingRe.FindStringSubmatch(line); m != nil {
			flush()
			sawFeatureHeading = true
			num := atoi(m[1])
			f := models.Feature{
				Number: num,
				Title:  strings.TrimSpace(m[2]),
			}
			plan.Features = append(plan.Features, f)
			currentFeature = &plan.Features[len(plan.Features)-1]
			currentTask = nil
			cur = stateFeature
			continue
		}

		// Check for ### Overview (feature-level overview).
		if overviewH3Re.MatchString(line) {
			flush()
			if currentFeature != nil {
				cur = stateFeatureOver
			}
			continue
		}

		// Check for ### Task heading.
		if m := taskHeadingRe.FindStringSubmatch(line); m != nil {
			flush()

			featureNum := atoi(m[1])
			taskNum := 0
			suffix := m[3] // letter suffix (e.g. "b") or ""
			title := strings.TrimSpace(m[4])

			if m[2] != "" {
				// Multi-feature format: Task N.M or Task N.Mb
				taskNum = atoi(m[2])
			} else {
				// Single-feature format: Task N → taskNum = N, featureNum = 1
				taskNum = featureNum
				featureNum = 1
			}

			// Extract status tag from title.
			status := models.StatusPending
			if sm := statusTagRe.FindStringSubmatch(title); sm != nil {
				status = models.ParseStatus(sm[1])
				title = strings.TrimSpace(statusTagRe.ReplaceAllString(title, ""))
			}

			// If no feature heading seen yet, create implicit feature.
			if !sawFeatureHeading && currentFeature == nil {
				f := models.Feature{
					Number: 1,
					Title:  plan.Title,
				}
				plan.Features = append(plan.Features, f)
				currentFeature = &plan.Features[len(plan.Features)-1]
			}

			t := models.Task{
				FeatureNumber: featureNum,
				TaskNumber:    taskNum,
				Suffix:        suffix,
				Title:         title,
				Status:        status,
			}
			currentFeature.Tasks = append(currentFeature.Tasks, t)
			currentTask = &currentFeature.Tasks[len(currentFeature.Tasks)-1]
			cur = stateTask
			continue
		}

		// Check for other ## headings (skip their content).
		if h2Re.MatchString(line) {
			flush()
			currentTask = nil
			cur = stateOther
			continue
		}

		// Check for other ### headings within a task — treat as task description content.
		// (Don't change state, just accumulate.)

		// Parse plan-level metadata (before any feature heading).
		if cur == statePlanLevel {
			if m := priorityRe.FindStringSubmatch(line); m != nil {
				if n, err := strconv.Atoi(m[1]); err == nil {
					plan.Priority = n
				}
				continue
			}
		}

		// Accumulate content into current section.
		switch cur {
		case stateOverview, stateFeature, stateFeatureOver:
			descBuf.WriteString(line)
			descBuf.WriteString("\n")
		case stateTask:
			if currentTask == nil {
				break
			}
			// Try to extract task metadata before falling through to description.
			if m := complexityRe.FindStringSubmatch(line); m != nil {
				currentTask.Complexity = models.Complexity(strings.TrimSpace(m[1]))
				continue
			}
			if m := filesRe.FindStringSubmatch(line); m != nil {
				for _, f := range strings.Split(m[1], ",") {
					f = strings.TrimSpace(f)
					if f != "" {
						currentTask.Files = append(currentTask.Files, f)
					}
				}
				continue
			}
			if m := dependsOnRe.FindStringSubmatch(line); m != nil {
				for _, d := range strings.Split(m[1], ",") {
					d = strings.TrimSpace(d)
					if d != "" {
						currentTask.DependsOn = append(currentTask.DependsOn, d)
					}
				}
				continue
			}
			if criteriaHeadingRe.MatchString(line) {
				continue
			}
			if m := criterionRe.FindStringSubmatch(line); m != nil {
				inComment = false
				currentTask.Criteria = append(currentTask.Criteria, models.Criterion{
					Description: strings.TrimSpace(m[2]),
					IsMet:       m[1] == "x",
				})
				continue
			}
			if m := commentRe.FindStringSubmatch(line); m != nil {
				inComment = true
				currentTask.Comments = append(currentTask.Comments, strings.TrimSpace(m[1]))
				continue
			}
			// Multi-line comment continuation: > lines following a 💬 line.
			if inComment {
				if m := commentContRe.FindStringSubmatch(line); m != nil {
					idx := len(currentTask.Comments) - 1
					currentTask.Comments[idx] += "\n" + strings.TrimSpace(m[1])
					continue
				}
				inComment = false
			}
			descBuf.WriteString(line)
			descBuf.WriteString("\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading plan: %w", err)
	}

	// Final flush.
	flush()

	if plan.Title == "" {
		return nil, fmt.Errorf("invalid plan file: no '# Plan:' heading found")
	}

	return plan, nil
}

// atoi converts a string to int, returning 0 on failure.
func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
