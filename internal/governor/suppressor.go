package governor

import "sync"

type injectionRecord struct {
	InjectCount      int
	LastInjectedTurn int
}

type FrequencySuppressor struct {
	mu        sync.RWMutex
	records   map[string]map[string]*injectionRecord // sessionID -> memoryID -> record
	threshold int
	cooldown  int
}

func NewFrequencySuppressor(threshold, cooldown int) *FrequencySuppressor {
	return &FrequencySuppressor{
		records:   make(map[string]map[string]*injectionRecord),
		threshold: threshold,
		cooldown:  cooldown,
	}
}

func (fs *FrequencySuppressor) ShouldSuppress(sessionID, memoryID string, currentTurn int) bool {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	sessionRecords, ok := fs.records[sessionID]
	if !ok {
		return false
	}
	rec, ok := sessionRecords[memoryID]
	if !ok {
		return false
	}

	return rec.InjectCount >= fs.threshold &&
		(currentTurn-rec.LastInjectedTurn) < fs.cooldown
}

func (fs *FrequencySuppressor) RecordInjection(sessionID, memoryID string, currentTurn int) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, ok := fs.records[sessionID]; !ok {
		fs.records[sessionID] = make(map[string]*injectionRecord)
	}

	rec, ok := fs.records[sessionID][memoryID]
	if !ok {
		rec = &injectionRecord{}
		fs.records[sessionID][memoryID] = rec
	}
	rec.InjectCount++
	rec.LastInjectedTurn = currentTurn
}
