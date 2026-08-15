package labflow

import "testing"

func TestValidTransitions(t *testing.T) {
	steps := []struct{ from, to Status }{
		{StatusReceived, StatusAssigned},
		{StatusAssigned, StatusInProgress},
		{StatusInProgress, StatusQCReview},
		{StatusQCReview, StatusReleased},
		{StatusReleased, StatusArchived},
	}
	for _, s := range steps {
		if _, err := Transition(s.from, s.to); err != nil {
			t.Fatalf("%s → %s: %v", s.from, s.to, err)
		}
	}
}

func TestInvalidTransition(t *testing.T) {
	if _, err := Transition(StatusReceived, StatusReleased); err == nil {
		t.Fatal("expected error for RECEIVED → RELEASED")
	}
}

func TestFlagFromInProgress(t *testing.T) {
	if _, err := Transition(StatusInProgress, StatusFlagged); err != nil {
		t.Fatal(err)
	}
}
