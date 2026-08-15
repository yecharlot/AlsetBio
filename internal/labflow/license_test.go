package labflow

import "testing"

func TestLicenseLimits(t *testing.T) {
	dir := t.TempDir()
	s := NewLicenseStore(dir)
	st := s.Status(0)
	if st.Active {
		t.Fatal("expected inactive trial")
	}
	if _, err := s.Activate("TEST-KEY-123", PlanPilot, "lab-1", "Demo Lab", 30); err != nil {
		t.Fatal(err)
	}
	st = s.Status(10)
	if !st.Active || st.Plan != PlanPilot {
		t.Fatalf("%+v", st)
	}
	if err := s.AllowCreate(499); err != nil {
		t.Fatal(err)
	}
	if err := s.AllowCreate(500); err == nil {
		t.Fatal("expected limit")
	}
}
