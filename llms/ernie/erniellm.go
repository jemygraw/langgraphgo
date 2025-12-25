package ernie

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/llms"

	"github.com/smallnest/langgraphgo/llms/ernie/client"
)

var (
	ErrEmptyResponse = errors.New("no response")
	ErrCodeResponse  = errors.New("has error code")
)

// LLM is a client for Baidu Qianfan (Ernie) LLM.
type LLM struct {
	client           *client.Client
	model            ModelName
	CallbacksHandler callbacks.Handler
}

var _ llms.Model = (*LLM)(nil)

// New returns a new Ernie LLM client using API Key authentication.
//
// Authentication options:
// 1. WithAPIKey(apiKey) - pass API key directly
// 2. Set ERNIE_API_KEY environment variable
//
// Example:
//
//	llm, err := ernie.New(
//		ernie.WithAPIKey("your-api-key"),
//		ernie.WithModel(ernie.ModelNameERNIE4),
//	)
func New(opts ...Option) (*LLM, error) {
	options := &options{
		apiKey:    getEnvOrDefault("ERNIE_API_KEY", ""),
		modelName: ModelNameERNIESpeed8K, // 默认使用ERNIE Speed 8K
		baseURL:   "https://qianfan.baidubce.com",
	}

	for _, opt := range opts {
		opt(options)
	}

	if options.apiKey == "" {
		return nil, fmt.Errorf(`%w
You can pass auth info by using ernie.New(ernie.WithAPIKey("{API Key}"))
or
export ERNIE_API_KEY={API Key}
doc: https://cloud.baidu.com/doc/qianfan-api/s/3m9b5lqft`, client.ErrNotSetAuth)
	}

	clientOpts := []client.Option{
		client.WithAPIKey(options.apiKey),
		client.WithBaseURL(options.baseURL),
	}

	if options.httpClient != nil {
		clientOpts = append(clientOpts, client.WithHTTPClient(options.httpClient))
	}

	c, err := client.New(clientOpts...)
	if err != nil {
		return nil, err
	}

	return &LLM{
		client:           c,
		model:            options.modelName,
		CallbacksHandler: options.callbacksHandler,
	}, nil
}

// Call generates a response from the LLM for the given prompt.
func (o *LLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return llms.GenerateFromSinglePrompt(ctx, o, prompt, options...)
}

// GenerateContent implements the Model interface.
func (o *LLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if o.CallbacksHandler != nil {
		o.CallbacksHandler.HandleLLMGenerateContentStart(ctx, messages)
	}

	opts := &llms.CallOptions{}
	for _, opt := range options {
		opt(opts)
	}

	// Convert messages to Ernie format
	ernieMessages := make([]client.Message, 0, len(messages))
	for _, msg := range messages {
		role := string(msg.Role)
		// Map ChatMessageType to Ernie role format
		switch role {
		case "":
			role = "user"
		case "human":
			role = "user"
		case "ai":
			role = "assistant"
		case "system":
			role = "system"
		}

		var content strings.Builder
		for _, part := range msg.Parts {
			if text, ok := part.(llms.TextContent); ok {
				content.WriteString(text.Text)
			}
		}

		ernieMessages = append(ernieMessages, client.Message{
			Role:    role,
			Content: content.String(),
		})
	}

	result, err := o.client.CreateCompletion(ctx, o.getModelString(*opts), &client.CompletionRequest{
		Messages:      ernieMessages,
		Temperature:   opts.Temperature,
		TopP:          opts.TopP,
		PenaltyScore:  opts.RepetitionPenalty,
		StreamingFunc: opts.StreamingFunc,
		Stream:        opts.StreamingFunc != nil,
		MaxTokens:     int(opts.MaxTokens),
	})
	if err != nil {
		if o.CallbacksHandler != nil {
			o.CallbacksHandler.HandleLLMError(ctx, err)
		}
		return nil, err
	}

	if result.ErrorCode > 0 {
		err = fmt.Errorf("%w, error_code:%v, error_msg:%v, id:%v",
			ErrCodeResponse, result.ErrorCode, result.ErrorMsg, result.ID)
		if o.CallbacksHandler != nil {
			o.CallbacksHandler.HandleLLMError(ctx, err)
		}
		return nil, err
	}

	resp := &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content:    result.Result,
				StopReason: "", // Will be set if available
			},
		},
	}

	// Set StopReason from Choices if available
	if len(result.Choices) > 0 && result.Choices[0].FinishReason != "" {
		resp.Choices[0].StopReason = result.Choices[0].FinishReason
	} else if result.Result == "" {
		resp.Choices[0].StopReason = "stop"
	}

	// Add usage information to GenerationInfo
	if result.Usage.TotalTokens > 0 {
		resp.Choices[0].GenerationInfo = map[string]any{
			"prompt_tokens":     result.Usage.PromptTokens,
			"completion_tokens": result.Usage.CompletionTokens,
			"total_tokens":      result.Usage.TotalTokens,
		}
	} else {
		resp.Choices[0].GenerationInfo = make(map[string]any)
	}

	if o.CallbacksHandler != nil {
		o.CallbacksHandler.HandleLLMGenerateContentEnd(ctx, resp)
	}

	return resp, nil
}

// CreateEmbedding generates embeddings for the given texts using Ernie embedding models.
//
// The embedding model has the following limitations:
//   - Embedding-V1: token count <= 384, text length <= 1000 characters
//   - bge-large-zh or bge-large-en: token count <= 512, text length <= 2000 characters
//   - tao-8k: token count <= 8192, text length <= 28000 characters
//   - Qwen3-Embedding-0.6B or Qwen3-Embedding-4B: max 8k tokens per text
//
// Text count limits:
//   - tao-8k: only 1 text
//   - Others: max 16 texts
//
// API documentation: https://cloud.baidu.com/doc/qianfan-api/s/Fm7u3ropn
func (o *LLM) CreateEmbedding(ctx context.Context, texts []string) ([][]float32, error) {
	resp, err := o.client.CreateEmbedding(ctx, o.getModelString(llms.CallOptions{}), texts)
	if err != nil {
		return nil, err
	}

	if resp.ErrorCode > 0 {
		return nil, fmt.Errorf("%w, error_code:%v, error_msg:%v, id:%v",
			ErrCodeResponse, resp.ErrorCode, resp.ErrorMsg, resp.ID)
	}

	emb := make([][]float32, 0, len(resp.Data))
	for i := range resp.Data {
		emb = append(emb, resp.Data[i].Embedding)
	}

	return emb, nil
}

func (o *LLM) getModelString(opts llms.CallOptions) string {
	model := o.model

	if model == "" {
		model = ModelName(opts.Model)
	}

	return modelToModelString(model)
}

// modelToModelString returns the model string for API calls.
// Since ModelName constants already use the correct API IDs, this just returns the string value.
func modelToModelString(model ModelName) string {
	return string(model)
}
