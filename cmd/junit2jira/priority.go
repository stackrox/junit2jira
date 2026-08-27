package main

type Priority int

const (
	Undefined Priority = iota
	Minor
	Normal
	Major
	Blocker
	Critical
)

func (p Priority) String() string {
	switch p {
	case Critical:
		return "Critical"
	case Blocker:
		return "Blocker"
	case Major:
		return "Major"
	case Normal:
		return "Normal"
	case Minor:
		return "Minor"
	default:
		return "Undefined"
	}
}

func parsePriority(name string) (Priority, bool) {
	switch name {
	case "Critical":
		return Critical, true
	case "Blocker":
		return Blocker, true
	case "Major":
		return Major, true
	case "Normal":
		return Normal, true
	case "Minor":
		return Minor, true
	case "Undefined":
		return Undefined, true
	default:
		return Undefined, false
	}
}

var defaultPriorityThresholds = []int{2, 10, 50, 100, 200}

const defaultPriorityThresholdsStr = "2,10,50,100,200"

// calculatePriority maps comment count to JIRA priority using default thresholds
func calculatePriority(commentCount int) Priority {
	return calculatePriorityWithThresholds(commentCount, defaultPriorityThresholds)
}

// calculatePriorityWithThresholds maps comment count to JIRA priority using custom thresholds
// thresholds should contain exactly 5 non-negative, strictly ascending values for: Minor, Normal, Major, Blocker, Critical
func calculatePriorityWithThresholds(commentCount int, thresholds []int) Priority {
	// Validate thresholds: must have exactly 5 values, all non-negative, and strictly ascending
	if len(thresholds) != 5 {
		thresholds = defaultPriorityThresholds
	} else {
		valid := true
		for i := 0; i < 5; i++ {
			if thresholds[i] < 0 {
				valid = false
				break
			}
			if i > 0 && thresholds[i] <= thresholds[i-1] {
				valid = false
				break
			}
		}
		if !valid {
			thresholds = defaultPriorityThresholds
		}
	}

	switch {
	case commentCount >= thresholds[4]:
		return Critical
	case commentCount >= thresholds[3]:
		return Blocker
	case commentCount >= thresholds[2]:
		return Major
	case commentCount >= thresholds[1]:
		return Normal
	case commentCount >= thresholds[0]:
		return Minor
	default:
		return Undefined
	}
}

// maxPriority returns the higher of two priorities
func maxPriority(a, b Priority) Priority {
	if a > b {
		return a
	}
	return b
}

// calculatePriorityWithTimeEscalation applies layered escalation based on total comments
// and recent activity (last 30 days and last 10 days).
// This catches both persistent issues (high total) and hot issues (high recent activity).
func calculatePriorityWithTimeEscalation(totalComments, last30Days, last10Days int, thresholds []int) Priority {
	// Step 1: Calculate base priority from total comments
	basePriority := calculatePriorityWithThresholds(totalComments, thresholds)

	// Step 2: Check last 10 days for HOT issues (immediate escalation)
	if last10Days >= 10 {
		return Critical // Top 11% - extremely hot
	}
	if last10Days >= 5 {
		return maxPriority(basePriority, Major) // Top 17% - very active
	}

	// Step 3: Check last 30 days for active trends
	if last30Days >= 50 {
		return maxPriority(basePriority, Critical) // Top 7% - sustained high
	}
	if last30Days >= 20 {
		return maxPriority(basePriority, Blocker) // Top 13% - very active
	}
	if last30Days >= 10 {
		return maxPriority(basePriority, Major) // Top 17% - active
	}
	if last30Days >= 5 {
		return maxPriority(basePriority, Normal) // Top 28% - noticeable
	}

	// Step 4: Fall back to base priority
	return basePriority
}
