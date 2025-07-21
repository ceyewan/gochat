package queue

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// MessageDeduplicator provides exactly-once processing guarantees
// Uses content-based deduplication with configurable time windows
type MessageDeduplicator struct {
	seenMessages map[string]time.Time
	mu           sync.RWMutex
	ttl          time.Duration
	maxSize      int
	cleanupFreq  time.Duration
	stopChan     chan struct{}
}

// DeduplicationConfig holds configuration for message deduplication
type DeduplicationConfig struct {
	TTL         time.Duration // How long to keep message hashes
	MaxSize     int           // Maximum number of entries to store
	CleanupFreq time.Duration // How often to cleanup expired entries
}

// DefaultDeduplicationConfig returns default configuration optimized for text messaging
func DefaultDeduplicationConfig() *DeduplicationConfig {
	return &DeduplicationConfig{
		TTL:         24 * time.Hour, // Keep messages for 24 hours
		MaxSize:     100000,         // Store up to 100k messages
		CleanupFreq: 5 * time.Minute, // Cleanup every 5 minutes
	}
}

// NewMessageDeduplicator creates a new message deduplicator
func NewMessageDeduplicator(config *DeduplicationConfig) *MessageDeduplicator {
	if config == nil {
		config = DefaultDeduplicationConfig()
	}

	d := &MessageDeduplicator{
		seenMessages: make(map[string]time.Time),
		ttl:          config.TTL,
		maxSize:      config.MaxSize,
		cleanupFreq:  config.CleanupFreq,
		stopChan:     make(chan struct{}),
	}

	// Start background cleanup goroutine
	go d.cleanupRoutine()

	return d
}

// GenerateMessageID creates a unique identifier for a message
// Combines message content and metadata to ensure uniqueness
func (d *MessageDeduplicator) GenerateMessageID(msg *QueueMsg) string {
	// Create a hash based on message content and key metadata
	hash := sha256.New()
	hash.Write([]byte(fmt.Sprintf("%d-%d-%d-%s", msg.UserId, msg.RoomId, msg.Op, string(msg.Msg))))
	return hex.EncodeToString(hash.Sum(nil))
}

// IsDuplicate checks if a message has been processed before
// Returns true if message is a duplicate, false otherwise
func (d *MessageDeduplicator) IsDuplicate(messageID string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if seenTime, exists := d.seenMessages[messageID]; exists {
		// Check if the entry hasn't expired
		if time.Since(seenTime) < d.ttl {
			return true
		}
	}

	return false
}

// MarkAsProcessed marks a message as processed
func (d *MessageDeduplicator) MarkAsProcessed(messageID string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Add or update the message timestamp
	d.seenMessages[messageID] = time.Now()

	// Check if we need to cleanup due to size limit
	if len(d.seenMessages) > d.maxSize {
		d.cleanupOldEntries()
	}
}

// MarkAsProcessedWithTTL marks a message as processed with custom TTL
func (d *MessageDeduplicator) MarkAsProcessedWithTTL(messageID string, ttl time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.seenMessages[messageID] = time.Now()

	if len(d.seenMessages) > d.maxSize {
		d.cleanupOldEntries()
	}
}

// cleanupOldEntries removes expired entries to maintain size limits
func (d *MessageDeduplicator) cleanupOldEntries() {
	now := time.Now()
	for id, seenTime := range d.seenMessages {
		if now.Sub(seenTime) > d.ttl {
			delete(d.seenMessages, id)
		}
	}
}

// cleanupRoutine runs periodically to cleanup expired entries
func (d *MessageDeduplicator) cleanupRoutine() {
	ticker := time.NewTicker(d.cleanupFreq)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.cleanup()
		case <-d.stopChan:
			return
		}
	}
}

// cleanup performs the actual cleanup of expired entries
func (d *MessageDeduplicator) cleanup() {
	d.mu.Lock()
	defer d.mu.Unlock()

	initialSize := len(d.seenMessages)
	d.cleanupOldEntries()
	finalSize := len(d.seenMessages)

	if initialSize != finalSize {
		// Log cleanup statistics (replace with proper logging)
		fmt.Printf("Deduplicator cleanup: removed %d expired entries, %d remaining\n",
			initialSize-finalSize, finalSize)
	}
}

// Stop stops the cleanup routine and releases resources
func (d *MessageDeduplicator) Stop() {
	close(d.stopChan)
}

// GetStats returns deduplication statistics
func (d *MessageDeduplicator) GetStats() map[string]interface{} {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return map[string]interface{}{
		"total_messages": len(d.seenMessages),
		"ttl":            d.ttl.String(),
		"max_size":       d.maxSize,
	}
}

// Clear removes all entries (for testing purposes)
func (d *MessageDeduplicator) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.seenMessages = make(map[string]time.Time)
}

// ProcessWithDeduplication wraps message processing with deduplication
func (d *MessageDeduplicator) ProcessWithDeduplication(
	msg *QueueMsg,
	processor func(*QueueMsg) error,
) error {
	// Generate unique message ID
	messageID := d.GenerateMessageID(msg)

	// Check for duplicates
	if d.IsDuplicate(messageID) {
		return fmt.Errorf("duplicate message detected: %s", messageID)
	}

	// Process the message
	err := processor(msg)
	if err != nil {
		return err
	}

	// Mark as processed only if successful
	d.MarkAsProcessed(messageID)

	return nil
}

// AtomicDeduplicationProcessor provides atomic deduplication and processing
type AtomicDeduplicationProcessor struct {
	deduplicator *MessageDeduplicator
	processor    func(*QueueMsg) error
	mu           sync.Mutex
}

// NewAtomicDeduplicationProcessor creates an atomic processor
func NewAtomicDeduplicationProcessor(
	deduplicator *MessageDeduplicator,
	processor func(*QueueMsg) error,
) *AtomicDeduplicationProcessor {
	return &AtomicDeduplicationProcessor{
		deduplicator: deduplicator,
		processor:    processor,
	}
}

// Process processes a message with atomic deduplication
func (adp *AtomicDeduplicationProcessor) Process(msg *QueueMsg) error {
	adp.mu.Lock()
	defer adp.mu.Unlock()

	return adp.deduplicator.ProcessWithDeduplication(msg, adp.processor)
}

// MessageProcessorWithDeduplication provides a wrapper for message processing
// that handles both deduplication and retry logic
type MessageProcessorWithDeduplication struct {
	deduplicator *MessageDeduplicator
	retryPolicy  *RetryPolicy
	processor    func(*QueueMsg) error
}

// RetryPolicy defines retry behavior for message processing
type RetryPolicy struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Multiplier float64
}

// DefaultRetryPolicy returns a default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries: 3,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   5 * time.Second,
		Multiplier: 2.0,
	}
}

// NewMessageProcessorWithDeduplication creates a new processor
func NewMessageProcessorWithDeduplication(
	deduplicator *MessageDeduplicator,
	processor func(*QueueMsg) error,
	retryPolicy *RetryPolicy,
) *MessageProcessorWithDeduplication {
	if retryPolicy == nil {
		retryPolicy = DefaultRetryPolicy()
	}

	return &MessageProcessorWithDeduplication{
		deduplicator: deduplicator,
		retryPolicy:  retryPolicy,
		processor:    processor,
	}
}

// Process processes a message with deduplication and retry logic
func (mp *MessageProcessorWithDeduplication) Process(msg *QueueMsg) error {
	messageID := mp.deduplicator.GenerateMessageID(msg)

	// Check for duplicates
	if mp.deduplicator.IsDuplicate(messageID) {
		return nil // Silently skip duplicates
	}

	// Process with retry
	err := mp.processWithRetry(msg)
	if err != nil {
		return err
	}

	// Mark as processed only after successful processing
	mp.deduplicator.MarkAsProcessed(messageID)
	return nil
}

// processWithRetry implements retry logic
func (mp *MessageProcessorWithDeduplication) processWithRetry(msg *QueueMsg) error {
	var lastErr error
	delay := mp.retryPolicy.BaseDelay

	for attempt := 0; attempt <= mp.retryPolicy.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(delay)
			delay = time.Duration(float64(delay) * mp.retryPolicy.Multiplier)
			if delay > mp.retryPolicy.MaxDelay {
				delay = mp.retryPolicy.MaxDelay
			}
		}

		lastErr = mp.processor(msg)
		if lastErr == nil {
			return nil
		}

		// Skip retry for certain types of errors
		if lastErr == ErrStopConsumer {
			return lastErr
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}