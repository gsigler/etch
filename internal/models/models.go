package models

import "fmt"

// Status represents the current state of a task.
type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusBlocked    Status = "blocked"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

// ParseStatus converts a string to a Status, defaulting to StatusPending for unknown values.
func ParseStatus(s string) Status {
	switch Status(s) {
	case StatusPending, StatusInProgress, StatusBlocked, StatusCompleted, StatusFailed:
		return Status(s)
	default:
		return StatusPending
	}
}

// Icon returns a unicode icon representing the status.
func (s Status) Icon() string {
	switch s {
	case StatusCompleted:
		return "✓"
	case StatusInProgress:
		return "▶"
	case StatusPending:
		return "○"
	case StatusFailed:
		return "✗"
	case StatusBlocked:
		return "⊘"
	default:
		return "○"
	}
}

// Complexity represents the estimated size of a task.
type Complexity string

const (
	ComplexitySmall  Complexity = "small"
	ComplexityMedium Complexity = "medium"
	ComplexityLarge  Complexity = "large"
)

// Plan represents an implementation plan parsed from a markdown file.
type Plan struct {
	Title    string    `json:"title"`
	Overview string    `json:"overview"`
	Features []Feature `json:"features"`
	FilePath string    `json:"file_path"`
	Slug     string    `json:"slug"`
}

// TaskByID finds a task by its full ID (e.g. "1.2"). Returns nil if not found.
func (p *Plan) TaskByID(id string) *Task {
	for i := range p.Features {
		for j := range p.Features[i].Tasks {
			if p.Features[i].Tasks[j].FullID() == id {
				return &p.Features[i].Tasks[j]
			}
		}
	}
	return nil
}

// Feature represents a group of related tasks within a plan.
type Feature struct {
	Number   int    `json:"number"`
	Title    string `json:"title"`
	Overview string `json:"overview"`
	Tasks    []Task `json:"tasks"`
}

// Task represents a single unit of work within a feature.
type Task struct {
	FeatureNumber int        `json:"feature_number"`
	TaskNumber    int        `json:"task_number"`
	Suffix        string     `json:"suffix,omitempty"`
	Title         string     `json:"title"`
	Status        Status     `json:"status"`
	Complexity    Complexity `json:"complexity"`
	Files         []string   `json:"files"`
	DependsOn     []string   `json:"depends_on"`
	Description   string     `json:"description"`
	Criteria      []Criterion `json:"criteria"`
	Comments      []string   `json:"comments"`
}

// FullID returns the task identifier in "feature.task" format (e.g. "1.2" or "1.3b").
func (t *Task) FullID() string {
	return fmt.Sprintf("%d.%d%s", t.FeatureNumber, t.TaskNumber, t.Suffix)
}

// Criterion represents a single acceptance criterion for a task.
type Criterion struct {
	Description string `json:"description"`
	IsMet       bool   `json:"is_met"`
}

// SessionProgress tracks work done on a task in a single session.
type SessionProgress struct {
	PlanSlug        string      `json:"plan_slug"`
	TaskID          string      `json:"task_id"`
	SessionNumber   int         `json:"session_number"`
	Started         string      `json:"started"`
	Status          string      `json:"status"`
	ChangesMade     []string    `json:"changes_made"`
	CriteriaUpdates []Criterion `json:"criteria_updates"`
	Decisions       string      `json:"decisions"`
	Blockers        string      `json:"blockers"`
	Next            string      `json:"next"`
}
