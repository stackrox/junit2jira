package main

import (
	"strconv"
	"strings"
)

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

// defaultPriorityThresholds are minimum comment counts within the configured
// activity window for Minor, Normal, Major, Blocker, and Critical, respectively.
// Keep the defaults and count ranges in README.md in sync when changing these.
var defaultPriorityThresholds = []int{2, 10, 50, 100, 200}

func priorityThresholdsString(thresholds []int) string {
	values := make([]string, len(thresholds))
	for i, threshold := range thresholds {
		values[i] = strconv.Itoa(threshold)
	}
	return strings.Join(values, ",")
}

// priorityForCommentCount maps the selected activity-window count to a priority.
func priorityForCommentCount(commentCount int, thresholds []int) Priority {
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
