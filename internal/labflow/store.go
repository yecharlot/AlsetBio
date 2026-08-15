package labflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// BlockBackend is the IPFS-like content-addressed store provided by the Alset node.
type BlockBackend interface {
	Put(data []byte) (cid string, error error)
	Get(cid string) ([]byte, error)
}

// indexDoc is itself stored as an IPFS block; its CID is the LabFlow root.
type indexDoc struct {
	Version   int               `json:"version"`
	UpdatedAt time.Time         `json:"updated_at"`
	Samples   map[string]string `json:"samples"` // sampleID -> sample block CID
	Events    map[string][]string `json:"events"` // sampleID -> event block CIDs (ordered)
}

// Store persists samples and custody events as content-addressed IPFS blocks.
type Store struct {
	mu      sync.RWMutex
	backend BlockBackend
	index   indexDoc
	rootCID string
	metaPath string // local pointer to root CID so restarts recover the index
}

func NewStore(backend BlockBackend, dataDir string) *Store {
	if dataDir == "" {
		dataDir = "alset_data"
	}
	s := &Store{
		backend:  backend,
		metaPath: filepath.Join(dataDir, "labflow_root.cid"),
		index: indexDoc{
			Version: 1,
			Samples: make(map[string]string),
			Events:  make(map[string][]string),
		},
	}
	_ = os.MkdirAll(dataDir, 0755)
	s.loadRoot()
	return s
}

func (s *Store) RootCID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rootCID
}

func (s *Store) loadRoot() {
	b, err := os.ReadFile(s.metaPath)
	if err != nil || len(b) == 0 {
		return
	}
	cid := string(b)
	data, err := s.backend.Get(cid)
	if err != nil {
		return
	}
	var idx indexDoc
	if json.Unmarshal(data, &idx) != nil {
		return
	}
	if idx.Samples == nil {
		idx.Samples = make(map[string]string)
	}
	if idx.Events == nil {
		idx.Events = make(map[string][]string)
	}
	s.index = idx
	s.rootCID = cid
}

func (s *Store) commitIndex() error {
	s.index.UpdatedAt = time.Now().UTC()
	raw, err := json.Marshal(s.index)
	if err != nil {
		return err
	}
	cid, err := s.backend.Put(raw)
	if err != nil {
		return err
	}
	s.rootCID = cid
	return os.WriteFile(s.metaPath, []byte(cid), 0644)
}

// PutSample stores the sample JSON as an IPFS block and updates the root index.
func (s *Store) PutSample(sample *Sample) (sampleCID string, rootCID string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := json.Marshal(sample)
	if err != nil {
		return "", "", err
	}
	cid, err := s.backend.Put(raw)
	if err != nil {
		return "", "", err
	}
	s.index.Samples[sample.ID] = cid
	if err := s.commitIndex(); err != nil {
		return cid, "", err
	}
	return cid, s.rootCID, nil
}

// AppendEvent stores an event block and links it in the index.
func (s *Store) AppendEvent(ev *CustodyEvent) (eventCID string, rootCID string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := json.Marshal(ev)
	if err != nil {
		return "", "", err
	}
	cid, err := s.backend.Put(raw)
	if err != nil {
		return "", "", err
	}
	s.index.Events[ev.SampleID] = append(s.index.Events[ev.SampleID], cid)
	if err := s.commitIndex(); err != nil {
		return cid, "", err
	}
	return cid, s.rootCID, nil
}

func (s *Store) GetSample(id string) (*Sample, string, error) {
	s.mu.RLock()
	cid, ok := s.index.Samples[id]
	s.mu.RUnlock()
	if !ok {
		return nil, "", fmt.Errorf("sample not found: %s", id)
	}
	data, err := s.backend.Get(cid)
	if err != nil {
		return nil, cid, err
	}
	var sample Sample
	if err := json.Unmarshal(data, &sample); err != nil {
		return nil, cid, err
	}
	return &sample, cid, nil
}

func (s *Store) ListSamples() ([]Sample, error) {
	s.mu.RLock()
	ids := make([]string, 0, len(s.index.Samples))
	for id := range s.index.Samples {
		ids = append(ids, id)
	}
	s.mu.RUnlock()
	out := make([]Sample, 0, len(ids))
	for _, id := range ids {
		sample, _, err := s.GetSample(id)
		if err != nil {
			continue
		}
		out = append(out, *sample)
	}
	return out, nil
}

func (s *Store) ListEvents(sampleID string) ([]CustodyEvent, error) {
	s.mu.RLock()
	cids := append([]string(nil), s.index.Events[sampleID]...)
	s.mu.RUnlock()
	out := make([]CustodyEvent, 0, len(cids))
	for _, cid := range cids {
		data, err := s.backend.Get(cid)
		if err != nil {
			continue
		}
		var ev CustodyEvent
		if json.Unmarshal(data, &ev) != nil {
			continue
		}
		out = append(out, ev)
	}
	return out, nil
}
