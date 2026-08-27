package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculatePriority(t *testing.T) {
	tests := []struct {
		name         string
		commentCount int
		expected     Priority
	}{
		// Undefined: 0-3 comments
		{"zero comments", 0, Undefined},
		{"one comment", 1, Undefined},
		{"three comments", 3, Undefined},

		// Minor: 4-15 comments
		{"four comments (threshold)", 4, Minor},
		{"ten comments", 10, Minor},
		{"fifteen comments", 15, Minor},

		// Normal: 16-63 comments
		{"sixteen comments (threshold)", 16, Normal},
		{"thirty comments", 30, Normal},
		{"sixty-three comments", 63, Normal},

		// Major: 64-127 comments
		{"sixty-four comments (threshold)", 64, Major},
		{"one hundred comments", 100, Major},
		{"one hundred twenty-seven comments", 127, Major},

		// Blocker: 128-255 comments
		{"one hundred twenty-eight comments (threshold)", 128, Blocker},
		{"two hundred comments", 200, Blocker},
		{"two hundred fifty-five comments", 255, Blocker},

		// Critical: 256+ comments
		{"two hundred fifty-six comments (threshold)", 256, Critical},
		{"five hundred comments", 500, Critical},
		{"one thousand comments", 1000, Critical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculatePriority(tt.commentCount)
			assert.Equal(t, tt.expected, result, "Comment count %d should map to priority %s", tt.commentCount, tt.expected)
		})
	}
}

func TestCalculatePriorityWithCustomThresholds(t *testing.T) {
	tests := []struct {
		name         string
		commentCount int
		thresholds   []int
		expected     Priority
	}{
		// Default thresholds: 4, 16, 64, 128, 256
		{"default: 3 comments", 3, []int{4, 16, 64, 128, 256}, Undefined},
		{"default: 4 comments", 4, []int{4, 16, 64, 128, 256}, Minor},
		{"default: 16 comments", 16, []int{4, 16, 64, 128, 256}, Normal},
		{"default: 64 comments", 64, []int{4, 16, 64, 128, 256}, Major},
		{"default: 128 comments", 128, []int{4, 16, 64, 128, 256}, Blocker},
		{"default: 256 comments", 256, []int{4, 16, 64, 128, 256}, Critical},

		// Custom thresholds: 10, 50, 100, 200, 400
		{"custom: 9 comments", 9, []int{10, 50, 100, 200, 400}, Undefined},
		{"custom: 10 comments", 10, []int{10, 50, 100, 200, 400}, Minor},
		{"custom: 50 comments", 50, []int{10, 50, 100, 200, 400}, Normal},
		{"custom: 100 comments", 100, []int{10, 50, 100, 200, 400}, Major},
		{"custom: 200 comments", 200, []int{10, 50, 100, 200, 400}, Blocker},
		{"custom: 400 comments", 400, []int{10, 50, 100, 200, 400}, Critical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculatePriorityWithThresholds(tt.commentCount, tt.thresholds)
			assert.Equal(t, tt.expected, result, "Comment count %d with thresholds %v should map to priority %s", tt.commentCount, tt.thresholds, tt.expected)
		})
	}
}

func TestPriorityOrdering(t *testing.T) {
	tests := []struct {
		name     string
		lower    Priority
		higher   Priority
		expected bool
	}{
		{"Undefined < Minor", Undefined, Minor, true},
		{"Minor < Normal", Minor, Normal, true},
		{"Normal < Major", Normal, Major, true},
		{"Major < Blocker", Major, Blocker, true},
		{"Blocker < Critical", Blocker, Critical, true},
		{"Critical = Critical", Critical, Critical, false},
		{"Major > Minor", Major, Minor, false},
		{"Critical > Undefined", Critical, Undefined, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.higher > tt.lower
			assert.Equal(t, tt.expected, result, "%s should be %v when comparing %s > %s", tt.name, tt.expected, tt.higher, tt.lower)
		})
	}
}

func TestPriorityString(t *testing.T) {
	tests := []struct {
		priority Priority
		expected string
	}{
		{Undefined, "Undefined"},
		{Minor, "Minor"},
		{Normal, "Normal"},
		{Major, "Major"},
		{Blocker, "Blocker"},
		{Critical, "Critical"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.priority.String())
		})
	}
}

func TestParsePriority(t *testing.T) {
	tests := []struct {
		name     string
		expected Priority
	}{
		{"Undefined", Undefined},
		{"Minor", Minor},
		{"Normal", Normal},
		{"Major", Major},
		{"Blocker", Blocker},
		{"Critical", Critical},
		{"Unknown", Undefined},
		{"", Undefined},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePriority(tt.name)
			assert.Equal(t, tt.expected, result)
		})
	}
}
