package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
	if issue.Fields != nil && issue.Fields.Comment != nil && issue.Fields.Comment.Comments != nil {
		commentCount = len(issue.Fields.Comment.Comments)
	}

	// Get current priority
	currentPriority := "Undefined"
	if issue.Fields != nil && issue.Fields.Priority != nil {
		currentPriority = issue.Fields.Priority.Name
	}

	// Calculate target priority
	targetPriority := calculatePriorityWithThresholds(commentCount, thresholds)

	// Update if changed
	if currentPriority != targetPriority {
		logEntry(issueKey, "").Infof("Auto-escalating priority from %s to %s (comment count: %d)", currentPriority, targetPriority, commentCount)

		if j.dryRun {
			logEntry(issueKey, "").Debug("Dry run: would update priority")
			return nil
		}

		// Update the issue priority
		updatePayload := &models.IssueScheme{
			Fields: &models.IssueFieldsScheme{
				Priority: &models.PriorityScheme{
					Name: targetPriority,
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
