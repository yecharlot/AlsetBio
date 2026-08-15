package labflow

import "fmt"

// allowedTransitions defines the MVP state machine.
var allowedTransitions = map[Status][]Status{
	StatusReceived:   {StatusAssigned},
	StatusAssigned:   {StatusInProgress},
	StatusInProgress: {StatusQCReview, StatusFlagged},
	StatusQCReview:   {StatusReleased, StatusFlagged},
	StatusReleased:   {StatusArchived},
	StatusFlagged:    {StatusInProgress, StatusQCReview, StatusArchived},
	StatusArchived:   {},
}

// CanTransition reports whether from → to is allowed.
func CanTransition(from, to Status) bool {
	for _, s := range allowedTransitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

// Transition validates and returns the new status.
func Transition(from, to Status) (Status, error) {
	if !CanTransition(from, to) {
		return from, fmt.Errorf("labflow: invalid transition %s → %s", from, to)
	}
	return to, nil
}
