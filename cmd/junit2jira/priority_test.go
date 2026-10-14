package main

import (
	"testing"
	"time"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/stretchr/testify/require"
)

func TestPriorityForCommentCount(t *testing.T) {
	thresholds := []int{2, 10, 50, 100, 200}
	for _, test := range []struct {
		count    int
		expected Priority
	}{
		{0, Undefined}, {1, Undefined}, {2, Minor}, {10, Normal},
		{50, Major}, {100, Blocker}, {200, Critical}, {1000, Critical},
	} {
		require.Equal(t, test.expected, priorityForCommentCount(test.count, thresholds))
	}
}

func TestPriorityForCommentCountCustomThresholds(t *testing.T) {
	customThresholds, err := parsePriorityThresholds("10,50,100,200,400")
	require.NoError(t, err)
	for name, test := range map[string]struct {
		count           int
		defaultPriority Priority
		customPriority  Priority
	}{
		"below Minor":                   {9, Minor, Undefined},
		"Minor boundary":                {10, Normal, Minor},
		"same count different priority": {12, Normal, Minor},
		"below Normal":                  {49, Normal, Minor},
		"Normal boundary":               {50, Major, Normal},
		"below Major":                   {99, Major, Normal},
		"Major boundary":                {100, Blocker, Major},
		"below Blocker":                 {199, Blocker, Major},
		"Blocker boundary":              {200, Critical, Blocker},
		"below Critical":                {399, Critical, Blocker},
		"Critical boundary":             {400, Critical, Critical},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.defaultPriority, priorityForCommentCount(test.count, defaultPriorityThresholds))
			require.Equal(t, test.customPriority, priorityForCommentCount(test.count, customThresholds))
		})
	}
}

func TestCountCommentsInWindow(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	comments := []*models.IssueCommentScheme{
		{Created: "2026-09-11T11:00:00.000+0000"},
		{Created: "2026-08-11T12:00:00.000+0000"},
		{Created: "2026-08-12T12:00:00.000+0000"},
		{Created: "not-a-timestamp"},
		nil,
		{Created: "another-invalid-timestamp"},
	}
	warnings := 0
	count := countCommentsInWindow(comments, 30, now, func(index int, timestamp string) {
		require.Equal(t, 3, index)
		require.Equal(t, "not-a-timestamp", timestamp)
		warnings++
	})
	require.Equal(t, 1, count)
	require.Equal(t, 1, warnings)
}
