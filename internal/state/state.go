package state

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// State tracks alert state to prevent duplicate notifications
type State interface {
	// ShouldAlert returns true if this is a new alert condition
	ShouldAlert(checkName string, resultHash string) bool
	// MarkAlerted records that an alert was sent
	MarkAlerted(checkName string, resultHash string) error
	// Clear resets state for a check (when condition clears)
	Clear(checkName string) error
	// Close cleans up resources
	Close() error
}

// Hash generates a hash from alert content for deduplication
func Hash(title, message string) string {
	h := sha256.New()
	h.Write([]byte(title))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// alertRecord tracks when an alert was sent
type alertRecord struct {
	hash      string
	alertedAt time.Time
}

// Memory implements in-memory state tracking
type Memory struct {
	mu     sync.RWMutex
	alerts map[string]alertRecord
}

// NewMemory creates a new in-memory state tracker
func NewMemory() *Memory {
	return &Memory{
		alerts: make(map[string]alertRecord),
	}
}

// ShouldAlert returns true if this hash hasn't been alerted recently
func (m *Memory) ShouldAlert(checkName string, resultHash string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	record, exists := m.alerts[checkName]
	if !exists {
		return true
	}

	return record.hash != resultHash
}

// MarkAlerted records that an alert was sent
func (m *Memory) MarkAlerted(checkName string, resultHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.alerts[checkName] = alertRecord{
		hash:      resultHash,
		alertedAt: time.Now(),
	}

	return nil
}

// Clear removes state for a check
func (m *Memory) Clear(checkName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.alerts, checkName)
	return nil
}

// Close is a no-op for memory state
func (m *Memory) Close() error {
	return nil
}
