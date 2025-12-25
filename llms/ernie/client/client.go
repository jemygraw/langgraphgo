package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	ErrNotSetAuth      = errors.New("API key not set")
	ErrEmptyResponse   = errors.New("empty response")
	ErrInvalidResponse = errors.New("invalid response")
)

const (
	defaultChatEndpoint      = "/v2/chat/completions"
	defaultEmbeddingEndpoint = "/v2/embeddings"
)

// Client is a client for Baidu Qianfan API using API Key authentication.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// Option is a function that configures a Client.
type Option func(*clientOptions)

type clientOptions struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// WithAPIKey sets the API key for the client.
func WithAPIKey(apiKey string) Option {
	return func(opts *clientOptions) {
		opts.apiKey = apiKey
	}
}

// WithBaseURL sets the base URL for the API.
func WithBaseURL(baseURL string) Option {
	return func(opts *clientOptions) {
		opts.baseURL = baseURL
	}
}

// WithHTTPClient sets the HTTP client for the API.
func WithHTTPClient(client *http.Client) Option {
	return func(opts *clientOptions) {
		opts.httpClient = client
	}
}

// New creates a new Client with the given options.
func New(opts ...Option) (*Client, error) {
	options := &clientOptions{
		baseURL:    "https://qianfan.baidubce.com",
		httpClient: http.DefaultClient,
	}

	for _, opt := range opts {
		opt(options)
	}

	if options.apiKey == "" {
		return nil, ErrNotSetAuth
	}

	return &Client{
		apiKey:     options.apiKey,
		baseURL:    strings.TrimSuffix(options.baseURL, "/"),
		httpClient: options.httpClient,
	}, nil
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompletionRequest represents a request to the chat completion API.
type CompletionRequest struct {
	Model         string                                        `json:"model"`
	Messages      []Message                                     `json:"messages"`
	Temperature   float64                                       `json:"temperature,omitempty"`
	TopP          float64                                       `json:"top_p,omitempty"`
	PenaltyScore  float64                                       `json:"penalty_score,omitempty"`
	MaxTokens     int                                           `json:"max_tokens,omitempty"`
	Stream        bool                                          `json:"stream,omitempty"`
	StreamingFunc func(ctx context.Context, chunk []byte) error `json:"-"`
}

// CompletionResponse represents a response from the chat completion API.
type CompletionResponse struct {
	ID        string   `json:"id"`
	Object    string   `json:"object"`
	Created   int64    `json:"created"`
	Result    string   `json:"result,omitempty"`
	Choices   []Choice `json:"choices"`
	Usage     Usage    `json:"usage"`
	ErrorCode int      `json:"error_code,omitempty"`
	ErrorMsg  string   `json:"error_msg,omitempty"`
}

// Choice represents a choice in the completion response.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	Delta        Delta   `json:"delta"`
	FinishReason string  `json:"finish_reason"`
}

// Delta represents the delta content in streaming responses.
type Delta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// Usage represents token usage information.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// EmbeddingRequest represents a request to the embedding API.
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbeddingResponse represents a response from the embedding API.
type EmbeddingResponse struct {
	ID        string      `json:"id"`
	Object    string      `json:"object"`
	Created   int64       `json:"created"`
	Data      []EmbedData `json:"data"`
	Model     string      `json:"model"`
	Usage     Usage       `json:"usage"`
	ErrorCode int         `json:"error_code,omitempty"`
	ErrorMsg  string      `json:"error_msg,omitempty"`
}

// EmbedData represents embedding data in the response.
type EmbedData struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

// CreateCompletion sends a chat completion request.
func (c *Client) CreateCompletion(ctx context.Context, model string, req *CompletionRequest) (*CompletionResponse, error) {
	req.Model = model

	if req.Stream {
		return c.createStreamingCompletion(ctx, req)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + defaultChatEndpoint
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result CompletionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Handle OpenAI-compatible format with choices
	if len(result.Choices) > 0 {
		result.Result = result.Choices[0].Message.Content
	}

	return &result, nil
}

func (c *Client) createStreamingCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + defaultChatEndpoint
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var fullContent strings.Builder
	var usage Usage

	for {
		var line string
		if _, err := fmt.Fscanf(resp.Body, "data: %s\n", &line); err != nil {
			if err == io.EOF {
				break
			}
			// Skip lines that don't match the SSE format
			continue
		}

		// Check for [DONE] marker
		if strings.TrimSpace(line) == "[DONE]" {
			break
		}

		var chunk struct {
			ID      string   `json:"id"`
			Object  string   `json:"object"`
			Created int64    `json:"created"`
			Choices []Choice `json:"choices"`
			Usage   *Usage   `json:"usage,omitempty"`
		}

		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta.Content
			fullContent.WriteString(delta)

			if req.StreamingFunc != nil {
				if err := req.StreamingFunc(ctx, []byte(delta)); err != nil {
					return nil, fmt.Errorf("streaming function error: %w", err)
				}
			}
		}

		if chunk.Usage != nil {
			usage = *chunk.Usage
		}
	}

	result := &CompletionResponse{
		Result: fullContent.String(),
		Usage:  usage,
	}

	return result, nil
}

// CreateEmbedding sends an embedding request.
func (c *Client) CreateEmbedding(ctx context.Context, model string, texts []string) (*EmbeddingResponse, error) {
	if len(texts) == 0 {
		return nil, errors.New("texts cannot be empty")
	}

	req := EmbeddingRequest{
		Model: model,
		Input: texts,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + defaultEmbeddingEndpoint
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result EmbeddingResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
}
