package memory

import (
	"context"
	"testing"
)

func TestSequentialMemory(t *testing.T) {
	ctx := context.Background()
	mem := NewSequentialMemory()

	// Add messages
	msg1 := NewMessage("user", "Hello")
	msg2 := NewMessage("assistant", "Hi there!")
	msg3 := NewMessage("user", "How are you?")

	if err := mem.AddMessage(ctx, msg1); err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}
	if err := mem.AddMessage(ctx, msg2); err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}
	if err := mem.AddMessage(ctx, msg3); err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}

	// Get context
	messages, err := mem.GetContext(ctx, "")
	if err != nil {
		t.Fatalf("Failed to get context: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	// Check stats
	stats, err := mem.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.TotalMessages != 3 {
		t.Errorf("Expected 3 total messages, got %d", stats.TotalMessages)
	}

	// Clear
	if err := mem.Clear(ctx); err != nil {
		t.Fatalf("Failed to clear: %v", err)
	}

	messages, _ = mem.GetContext(ctx, "")
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", len(messages))
	}
}

func TestSlidingWindowMemory(t *testing.T) {
	ctx := context.Background()
	mem := NewSlidingWindowMemory(2) // Window size of 2

	// Add 3 messages
	msg1 := NewMessage("user", "Message 1")
	msg2 := NewMessage("user", "Message 2")
	msg3 := NewMessage("user", "Message 3")

	mem.AddMessage(ctx, msg1)
	mem.AddMessage(ctx, msg2)
	mem.AddMessage(ctx, msg3)

	// Should only keep last 2
	messages, err := mem.GetContext(ctx, "")
	if err != nil {
		t.Fatalf("Failed to get context: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages in window, got %d", len(messages))
	}

	// Should have message 2 and 3
	if messages[0].Content != "Message 2" || messages[1].Content != "Message 3" {
		t.Errorf("Window contains wrong messages")
	}
}

func TestBufferMemory(t *testing.T) {
	ctx := context.Background()

	// Test with message limit
	mem := NewBufferMemory(&BufferConfig{
		MaxMessages: 2,
	})

	msg1 := NewMessage("user", "Message 1")
	msg2 := NewMessage("user", "Message 2")
	msg3 := NewMessage("user", "Message 3")

	mem.AddMessage(ctx, msg1)
	mem.AddMessage(ctx, msg2)
	mem.AddMessage(ctx, msg3)

	messages, _ := mem.GetContext(ctx, "")
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages with limit, got %d", len(messages))
	}

	// Test GetMessages
	msgs := mem.GetMessages()
	if len(msgs) != 2 {
		t.Errorf("Expected 2 messages from GetMessages, got %d", len(msgs))
	}

	// Test LoadMessages
	newMessages := []*Message{
		NewMessage("user", "Loaded 1"),
		NewMessage("user", "Loaded 2"),
	}
	mem.LoadMessages(newMessages)

	messages, _ = mem.GetContext(ctx, "")
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages after load, got %d", len(messages))
	}
	if messages[0].Content != "Loaded 1" {
		t.Errorf("Loaded messages incorrect")
	}
}

func TestSummarizationMemory(t *testing.T) {
	ctx := context.Background()

	mem := NewSummarizationMemory(&SummarizationConfig{
		RecentWindowSize: 2,
		SummarizeAfter:   3,
	})

	// Add messages
	for i := 1; i <= 4; i++ {
		msg := NewMessage("user", "Message content")
		if err := mem.AddMessage(ctx, msg); err != nil {
			t.Fatalf("Failed to add message %d: %v", i, err)
		}
	}

	messages, err := mem.GetContext(ctx, "")
	if err != nil {
		t.Fatalf("Failed to get context: %v", err)
	}

	// Should have summary + recent messages
	if len(messages) < 2 {
		t.Errorf("Expected at least 2 messages (summary + recent), got %d", len(messages))
	}

	// First message should be a summary
	if messages[0].Role != "system" {
		t.Errorf("Expected first message to be system (summary), got %s", messages[0].Role)
	}
}

func TestRetrievalMemory(t *testing.T) {
	ctx := context.Background()

	mem := NewRetrievalMemory(&RetrievalConfig{
		TopK: 2,
	})

	// Add messages
	msg1 := NewMessage("user", "Hello world")
	msg2 := NewMessage("user", "Goodbye world")
	msg3 := NewMessage("user", "Python programming")

	mem.AddMessage(ctx, msg1)
	mem.AddMessage(ctx, msg2)
	mem.AddMessage(ctx, msg3)

	// Query similar to "Hello"
	messages, err := mem.GetContext(ctx, "Hello")
	if err != nil {
		t.Fatalf("Failed to get context: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("Expected top 2 messages, got %d", len(messages))
	}
}

func TestHierarchicalMemory(t *testing.T) {
	ctx := context.Background()

	mem := NewHierarchicalMemory(&HierarchicalConfig{
		RecentLimit:    2,
		ImportantLimit: 2,
	})

	// Add messages with varying importance
	msg1 := NewMessage("user", "Regular message")
	msg2 := NewMessage("user", "Important message")
	msg2.Metadata["importance"] = 0.9

	msg3 := NewMessage("user", "Another regular")

	mem.AddMessage(ctx, msg1)
	mem.AddMessage(ctx, msg2)
	mem.AddMessage(ctx, msg3)

	messages, err := mem.GetContext(ctx, "")
	if err != nil {
		t.Fatalf("Failed to get context: %v", err)
	}

	// Should include important and recent messages
	if len(messages) == 0 {
		t.Error("Expected some messages from hierarchical memory")
	}

	stats, err := mem.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.TotalMessages == 0 {
		t.Error("Expected non-zero total messages")
	}
}

func TestMessageCreation(t *testing.T) {
	msg := NewMessage("user", "Test content")

	if msg.Role != "user" {
		t.Errorf("Expected role 'user', got %s", msg.Role)
	}

	if msg.Content != "Test content" {
		t.Errorf("Expected content 'Test content', got %s", msg.Content)
	}

	if msg.ID == "" {
		t.Error("Expected non-empty ID")
	}

	if msg.TokenCount == 0 {
		t.Error("Expected non-zero token count")
	}

	if msg.Metadata == nil {
		t.Error("Expected non-nil metadata")
	}
}

func TestGraphBasedMemory(t *testing.T) {
	ctx := context.Background()

	mem := NewGraphBasedMemory(&GraphConfig{
		TopK: 5,
	})

	// Add related messages
	msg1 := NewMessage("user", "What's the price of the product?")
	msg2 := NewMessage("assistant", "The price is $99")
	msg3 := NewMessage("user", "Tell me about features")
	msg4 := NewMessage("user", "What's the price again?")

	mem.AddMessage(ctx, msg1)
	mem.AddMessage(ctx, msg2)
	mem.AddMessage(ctx, msg3)
	mem.AddMessage(ctx, msg4)

	// Query should retrieve related messages
	messages, err := mem.GetContext(ctx, "price information")
	if err != nil {
		t.Fatalf("Failed to get context: %v", err)
	}

	if len(messages) == 0 {
		t.Error("Expected some messages from graph memory")
	}

	// Check stats
	stats, err := mem.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.TotalMessages != 4 {
		t.Errorf("Expected 4 total messages, got %d", stats.TotalMessages)
	}

	// Check relationships
	relations := mem.GetRelationships()
	if len(relations) == 0 {
		t.Error("Expected some relationships in graph")
	}
}

func TestCompressionMemory(t *testing.T) {
	ctx := context.Background()

	mem := NewCompressionMemory(&CompressionConfig{
		CompressionTrigger: 3, // Compress after 3 messages
	})

	// Add messages to trigger compression
	for range 5 {
		msg := NewMessage("user", "Message content for compression")
		if err := mem.AddMessage(ctx, msg); err != nil {
			t.Fatalf("Failed to add message: %v", err)
		}
	}

	messages, err := mem.GetContext(ctx, "")
	if err != nil {
		t.Fatalf("Failed to get context: %v", err)
	}

	// Should have compressed block(s) plus remaining messages
	if len(messages) == 0 {
		t.Error("Expected some messages from compression memory")
	}

	stats, err := mem.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	// Should show compression
	if stats.CompressionRate >= 1.0 {
		t.Logf("Compression rate: %.2f (expected < 1.0 for compression)", stats.CompressionRate)
	}
}

func TestOSLikeMemory(t *testing.T) {
	ctx := context.Background()

	mem := NewOSLikeMemory(&OSLikeConfig{
		ActiveLimit: 2,
		CacheLimit:  3,
	})

	// Add messages
	for range 10 {
		msg := NewMessage("user", "Message content")
		if err := mem.AddMessage(ctx, msg); err != nil {
			t.Fatalf("Failed to add message: %v", err)
		}
	}

	messages, err := mem.GetContext(ctx, "")
	if err != nil {
		t.Fatalf("Failed to get context: %v", err)
	}

	if len(messages) == 0 {
		t.Error("Expected some messages from OS-like memory")
	}

	stats, err := mem.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.TotalMessages == 0 {
		t.Error("Expected non-zero total messages")
	}

	// Check memory info
	info := mem.GetMemoryInfo()
	if info == nil {
		t.Error("Expected memory info")
	}

	if activePages, ok := info["active_pages"].(int); ok {
		t.Logf("Active pages: %d", activePages)
	}
}
