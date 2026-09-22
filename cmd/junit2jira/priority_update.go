package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
)

func parsePriorityThresholds(thresholdsStr string) ([]int, error) {
	parts := strings.Split(thresholdsStr, ",")
	if len(parts) != len(defaultPriorityThresholds) {
		return nil, fmt.Errorf("expected %d thresholds, got %d", len(defaultPriorityThresholds), len(parts))
	}

	thresholds := make([]int, len(parts))
	for i, part := range parts {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("invalid threshold value %q: %w", part, err)
		}
		if value < 0 {
			return nil, fmt.Errorf("threshold %d must not be negative", i+1)
		}
		if i > 0 && value <= thresholds[i-1] {
			return nil, fmt.Errorf("thresholds must be strictly ascending")
		}
		thresholds[i] = value
	}
	return thresholds, nil
}

func countCommentsInWindow(comments []*models.IssueCommentScheme, windowDays int, now time.Time, warn func(index int, timestamp string)) int {
	windowStart := now.AddDate(0, 0, -windowDays)
	count := 0
	warned := false
	for index, comment := range comments {
		if comment == nil || comment.Created == "" {
			continue
		}

		created, err := parseCommentTime(comment.Created)
		if err != nil {
			if !warned {
				warn(index, comment.Created)
				warned = true
			}
			continue
		}
		if created.After(windowStart) {
			count++
		}
	}
	return count
}

func parseCommentTime(value string) (time.Time, error) {
	for _, layout := range []string{
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05-0700",
		time.RFC3339,
	} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported timestamp format")
}

func (j junit2jira) updatePriorityIfNeeded(ctx context.Context, issueKey string) error {
	if !j.enableAutoPriority {
		return nil
	}
	if j.priorityWindowDays <= 0 {
		return fmt.Errorf("priority window must be positive")
	}
	if len(j.parsedPriorityThresholds) != len(defaultPriorityThresholds) {
		return fmt.Errorf("priority thresholds have not been configured")
	}

	issue, response, err := j.jiraClient.Issue.Get(ctx, issueKey, []string{"priority", "comment"}, nil)
	if err != nil {
		logError(err, response)
		return fmt.Errorf("could not fetch issue %s: %w", issueKey, err)
	}

	totalComments := 0
	if issue != nil && issue.Fields != nil && issue.Fields.Comment != nil {
		totalComments = issue.Fields.Comment.Total
	}

	const pageSize = 50
	comments := make([]*models.IssueCommentScheme, 0, totalComments)
	for startAt := 0; startAt < totalComments; {
		page, response, err := j.jiraClient.Issue.Comment.Gets(ctx, issueKey, "created", nil, startAt, pageSize)
		if err != nil {
			logError(err, response)
			return fmt.Errorf("could not fetch comments for issue %s: %w", issueKey, err)
		}
		if page == nil || len(page.Comments) == 0 {
			break
		}
		comments = append(comments, page.Comments...)
		startAt += len(page.Comments)
		if page.Total > totalComments {
			totalComments = page.Total
		}
	}

	windowCount := countCommentsInWindow(comments, j.priorityWindowDays, time.Now(), func(index int, timestamp string) {
		logEntry(issueKey, "").Warnf("Skipping comment %d with unparseable timestamp %q", index, timestamp)
	})

	currentPriority := Undefined
	if issue != nil && issue.Fields != nil && issue.Fields.Priority != nil {
		var recognized bool
		currentPriority, recognized = parsePriority(issue.Fields.Priority.Name)
		if !recognized {
			logEntry(issueKey, "").Warnf("Unrecognized priority %q, skipping auto-escalation", issue.Fields.Priority.Name)
			return nil
		}
	}

	targetPriority := priorityForCommentCount(windowCount, j.parsedPriorityThresholds)
	if targetPriority <= currentPriority {
		logEntry(issueKey, "").Debugf("Priority %s is already high enough for %d comments in the last %d days", currentPriority, windowCount, j.priorityWindowDays)
		return nil
	}

	logEntry(issueKey, "").Infof("Auto-escalating priority from %s to %s (%d comments in the last %d days)", currentPriority, targetPriority, windowCount, j.priorityWindowDays)
	if j.dryRun {
		logEntry(issueKey, "").Debug("Dry run: would update priority")
		return nil
	}

	payload := &models.IssueScheme{Fields: &models.IssueFieldsScheme{Priority: &models.PriorityScheme{Name: targetPriority.String()}}}
	response, err = j.jiraClient.Issue.Update(ctx, issueKey, true, payload, nil, nil)
	if err != nil {
		logError(err, response)
		return fmt.Errorf("could not update priority for issue %s: %w", issueKey, err)
	}
	logEntry(issueKey, "").Infof("Updated priority to %s", targetPriority)
	return nil
}
