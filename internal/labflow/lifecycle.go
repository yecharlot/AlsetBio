package labflow

import "fmt"

// allowedTransitions is the legacy default map (kept for tests / fallback).
var allowedTransitions = map[Status][]Status{
	StatusReceived:   {StatusAssigned},
	StatusAssigned:   {StatusInProgress},
	StatusInProgress: {StatusQCReview, StatusFlagged},
	StatusQCReview:   {StatusReleased, StatusFlagged},
	StatusReleased:   {StatusArchived},
	StatusFlagged:    {StatusInProgress, StatusQCReview, StatusArchived},
	StatusArchived:   {},
}

// CanTransition reports whether from → to is allowed on the default machine.
func CanTransition(from, to Status) bool {
	for _, s := range allowedTransitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

// Transition validates and returns the new status on the default machine.
func Transition(from, to Status) (Status, error) {
	if !CanTransition(from, to) {
		return from, fmt.Errorf("labflow: invalid transition %s → %s", from, to)
	}
	return to, nil
}

// TransitionWithWorkflow uses a specific workflow definition.
func TransitionWithWorkflow(w *Workflow, from, to Status) (Status, error) {
	if w == nil {
		return Transition(from, to)
	}
	return w.Transition(from, to)
}
