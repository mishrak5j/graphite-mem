package vault

import (
	"testing"

	"github.com/mishrak5j/graphite-mem/internal/storage"
)

func TestNewSession(t *testing.T) {
	sm := NewSessionManager()
	s := sm.NewSession(false, []string{"/default"})

	if s.ID == "" {
		t.Fatal("session ID should not be empty")
	}
	if s.InhibitPastContext {
		t.Fatal("inhibit should be false")
	}
	if len(s.MountedScopes) != 1 || s.MountedScopes[0] != "/default" {
		t.Fatalf("expected mounted scope /default, got %v", s.MountedScopes)
	}
}

func TestSessionInhibit(t *testing.T) {
	sm := NewSessionManager()
	s := sm.NewSession(false, nil)

	if s.InhibitPastContext {
		t.Fatal("should start uninhibited")
	}

	sm.SetInhibit(s.ID, true)
	s2 := sm.GetSession(s.ID)
	if !s2.InhibitPastContext {
		t.Fatal("should be inhibited after SetInhibit(true)")
	}

	sm.SetInhibit(s.ID, false)
	s3 := sm.GetSession(s.ID)
	if s3.InhibitPastContext {
		t.Fatal("should be uninhibited after SetInhibit(false)")
	}
}

func TestIncrementTurnDecrementsTTL(t *testing.T) {
	sm := NewSessionManager()
	s := sm.NewSession(false, nil)

	SuppressTerm(sm, s.ID, "C++", 3, 0.0)

	sm.IncrementTurn(s.ID)
	sups := GetActiveSuppressions(sm, s.ID)
	if len(sups) != 1 || sups[0].RemainingTTL != 2 {
		t.Fatalf("expected TTL=2 after 1 turn, got %v", sups)
	}

	sm.IncrementTurn(s.ID)
	sm.IncrementTurn(s.ID)

	sups = GetActiveSuppressions(sm, s.ID)
	if len(sups) != 0 {
		t.Fatalf("expected suppression to expire, got %v", sups)
	}
}

func TestScopeRegistryMountUnmount(t *testing.T) {
	sm := NewSessionManager()
	sr := NewScopeRegistry("/default")
	s := sm.NewSession(false, []string{"/default"})

	sr.CreateScope("/projects/test", "Test Project")
	mounted, err := sr.MountScope(sm, s.ID, "/projects/test")
	if err != nil {
		t.Fatal(err)
	}
	if len(mounted) != 2 {
		t.Fatalf("expected 2 mounted scopes, got %d", len(mounted))
	}

	mounted, err = sr.UnmountScope(sm, s.ID, "/projects/test")
	if err != nil {
		t.Fatal(err)
	}
	if len(mounted) != 1 || mounted[0] != "/default" {
		t.Fatalf("expected only /default after unmount, got %v", mounted)
	}
}

func TestScopeFilterResolution(t *testing.T) {
	sr := NewScopeRegistry("/default")
	session := &Session{MountedScopes: []string{"/a", "/b"}}

	f := sr.ResolveScopeFilter(session, false)
	if f.CrossScope {
		t.Fatal("should not be cross-scope")
	}
	if len(f.Scopes) != 2 {
		t.Fatalf("expected 2 scopes, got %d", len(f.Scopes))
	}

	f2 := sr.ResolveScopeFilter(session, true)
	if !f2.CrossScope {
		t.Fatal("should be cross-scope")
	}
}

func TestMountScopeIdempotent(t *testing.T) {
	sm := NewSessionManager()
	sr := NewScopeRegistry("/default")
	s := sm.NewSession(false, []string{"/default"})

	sr.MountScope(sm, s.ID, "/projects/test")
	sr.MountScope(sm, s.ID, "/projects/test")

	session := sm.GetSession(s.ID)
	count := 0
	for _, sc := range session.MountedScopes {
		if sc == "/projects/test" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 instance of /projects/test, got %d", count)
	}
}

func TestNegativeWeightingSuppression(t *testing.T) {
	memories := []RankedMemory{
		{ID: "1", Text: "I love C++ programming", Score: 0.9},
		{ID: "2", Text: "Go is great for servers", Score: 0.85},
		{ID: "3", Text: "Python for data science", Score: 0.8},
	}

	suppressions := []Suppression{
		{Term: "C++", RemainingTTL: 10, Weight: 0.0},
	}

	result := ApplyNegativeWeights(memories, suppressions)
	if len(result) != 2 {
		t.Fatalf("expected 2 memories after suppression, got %d", len(result))
	}

	for _, m := range result {
		if m.ID == "1" {
			t.Fatal("C++ memory should have been fully suppressed")
		}
	}
}

func TestNegativeWeightingPartial(t *testing.T) {
	memories := []RankedMemory{
		{ID: "1", Text: "I love C++ programming", Score: 1.0},
		{ID: "2", Text: "Go is great", Score: 1.0},
	}

	suppressions := []Suppression{
		{Term: "C++", RemainingTTL: 10, Weight: 0.5},
	}

	result := ApplyNegativeWeights(memories, suppressions)
	if len(result) != 2 {
		t.Fatalf("expected 2 memories with partial suppression, got %d", len(result))
	}

	for _, m := range result {
		if m.ID == "1" && m.Score != 0.5 {
			t.Fatalf("C++ memory score should be 0.5, got %f", m.Score)
		}
	}
}

func TestGetOrCreateDefault(t *testing.T) {
	sm := NewSessionManager()

	s1 := sm.GetOrCreateDefault("/default")
	s2 := sm.GetOrCreateDefault("/default")

	if s1.ID != s2.ID {
		t.Fatal("should return the same session")
	}
}

func TestVaultNew(t *testing.T) {
	v := New("/default")
	if v.Sessions == nil || v.Scopes == nil {
		t.Fatal("vault components should not be nil")
	}
	if v.DefaultScope != "/default" {
		t.Fatalf("expected /default, got %s", v.DefaultScope)
	}

	scopes := v.Scopes.ListScopes()
	if len(scopes) != 1 || scopes[0].Path != "/default" {
		t.Fatal("should have default scope")
	}
}

func TestResolveScopeFilterWithScopeFilter(t *testing.T) {
	_ = storage.ScopeFilter{Scopes: []string{"/a"}, CrossScope: false}
}
