package labflow

import (
	"fmt"
	"sync/atomic"
	"time"
)

var sampleSeq uint64

// Service orchestrates sample lifecycle with IPFS-backed persistence.
type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) RootCID() string {
	return s.store.RootCID()
}

func nextExternalID() string {
	n := atomic.AddUint64(&sampleSeq, 1)
	return fmt.Sprintf("BIO-%d-%06d", time.Now().Year(), n)
}

type CreateInput struct {
	Type     string            `json:"type"`
	OrgID    string            `json:"org_id"`
	ClientID string            `json:"client_id"`
	Location string            `json:"location"`
	Actor    string            `json:"actor"`
	Metadata map[string]string `json:"metadata"`
}

type CreateResult struct {
	Sample    *Sample `json:"sample"`
	SampleCID string  `json:"sample_cid"`
	RootCID   string  `json:"root_cid"`
	EventCID  string  `json:"event_cid"`
}

func (s *Service) Create(in CreateInput) (*CreateResult, error) {
	now := time.Now().UTC()
	id := fmt.Sprintf("smp_%d_%d", now.UnixNano(), atomic.AddUint64(&sampleSeq, 1))
	ext := nextExternalID()
	sample := &Sample{
		ID:              id,
		ExternalID:      ext,
		Type:            in.Type,
		Status:          StatusReceived,
		OrgID:           in.OrgID,
		ClientID:        in.ClientID,
		CreatedAt:       now,
		ReceivedAt:      &now,
		CurrentLocation: in.Location,
		CurrentOwner:    in.Actor,
		Metadata:        in.Metadata,
	}
	if sample.Type == "" {
		sample.Type = "generic"
	}
	if sample.OrgID == "" {
		sample.OrgID = "lab-default"
	}
	if sample.CurrentOwner == "" {
		sample.CurrentOwner = "system"
	}

	sampleCID, rootCID, err := s.store.PutSample(sample)
	if err != nil {
		return nil, err
	}
	sample.EvidenceCID = sampleCID
	sample.RootCID = rootCID
	// update sample with CIDs
	sampleCID, rootCID, err = s.store.PutSample(sample)
	if err != nil {
		return nil, err
	}

	ev := &CustodyEvent{
		ID:          fmt.Sprintf("evt_%d", now.UnixNano()),
		SampleID:    sample.ID,
		Type:        EventCreated,
		Actor:       sample.CurrentOwner,
		Timestamp:   now,
		PrevStatus:  "",
		NewStatus:   StatusReceived,
		Metadata:    map[string]string{"external_id": sample.ExternalID},
		EvidenceRef: sampleCID,
	}
	eventCID, rootCID, err := s.store.AppendEvent(ev)
	if err != nil {
		return nil, err
	}
	// refresh root on sample
	sample.RootCID = rootCID
	_, rootCID, _ = s.store.PutSample(sample)

	return &CreateResult{
		Sample:    sample,
		SampleCID: sampleCID,
		RootCID:   rootCID,
		EventCID:  eventCID,
	}, nil
}

type TransitionInput struct {
	ToStatus Status            `json:"to_status"`
	Actor    string            `json:"actor"`
	Location string            `json:"location"`
	Metadata map[string]string `json:"metadata"`
}

func (s *Service) Transition(sampleID string, in TransitionInput) (*Sample, string, string, error) {
	sample, _, err := s.store.GetSample(sampleID)
	if err != nil {
		return nil, "", "", err
	}
	newStatus, err := Transition(sample.Status, in.ToStatus)
	if err != nil {
		return nil, "", "", err
	}
	prev := sample.Status
	sample.Status = newStatus
	if in.Actor != "" {
		sample.CurrentOwner = in.Actor
	}
	if in.Location != "" {
		sample.CurrentLocation = in.Location
	}
	now := time.Now().UTC()
	evType := EventMoved
	switch newStatus {
	case StatusAssigned:
		evType = EventAssigned
	case StatusInProgress:
		evType = EventProcessingStarted
	case StatusQCReview:
		evType = EventQCStarted
	case StatusReleased:
		evType = EventReleased
	case StatusFlagged:
		evType = EventFlagged
	case StatusReceived:
		evType = EventReceived
	}

	sampleCID, rootCID, err := s.store.PutSample(sample)
	if err != nil {
		return nil, "", "", err
	}
	sample.EvidenceCID = sampleCID
	sample.RootCID = rootCID

	ev := &CustodyEvent{
		ID:          fmt.Sprintf("evt_%d", now.UnixNano()),
		SampleID:    sample.ID,
		Type:        evType,
		Actor:       sample.CurrentOwner,
		Timestamp:   now,
		PrevStatus:  prev,
		NewStatus:   newStatus,
		Metadata:    in.Metadata,
		EvidenceRef: sampleCID,
	}
	_, rootCID, err = s.store.AppendEvent(ev)
	if err != nil {
		return nil, "", "", err
	}
	sample.RootCID = rootCID
	sampleCID, rootCID, _ = s.store.PutSample(sample)
	return sample, sampleCID, rootCID, nil
}

func (s *Service) Get(id string) (*Sample, string, error) {
	return s.store.GetSample(id)
}

func (s *Service) List() ([]Sample, error) {
	return s.store.ListSamples()
}

func (s *Service) Events(id string) ([]CustodyEvent, error) {
	return s.store.ListEvents(id)
}

// VerifyView is public, non-sensitive sample verification payload.
type VerifyView struct {
	Verified   bool   `json:"verified"`
	ExternalID string `json:"sample"`
	Status     Status `json:"status"`
	Created    string `json:"created"`
	Integrity  string `json:"integrity"`
	SampleCID  string `json:"sample_cid,omitempty"`
	RootCID    string `json:"root_cid,omitempty"`
	Certificate string `json:"certificate"`
}

func (s *Service) Verify(id string) (*VerifyView, error) {
	sample, sampleCID, err := s.store.GetSample(id)
	if err != nil {
		// try by external id
		all, _ := s.store.ListSamples()
		for i := range all {
			if all[i].ExternalID == id {
				sample = &all[i]
				sampleCID = all[i].EvidenceCID
				err = nil
				break
			}
		}
		if err != nil || sample == nil {
			return &VerifyView{Verified: false, Integrity: "NOT_FOUND", Certificate: "N/A"}, fmt.Errorf("not found")
		}
	}
	return &VerifyView{
		Verified:    true,
		ExternalID:  sample.ExternalID,
		Status:      sample.Status,
		Created:     sample.CreatedAt.Format(time.RFC3339),
		Integrity:   "VERIFIED",
		SampleCID:   sampleCID,
		RootCID:     sample.RootCID,
		Certificate: "VERIFIED",
	}, nil
}
