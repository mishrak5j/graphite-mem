package vault

import (
	"strings"
)

type RankedMemory struct {
	ID    string
	Text  string
	Scope string
	Score float64
}

func SuppressTerm(sm *SessionManager, sessionID, term string, ttl int, weight float64) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[sessionID]
	if !ok {
		return false
	}

	s.Suppressions[term] = Suppression{
		Term:         term,
		RemainingTTL: ttl,
		Weight:       weight,
	}
	return true
}

func GetActiveSuppressions(sm *SessionManager, sessionID string) []Suppression {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	s, ok := sm.sessions[sessionID]
	if !ok {
		return nil
	}

	result := make([]Suppression, 0, len(s.Suppressions))
	for _, sup := range s.Suppressions {
		result = append(result, sup)
	}
	return result
}

func ApplyNegativeWeights(memories []RankedMemory, suppressions []Suppression) []RankedMemory {
	if len(suppressions) == 0 {
		return memories
	}

	result := make([]RankedMemory, 0, len(memories))
	for _, mem := range memories {
		textLower := strings.ToLower(mem.Text)
		finalScore := mem.Score

		for _, sup := range suppressions {
			if strings.Contains(textLower, strings.ToLower(sup.Term)) {
				finalScore *= sup.Weight
			}
		}

		if finalScore > 0.001 {
			mem.Score = finalScore
			result = append(result, mem)
		}
	}
	return result
}
