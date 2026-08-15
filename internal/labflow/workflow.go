package labflow

import (
	"fmt"
	"sync"
)

// Workflow defines allowed states and transitions for a lab process family.
type Workflow struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	States      []Status            `json:"states"`
	Transitions map[Status][]Status `json:"transitions"`
}

func (w *Workflow) CanTransition(from, to Status) bool {
	if w == nil {
		return false
	}
	for _, s := range w.Transitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

func (w *Workflow) Transition(from, to Status) (Status, error) {
	if !w.CanTransition(from, to) {
		id := "unknown"
		if w != nil {
			id = w.ID
		}
		return from, fmt.Errorf("labflow: workflow %s rejects %s → %s", id, from, to)
	}
	return to, nil
}

// Registry holds named workflows.
type Registry struct {
	mu   sync.RWMutex
	byID map[string]*Workflow
}

func NewRegistry() *Registry {
	r := &Registry{byID: make(map[string]*Workflow)}
	r.Register(DefaultWorkflow())
	r.Register(WaterTestingWorkflow())
	r.Register(ClinicalWorkflow())
	return r
}

func (r *Registry) Register(w *Workflow) {
	if w == nil || w.ID == "" {
		return
	}
	r.mu.Lock()
	r.byID[w.ID] = w
	r.mu.Unlock()
}

func (r *Registry) Get(id string) (*Workflow, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if id == "" {
		id = "default"
	}
	w, ok := r.byID[id]
	return w, ok
}

func (r *Registry) List() []*Workflow {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Workflow, 0, len(r.byID))
	for _, w := range r.byID {
		out = append(out, w)
	}
	return out
}

// DefaultWorkflow is the MVP LabFlow state machine.
func DefaultWorkflow() *Workflow {
	return &Workflow{
		ID:          "default",
		Name:        "LabFlow default",
		Description: "Generic laboratory sample lifecycle",
		States: []Status{
			StatusReceived, StatusAssigned, StatusInProgress, StatusQCReview,
			StatusReleased, StatusFlagged, StatusArchived,
		},
		Transitions: map[Status][]Status{
			StatusReceived:   {StatusAssigned},
			StatusAssigned:   {StatusInProgress},
			StatusInProgress: {StatusQCReview, StatusFlagged},
			StatusQCReview:   {StatusReleased, StatusFlagged},
			StatusReleased:   {StatusArchived},
			StatusFlagged:    {StatusInProgress, StatusQCReview, StatusArchived},
			StatusArchived:   {},
		},
	}
}

// WaterTestingWorkflow — example vertical without rewriting the core.
func WaterTestingWorkflow() *Workflow {
	return &Workflow{
		ID:          "water-testing",
		Name:        "Water testing",
		Description: "Environmental / water quality samples",
		States: []Status{
			StatusReceived, StatusAssigned, StatusInProgress, StatusQCReview,
			StatusReleased, StatusFlagged, StatusArchived,
		},
		Transitions: map[Status][]Status{
			StatusReceived:   {StatusAssigned},
			StatusAssigned:   {StatusInProgress},
			StatusInProgress: {StatusQCReview, StatusFlagged},
			StatusQCReview:   {StatusReleased, StatusFlagged, StatusInProgress}, // retest path
			StatusReleased:   {StatusArchived},
			StatusFlagged:    {StatusInProgress, StatusArchived},
			StatusArchived:   {},
		},
	}
}

// ClinicalWorkflow — stricter path (no skip back from QC to process without flag).
func ClinicalWorkflow() *Workflow {
	return &Workflow{
		ID:          "clinical",
		Name:        "Clinical samples",
		Description: "Stricter clinical sample handling",
		States: []Status{
			StatusReceived, StatusAssigned, StatusInProgress, StatusQCReview,
			StatusReleased, StatusFlagged, StatusArchived,
		},
		Transitions: map[Status][]Status{
			StatusReceived:   {StatusAssigned},
			StatusAssigned:   {StatusInProgress},
			StatusInProgress: {StatusQCReview, StatusFlagged},
			StatusQCReview:   {StatusReleased, StatusFlagged},
			StatusReleased:   {StatusArchived},
			StatusFlagged:    {StatusArchived}, // clinical: flagged samples archive only
			StatusArchived:   {},
		},
	}
}
