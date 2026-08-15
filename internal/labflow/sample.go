package labflow

import "time"

// Status of a laboratory sample (MVP state machine).
type Status string

const (
	StatusReceived   Status = "RECEIVED"
	StatusAssigned   Status = "ASSIGNED"
	StatusInProgress Status = "IN_PROGRESS"
	StatusQCReview   Status = "QC_REVIEW"
	StatusReleased   Status = "RELEASED"
	StatusFlagged    Status = "FLAGGED"
	StatusArchived   Status = "ARCHIVED"
)

// Sample is the core laboratory sample entity.
type Sample struct {
	ID             string            `json:"id"`
	ExternalID     string            `json:"external_id"`
	Type           string            `json:"type"`
	Status         Status            `json:"status"`
	OrgID          string            `json:"org_id"`
	ClientID       string            `json:"client_id,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	ReceivedAt     *time.Time        `json:"received_at,omitempty"`
	CurrentLocation string           `json:"current_location,omitempty"`
	CurrentOwner   string            `json:"current_owner,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	EvidenceCID    string            `json:"evidence_cid,omitempty"`
	RootCID        string            `json:"root_cid,omitempty"`
}
