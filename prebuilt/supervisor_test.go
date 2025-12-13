package prebuilt

import (
	"context"
	"errors"
	"testing"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

// SupervisorMockLLM for supervisor testing
type SupervisorMockLLM struct {
	responses   []llms.ContentResponse
	currentIdx  int
	returnError error
}

func (m *SupervisorMockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	if m.currentIdx >= len(m.responses) {
		return &llms.ContentResponse{
			Choices: []*llms.ContentChoice{
				{Content: "No more responses"},
			},
		}, nil
	}
	resp := m.responses[m.currentIdx]
	m.currentIdx++
	return &resp, nil
}

func (m *SupervisorMockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return "", nil
}

// MockAgent for testing supervisor with various behaviors
type MockAgent struct {
	name        string
	response    string
	shouldError bool
	errorMsg    string
}

func NewMockAgent(name, response string) *MockAgent {
	return &MockAgent{
		name:     name,
		response: response,
	}
}

func NewMockErrorAgent(name, errorMsg string) *MockAgent {
	return &MockAgent{
		name:        name,
		shouldError: true,
		errorMsg:    errorMsg,
	}
}

func (a *MockAgent) Invoke(ctx context.Context, state any) (any, error) {
	if a.shouldError {
		return nil, errors.New(a.errorMsg)
	}

	// Extract existing messages
	mState, ok := state.(map[string]any)
	if !ok {
		return nil, errors.New("invalid state type")
	}

	messages, ok := mState["messages"].([]llms.MessageContent)
	if !ok {
		return nil, errors.New("messages key not found or invalid type")
	}

	// Append agent response
	newMessages := append(messages, llms.TextParts(llms.ChatMessageTypeAI, a.response))

	return map[string]any{
		"messages": newMessages,
	}, nil
}

func (a *MockAgent) Compile() (*graph.StateRunnable, error) {
	workflow := graph.NewStateGraph()

	// Define state schema
	schema := graph.NewMapSchema()
	schema.RegisterReducer("messages", graph.AppendReducer)
	workflow.SetSchema(schema)

	workflow.AddNode("run", "Agent run node", a.Invoke)
	workflow.SetEntryPoint("run")
	workflow.AddEdge("run", graph.END)

	return workflow.Compile()
}

func TestCreateSupervisor_DirectFinish(t *testing.T) {
	// Test supervisor that directly routes to FINISH
	mockLLM := &SupervisorMockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								FunctionCall: &llms.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "FINISH"}`,
								},
							},
						},
					},
				},
			},
		},
	}

	agent := NewMockAgent("Agent", "Should not be called")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable{
		"Agent": agentRunnable,
	}
	supervisor, err := CreateSupervisor(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Complete immediately"),
		},
	}

	res, err := supervisor.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	mState := res.(map[string]any)
	messages := mState["messages"].([]llms.MessageContent)
	// Should only have initial message, no agent responses
	assert.Equal(t, 1, len(messages))
	assert.Equal(t, "Complete immediately", messages[0].Parts[0].(llms.TextContent).Text)
	assert.Equal(t, "FINISH", mState["next"])
}

func TestCreateSupervisor_AgentError(t *testing.T) {
	// Test handling of agent errors
	mockLLM := &SupervisorMockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								FunctionCall: &llms.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "ErrorAgent"}`,
								},
							},
						},
					},
				},
			},
		},
	}

	errorAgent := NewMockErrorAgent("ErrorAgent", "Agent failed to process")
	errorAgentRunnable, err := errorAgent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable{
		"ErrorAgent": errorAgentRunnable,
	}
	supervisor, err := CreateSupervisor(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Trigger error"),
		},
	}

	// Should return error from agent
	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Agent failed to process")
}

func TestCreateSupervisor_NoToolCall(t *testing.T) {
	// Test when LLM doesn't make a tool call
	mockLLM := &SupervisorMockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{
						Content: "I don't know what to do",
					},
				},
			},
		},
	}

	agent := NewMockAgent("Agent", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable{
		"Agent": agentRunnable,
	}
	supervisor, err := CreateSupervisor(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Test"),
		},
	}

	// Should return error about not selecting next step
	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "supervisor did not select a next step")
}

func TestCreateSupervisor_InvalidRouteArguments(t *testing.T) {
	// Test when route tool has invalid JSON
	mockLLM := &SupervisorMockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								FunctionCall: &llms.FunctionCall{
									Name:      "route",
									Arguments: `{invalid json`,
								},
							},
						},
					},
				},
			},
		},
	}

	agent := NewMockAgent("Agent", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable{
		"Agent": agentRunnable,
	}
	supervisor, err := CreateSupervisor(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Test"),
		},
	}

	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse route arguments")
}

func TestCreateSupervisor_InvalidStateType(t *testing.T) {
	mockLLM := &SupervisorMockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								FunctionCall: &llms.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "Agent1"}`,
								},
							},
						},
					},
				},
			},
		},
	}

	agent := NewMockAgent("Agent1", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable{
		"Agent1": agentRunnable,
	}
	supervisor, err := CreateSupervisor(mockLLM, members)
	assert.NoError(t, err)

	// Pass invalid state (string instead of map)
	_, err = supervisor.Invoke(context.Background(), "invalid state")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid state type")
}

func TestCreateSupervisor_MissingMessages(t *testing.T) {
	mockLLM := &SupervisorMockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								FunctionCall: &llms.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "Agent1"}`,
								},
							},
						},
					},
				},
			},
		},
	}

	agent := NewMockAgent("Agent1", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable{
		"Agent1": agentRunnable,
	}
	supervisor, err := CreateSupervisor(mockLLM, members)
	assert.NoError(t, err)

	// Pass state without messages
	invalidState := map[string]any{
		"other": "value",
	}
	_, err = supervisor.Invoke(context.Background(), invalidState)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "messages key not found or invalid type")
}

func TestCreateSupervisor_LLMError(t *testing.T) {
	// Test when LLM returns an error
	mockLLM := &SupervisorMockLLM{
		responses:   []llms.ContentResponse{},
		returnError: errors.New("LLM connection failed"),
	}

	agent := NewMockAgent("Agent", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable{
		"Agent": agentRunnable,
	}
	supervisor, err := CreateSupervisor(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Test"),
		},
	}

	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM connection failed")
}

func TestCreateSupervisor_EmptyMembers(t *testing.T) {
	// Test with no members
	mockLLM := &SupervisorMockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								FunctionCall: &llms.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "FINISH"}`,
								},
							},
						},
					},
				},
			},
		},
	}

	// Empty members map
	members := map[string]*graph.StateRunnable{}
	supervisor, err := CreateSupervisor(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Test"),
		},
	}

	res, err := supervisor.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	mState := res.(map[string]any)
	messages := mState["messages"].([]llms.MessageContent)
	assert.Equal(t, 1, len(messages)) // Only initial message
	assert.Equal(t, "FINISH", mState["next"])
}

func TestCreateSupervisor_UnknownAgent(t *testing.T) {
	// Test when LLM routes to an unknown agent
	mockLLM := &SupervisorMockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								FunctionCall: &llms.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "UnknownAgent"}`,
								},
							},
						},
					},
				},
			},
		},
	}

	agent := NewMockAgent("KnownAgent", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable{
		"KnownAgent": agentRunnable,
	}
	supervisor, err := CreateSupervisor(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Test unknown agent"),
		},
	}

	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
	// The error message will depend on graph implementation
	assert.Error(t, err)
}

func TestCreateSupervisor_RouteWithoutFunctionCall(t *testing.T) {
	// Test when tool call has no function call
	mockLLM := &SupervisorMockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								// No FunctionCall field
							},
						},
					},
				},
			},
		},
	}

	agent := NewMockAgent("Agent", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable{
		"Agent": agentRunnable,
	}
	supervisor, err := CreateSupervisor(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Test"),
		},
	}

	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
}

func TestCreateSupervisor_NoChoices(t *testing.T) {
	// Test when LLM returns no choices
	mockLLM := &SupervisorMockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{}, // Empty choices
			},
		},
	}

	agent := NewMockAgent("Agent", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable{
		"Agent": agentRunnable,
	}
	supervisor, err := CreateSupervisor(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Test"),
		},
	}

	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
}

func TestCreateSupervisor_EmptyRouteName(t *testing.T) {
	// Test when route tool call has empty name
	mockLLM := &SupervisorMockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								FunctionCall: &llms.FunctionCall{
									Name:      "",
									Arguments: `{"next": "Agent"}`,
								},
							},
						},
					},
				},
			},
		},
	}

	agent := NewMockAgent("Agent", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable{
		"Agent": agentRunnable,
	}
	supervisor, err := CreateSupervisor(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Test"),
		},
	}

	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "supervisor did not select a next step")
}

func TestCreateSupervisor_SingleAgent(t *testing.T) {
	// Test with single agent
	mockLLM := &SupervisorMockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								FunctionCall: &llms.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "Worker"}`,
								},
							},
						},
					},
				},
			},
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								FunctionCall: &llms.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "FINISH"}`,
								},
							},
						},
					},
				},
			},
		},
	}

	agent := NewMockAgent("Worker", "Task completed")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable{
		"Worker": agentRunnable,
	}
	supervisor, err := CreateSupervisor(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Single task"),
		},
	}

	res, err := supervisor.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	mState := res.(map[string]any)
	messages := mState["messages"].([]llms.MessageContent)
	// Should have initial + worker message + potential routing messages
	assert.True(t, len(messages) >= 2)
	assert.Equal(t, "Single task", messages[0].Parts[0].(llms.TextContent).Text)
	// Find the worker response
	found := false
	for _, msg := range messages[1:] {
		if msg.Role == llms.ChatMessageTypeAI {
			if txt, ok := msg.Parts[0].(llms.TextContent); ok && txt.Text == "Task completed" {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "Worker response should be in messages")
}
