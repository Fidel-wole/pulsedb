package store

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"sort"
	"sync"
	"time"
)

const (
	ShardCount       = 64
	MaxVersions      = 10 // Maximum versions to keep per key
	TTLCheckInterval = 1 * time.Second
)

// Value represents a versioned value in the store
type Value struct {
	Data      string
	Timestamp int64 // Unix milliseconds
	TTL       int64 // Unix milliseconds when key expires, 0 means no expiration
}

// KeyHistory holds multiple versions of a key
type KeyHistory struct {
	Versions []Value
	mu       sync.RWMutex
}

// Shard represents a single shard of the store
type Shard struct {
	data map[string]*KeyHistory
	mu   sync.RWMutex
}

// Store represents the main in-memory store with MVCC support
type Store struct {
	shards   [ShardCount]*Shard
	ttlWheel *TTLWheel
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewStore creates a new store instance
func NewStore() *Store {
	ctx, cancel := context.WithCancel(context.Background())

	store := &Store{
		ttlWheel: NewTTLWheel(),
		ctx:      ctx,
		cancel:   cancel,
	}

	// Initialize shards
	for i := 0; i < ShardCount; i++ {
		store.shards[i] = &Shard{
			data: make(map[string]*KeyHistory),
		}
	}

	return store
}

// hash returns the shard index for a given key
func (s *Store) hash(key string) int {
	h := sha256.Sum256([]byte(key))
	return int(binary.BigEndian.Uint64(h[:8]) % ShardCount)
}

// getShard returns the shard for a given key
func (s *Store) getShard(key string) *Shard {
	return s.shards[s.hash(key)]
}

// Set sets a key-value pair with optional TTL
func (s *Store) Set(key, value string, ttlMs int64) {
	now := time.Now().UnixMilli()
	shard := s.getShard(key)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	var expiration int64
	if ttlMs > 0 {
		expiration = now + ttlMs
		s.ttlWheel.Add(key, expiration)
	}

	val := Value{
		Data:      value,
		Timestamp: now,
		TTL:       expiration,
	}

	history, exists := shard.data[key]
	if !exists {
		history = &KeyHistory{
			Versions: make([]Value, 0, MaxVersions),
		}
		shard.data[key] = history
	}

	history.mu.Lock()
	defer history.mu.Unlock()

	// Add new version
	history.Versions = append(history.Versions, val)

	// Keep only the latest MaxVersions
	if len(history.Versions) > MaxVersions {
		history.Versions = history.Versions[len(history.Versions)-MaxVersions:]
	}
}

// Get retrieves the current value of a key
func (s *Store) Get(key string) (string, bool) {
	return s.GetAt(key, time.Now().UnixMilli())
}

// GetAt retrieves the value of a key at a specific timestamp (MVCC)
func (s *Store) GetAt(key string, timestamp int64) (string, bool) {
	shard := s.getShard(key)

	shard.mu.RLock()
	history, exists := shard.data[key]
	shard.mu.RUnlock()

	if !exists {
		return "", false
	}

	history.mu.RLock()
	defer history.mu.RUnlock()

	// Find the latest version at or before the timestamp
	var latestValue *Value
	for i := len(history.Versions) - 1; i >= 0; i-- {
		version := &history.Versions[i]
		if version.Timestamp <= timestamp {
			// Check if the key was expired at the requested timestamp
			if version.TTL > 0 && timestamp >= version.TTL {
				return "", false
			}
			latestValue = version
			break
		}
	}

	if latestValue == nil {
		return "", false
	}

	return latestValue.Data, true
}

// Delete removes a key
func (s *Store) Delete(key string) bool {
	shard := s.getShard(key)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	_, exists := shard.data[key]
	if exists {
		delete(shard.data, key)
		s.ttlWheel.Remove(key)
		return true
	}

	return false
}

// Expire sets TTL for a key
func (s *Store) Expire(key string, ttlMs int64) bool {
	shard := s.getShard(key)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	history, exists := shard.data[key]
	if !exists {
		return false
	}

	history.mu.Lock()
	defer history.mu.Unlock()

	if len(history.Versions) == 0 {
		return false
	}

	// Update TTL of the latest version
	expiration := time.Now().UnixMilli() + ttlMs
	latestVersion := &history.Versions[len(history.Versions)-1]
	latestVersion.TTL = expiration

	s.ttlWheel.Add(key, expiration)
	return true
}

// TTL returns the time to live for a key in milliseconds
func (s *Store) TTL(key string) int64 {
	shard := s.getShard(key)

	shard.mu.RLock()
	history, exists := shard.data[key]
	shard.mu.RUnlock()

	if !exists {
		return -2 // Key doesn't exist
	}

	history.mu.RLock()
	defer history.mu.RUnlock()

	if len(history.Versions) == 0 {
		return -2
	}

	latestVersion := &history.Versions[len(history.Versions)-1]
	if latestVersion.TTL == 0 {
		return -1 // No expiration
	}

	now := time.Now().UnixMilli()
	if now >= latestVersion.TTL {
		return -2 // Already expired
	}

	return latestVersion.TTL - now
}

// History returns the version history for a key
func (s *Store) History(key string, limit int) []Value {
	shard := s.getShard(key)

	shard.mu.RLock()
	history, exists := shard.data[key]
	shard.mu.RUnlock()

	if !exists {
		return []Value{}
	}

	history.mu.RLock()
	defer history.mu.RUnlock()

	versions := make([]Value, len(history.Versions))
	copy(versions, history.Versions)

	// Sort by timestamp (newest first)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Timestamp > versions[j].Timestamp
	})

	if limit > 0 && limit < len(versions) {
		versions = versions[:limit]
	}

	return versions
}

// StartBackgroundProcesses starts background goroutines for TTL management
func (s *Store) StartBackgroundProcesses(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(TTLCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.expireKeys()
			}
		}
	}()
}

// expireKeys removes expired keys
func (s *Store) expireKeys() {
	now := time.Now().UnixMilli()
	expiredKeys := s.ttlWheel.GetExpired(now)

	for _, key := range expiredKeys {
		shard := s.getShard(key)

		shard.mu.Lock()
		history, exists := shard.data[key]
		if exists {
			history.mu.RLock()
			if len(history.Versions) > 0 {
				latestVersion := &history.Versions[len(history.Versions)-1]
				if latestVersion.TTL > 0 && now >= latestVersion.TTL {
					delete(shard.data, key)
				}
			}
			history.mu.RUnlock()
		}
		shard.mu.Unlock()
	}
}

// Close gracefully shuts down the store
func (s *Store) Close() {
	s.cancel()
	s.wg.Wait()
}

// Stats returns store statistics
func (s *Store) Stats() map[string]interface{} {
	totalKeys := 0
	totalVersions := 0

	for _, shard := range s.shards {
		shard.mu.RLock()
		totalKeys += len(shard.data)
		for _, history := range shard.data {
			history.mu.RLock()
			totalVersions += len(history.Versions)
			history.mu.RUnlock()
		}
		shard.mu.RUnlock()
	}

	return map[string]interface{}{
		"total_keys":     totalKeys,
		"total_versions": totalVersions,
		"shard_count":    ShardCount,
	}
}
