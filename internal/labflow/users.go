package labflow

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User is a multi-tenant LabFlow account bound to an organization.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"password_hash"`
	OrgID        string    `json:"org_id"`
	OrgName      string    `json:"org_name"`
	Role         string    `json:"role"` // LAB_ADMIN, LAB_MANAGER, TECHNICIAN, REVIEWER, CLIENT
	CreatedAt    time.Time `json:"created_at"`
	Active       bool      `json:"active"`
}

type userFile struct {
	Users []User `json:"users"`
}

// UserStore persists accounts on disk (one installation can host many orgs).
type UserStore struct {
	mu   sync.RWMutex
	path string
	users map[string]User // email lower -> user
}

func NewUserStore(dataDir string) *UserStore {
	if dataDir == "" {
		dataDir = "alset_data"
	}
	_ = os.MkdirAll(dataDir, 0755)
	s := &UserStore{
		path:  filepath.Join(dataDir, "labflow_users.json"),
		users: make(map[string]User),
	}
	s.load()
	return s
}

func (s *UserStore) load() {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var f userFile
	if json.Unmarshal(b, &f) != nil {
		return
	}
	for _, u := range f.Users {
		s.users[strings.ToLower(u.Email)] = u
	}
}

func (s *UserStore) save() error {
	list := make([]User, 0, len(s.users))
	for _, u := range s.users {
		list = append(list, u)
	}
	b, err := json.MarshalIndent(userFile{Users: list}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0600)
}

type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	OrgName  string `json:"org_name"`
	OrgID    string `json:"org_id"`
	Role     string `json:"role"`
}

func (s *UserStore) Register(in RegisterInput) (*User, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, fmt.Errorf("valid email required")
	}
	if len(in.Password) < 6 {
		return nil, fmt.Errorf("password must be at least 6 characters")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.users[email]; exists {
		return nil, fmt.Errorf("email already registered")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	orgName := strings.TrimSpace(in.OrgName)
	if orgName == "" {
		orgName = "Laboratory"
	}
	orgID := strings.TrimSpace(in.OrgID)
	if orgID == "" {
		orgID = slugOrg(orgName)
	}
	role := strings.ToUpper(strings.TrimSpace(in.Role))
	if role == "" {
		role = RoleAdmin // first user of an org is admin
	}
	id, _ := randomID("usr")
	u := User{
		ID:           id,
		Email:        email,
		Name:         strings.TrimSpace(in.Name),
		PasswordHash: string(hash),
		OrgID:        orgID,
		OrgName:      orgName,
		Role:         role,
		CreatedAt:    time.Now().UTC(),
		Active:       true,
	}
	if u.Name == "" {
		u.Name = strings.Split(email, "@")[0]
	}
	s.users[email] = u
	if err := s.save(); err != nil {
		return nil, err
	}
	out := u
	out.PasswordHash = ""
	return &out, nil
}

func (s *UserStore) Authenticate(email, password string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	s.mu.RLock()
	u, ok := s.users[email]
	s.mu.RUnlock()
	if !ok || !u.Active {
		return nil, fmt.Errorf("invalid email or password")
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, fmt.Errorf("invalid email or password")
	}
	out := u
	out.PasswordHash = ""
	return &out, nil
}

func (s *UserStore) GetByEmail(email string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		return nil, false
	}
	out := u
	out.PasswordHash = ""
	return &out, true
}

func (s *UserStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users)
}

func slugOrg(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			b.WriteByte('-')
		}
	}
	s := b.String()
	if s == "" {
		id, _ := randomID("org")
		return id
	}
	return "org-" + s
}

func randomID(prefix string) (string, error) {
	var b [6]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return prefix + "-x", err
	}
	return prefix + "-" + hex.EncodeToString(b[:]), nil
}

// Session is a lightweight server-side session tied to a LabFlow user.
type Session struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	OrgID     string    `json:"org_id"`
	OrgName   string    `json:"org_name"`
	Role      string    `json:"role"`
	Name      string    `json:"name"`
	ExpiresAt time.Time `json:"expires_at"`
}

type SessionStore struct {
	mu   sync.RWMutex
	path string
	byToken map[string]Session
}

func NewSessionStore(dataDir string) *SessionStore {
	if dataDir == "" {
		dataDir = "alset_data"
	}
	s := &SessionStore{
		path:    filepath.Join(dataDir, "labflow_sessions.json"),
		byToken: make(map[string]Session),
	}
	s.load()
	return s
}

func (s *SessionStore) load() {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var list []Session
	if json.Unmarshal(b, &list) != nil {
		return
	}
	now := time.Now().UTC()
	for _, sess := range list {
		if sess.ExpiresAt.After(now) {
			s.byToken[sess.Token] = sess
		}
	}
}

func (s *SessionStore) save() error {
	list := make([]Session, 0, len(s.byToken))
	for _, sess := range s.byToken {
		list = append(list, sess)
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0600)
}

func (s *SessionStore) Create(u *User, hours int) (*Session, error) {
	if hours <= 0 {
		hours = 72
	}
	tok, err := randomID("sess")
	if err != nil {
		return nil, err
	}
	// stronger token
	var extra [16]byte
	_, _ = rand.Read(extra[:])
	tok = hex.EncodeToString(extra[:])
	sess := Session{
		Token:     tok,
		UserID:    u.ID,
		Email:     u.Email,
		OrgID:     u.OrgID,
		OrgName:   u.OrgName,
		Role:      u.Role,
		Name:      u.Name,
		ExpiresAt: time.Now().UTC().Add(time.Duration(hours) * time.Hour),
	}
	s.mu.Lock()
	s.byToken[tok] = sess
	_ = s.save()
	s.mu.Unlock()
	return &sess, nil
}

func (s *SessionStore) Get(token string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.byToken[token]
	if !ok || time.Now().UTC().After(sess.ExpiresAt) {
		return nil, false
	}
	cp := sess
	return &cp, true
}

func (s *SessionStore) Revoke(token string) {
	s.mu.Lock()
	delete(s.byToken, token)
	_ = s.save()
	s.mu.Unlock()
}
