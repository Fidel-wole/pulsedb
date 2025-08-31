package store

import (
	"testing"
	"time"
)

func TestStoreBasicOperations(t *testing.T) {
	store := NewStore()
	defer store.Close()

	// Test Set and Get
	store.Set("key1", "value1", 0)
	value, found := store.Get("key1")
	if !found {
		t.Error("Expected to find key1")
	}
	if value != "value1" {
		t.Errorf("Expected value1, got %s", value)
	}

	// Test Get non-existent key
	_, found = store.Get("nonexistent")
	if found {
		t.Error("Expected not to find nonexistent key")
	}

	// Test Delete
	deleted := store.Delete("key1")
	if !deleted {
		t.Error("Expected key1 to be deleted")
	}

	_, found = store.Get("key1")
	if found {
		t.Error("Expected key1 to be deleted")
	}

	// Test Delete non-existent key
	deleted = store.Delete("nonexistent")
	if deleted {
		t.Error("Expected delete of nonexistent key to return false")
	}
}

func TestStoreTTL(t *testing.T) {
	store := NewStore()
	defer store.Close()

	// Set key with TTL
	store.Set("ttl_key", "value", 100) // 100ms TTL

	// Should exist immediately
	_, found := store.Get("ttl_key")
	if !found {
		t.Error("Expected to find ttl_key immediately after setting")
	}

	// Check TTL
	ttl := store.TTL("ttl_key")
	if ttl <= 0 {
		t.Errorf("Expected positive TTL, got %d", ttl)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, found = store.Get("ttl_key")
	if found {
		t.Error("Expected ttl_key to be expired")
	}

	// TTL should be -2 (expired/non-existent)
	ttl = store.TTL("ttl_key")
	if ttl != -2 {
		t.Errorf("Expected TTL -2 for expired key, got %d", ttl)
	}
}

func TestStoreExpire(t *testing.T) {
	store := NewStore()
	defer store.Close()

	// Set key without TTL
	store.Set("expire_key", "value", 0)

	// TTL should be -1 (no expiration)
	ttl := store.TTL("expire_key")
	if ttl != -1 {
		t.Errorf("Expected TTL -1 for key without expiration, got %d", ttl)
	}

	// Set expiration
	success := store.Expire("expire_key", 100) // 100ms from now
	if !success {
		t.Error("Expected expire to succeed")
	}

	// TTL should be positive
	ttl = store.TTL("expire_key")
	if ttl <= 0 {
		t.Errorf("Expected positive TTL after expire, got %d", ttl)
	}

	// Try to expire non-existent key
	success = store.Expire("nonexistent", 100)
	if success {
		t.Error("Expected expire on non-existent key to fail")
	}
}

func TestStoreMVCC(t *testing.T) {
	store := NewStore()
	defer store.Close()

	now := time.Now().UnixMilli()

	// Set initial value
	store.Set("mvcc_key", "v1", 0)
	time.Sleep(10 * time.Millisecond) // Ensure different timestamp

	// Set second value
	store.Set("mvcc_key", "v2", 0)
	time.Sleep(10 * time.Millisecond)

	// Set third value
	store.Set("mvcc_key", "v3", 0)

	// Get current value
	value, found := store.Get("mvcc_key")
	if !found || value != "v3" {
		t.Errorf("Expected current value v3, got %s (found: %t)", value, found)
	}

	// Get value at time before any writes
	value, found = store.GetAt("mvcc_key", now-1000)
	if found {
		t.Error("Expected no value before first write")
	}

	// Get history
	history := store.History("mvcc_key", 0)
	if len(history) == 0 {
		t.Error("Expected non-empty history")
	}

	// Should have all versions (newest first due to sorting)
	expectedValues := []string{"v3", "v2", "v1"}
	for i, version := range history {
		if i >= len(expectedValues) {
			break
		}
		if version.Data != expectedValues[i] {
			t.Errorf("Expected history[%d] to be %s, got %s", i, expectedValues[i], version.Data)
		}
	}

	// Test history with limit
	limitedHistory := store.History("mvcc_key", 2)
	if len(limitedHistory) != 2 {
		t.Errorf("Expected history with limit 2 to have 2 entries, got %d", len(limitedHistory))
	}
}

func TestStoreStats(t *testing.T) {
	store := NewStore()
	defer store.Close()

	// Add some keys
	store.Set("key1", "value1", 0)
	store.Set("key2", "value2", 0)
	store.Set("key1", "value1_updated", 0) // Update key1

	stats := store.Stats()

	totalKeys, ok := stats["total_keys"].(int)
	if !ok || totalKeys != 2 {
		t.Errorf("Expected 2 total keys, got %v", stats["total_keys"])
	}

	totalVersions, ok := stats["total_versions"].(int)
	if !ok || totalVersions < 2 {
		t.Errorf("Expected at least 2 total versions, got %v", stats["total_versions"])
	}

	shardCount, ok := stats["shard_count"].(int)
	if !ok || shardCount != ShardCount {
		t.Errorf("Expected shard count %d, got %v", ShardCount, stats["shard_count"])
	}
}

func TestStoreSharding(t *testing.T) {
	store := NewStore()
	defer store.Close()

	// Test that different keys might go to different shards
	key1 := "test_key_1"
	key2 := "test_key_2_different"

	shard1 := store.hash(key1)
	shard2 := store.hash(key2)

	if shard1 < 0 || shard1 >= ShardCount {
		t.Errorf("Shard index %d out of range [0, %d)", shard1, ShardCount)
	}

	if shard2 < 0 || shard2 >= ShardCount {
		t.Errorf("Shard index %d out of range [0, %d)", shard2, ShardCount)
	}

	// The same key should always go to the same shard
	if store.hash(key1) != shard1 {
		t.Error("Hash function should be deterministic")
	}
}
