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

func parsePriority(name string) Priority {
	switch name {
	case "Critical":
		return Critical
	case "Blocker":
		return Blocker
	case "Major":
		return Major
	case "Normal":
		return Normal
	case "Minor":
		return Minor
	default:
		return Undefined
	}
}

var defaultPriorityThresholds = []int{4, 16, 64, 128, 256}

// calculatePriority maps comment count to JIRA priority using default thresholds
func calculatePriority(commentCount int) Priority {
	return calculatePriorityWithThresholds(commentCount, defaultPriorityThresholds)
}

// calculatePriorityWithThresholds maps comment count to JIRA priority using custom thresholds
// thresholds should contain exactly 5 values for: Minor, Normal, Major, Blocker, Critical
func calculatePriorityWithThresholds(commentCount int, thresholds []int) Priority {
	if len(thresholds) != 5 {
		thresholds = defaultPriorityThresholds
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
