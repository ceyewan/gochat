package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/es"
)

// MyMessage is a sample struct that implements the es.Indexable interface.
type MyMessage struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// GetID returns the unique identifier for the message.
func (m MyMessage) GetID() string {
	return m.ID
}

func main() {
	// 1. Initialize logger
	logger := clog.Namespace("es-example")
	ctx := context.Background()

	// 2. Get default config and create a new es provider
	cfg := es.GetDefaultConfig("development")
	// Use a real ES address if you have one running, e.g., http://localhost:9200
	// For this example, we assume it might fail if ES is not running, which is fine.
	esProvider, err := es.New[MyMessage](ctx, cfg, es.WithLogger(logger))
	if err != nil {
		log.Fatalf("Failed to create es provider: %v", err)
	}
	defer esProvider.Close()

	// 3. Create some sample messages
	messages := make([]MyMessage, 0, 10)
	for i := 0; i < 10; i++ {
		messages = append(messages, MyMessage{
			ID:        strconv.Itoa(i),
			SessionID: "session_123",
			Content:   fmt.Sprintf("Hello, Elasticsearch! This is message %d", i),
			Timestamp: time.Now(),
		})
	}

	// 4. Bulk index the messages
	indexName := "test-messages"
	logger.Info("indexing documents...", clog.Int("count", len(messages)))
	if err := esProvider.BulkIndex(ctx, indexName, messages); err != nil {
		log.Fatalf("Failed to bulk index messages: %v", err)
	}

	logger.Info("successfully indexed documents. waiting for bulk indexer to flush...")

	// In a real application, you don't need to sleep. The provider's Close() method
	// will handle flushing remaining documents gracefully.
	// We sleep here just to give the async bulk indexer time to process.
	log.Println("Waiting for bulk indexer to flush...")
	time.Sleep(6 * time.Second)

	// 5. Search for the messages
	log.Println("Searching for messages with keyword 'Elasticsearch'...")
	searchResult, err := esProvider.SearchGlobal(ctx, indexName, "Elasticsearch", 1, 10)
	if err != nil {
		log.Fatalf("Failed to search messages: %v", err)
	}

	log.Printf("Found %d messages:\n", searchResult.Total)
	for _, msg := range searchResult.Items {
		log.Printf("  - ID: %s, Content: %s\n", (*msg).GetID(), (*msg).Content)
	}

	log.Println("\nSearching for messages in session 'session_123'...")
	sessionResult, err := esProvider.SearchInSession(ctx, indexName, "session_123", "message 5", 1, 5)
	if err != nil {
		log.Fatalf("Failed to search messages in session: %v", err)
	}

	log.Printf("Found %d messages in session:\n", sessionResult.Total)
	for _, msg := range sessionResult.Items {
		log.Printf("  - ID: %s, Content: %s\n", (*msg).GetID(), (*msg).Content)
	}

	log.Println("\nExample finished.")
}
