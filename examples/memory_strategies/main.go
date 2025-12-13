package main

import (
	"context"
	"fmt"
	"time"

	"github.com/smallnest/langgraphgo/memory"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Memory Strategies Examples ===")

	// 1. Sequential Memory
	fmt.Println("1. Sequential Memory (Keep-It-All)")
	fmt.Println("   - Stores all messages without limit")
	demoSequentialMemory(ctx)

	// 2. Sliding Window Memory
	fmt.Println("\n2. Sliding Window Memory")
	fmt.Println("   - Keeps only recent N messages")
	demoSlidingWindowMemory(ctx)

	// 3. Buffer Memory
	fmt.Println("\n3. Buffer Memory")
	fmt.Println("   - Flexible limits by messages or tokens")
	demoBufferMemory(ctx)

	// 4. Summarization Memory
	fmt.Println("\n4. Summarization Memory")
	fmt.Println("   - Summarizes old messages, keeps recent ones")
	demoSummarizationMemory(ctx)

	// 5. Retrieval Memory
	fmt.Println("\n5. Retrieval Memory")
	fmt.Println("   - Retrieves most relevant messages using embeddings")
	demoRetrievalMemory(ctx)

	// 6. Hierarchical Memory
	fmt.Println("\n6. Hierarchical Memory")
	fmt.Println("   - Separates important and recent messages")
	demoHierarchicalMemory(ctx)

	// 7. Graph-Based Memory
	fmt.Println("\n7. Graph-Based Memory")
	fmt.Println("   - Tracks relationships between messages")
	demoGraphBasedMemory(ctx)

	// 8. Compression Memory
	fmt.Println("\n8. Compression Memory")
	fmt.Println("   - Compresses and consolidates old messages")
	demoCompressionMemory(ctx)

	// 9. OS-Like Memory
	fmt.Println("\n9. OS-Like Memory")
	fmt.Println("   - Multi-tier memory with paging and eviction")
	demoOSLikeMemory(ctx)
}

func demoSequentialMemory(ctx context.Context) {
	mem := memory.NewSequentialMemory()

	// Add some messages
	messages := []struct {
		role    string
		content string
	}{
		{"user", "Hello!"},
		{"assistant", "Hi there! How can I help you?"},
		{"user", "What's the weather like?"},
		{"assistant", "I don't have real-time weather data, but I can help you find it!"},
	}

	for _, msg := range messages {
		mem.AddMessage(ctx, memory.NewMessage(msg.role, msg.content))
	}

	// Get all context
	result, _ := mem.GetContext(ctx, "")
	stats, _ := mem.GetStats(ctx)

	fmt.Printf("   Total messages: %d, Active messages: %d\n", stats.TotalMessages, stats.ActiveMessages)
	fmt.Printf("   Latest message: %s\n", result[len(result)-1].Content)
}

func demoSlidingWindowMemory(ctx context.Context) {
	// Keep only last 3 messages
	mem := memory.NewSlidingWindowMemory(3)

	// Add 5 messages
	for i := 1; i <= 5; i++ {
		mem.AddMessage(ctx, memory.NewMessage("user", fmt.Sprintf("Message %d", i)))
	}

	result, _ := mem.GetContext(ctx, "")
	stats, _ := mem.GetStats(ctx)

	fmt.Printf("   Window size: 3, Total added: 5, Kept: %d\n", stats.TotalMessages)
	fmt.Printf("   Messages in window: ")
	for i, msg := range result {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Printf("\"%s\"", msg.Content)
	}
	fmt.Println()
}

func demoBufferMemory(ctx context.Context) {
	// Limit by message count
	mem := memory.NewBufferMemory(&memory.BufferConfig{
		MaxMessages: 3,
		MaxTokens:   1000,
	})

	// Add messages
	for i := 1; i <= 5; i++ {
		mem.AddMessage(ctx, memory.NewMessage("user", fmt.Sprintf("Buffer message %d", i)))
	}

	result, _ := mem.GetContext(ctx, "")
	stats, _ := mem.GetStats(ctx)

	fmt.Printf("   Max messages: 3, Added: 5, Kept: %d\n", len(result))
	fmt.Printf("   Total tokens: %d\n", stats.TotalTokens)
}

func demoSummarizationMemory(ctx context.Context) {
	mem := memory.NewSummarizationMemory(&memory.SummarizationConfig{
		RecentWindowSize: 2, // Keep last 2 messages
		SummarizeAfter:   4, // Summarize after 4 messages
	})

	// Add messages to trigger summarization
	topics := []string{"weather", "sports", "news", "tech", "music"}
	for _, topic := range topics {
		mem.AddMessage(ctx, memory.NewMessage("user", fmt.Sprintf("Tell me about %s", topic)))
		mem.AddMessage(ctx, memory.NewMessage("assistant", fmt.Sprintf("Here's info about %s...", topic)))
	}

	result, _ := mem.GetContext(ctx, "")
	stats, _ := mem.GetStats(ctx)

	fmt.Printf("   Total messages: %d, Active (recent + summary): %d\n", stats.TotalMessages, stats.ActiveMessages)

	// Check if summary was created
	hasSummary := false
	for _, msg := range result {
		if msg.Role == "system" {
			hasSummary = true
			break
		}
	}
	fmt.Printf("   Summary created: %v\n", hasSummary)
}

func demoRetrievalMemory(ctx context.Context) {
	mem := memory.NewRetrievalMemory(&memory.RetrievalConfig{
		TopK: 2, // Retrieve top 2 relevant messages
	})

	// Add messages about different topics
	messages := []struct {
		content string
	}{
		{"Python is a programming language"},
		{"The weather is sunny today"},
		{"Go is great for concurrency"},
		{"It might rain tomorrow"},
		{"JavaScript runs in browsers"},
	}

	for _, msg := range messages {
		mem.AddMessage(ctx, memory.NewMessage("user", msg.content))
	}

	// Query for programming-related content
	result, _ := mem.GetContext(ctx, "programming languages")
	stats, _ := mem.GetStats(ctx)

	fmt.Printf("   Total stored: %d, Query: \"programming languages\", Retrieved: %d\n",
		stats.TotalMessages, len(result))
	if len(result) > 0 {
		fmt.Printf("   Most relevant: \"%s\"\n", result[0].Content)
	}
}

func demoHierarchicalMemory(ctx context.Context) {
	mem := memory.NewHierarchicalMemory(&memory.HierarchicalConfig{
		RecentLimit:    2,
		ImportantLimit: 2,
	})

	// Add messages with varying importance
	messages := []struct {
		content    string
		importance float64
	}{
		{"Regular message 1", 0.3},
		{"IMPORTANT: Remember this key fact", 0.9},
		{"Regular message 2", 0.4},
		{"CRITICAL: System alert", 0.95},
		{"Regular message 3", 0.3},
	}

	for _, msg := range messages {
		m := memory.NewMessage("user", msg.content)
		if msg.importance > 0.7 {
			m.Metadata["importance"] = msg.importance
		}
		mem.AddMessage(ctx, m)
	}

	result, _ := mem.GetContext(ctx, "")
	stats, _ := mem.GetStats(ctx)

	fmt.Printf("   Total: %d, Active (important + recent): %d\n", stats.TotalMessages, len(result))

	// Count important messages in result
	importantCount := 0
	for _, msg := range result {
		if imp, ok := msg.Metadata["importance"].(float64); ok && imp > 0.7 {
			importantCount++
		}
	}
	fmt.Printf("   Important messages in context: %d\n", importantCount)
}

func demoGraphBasedMemory(ctx context.Context) {
	mem := memory.NewGraphBasedMemory(&memory.GraphConfig{
		TopK: 3,
	})

	// Add related messages
	messages := []string{
		"What's the price of the product?",
		"The price is $99",
		"Tell me about the features",
		"What's the price again?",
		"Does it have a warranty?",
	}

	for _, content := range messages {
		mem.AddMessage(ctx, memory.NewMessage("user", content))
	}

	// Query for price-related messages
	result, _ := mem.GetContext(ctx, "price information")
	stats, _ := mem.GetStats(ctx)

	fmt.Printf("   Total: %d, Query: \"price information\", Retrieved: %d\n",
		stats.TotalMessages, len(result))

	relations := mem.GetRelationships()
	fmt.Printf("   Topic relationships tracked: %d\n", len(relations))
}

func demoCompressionMemory(ctx context.Context) {
	mem := memory.NewCompressionMemory(&memory.CompressionConfig{
		CompressionTrigger: 3, // Compress after 3 messages
	})

	// Add messages to trigger compression
	for i := 1; i <= 7; i++ {
		mem.AddMessage(ctx, memory.NewMessage("user", fmt.Sprintf("Message %d with some content", i)))
	}

	result, _ := mem.GetContext(ctx, "")
	stats, _ := mem.GetStats(ctx)

	fmt.Printf("   Total original: %d, Active (blocks + recent): %d\n",
		stats.TotalMessages, stats.ActiveMessages)
	fmt.Printf("   Compression rate: %.2f\n", stats.CompressionRate)

	// Check for compressed blocks
	hasCompressed := false
	for _, msg := range result {
		if msg.Role == "system" {
			hasCompressed = true
			break
		}
	}
	fmt.Printf("   Compressed blocks: %v\n", hasCompressed)
}

func demoOSLikeMemory(ctx context.Context) {
	mem := memory.NewOSLikeMemory(&memory.OSLikeConfig{
		ActiveLimit:  2, // 2 pages in active memory
		CacheLimit:   3, // 3 pages in cache
		AccessWindow: time.Minute * 5,
	})

	// Add messages over time
	for i := 1; i <= 15; i++ {
		mem.AddMessage(ctx, memory.NewMessage("user", fmt.Sprintf("Message %d", i)))
	}

	result, _ := mem.GetContext(ctx, "")
	stats, _ := mem.GetStats(ctx)
	info := mem.GetMemoryInfo()

	fmt.Printf("   Total: %d, Active: %d\n", stats.TotalMessages, stats.ActiveMessages)
	fmt.Printf("   Memory tiers - Active: %d, Cache: %d, Archive: %d pages\n",
		info["active_pages"].(int),
		info["cached_pages"].(int),
		info["archived_pages"].(int))
	fmt.Printf("   Retrieved from active memory: %d messages\n", len(result))
}
