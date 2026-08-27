package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	log "github.com/sirupsen/logrus"
)

// parsePriorityThresholds parses comma-separated threshold string into int slice
func parsePriorityThresholds(thresholdsStr string) ([]int, error) {
	parts := strings.Split(thresholdsStr, ",")
	if len(parts) != 5 {
		return defaultPriorityThresholds, fmt.Errorf("expected 5 thresholds, got %d", len(parts))
	}

	thresholds := make([]int, 5)
	for i, part := range parts {
		val, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return defaultPriorityThresholds, fmt.Errorf("invalid threshold value %q: %w", part, err)
		}
		thresholds[i] = val
	}

	return thresholds, nil
}

// countCommentsInTimeWindows counts comments in the last 10 and 30 days
func countCommentsInTimeWindows(comments []*models.IssueCommentScheme) (last30Days, last10Days int) {
	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	tenDaysAgo := now.AddDate(0, 0, -10)

	for _, comment := range comments {
		if comment.Created == "" {
			continue
		}

		// Parse comment created timestamp
		// JIRA format: "2006-01-02T15:04:05.000-0700"
		created, err := time.Parse("2006-01-02T15:04:05.000-0700", comment.Created)
		if err != nil {
			// Try RFC3339 format as fallback
			created, err = time.Parse(time.RFC3339, comment.Created)
			if err != nil {
				continue
			}
		}

		if created.After(thirtyDaysAgo) {
			last30Days++
		}
		if created.After(tenDaysAgo) {
			last10Days++
		}
	}

	return last30Days, last10Days
}

// updatePriorityIfNeeded updates issue priority based on comment count
func (j junit2jira) updatePriorityIfNeeded(issueKey string) error {
	if !j.enableAutoPriority {
		return nil
	}

	// Parse thresholds
	thresholds, err := parsePriorityThresholds(j.priorityThresholds)
	if err != nil {
		log.WithError(err).Warn("Failed to parse priority thresholds, using defaults")
		thresholds = defaultPriorityThresholds
	}

	// Get issue with comments to count them
	issue, response, err := j.jiraClient.Issue.Get(
		context.TODO(),
		issueKey,
		[]string{"priority", "comment"}, // fields
		nil,                              // expand
	)
	if err != nil {
		logError(err, response)
		return fmt.Errorf("could not fetch issue %s: %w", issueKey, err)
	}

	// Count comments
	commentCount := 0
	if issue.Fields != nil && issue.Fields.Comment != nil {
		commentCount = issue.Fields.Comment.Total
	}

	// Fetch all comment pages
	var allComments []*models.IssueCommentScheme
	if commentCount > 0 {
		startAt := 0
		maxResults := 50
		for {
			commentsPage, response, err := j.jiraClient.Issue.Comment.Gets(
				context.TODO(),
				issueKey,
				"created", // orderBy
				nil,       // expand
				startAt,
				maxResults,
			)
			if err != nil {
				logError(err, response)
				return fmt.Errorf("could not fetch comments for issue %s: %w", issueKey, err)
			}

			if commentsPage != nil && commentsPage.Comments != nil {
				allComments = append(allComments, commentsPage.Comments...)
			}

			// Check if we've fetched all comments
			if commentsPage == nil || len(commentsPage.Comments) < maxResults {
				break
			}

			startAt += maxResults
		}
	}

	// Count comments in time windows for time-based escalation
	last30Days, last10Days := countCommentsInTimeWindows(allComments)

	// Get current priority
	currentPriority := Undefined
	currentPriorityRecognized := true
	if issue.Fields != nil && issue.Fields.Priority != nil {
		currentPriority, currentPriorityRecognized = parsePriority(issue.Fields.Priority.Name)
		if !currentPriorityRecognized {
			logEntry(issueKey, "").Warnf("Unrecognized priority %q, skipping auto-escalation", issue.Fields.Priority.Name)
			return nil
		}
	}

	// Calculate target priority with time-based escalation
	targetPriority := calculatePriorityWithTimeEscalation(commentCount, last30Days, last10Days, thresholds)

	// Only escalate if target priority is higher than current
	if targetPriority > currentPriority {
		logEntry(issueKey, "").Infof("Auto-escalating priority from %s to %s (total: %d, last 30d: %d, last 10d: %d)",
			currentPriority, targetPriority, commentCount, last30Days, last10Days)

		if j.dryRun {
			logEntry(issueKey, "").Debug("Dry run: would update priority")
			return nil
		}

		// Update the issue priority
		updatePayload := &models.IssueScheme{
			Fields: &models.IssueFieldsScheme{
				Priority: &models.PriorityScheme{
					Name: targetPriority.String(),
				},
			},
		}

		// Set notify to true - false requires admin permissions to suppress notifications
		response, err := j.jiraClient.Issue.Update(context.TODO(), issueKey, true, updatePayload, nil, nil)
		if err != nil {
			logError(err, response)
			return fmt.Errorf("could not update priority for issue %s: %w", issueKey, err)
		}

		logEntry(issueKey, "").Infof("Updated priority to %s", targetPriority)
	} else {
		logEntry(issueKey, "").Debugf("Priority %s is already correct for %d comments", currentPriority, commentCount)
	}

	return nil
}
