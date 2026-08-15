package labflow

import "testing"

func TestRolePermissions(t *testing.T) {
	tech := Principal{Roles: []string{RoleTechnician}, OrgID: "lab-1"}
	if !tech.CanCreateSample() {
		t.Fatal("tech should create")
	}
	if !tech.CanTransition(StatusAssigned) {
		t.Fatal("tech assign")
	}
	if tech.CanTransition(StatusReleased) {
		t.Fatal("tech should not release")
	}
	rev := Principal{Roles: []string{RoleReviewer}}
	if !rev.CanTransition(StatusReleased) {
		t.Fatal("reviewer release")
	}
	client := Principal{Roles: []string{RoleClient}, AgentID: "c1"}
	if client.CanCreateSample() {
		t.Fatal("client no create")
	}
	s := &Sample{OrgID: "lab-1", ClientID: "c1"}
	if !client.CanViewSample(s) {
		t.Fatal("client view own")
	}
	s2 := &Sample{OrgID: "lab-1", ClientID: "other"}
	if client.CanViewSample(s2) {
		t.Fatal("client not view other")
	}
}
