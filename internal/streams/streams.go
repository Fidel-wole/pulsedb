package streams

import (
	"fmt"
	"sync"
	"time"
)

// StreamEntry represents an entry in a stream
type StreamEntry struct {
	ID        string
	Timestamp int64
	Fields    map[string]string
	UUID      string // For idempotent operations
}

// ConsumerGroup represents a consumer group
type ConsumerGroup struct {
	Name      string
	Consumers map[string]*Consumer
	LastID    string
	mu        sync.RWMutex
}

// Consumer represents a stream consumer
type Consumer struct {
	Name         string
	Group        string
	PendingCount int
	LastSeen     int64
}

// Stream represents a PulseDB stream with enhanced features
type Stream struct {
	Name    string
	Entries []StreamEntry
	Groups  map[string]*ConsumerGroup
	UUIDs   map[string]bool // For idempotency checking
	mu      sync.RWMutex
}

// StreamManager manages all streams
type StreamManager struct {
	streams map[string]*Stream
	mu      sync.RWMutex
}

// NewStreamManager creates a new stream manager
func NewStreamManager() *StreamManager {
	return &StreamManager{
		streams: make(map[string]*Stream),
	}
}

// AddEntry adds an entry to a stream with optional idempotency
func (sm *StreamManager) AddEntry(streamName string, fields map[string]string, uuid string) (string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	stream, exists := sm.streams[streamName]
	if !exists {
		stream = &Stream{
			Name:    streamName,
			Entries: make([]StreamEntry, 0),
			Groups:  make(map[string]*ConsumerGroup),
			UUIDs:   make(map[string]bool),
		}
		sm.streams[streamName] = stream
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	// Check idempotency
	if uuid != "" {
		if stream.UUIDs[uuid] {
			// Entry already exists, return existing ID
			for _, entry := range stream.Entries {
				if entry.UUID == uuid {
					return entry.ID, nil
				}
			}
		}
		stream.UUIDs[uuid] = true
	}

	// Generate ID (simplified - real implementation would be more sophisticated)
	timestamp := time.Now().UnixMilli()
	id := fmt.Sprintf("%d-0", timestamp)

	entry := StreamEntry{
		ID:        id,
		Timestamp: timestamp,
		Fields:    fields,
		UUID:      uuid,
	}

	stream.Entries = append(stream.Entries, entry)

	return id, nil
}

// CreateConsumerGroup creates a new consumer group
func (sm *StreamManager) CreateConsumerGroup(streamName, groupName string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	stream, exists := sm.streams[streamName]
	if !exists {
		return fmt.Errorf("stream %s does not exist", streamName)
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	if _, exists := stream.Groups[groupName]; exists {
		return fmt.Errorf("consumer group %s already exists", groupName)
	}

	stream.Groups[groupName] = &ConsumerGroup{
		Name:      groupName,
		Consumers: make(map[string]*Consumer),
		LastID:    "0-0",
	}

	return nil
}

// ReadGroup reads entries from a stream as part of a consumer group
func (sm *StreamManager) ReadGroup(streamName, groupName, consumerName string, count int) ([]StreamEntry, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stream, exists := sm.streams[streamName]
	if !exists {
		return nil, fmt.Errorf("stream %s does not exist", streamName)
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	group, exists := stream.Groups[groupName]
	if !exists {
		return nil, fmt.Errorf("consumer group %s does not exist", groupName)
	}

	group.mu.Lock()
	defer group.mu.Unlock()

	// Create consumer if it doesn't exist
	if _, exists := group.Consumers[consumerName]; !exists {
		group.Consumers[consumerName] = &Consumer{
			Name:     consumerName,
			Group:    groupName,
			LastSeen: time.Now().Unix(),
		}
	}

	// Find entries after the group's last ID
	var result []StreamEntry
	found := false

	for _, entry := range stream.Entries {
		if entry.ID == group.LastID {
			found = true
			continue
		}
		if found && len(result) < count {
			result = append(result, entry)
		}
	}

	// Update group's last ID if we found entries
	if len(result) > 0 {
		group.LastID = result[len(result)-1].ID
	}

	return result, nil
}

// GetStreamInfo returns information about a stream
func (sm *StreamManager) GetStreamInfo(streamName string) (map[string]interface{}, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stream, exists := sm.streams[streamName]
	if !exists {
		return nil, fmt.Errorf("stream %s does not exist", streamName)
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	info := map[string]interface{}{
		"name":           stream.Name,
		"length":         len(stream.Entries),
		"groups":         len(stream.Groups),
		"unique_entries": len(stream.UUIDs),
	}

	if len(stream.Entries) > 0 {
		info["first_entry"] = stream.Entries[0].ID
		info["last_entry"] = stream.Entries[len(stream.Entries)-1].ID
	}

	return info, nil
}

// ListStreams returns a list of all stream names
func (sm *StreamManager) ListStreams() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	names := make([]string, 0, len(sm.streams))
	for name := range sm.streams {
		names = append(names, name)
	}

	return names
}
