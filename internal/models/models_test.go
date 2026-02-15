package models

import "testing"

func TestParseStatus(t *testing.T) {
	tests := []struct {
		input string
		want  Status
	}{
		{"pending", StatusPending},
		{"in_progress", StatusInProgress},
		{"blocked", StatusBlocked},
		{"completed", StatusCompleted},
		{"failed", StatusFailed},
		{"", StatusPending},
		{"unknown", StatusPending},
		{"PENDING", StatusPending},
		{"done", StatusPending},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseStatus(tt.input)
			if got != tt.want {
				t.Errorf("ParseStatus(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusCompleted, "✓"},
		{StatusInProgress, "▶"},
		{StatusPending, "○"},
		{StatusFailed, "✗"},
		{StatusBlocked, "⊘"},
		{Status("unknown"), "○"},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := tt.status.Icon()
			if got != tt.want {
				t.Errorf("Status(%q).Icon() = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestTaskFullID(t *testing.T) {
	tests := []struct {
		name          string
		featureNumber int
		taskNumber    int
		suffix        string
		want          string
	}{
		{"single digit", 1, 2, "", "1.2"},
		{"first task", 1, 1, "", "1.1"},
		{"higher feature", 3, 5, "", "3.5"},
		{"double digits", 12, 34, "", "12.34"},
		{"with suffix", 1, 3, "b", "1.3b"},
		{"suffix a", 2, 1, "a", "2.1a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{
				FeatureNumber: tt.featureNumber,
				TaskNumber:    tt.taskNumber,
				Suffix:        tt.suffix,
			}
			got := task.FullID()
			if got != tt.want {
				t.Errorf("Task{%d, %d, %q}.FullID() = %q, want %q", tt.featureNumber, tt.taskNumber, tt.suffix, got, tt.want)
			}
		})
	}
}

func TestPlanTaskByID(t *testing.T) {
	plan := &Plan{
		Features: []Feature{
			{
				Number: 1,
				Tasks: []Task{
					{FeatureNumber: 1, TaskNumber: 1, Title: "Task 1.1"},
					{FeatureNumber: 1, TaskNumber: 2, Title: "Task 1.2"},
					{FeatureNumber: 1, TaskNumber: 2, Suffix: "b", Title: "Task 1.2b"},
				},
			},
			{
				Number: 2,
				Tasks: []Task{
					{FeatureNumber: 2, TaskNumber: 1, Title: "Task 2.1"},
				},
			},
		},
	}

	tests := []struct {
		id        string
		wantTitle string
		wantNil   bool
	}{
		{"1.1", "Task 1.1", false},
		{"1.2", "Task 1.2", false},
		{"1.2b", "Task 1.2b", false},
		{"2.1", "Task 2.1", false},
		{"3.1", "", true},
		{"0.0", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := plan.TaskByID(tt.id)
			if tt.wantNil {
				if got != nil {
					t.Errorf("TaskByID(%q) = %v, want nil", tt.id, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("TaskByID(%q) = nil, want task with title %q", tt.id, tt.wantTitle)
			}
			if got.Title != tt.wantTitle {
				t.Errorf("TaskByID(%q).Title = %q, want %q", tt.id, got.Title, tt.wantTitle)
			}
		})
	}
}
