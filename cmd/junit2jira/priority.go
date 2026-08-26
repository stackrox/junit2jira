package main

var defaultPriorityThresholds = []int{4, 16, 64, 128, 256}

// calculatePriority maps comment count to JIRA priority name using default thresholds
func calculatePriority(commentCount int) string {
	return calculatePriorityWithThresholds(commentCount, defaultPriorityThresholds)
}

// calculatePriorityWithThresholds maps comment count to JIRA priority name using custom thresholds
// thresholds should contain exactly 5 values for: Minor, Normal, Major, Blocker, Critical
func calculatePriorityWithThresholds(commentCount int, thresholds []int) string {
	if len(thresholds) != 5 {
		thresholds = defaultPriorityThresholds
	}

	switch {
	case commentCount >= thresholds[4]:
		return "Critical"
	case commentCount >= thresholds[3]:
		return "Blocker"
	case commentCount >= thresholds[2]:
		return "Major"
	case commentCount >= thresholds[1]:
		return "Normal"
	case commentCount >= thresholds[0]:
		return "Minor"
	default:
		return "Undefined"
	}
}
