package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestClientNew tests the Client creation with various options.
func TestClientNew(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "no api key",
			opts:    []Option{},
			wantErr: true,
		},
		{
			name: "with api key",
			opts: []Option{
				WithAPIKey("test-key"),
			},
			wantErr: false,
		},
		{
			name: "with api key and base url",
			opts: []Option{
				WithAPIKey("test-key"),
				WithBaseURL("https://custom.example.com"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("New() returned nil client")
			}
		})
	}
}

// TestClientHeaders tests that the correct headers are set.
func TestClientHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Errorf("Expected Authorization header to start with 'Bearer ', got: %s", auth)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got: %s", contentType)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test","object":"chat.completion","created":123456,"choices":[{"index":0,"message":{"role":"assistant","content":"Hello!"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`))
	}))
	defer server.Close()

	client, err := New(WithAPIKey("test-key"), WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	_, err = client.CreateCompletion(context.Background(), "test-model", &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("Failed to create completion: %v", err)
	}
}

// TestClientCreateEmbedding_RealAPI tests embedding generation with real API.
// Skipped if QIANFAN_TOKEN is not set.
func TestClientCreateEmbedding_RealAPI(t *testing.T) {
	apiKey := os.Getenv("QIANFAN_TOKEN")
	if apiKey == "" {
		t.Skip("QIANFAN_TOKEN not set")
	}

	client, err := New(WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("single text", func(t *testing.T) {
		resp, err := client.CreateEmbedding(context.Background(), "embedding-v1", []string{"Hello world"})
		if err != nil {
			t.Fatalf("Failed to create embedding: %v", err)
		}
		if len(resp.Data) != 1 {
			t.Fatalf("Expected 1 embedding, got %d", len(resp.Data))
		}
		if len(resp.Data[0].Embedding) == 0 {
			t.Fatal("Empty embedding")
		}
		t.Logf("Embedding dimension: %d", len(resp.Data[0].Embedding))
	})

	t.Run("multiple texts", func(t *testing.T) {
		resp, err := client.CreateEmbedding(context.Background(), "embedding-v1", []string{"Hello", "World"})
		if err != nil {
			t.Fatalf("Failed to create embedding: %v", err)
		}
		if len(resp.Data) != 2 {
			t.Fatalf("Expected 2 embeddings, got %d", len(resp.Data))
		}
		for i, data := range resp.Data {
			if len(data.Embedding) == 0 {
				t.Errorf("Empty embedding at index %d", i)
			}
			t.Logf("Embedding %d dimension: %d", i, len(data.Embedding))
		}
	})
}

// TestClientCreateCompletion_RealAPI tests chat completion with real API.
// Skipped if QIANFAN_TOKEN is not set.
func TestClientCreateCompletion_RealAPI(t *testing.T) {
	apiKey := os.Getenv("QIANFAN_TOKEN")
	if apiKey == "" {
		t.Skip("QIANFAN_TOKEN not set")
	}

	client, err := New(WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	resp, err := client.CreateCompletion(context.Background(), "ernie-speed-8k", &CompletionRequest{
		Messages: []Message{
			{Role: "user", Content: "Hello, how are you?"},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create completion: %v", err)
	}

	if resp.Result == "" && len(resp.Choices) == 0 {
		t.Fatal("Empty result and no choices")
	}

	t.Logf("Result: %s", resp.Result)
	t.Logf("Usage: %+v", resp.Usage)
}

// TestClientCreateCompletion_Messages tests message handling.
func TestClientCreateCompletion_Messages(t *testing.T) {
	apiKey := os.Getenv("QIANFAN_TOKEN")
	if apiKey == "" {
		t.Skip("QIANFAN_TOKEN not set")
	}

	client, err := New(WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tests := []struct {
		name     string
		messages []Message
	}{
		{
			name: "single user message",
			messages: []Message{
				{Role: "user", Content: "What is 2+2?"},
			},
		},
		{
			name: "multi-turn conversation",
			messages: []Message{
				{Role: "user", Content: "My name is Bob"},
				{Role: "assistant", Content: "Hello Bob!"},
				{Role: "user", Content: "What's my name?"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.CreateCompletion(context.Background(), "ernie-speed-8k", &CompletionRequest{
				Messages: tt.messages,
			})
			if err != nil {
				t.Fatalf("Failed to create completion: %v", err)
			}

			if resp.Result == "" && len(resp.Choices) == 0 {
				t.Error("Empty result and no choices")
			}

			t.Logf("Response: %s", resp.Result)
		})
	}
}

// TestClientCreateCompletion_Temperature tests temperature parameter.
func TestClientCreateCompletion_Temperature(t *testing.T) {
	apiKey := os.Getenv("QIANFAN_TOKEN")
	if apiKey == "" {
		t.Skip("QIANFAN_TOKEN not set")
	}

	client, err := New(WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	resp, err := client.CreateCompletion(context.Background(), "ernie-speed-8k", &CompletionRequest{
		Messages: []Message{
			{Role: "user", Content: "Tell me a joke"},
		},
		Temperature: 0.9,
	})
	if err != nil {
		t.Fatalf("Failed to create completion: %v", err)
	}

	if resp.Result == "" {
		t.Error("Empty result")
	}

	t.Logf("Response with temperature 0.9: %s", resp.Result)
}

// TestClientCreateCompletion_MaxTokens tests max_tokens parameter.
func TestClientCreateCompletion_MaxTokens(t *testing.T) {
	apiKey := os.Getenv("QIANFAN_TOKEN")
	if apiKey == "" {
		t.Skip("QIANFAN_TOKEN not set")
	}

	client, err := New(WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	resp, err := client.CreateCompletion(context.Background(), "ernie-speed-8k", &CompletionRequest{
		Messages: []Message{
			{Role: "user", Content: "Tell me a short story"},
		},
		MaxTokens: 50,
	})
	if err != nil {
		t.Fatalf("Failed to create completion: %v", err)
	}

	if resp.Result == "" {
		t.Error("Empty result")
	}

	t.Logf("Response (max 50 tokens): %s", resp.Result)
	t.Logf("Tokens used: %d", resp.Usage.CompletionTokens)
}

// TestClient_EmptyInput tests error handling for empty input.
func TestClient_EmptyInput(t *testing.T) {
	client, err := New(WithAPIKey("test-key"))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("CreateEmbedding with empty texts", func(t *testing.T) {
		_, err := client.CreateEmbedding(context.Background(), "embedding-v1", []string{})
		if err == nil {
			t.Error("Expected error for empty texts, got nil")
		}
	})
}
