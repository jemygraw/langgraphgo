package prebuilt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

func TestCreateReactAgentWithCustomStateTyped_ToolExecutionError(t *testing.T) {
	type CustomState struct {
		Messages       []llms.MessageContent
		IterationCount int
	}

	tool := &MockToolError{name: "error_tool"}          // Use the mock tool that returns an error
	mockLLM := NewMockLLMWithToolCalls([]llms.ToolCall{ // LLM returns a tool call
		{
			ID: "call_1",
			FunctionCall: &llms.FunctionCall{
				Name:      "error_tool",
				Arguments: `{"input":"some input"}`,
			},
		},
	})

	getMessages := func(s CustomState) []llms.MessageContent { return s.Messages }
	setMessages := func(s CustomState, msgs []llms.MessageContent) CustomState {
		s.Messages = append(s.Messages, msgs...)
		return s
	}
	getIterationCount := func(s CustomState) int { return s.IterationCount }
	setIterationCount := func(s CustomState, count int) CustomState {
		s.IterationCount = count
		return s
	}
	hasToolCalls := func(msgs []llms.MessageContent) bool {
		if len(msgs) == 0 {
			return false
		}
		lastMsg := msgs[len(msgs)-1]
		for _, part := range lastMsg.Parts {
			if _, ok := part.(llms.ToolCall); ok {
				return true
			}
		}
		return false
	}

	agent, err := CreateReactAgentWithCustomStateTyped(
		mockLLM,
		[]tools.Tool{tool},
		getMessages,
		setMessages,
		getIterationCount,
		setIterationCount,
		hasToolCalls,
		3,
	)
	require.NoError(t, err)
	require.NotNil(t, agent)

	initialState := CustomState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Test"),
		},
	}

	_, err = agent.Invoke(context.Background(), initialState)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tool execution failed: mock tool execution error")
}
