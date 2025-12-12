package agui

import (
	"sync"
	"time"
)

// StateManager manages state persistence per threadId
type StateManager struct {
	mu     sync.RWMutex
	states map[string]map[string]interface{}
	// Optional: track last access time for cleanup
	lastAccess map[string]time.Time
}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	return &StateManager{
		states:     make(map[string]map[string]interface{}),
		lastAccess: make(map[string]time.Time),
	}
}

// Get retrieves state for a threadId
func (m *StateManager) Get(threadID string) map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.states[threadID]
	if !exists {
		return make(map[string]interface{})
	}

	// Update last access time
	m.lastAccess[threadID] = time.Now()

	// Return a copy to prevent external modifications
	result := make(map[string]interface{})
	for k, v := range state {
		result[k] = v
	}
	return result
}

// Set sets state for a threadId (replaces existing state)
func (m *StateManager) Set(threadID string, state map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if state == nil {
		state = make(map[string]interface{})
	}

	// Store a copy to prevent external modifications
	result := make(map[string]interface{})
	for k, v := range state {
		result[k] = v
	}

	m.states[threadID] = result
	m.lastAccess[threadID] = time.Now()
}

// Merge merges incoming state with existing state for a threadId
// Incoming state takes precedence for overlapping keys
func (m *StateManager) Merge(threadID string, incomingState map[string]interface{}) map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.states[threadID]
	if !exists {
		existing = make(map[string]interface{})
	}

	// Merge states - incoming state takes precedence
	merged := make(map[string]interface{})

	// First, copy existing state
	for k, v := range existing {
		merged[k] = v
	}

	// Then, overlay incoming state
	for k, v := range incomingState {
		merged[k] = v
	}

	m.states[threadID] = merged
	m.lastAccess[threadID] = time.Now()

	// Return a copy
	result := make(map[string]interface{})
	for k, v := range merged {
		result[k] = v
	}
	return result
}

// Delete removes state for a threadId
func (m *StateManager) Delete(threadID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.states, threadID)
	delete(m.lastAccess, threadID)
}

// Cleanup removes states older than the specified duration
// This is useful for memory management
func (m *StateManager) Cleanup(olderThan time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	removed := 0

	for threadID, lastAccess := range m.lastAccess {
		if now.Sub(lastAccess) > olderThan {
			delete(m.states, threadID)
			delete(m.lastAccess, threadID)
			removed++
		}
	}

	return removed
}
