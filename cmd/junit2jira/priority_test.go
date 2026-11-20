package main

import (
	"fmt"
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
		{50, Major}, {100, Critical}, {200, Blocker}, {1000, Blocker},
	} {
		require.Equal(t, test.expected, priorityForCommentCount(test.count, thresholds))
	}
}

func TestPriorityForCommentCountCustomThresholds(t *testing.T) {
	customThresholds, err := parsePriorityThresholds("10,50,100,200,400")
	require.NoError(t, err)
	for count, test := range map[int]struct {
		defaultPriority Priority
		customPriority  Priority
	}{
		0:   {Undefined, Undefined},
		1:   {Minor, Undefined},
		2:   {Normal, Undefined},
		11:  {Normal, Minor},
		12:  {Major, Minor},
		23:  {Major, Minor},
		24:  {Critical, Minor},
		47:  {Critical, Minor},
		49:  {Blocker, Minor},
		50:  {Blocker, Normal},
		99:  {Blocker, Normal},
		100: {Blocker, Major},
		199: {Blocker, Major},
		200: {Blocker, Critical},
		399: {Blocker, Critical},
		400: {Blocker, Blocker},
	} {
		t.Run(fmt.Sprintf("count=%d", count), func(t *testing.T) {
			defaultRes := priorityForCommentCount(count, defaultPriorityThresholds)
			require.Equal(t, test.defaultPriority, defaultRes, "priority for default thresholds does not match")

			customRes := priorityForCommentCount(count, customThresholds)
			require.Equal(t, test.customPriority, customRes, "priority for custom thresholds does not match")
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

func TestPriorityOrder(t *testing.T) {
	require.Less(t, Major, Critical)
	require.Less(t, Critical, Blocker)
}
