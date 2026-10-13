package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
)

// parsePriorityThresholds parses minimum comment counts for Minor, Normal, Major,
// Blocker, and Critical. Strict ordering gives each priority a distinct count range.
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

	priority, err := j.fetchIssuePriority(ctx, issueKey)
	if err != nil {
		return err
	}

	comments, err := j.fetchIssueComments(ctx, issueKey)
	if err != nil {
		return err
	}

	windowCount := countCommentsInWindow(comments, j.priorityWindowDays, time.Now(), func(index int, timestamp string) {
		logEntry(issueKey, "").Warnf("Skipping comment %d with unparseable timestamp %q", index, timestamp)
	})

	return j.escalateIssuePriority(ctx, issueKey, priority, windowCount)
}

func (j junit2jira) fetchIssuePriority(ctx context.Context, issueKey string) (*models.PriorityScheme, error) {
	issue, response, err := j.jiraClient.Issue.Get(ctx, issueKey, []string{"priority"}, nil)
	if err != nil {
		logError(err, response)
		return nil, fmt.Errorf("could not fetch issue %s: %w", issueKey, err)
	}
	if issue == nil || issue.Fields == nil {
		return nil, nil
	}
	return issue.Fields.Priority, nil
}

func (j junit2jira) escalateIssuePriority(ctx context.Context, issueKey string, priority *models.PriorityScheme, windowCount int) error {
	currentPriority := Undefined
	if priority != nil {
		var recognized bool
		currentPriority, recognized = parsePriority(priority.Name)
		if !recognized {
			logEntry(issueKey, "").Warnf("Unrecognized priority %q, skipping auto-escalation", priority.Name)
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

	return j.setIssuePriority(ctx, issueKey, targetPriority)
}

func (j junit2jira) fetchIssueComments(ctx context.Context, issueKey string) ([]*models.IssueCommentScheme, error) {
	// Match Jira Cloud's documented default comment page size; the server may return fewer comments.
	// https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issue-comments/#api-rest-api-3-issue-issueidorkey-comment-get
	const pageSize = 100
	var comments []*models.IssueCommentScheme
	// Use the comment endpoint's totals; the issue's embedded count can be stale.
	for startAt := 0; ; {
		page, response, err := j.jiraClient.Issue.Comment.Gets(ctx, issueKey, "created", nil, startAt, pageSize)
		if err != nil {
			logError(err, response)
			return nil, fmt.Errorf("could not fetch comments for issue %s: %w", issueKey, err)
		}
		if page == nil || len(page.Comments) == 0 {
			break
		}
		comments = append(comments, page.Comments...)
		startAt += len(page.Comments)
		if startAt >= page.Total {
			break
		}
	}

	return comments, nil
}

func (j junit2jira) setIssuePriority(ctx context.Context, issueKey string, priority Priority) error {
	payload := &models.IssueScheme{Fields: &models.IssueFieldsScheme{Priority: &models.PriorityScheme{Name: priority.String()}}}
	response, err := j.jiraClient.Issue.Update(ctx, issueKey, true, payload, nil, nil)
	if err != nil {
		logError(err, response)
		return fmt.Errorf("could not update priority for issue %s: %w", issueKey, err)
	}
	logEntry(issueKey, "").Infof("Updated priority to %s", priority)
	return nil
}
