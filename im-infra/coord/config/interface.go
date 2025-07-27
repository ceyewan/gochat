package config

import "context"

// EventType 事件类型
type EventType string

const (
	EventTypePut    EventType = "PUT"
	EventTypeDelete EventType = "DELETE"
)

// ConfigEvent represents a configuration change event.
// It is generic to support typed values.
type ConfigEvent[T any] struct {
	Type  EventType
	Key   string
	Value T
}

// Watcher is a generic interface for watching configuration changes.
type Watcher[T any] interface {
	// Chan returns a channel that receives configuration change events.
	Chan() <-chan ConfigEvent[T]
	// Close stops the watcher.
	Close()
}

// ConfigCenter is the interface for a key-value configuration store.
type ConfigCenter interface {
	// Get retrieves a configuration value and unmarshals it into the provided type.
	Get(ctx context.Context, key string, v interface{}) error
	// Set serializes and stores a configuration value.
	Set(ctx context.Context, key string, value interface{}) error
	// Delete removes a configuration key.
	Delete(ctx context.Context, key string) error
	// Watch watches for changes on a single key and attempts to unmarshal them into the given type.
	Watch(ctx context.Context, key string, v interface{}) (Watcher[any], error)
	// WatchPrefix watches for changes on all keys under a given prefix.
	WatchPrefix(ctx context.Context, prefix string, v interface{}) (Watcher[any], error)
	// List lists all keys under a given prefix.
	List(ctx context.Context, prefix string) ([]string, error)
}
