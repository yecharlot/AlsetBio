package labflow

import (
	"fmt"
	"strings"
)

// Laboratory roles (MVP).
const (
	RoleAdmin      = "LAB_ADMIN"
	RoleManager    = "LAB_MANAGER"
	RoleTechnician = "TECHNICIAN"
	RoleReviewer   = "REVIEWER"
	RoleClient     = "CLIENT"
)

// Principal is the authenticated caller for LabFlow.
type Principal struct {
	AgentID string
	Roles   []string
	OrgID   string // optional org scope from token/metadata
}

func (p Principal) HasRole(role string) bool {
	role = strings.ToUpper(role)
	for _, r := range p.Roles {
		if strings.ToUpper(r) == role || strings.ToUpper(r) == "ADMIN" || strings.ToUpper(r) == "LAB_ADMIN" {
			if role == RoleAdmin || strings.ToUpper(r) == role {
				return true
			}
		}
		if strings.ToUpper(r) == "ADMIN" {
			return true // global admin
		}
	}
	// LAB_ADMIN matches admin-style
	for _, r := range p.Roles {
		u := strings.ToUpper(r)
		if u == "LAB_ADMIN" || u == "ADMIN" {
			if role == RoleAdmin {
				return true
			}
			return true // admin can do anything checked via Can*
		}
	}
	return false
}

func (p Principal) isAdmin() bool {
	for _, r := range p.Roles {
		u := strings.ToUpper(r)
		if u == "LAB_ADMIN" || u == "ADMIN" || u == "LAB_MANAGER" && false {
			return u == "LAB_ADMIN" || u == "ADMIN"
		}
	}
	return false
}

func (p Principal) IsAdmin() bool {
	for _, r := range p.Roles {
		u := strings.ToUpper(r)
		if u == RoleAdmin || u == "ADMIN" || u == "LAB_ADMIN" {
			return true
		}
	}
	return false
}

func (p Principal) IsManager() bool {
	if p.IsAdmin() {
		return true
	}
	for _, r := range p.Roles {
		if strings.ToUpper(r) == RoleManager {
			return true
		}
	}
	return false
}

func (p Principal) IsTechnician() bool {
	if p.IsManager() {
		return true
	}
	for _, r := range p.Roles {
		if strings.ToUpper(r) == RoleTechnician {
			return true
		}
	}
	return false
}

func (p Principal) IsReviewer() bool {
	if p.IsManager() {
		return true
	}
	for _, r := range p.Roles {
		if strings.ToUpper(r) == RoleReviewer {
			return true
		}
	}
	return false
}

func (p Principal) IsClient() bool {
	for _, r := range p.Roles {
		if strings.ToUpper(r) == RoleClient {
			return true
		}
	}
	return false
}

// Permission checks for LabFlow operations.
func (p Principal) CanCreateSample() bool {
	return p.IsTechnician() || p.IsManager() || p.IsAdmin()
}

func (p Principal) CanTransition(to Status) bool {
	if p.IsAdmin() || p.IsManager() {
		return true
	}
	if p.IsTechnician() {
		switch to {
		case StatusAssigned, StatusInProgress, StatusFlagged:
			return true
		default:
			return false
		}
	}
	if p.IsReviewer() {
		switch to {
		case StatusQCReview, StatusReleased, StatusFlagged, StatusArchived:
			return true
		default:
			return false
		}
	}
	return false
}

func (p Principal) CanListAllOrgs() bool {
	return p.IsAdmin() || p.IsManager()
}

func (p Principal) CanViewSample(sample *Sample) bool {
	if sample == nil {
		return false
	}
	if p.IsAdmin() || p.IsManager() || p.IsTechnician() || p.IsReviewer() {
		if p.OrgID == "" || p.IsAdmin() {
			return true
		}
		return sample.OrgID == p.OrgID
	}
	if p.IsClient() {
		// client sees only own client_id or org if matched via metadata
		if p.AgentID != "" && sample.ClientID == p.AgentID {
			return true
		}
		if p.OrgID != "" && sample.ClientID == p.OrgID {
			return true
		}
		return false
	}
	return false
}

// PublicVerifyAllowed is always true for the verify endpoint (non-sensitive).
func PublicVerifyAllowed() bool { return true }

func NormalizeRoles(roles []string) []string {
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		out = append(out, strings.ToUpper(r))
	}
	return out
}

func Require(cond bool, msg string) error {
	if !cond {
		return fmt.Errorf("%s", msg)
	}
	return nil
}
