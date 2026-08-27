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
		// New thresholds: [2, 10, 50, 100, 200]

		// Undefined: 0-1 comments
		{"zero comments", 0, Undefined},
		{"one comment", 1, Undefined},

		// Minor: 2-9 comments
		{"two comments (threshold)", 2, Minor},
		{"five comments", 5, Minor},
		{"nine comments", 9, Minor},

		// Normal: 10-49 comments
		{"ten comments (threshold)", 10, Normal},
		{"thirty comments", 30, Normal},
		{"forty-nine comments", 49, Normal},

		// Major: 50-99 comments
		{"fifty comments (threshold)", 50, Major},
		{"seventy-five comments", 75, Major},
		{"ninety-nine comments", 99, Major},

		// Blocker: 100-199 comments
		{"one hundred comments (threshold)", 100, Blocker},
		{"one hundred fifty comments", 150, Blocker},
		{"one hundred ninety-nine comments", 199, Blocker},

		// Critical: 200+ comments
		{"two hundred comments (threshold)", 200, Critical},
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

		// Invalid thresholds - negative values (should fall back to defaults)
		{"negative threshold", 50, []int{-1, 10, 50, 100, 200}, Major},
		{"multiple negative", 100, []int{2, -5, 50, 100, 200}, Blocker},

		// Invalid thresholds - non-ascending (should fall back to defaults)
		{"equal values", 50, []int{2, 10, 10, 100, 200}, Major},
		{"descending", 100, []int{200, 100, 50, 10, 2}, Blocker},
		{"partially descending", 50, []int{2, 50, 30, 100, 200}, Major},

		// Invalid thresholds - wrong count (should fall back to defaults)
		{"too few", 50, []int{2, 10, 50}, Major},
		{"too many", 100, []int{2, 10, 50, 100, 200, 300}, Blocker},
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
		name       string
		expected   Priority
		recognized bool
	}{
		{"Undefined", Undefined, true},
		{"Minor", Minor, true},
		{"Normal", Normal, true},
		{"Major", Major, true},
		{"Blocker", Blocker, true},
		{"Critical", Critical, true},
		{"Unknown", Undefined, false},
		{"", Undefined, false},
		{"CustomPriority", Undefined, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, recognized := parsePriority(tt.name)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, tt.recognized, recognized)
		})
	}
}

func TestMaxPriority(t *testing.T) {
	tests := []struct {
		name     string
		a        Priority
		b        Priority
		expected Priority
	}{
		{"Critical vs Major", Critical, Major, Critical},
		{"Major vs Critical", Major, Critical, Critical},
		{"Normal vs Minor", Normal, Minor, Normal},
		{"Same priority", Major, Major, Major},
		{"Undefined vs Minor", Undefined, Minor, Minor},
		{"Critical vs Undefined", Critical, Undefined, Critical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maxPriority(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculatePriorityWithTimeEscalation(t *testing.T) {
	tests := []struct {
		name         string
		total        int
		last30       int
		last10       int
		expected     Priority
		reason       string
	}{
		// Hot issues (last 10 days triggers escalation)
		{"Very hot issue", 24, 21, 12, Critical, "12 in last 10 days"},
		{"Hot burst", 7, 7, 7, Major, "7 in last 10 days"},
		{"Recent spike", 39, 23, 7, Major, "7 in last 10 days"},
		{"Extremely hot", 5, 5, 15, Critical, "15 in last 10 days"},
		{"Just hot", 3, 3, 5, Major, "5 in last 10 days"},

		// Active issues (last 30 days triggers escalation)
		{"Sustained high", 36, 21, 1, Blocker, "21 in last 30 days"},
		{"Very active", 50, 50, 0, Critical, "50 in last 30 days"},
		{"Active trend", 24, 15, 0, Major, "15 in last 30 days → escalated to Major"},
		{"Moderate activity", 10, 7, 0, Normal, "7 in last 30 days"},
		{"Low recent", 100, 3, 0, Blocker, "High total, low recent"},

		// Persistent issues (total drives priority)
		{"Historic high", 842, 0, 0, Critical, "High total"},
		{"Stale but important", 65, 0, 0, Major, "High total, no recent"},
		{"Persistent moderate", 58, 2, 1, Major, "Total drives priority"},

		// Combined signals
		{"All signals high", 253, 100, 50, Critical, "All Critical"},
		{"Mixed signals", 14, 14, 4, Major, "Total=Normal, escalated by last 30d"},

		// Low activity
		{"Minimal", 1, 1, 1, Undefined, "Below all thresholds"},
		{"Some activity", 5, 2, 0, Minor, "5 total → Minor (no escalation with only 2 in last 30d)"},
		{"Low total, low recent", 3, 1, 0, Minor, "3 total -> Minor"},
	}

	thresholds := []int{2, 10, 50, 100, 200}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculatePriorityWithTimeEscalation(tt.total, tt.last30, tt.last10, thresholds)
			assert.Equal(t, tt.expected, result,
				"Issue with %d total, %d last30, %d last10 should be %s (%s)",
				tt.total, tt.last30, tt.last10, tt.expected, tt.reason)
		})
	}
}
