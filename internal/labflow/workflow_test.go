package labflow

import "testing"

func TestClinicalFlaggedCannotRestart(t *testing.T) {
	w := ClinicalWorkflow()
	if w.CanTransition(StatusFlagged, StatusInProgress) {
		t.Fatal("clinical should not allow FLAGGED → IN_PROGRESS")
	}
	if !DefaultWorkflow().CanTransition(StatusFlagged, StatusInProgress) {
		t.Fatal("default should allow FLAGGED → IN_PROGRESS")
	}
}

func TestServiceUsesWorkflow(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(NewStore(&memBackend{}, dir))
	res, err := svc.Create(CreateInput{Type: "clinical", WorkflowID: "clinical", Actor: "t"})
	if err != nil {
		t.Fatal(err)
	}
	id := res.Sample.ID
	for _, st := range []Status{StatusAssigned, StatusInProgress, StatusFlagged} {
		if _, _, _, err := svc.Transition(id, TransitionInput{ToStatus: st, Actor: "t"}); err != nil {
			t.Fatalf("%s: %v", st, err)
		}
	}
	if _, _, _, err := svc.Transition(id, TransitionInput{ToStatus: StatusInProgress, Actor: "t"}); err == nil {
		t.Fatal("expected clinical to reject FLAGGED → IN_PROGRESS")
	}
	if _, _, _, err := svc.Transition(id, TransitionInput{ToStatus: StatusArchived, Actor: "t"}); err != nil {
		t.Fatal(err)
	}
}

func TestWaterRetestPath(t *testing.T) {
	w := WaterTestingWorkflow()
	if !w.CanTransition(StatusQCReview, StatusInProgress) {
		t.Fatal("water should allow retest QC → IN_PROGRESS")
	}
}
