package vault

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID                 string
	CreatedAt          time.Time
	TurnCounter        int
	InhibitPastContext bool
	MountedScopes      []string
	Suppressions       map[string]Suppression
}

type Suppression struct {
	Term         string
	RemainingTTL int
	Weight       float64
}

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

func (sm *SessionManager) NewSession(inhibit bool, scopes []string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s := &Session{
		ID:                 uuid.New().String(),
		CreatedAt:          time.Now(),
		InhibitPastContext: inhibit,
		MountedScopes:      scopes,
		Suppressions:       make(map[string]Suppression),
	}
	sm.sessions[s.ID] = s
	return s
}

func (sm *SessionManager) GetSession(id string) *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[id]
}

func (sm *SessionManager) GetOrCreateDefault(defaultScope string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, s := range sm.sessions {
		return s
	}

	s := &Session{
		ID:            uuid.New().String(),
		CreatedAt:     time.Now(),
		MountedScopes: []string{defaultScope},
		Suppressions:  make(map[string]Suppression),
	}
	sm.sessions[s.ID] = s
	return s
}

func (sm *SessionManager) SetInhibit(sessionID string, inhibit bool) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if s, ok := sm.sessions[sessionID]; ok {
		s.InhibitPastContext = inhibit
		return true
	}
	return false
}

func (sm *SessionManager) IncrementTurn(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[sessionID]
	if !ok {
		return
	}

	s.TurnCounter++

	expired := make([]string, 0)
	for term, sup := range s.Suppressions {
		sup.RemainingTTL--
		if sup.RemainingTTL <= 0 {
			expired = append(expired, term)
		} else {
			s.Suppressions[term] = sup
		}
	}
	for _, term := range expired {
		delete(s.Suppressions, term)
	}
}
