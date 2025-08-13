package blazewave

import (
	"sync"
)

// Metadata is a simple key-value store for storing metadata
type Metadata struct {
	data map[string]string
	mu   sync.RWMutex
}

// NewMetadata creates a new metadata instance
func NewMetadata() *Metadata {
	return &Metadata{
		data: make(map[string]string),
	}
}

// Set sets a key-value pair in the metadata
func (m *Metadata) Set(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
}

// Get retrieves a value from the metadata by key
func (m *Metadata) Get(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, exists := m.data[key]
	return value, exists
}

// Delete removes a key-value pair from the metadata
func (m *Metadata) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}

// Clear removes all key-value pairs from the metadata
func (m *Metadata) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string]string)
}

// Has checks if a key exists in the metadata
func (m *Metadata) Has(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.data[key]
	return exists
}

// Count returns the number of key-value pairs in the metadata
func (m *Metadata) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}
