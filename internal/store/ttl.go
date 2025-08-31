package store

import (
	"sync"
)

// TTLWheel implements a timing wheel for efficient TTL management
type TTLWheel struct {
	entries map[string]int64 // key -> expiration timestamp
	mu      sync.RWMutex
}

// NewTTLWheel creates a new TTL wheel
func NewTTLWheel() *TTLWheel {
	return &TTLWheel{
		entries: make(map[string]int64),
	}
}

// Add adds a key with expiration timestamp
func (tw *TTLWheel) Add(key string, expiration int64) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.entries[key] = expiration
}

// Remove removes a key from the TTL wheel
func (tw *TTLWheel) Remove(key string) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	delete(tw.entries, key)
}

// GetExpired returns keys that have expired before the given timestamp
func (tw *TTLWheel) GetExpired(now int64) []string {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	var expired []string
	for key, expiration := range tw.entries {
		if now >= expiration {
			expired = append(expired, key)
			delete(tw.entries, key)
		}
	}

	return expired
}
