package vault

import (
	"fmt"
	"sync"
	"time"

	"github.com/mishrak5j/graphite-mem/internal/storage"
)

type Scope struct {
	Path        string
	DisplayName string
	CreatedAt   time.Time
	MemoryCount int
}

type ScopeRegistry struct {
	mu     sync.RWMutex
	scopes map[string]*Scope
}

func NewScopeRegistry(defaultScope string) *ScopeRegistry {
	sr := &ScopeRegistry{
		scopes: make(map[string]*Scope),
	}
	sr.scopes[defaultScope] = &Scope{
		Path:        defaultScope,
		DisplayName: "Default",
		CreatedAt:   time.Now(),
	}
	return sr
}

func (sr *ScopeRegistry) CreateScope(path, displayName string) *Scope {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if s, ok := sr.scopes[path]; ok {
		return s
	}

	s := &Scope{
		Path:        path,
		DisplayName: displayName,
		CreatedAt:   time.Now(),
	}
	sr.scopes[path] = s
	return s
}

func (sr *ScopeRegistry) GetScope(path string) (*Scope, bool) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	s, ok := sr.scopes[path]
	return s, ok
}

func (sr *ScopeRegistry) ListScopes() []*Scope {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	result := make([]*Scope, 0, len(sr.scopes))
	for _, s := range sr.scopes {
		result = append(result, s)
	}
	return result
}

func (sr *ScopeRegistry) MountScope(sm *SessionManager, sessionID, scopePath string) ([]string, error) {
	sr.mu.RLock()
	_, exists := sr.scopes[scopePath]
	sr.mu.RUnlock()

	if !exists {
		sr.CreateScope(scopePath, scopePath)
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	for _, sc := range s.MountedScopes {
		if sc == scopePath {
			return s.MountedScopes, nil
		}
	}

	s.MountedScopes = append(s.MountedScopes, scopePath)
	return s.MountedScopes, nil
}

func (sr *ScopeRegistry) UnmountScope(sm *SessionManager, sessionID, scopePath string) ([]string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	filtered := make([]string, 0, len(s.MountedScopes))
	for _, sc := range s.MountedScopes {
		if sc != scopePath {
			filtered = append(filtered, sc)
		}
	}
	s.MountedScopes = filtered
	return s.MountedScopes, nil
}

func (sr *ScopeRegistry) ResolveScopeFilter(session *Session, crossScope bool) storage.ScopeFilter {
	if crossScope {
		return storage.ScopeFilter{CrossScope: true}
	}
	return storage.ScopeFilter{Scopes: session.MountedScopes}
}

func (sr *ScopeRegistry) IncrementMemoryCount(scopePath string) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	if s, ok := sr.scopes[scopePath]; ok {
		s.MemoryCount++
	}
}
