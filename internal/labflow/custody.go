package labflow

import "time"

// EventType for chain-of-custody entries.
type EventType string

const (
	EventCreated             EventType = "SAMPLE_CREATED"
	EventReceived            EventType = "SAMPLE_RECEIVED"
	EventAssigned            EventType = "SAMPLE_ASSIGNED"
	EventMoved               EventType = "SAMPLE_MOVED"
	EventProcessingStarted   EventType = "SAMPLE_PROCESSING_STARTED"
	EventProcessingCompleted EventType = "SAMPLE_PROCESSING_COMPLETED"
	EventQCStarted           EventType = "SAMPLE_QC_STARTED"
	EventReleased            EventType = "SAMPLE_RELEASED"
	EventFlagged             EventType = "SAMPLE_FLAGGED"
)

// CustodyEvent is an append-only audit record. Events are never deleted;
// corrections are new events.
type CustodyEvent struct {
	ID           string            `json:"id"`
	SampleID     string            `json:"sample_id"`
	Type         EventType         `json:"type"`
	Actor        string            `json:"actor"`
	Timestamp    time.Time         `json:"timestamp"`
	PrevStatus   Status            `json:"previous_state,omitempty"`
	NewStatus    Status            `json:"new_state,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	EvidenceRef  string            `json:"evidence_ref,omitempty"`
}
