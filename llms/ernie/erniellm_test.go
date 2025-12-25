package ernie

import (
	"context"
	"os"
	"testing"

	"github.com/tmc/langchaingo/llms"
)

// TestLLM_Create tests the LLM creation with various options.
func TestLLM_Create(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name: "with api key",
			opts: []Option{
				WithAPIKey("test-key"),
			},
			wantErr: false,
		},
		{
			name: "with api key and model",
			opts: []Option{
				WithAPIKey("test-key"),
				WithModel(ModelNameERNIESpeed8K),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm, err := New(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && llm == nil {
				t.Error("New() returned nil LLM")
			}
		})
	}
}

// TestLLM_GenerateContent tests the content generation with real API.
// Skipped if QIANFAN_TOKEN is not set.
func TestLLM_GenerateContent(t *testing.T) {
	apiKey := os.Getenv("QIANFAN_TOKEN")
	if apiKey == "" {
		t.Skip("QIANFAN_TOKEN not set")
	}

	llm, err := New(
		WithAPIKey(apiKey),
		WithModel(ModelNameERNIESpeed8K),
	)
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	ctx := context.Background()
	messages := []llms.MessageContent{
		{
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextPart("Hello, how are you?"),
			},
		},
	}

	resp, err := llm.GenerateContent(ctx, messages)
	if err != nil {
		t.Fatalf("Failed to generate content: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("No choices in response")
	}

	content := resp.Choices[0].Content
	if content == "" {
		t.Error("Empty response content")
	}

	t.Logf("Response: %s", content)
	t.Logf("StopReason: %s", resp.Choices[0].StopReason)

	// Check GenerationInfo for token usage
	if resp.Choices[0].GenerationInfo != nil {
		if totalTokens, ok := resp.Choices[0].GenerationInfo["total_tokens"].(int); ok {
			t.Logf("Total tokens: %d", totalTokens)
		}
	}
}

// TestLLM_CreateEmbedding tests the embedding generation with real API.
// Skipped if QIANFAN_TOKEN is not set.
func TestLLM_CreateEmbedding(t *testing.T) {
	apiKey := os.Getenv("QIANFAN_TOKEN")
	if apiKey == "" {
		t.Skip("QIANFAN_TOKEN not set")
	}

	llm, err := New(
		WithAPIKey(apiKey),
		WithModel(ModelNameEmbeddingV1),
	)
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	ctx := context.Background()
	texts := []string{"Hello world"}

	embeddings, err := llm.CreateEmbedding(ctx, texts)
	if err != nil {
		t.Fatalf("Failed to create embedding: %v", err)
	}

	if len(embeddings) != 1 {
		t.Fatalf("Expected 1 embedding, got %d", len(embeddings))
	}

	if len(embeddings[0]) == 0 {
		t.Fatal("Empty embedding")
	}

	t.Logf("Embedding dimension: %d", len(embeddings[0]))
}

// TestLLM_CreateEmbeddingMultiple tests embedding generation for multiple texts.
func TestLLM_CreateEmbeddingMultiple(t *testing.T) {
	apiKey := os.Getenv("QIANFAN_TOKEN")
	if apiKey == "" {
		t.Skip("QIANFAN_TOKEN not set")
	}

	llm, err := New(
		WithAPIKey(apiKey),
		WithModel(ModelNameEmbeddingV1),
	)
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	ctx := context.Background()
	texts := []string{"Hello", "World"}

	embeddings, err := llm.CreateEmbedding(ctx, texts)
	if err != nil {
		t.Fatalf("Failed to create embedding: %v", err)
	}

	if len(embeddings) != 2 {
		t.Fatalf("Expected 2 embeddings, got %d", len(embeddings))
	}

	for i, emb := range embeddings {
		if len(emb) == 0 {
			t.Errorf("Empty embedding at index %d", i)
		}
		t.Logf("Embedding %d dimension: %d", i, len(emb))
	}
}

// TestLLM_Call tests the Call method.
func TestLLM_Call(t *testing.T) {
	apiKey := os.Getenv("QIANFAN_TOKEN")
	if apiKey == "" {
		t.Skip("QIANFAN_TOKEN not set")
	}

	llm, err := New(
		WithAPIKey(apiKey),
		WithModel(ModelNameERNIESpeed8K),
	)
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	ctx := context.Background()
	response, err := llm.Call(ctx, "What is 2+2?")
	if err != nil {
		t.Fatalf("Failed to call LLM: %v", err)
	}

	if response == "" {
		t.Error("Empty response")
	}

	t.Logf("Response: %s", response)
}

// TestLLM_ModelMapping tests model name mapping.
func TestLLM_ModelMapping(t *testing.T) {
	tests := []struct {
		name     string
		model    ModelName
		expected string
	}{
		// 推荐模型
		{"ERNIE 5.0 Thinking Preview", ModelNameERNIE5ThinkingPreview, "ernie-5.0-thinking-preview"},
		{"ERNIE 4.5 Turbo 128K", ModelNameERNIE45Turbo128K, "ernie-4.5-turbo-128k"},
		{"DeepSeek R1", ModelNameDeepSeekR1, "deepseek-r1"},

		// ERNIE系列
		{"ERNIE Speed 8K", ModelNameERNIESpeed8K, "ernie-speed-8k"},
		{"ERNIE Lite 8K", ModelNameERNIELite8K, "ernie-lite-8k"},
		{"ERNIE Tiny 8K", ModelNameERNIETiny8K, "ernie-tiny-8k"},

		// DeepSeek系列
		{"DeepSeek V3", ModelNameDeepSeekV3, "deepseek-v3"},
		{"DeepSeek V3.2", ModelNameDeepSeekV32, "deepseek-v3.2"},

		// Qwen系列
		{"Qwen3 8B", ModelNameQwen38B, "qwen3-8b"},
		{"Qwen3 32B", ModelNameQwen332B, "qwen3-32b"},

		// Embedding模型
		{"Embedding V1", ModelNameEmbeddingV1, "embedding-v1"},
		{"BGE Large ZH", ModelNameBgeLargeZh, "bge-large-zh"},
		{"BGE Large EN", ModelNameBgeLargeEn, "bge-large-en"},
		{"Tao 8k", ModelNameTao8k, "tao-8k"},
		{"Qwen3 Embedding 0.6B", ModelNameQwen3Embedding06B, "qwen3-embedding-0.6b"},
		{"Qwen3 Embedding 4B", ModelNameQwen3Embedding4B, "qwen3-embedding-4b"},

		// 兼容旧版本
		{"ERNIE Bot (legacy)", ModelNameERNIEBot, "ernie-speed-8k"},
		{"ERNIE Bot Turbo (legacy)", ModelNameERNIEBotTurbo, "ernie-speed-8k"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := modelToModelString(tt.model)
			if result != tt.expected {
				t.Errorf("modelToModelString(%v) = %s, want %s", tt.model, result, tt.expected)
			}
		})
	}
}

// TestLLM_Conversation tests a multi-turn conversation.
func TestLLM_Conversation(t *testing.T) {
	apiKey := os.Getenv("QIANFAN_TOKEN")
	if apiKey == "" {
		t.Skip("QIANFAN_TOKEN not set")
	}

	llm, err := New(
		WithAPIKey(apiKey),
		WithModel(ModelNameERNIESpeed8K),
	)
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	ctx := context.Background()
	messages := []llms.MessageContent{
		{
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextPart("My name is Alice"),
			},
		},
		{
			Role: llms.ChatMessageTypeAI,
			Parts: []llms.ContentPart{
				llms.TextPart("Hello Alice! Nice to meet you."),
			},
		},
		{
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextPart("What's my name?"),
			},
		},
	}

	resp, err := llm.GenerateContent(ctx, messages)
	if err != nil {
		t.Fatalf("Failed to generate content: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("No choices in response")
	}

	content := resp.Choices[0].Content
	t.Logf("Response: %s", content)

	// Check if the model remembers the name
	containsName := false
	lowerContent := content
	if len(lowerContent) > 0 {
		// Simple check for name presence
		for i := 0; i <= len(lowerContent)-5; i++ {
			if lowerContent[i:i+5] == "Alice" {
				containsName = true
				break
			}
		}
	}
	t.Logf("Model remembers name: %v", containsName)
}

// TestLLM_DifferentModels tests different model types.
func TestLLM_DifferentModels(t *testing.T) {
	apiKey := os.Getenv("QIANFAN_TOKEN")
	if apiKey == "" {
		t.Skip("QIANFAN_TOKEN not set")
	}

	models := []struct {
		name ModelName
		desc string
	}{
		{ModelNameERNIESpeed8K, "ERNIE Speed 8K - fast response"},
		{ModelNameERNIETiny8K, "ERNIE Tiny 8K - lightweight"},
		{ModelNameERNIELite8K, "ERNIE Lite 8K - basic"},
	}

	for _, m := range models {
		t.Run(m.desc, func(t *testing.T) {
			llm, err := New(
				WithAPIKey(apiKey),
				WithModel(m.name),
			)
			if err != nil {
				t.Fatalf("Failed to create LLM: %v", err)
			}

			ctx := context.Background()
			response, err := llm.Call(ctx, "Say hello")
			if err != nil {
				t.Logf("Model %s error: %v", m.name, err)
				return
			}

			if response == "" {
				t.Errorf("Model %s returned empty response", m.name)
			}

			t.Logf("Model %s response: %s", m.name, response)
		})
	}
}

// TestLLM_EmbeddingModels tests different embedding models.
func TestLLM_EmbeddingModels(t *testing.T) {
	apiKey := os.Getenv("QIANFAN_TOKEN")
	if apiKey == "" {
		t.Skip("QIANFAN_TOKEN not set")
	}

	models := []struct {
		name        ModelName
		expectedDim int
		description string
	}{
		{ModelNameEmbeddingV1, 384, "Embedding V1 - 384 dimensions"},
		{ModelNameBgeLargeZh, 1024, "BGE Large ZH - 1024 dimensions"},
		{ModelNameTao8k, 1024, "Tao 8k - 1024 dimensions"},
	}

	for _, m := range models {
		t.Run(m.description, func(t *testing.T) {
			llm, err := New(
				WithAPIKey(apiKey),
				WithModel(m.name),
			)
			if err != nil {
				t.Fatalf("Failed to create LLM: %v", err)
			}

			ctx := context.Background()
			embeddings, err := llm.CreateEmbedding(ctx, []string{"test text"})
			if err != nil {
				t.Logf("Model %s error: %v", m.name, err)
				return
			}

			if len(embeddings) != 1 {
				t.Fatalf("Expected 1 embedding, got %d", len(embeddings))
			}

			dim := len(embeddings[0])
			if dim != m.expectedDim {
				t.Errorf("Model %s: expected dimension %d, got %d", m.name, m.expectedDim, dim)
			}

			t.Logf("Model %s: dimension = %d", m.name, dim)
		})
	}
}
