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
