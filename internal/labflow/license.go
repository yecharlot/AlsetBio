package labflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Plan identifiers for commercial packaging.
const (
	PlanTrial   = "trial"
	PlanPilot   = "pilot"   // entry paid tier
	PlanLab     = "lab"     // standard lab
	PlanNetwork = "network" // multi-site
)

// License holds activation state for an installation / org.
type License struct {
	Key       string    `json:"key"`
	Plan      string    `json:"plan"`
	OrgID     string    `json:"org_id"`
	OrgName   string    `json:"org_name,omitempty"`
	MaxSamples int      `json:"max_samples"` // 0 = unlimited
	ActivatedAt time.Time `json:"activated_at"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	Active    bool      `json:"active"`
}

// LicenseStatus is the public commercial status payload.
type LicenseStatus struct {
	Active     bool   `json:"active"`
	Plan       string `json:"plan"`
	OrgID      string `json:"org_id,omitempty"`
	OrgName    string `json:"org_name,omitempty"`
	MaxSamples int    `json:"max_samples"`
	UsedSamples int   `json:"used_samples"`
	ExpiresAt  string `json:"expires_at,omitempty"`
	Message    string `json:"message,omitempty"`
	Sellable   bool   `json:"sellable"`
}

type LicenseStore struct {
	mu   sync.RWMutex
	path string
	lic  *License
}

func NewLicenseStore(dataDir string) *LicenseStore {
	if dataDir == "" {
		dataDir = "alset_data"
	}
	_ = os.MkdirAll(dataDir, 0755)
	s := &LicenseStore{path: filepath.Join(dataDir, "labflow_license.json")}
	s.load()
	// Env bootstrap: LABFLOW_LICENSE_KEY + LABFLOW_PLAN
	if s.lic == nil || !s.lic.Active {
		if key := strings.TrimSpace(os.Getenv("LABFLOW_LICENSE_KEY")); key != "" {
			plan := strings.TrimSpace(os.Getenv("LABFLOW_PLAN"))
			if plan == "" {
				plan = PlanPilot
			}
			org := strings.TrimSpace(os.Getenv("LABFLOW_ORG_ID"))
			if org == "" {
				org = "licensed-org"
			}
			_, _ = s.Activate(key, plan, org, os.Getenv("LABFLOW_ORG_NAME"), 0)
		}
	}
	return s
}

func (s *LicenseStore) load() {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var lic License
	if json.Unmarshal(b, &lic) == nil {
		s.lic = &lic
	}
}

func (s *LicenseStore) save() error {
	if s.lic == nil {
		return nil
	}
	b, err := json.MarshalIndent(s.lic, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0644)
}

func planLimits(plan string) (maxSamples int, days int) {
	switch strings.ToLower(plan) {
	case PlanTrial:
		return 50, 14
	case PlanPilot:
		return 500, 365
	case PlanLab:
		return 5000, 365
	case PlanNetwork:
		return 0, 365 // unlimited
	default:
		return 50, 14
	}
}

// NormalizeLicenseKey creates a stable display key from raw input.
func NormalizeLicenseKey(raw string) string {
	raw = strings.TrimSpace(strings.ToUpper(raw))
	raw = strings.ReplaceAll(raw, " ", "")
	if raw == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(raw))
	h := hex.EncodeToString(sum[:8])
	return "LF-" + strings.ToUpper(h[:4]) + "-" + strings.ToUpper(h[4:8]) + "-" + strings.ToUpper(h[8:12])
}

func (s *LicenseStore) Activate(rawKey, plan, orgID, orgName string, durationDays int) (*License, error) {
	if strings.TrimSpace(rawKey) == "" {
		return nil, fmt.Errorf("license key required")
	}
	plan = strings.ToLower(strings.TrimSpace(plan))
	if plan == "" {
		plan = PlanPilot
	}
	max, defDays := planLimits(plan)
	if durationDays <= 0 {
		durationDays = defDays
	}
	if orgID == "" {
		orgID = "org-default"
	}
	now := time.Now().UTC()
	lic := &License{
		Key:         NormalizeLicenseKey(rawKey),
		Plan:        plan,
		OrgID:       orgID,
		OrgName:     orgName,
		MaxSamples:  max,
		ActivatedAt: now,
		ExpiresAt:   now.Add(time.Duration(durationDays) * 24 * time.Hour),
		Active:      true,
	}
	s.mu.Lock()
	s.lic = lic
	err := s.save()
	s.mu.Unlock()
	return lic, err
}

func (s *LicenseStore) Get() *License {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.lic == nil {
		return nil
	}
	cp := *s.lic
	return &cp
}

func (s *LicenseStore) Status(usedSamples int) LicenseStatus {
	lic := s.Get()
	if lic == nil || !lic.Active {
		return LicenseStatus{
			Active:      false,
			Plan:        PlanTrial,
			MaxSamples:  50,
			UsedSamples: usedSamples,
			Message:     "Trial mode — activate a license to lift limits and remove the trial banner.",
			Sellable:    true,
		}
	}
	if !lic.ExpiresAt.IsZero() && time.Now().UTC().After(lic.ExpiresAt) {
		return LicenseStatus{
			Active:      false,
			Plan:        lic.Plan,
			OrgID:       lic.OrgID,
			OrgName:     lic.OrgName,
			MaxSamples:  lic.MaxSamples,
			UsedSamples: usedSamples,
			ExpiresAt:   lic.ExpiresAt.Format(time.RFC3339),
			Message:     "License expired — renew to continue creating samples.",
			Sellable:    true,
		}
	}
	st := LicenseStatus{
		Active:      true,
		Plan:        lic.Plan,
		OrgID:       lic.OrgID,
		OrgName:     lic.OrgName,
		MaxSamples:  lic.MaxSamples,
		UsedSamples: usedSamples,
		Sellable:    true,
		Message:     "License active",
	}
	if !lic.ExpiresAt.IsZero() {
		st.ExpiresAt = lic.ExpiresAt.Format(time.RFC3339)
	}
	return st
}

func (s *LicenseStore) AllowCreate(usedSamples int) error {
	st := s.Status(usedSamples)
	if !st.Active && st.Plan != PlanTrial {
		return fmt.Errorf("license inactive or expired")
	}
	// Trial allows up to max
	max := st.MaxSamples
	if max > 0 && usedSamples >= max {
		return fmt.Errorf("sample limit reached for plan %s (%d) — upgrade license", st.Plan, max)
	}
	return nil
}
