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

// SupervisorTypedMockLLM for testing supervisor typed
type SupervisorTypedMockLLM struct {
	responses   []llms.ContentResponse
	currentIdx  int
	returnError error
}

func (m *SupervisorTypedMockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
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

func (m *SupervisorTypedMockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return "", nil
}

// TypedMockAgent for testing supervisor typed with various behaviors
type TypedMockAgent struct {
	name        string
	response    string
	shouldError bool
	errorMsg    string
}

func NewTypedMockAgent(name, response string) *TypedMockAgent {
	return &TypedMockAgent{
		name:     name,
		response: response,
	}
}

func NewTypedMockErrorAgent(name, errorMsg string) *TypedMockAgent {
	return &TypedMockAgent{
		name:        name,
		shouldError: true,
		errorMsg:    errorMsg,
	}
}

func (a *TypedMockAgent) Invoke(ctx context.Context, state SupervisorState) (SupervisorState, error) {
	if a.shouldError {
		return SupervisorState{}, errors.New(a.errorMsg)
	}

	// Append agent response to messages
	newMessages := append(state.Messages, llms.TextParts(llms.ChatMessageTypeAI, a.response))

	return SupervisorState{
		Messages: newMessages,
		Next:     state.Next, // Preserve next for supervisor routing
	}, nil
}

func (a *TypedMockAgent) Compile() (*graph.StateRunnableTyped[SupervisorState], error) {
	workflow := graph.NewStateGraphTyped[SupervisorState]()

	// Define state schema
	schema := graph.NewStructSchema(
		SupervisorState{},
		func(current, new SupervisorState) (SupervisorState, error) {
			// Append new messages to current messages
			current.Messages = append(current.Messages, new.Messages...)
			if new.Next != "" {
				current.Next = new.Next
			}
			return current, nil
		},
	)
	workflow.SetSchema(schema)

	workflow.AddNode("run", "Agent run node", a.Invoke)
	workflow.SetEntryPoint("run")
	workflow.AddEdge("run", graph.END)

	return workflow.Compile()
}

func TestCreateSupervisorTyped(t *testing.T) {
	// Setup Mock LLM
	// 1. Route to Agent1
	// 2. Route to FINISH
	mockLLM := &SupervisorTypedMockLLM{
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

	// Setup Mock Agent
	agent1 := NewTypedMockAgent("Agent1", "Agent1 done")
	agent1Runnable, err := agent1.Compile()
	require.NoError(t, err)

	// Create Supervisor
	members := map[string]*graph.StateRunnableTyped[SupervisorState]{
		"Agent1": agent1Runnable,
	}
	supervisor, err := CreateSupervisorTyped(mockLLM, members)
	assert.NoError(t, err)

	// Initial State
	initialState := SupervisorState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Start"),
		},
	}

	// Run Supervisor
	res, err := supervisor.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	// Verify Result
	// Should have initial + agent message + supervisor routing messages
	assert.True(t, len(res.Messages) >= 2)
	assert.Equal(t, "Start", res.Messages[0].Parts[0].(llms.TextContent).Text)
	assert.Equal(t, "FINISH", res.Next)

	// Find the agent response
	found := false
	for _, msg := range res.Messages[1:] {
		if msg.Role == llms.ChatMessageTypeAI {
			if txt, ok := msg.Parts[0].(llms.TextContent); ok && txt.Text == "Agent1 done" {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "Agent1 response should be in messages")
}

func TestCreateSupervisorTyped_SingleAgent(t *testing.T) {
	// Test with single agent
	mockLLM := &SupervisorTypedMockLLM{
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

	agent := NewTypedMockAgent("Worker", "Task completed")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnableTyped[SupervisorState]{
		"Worker": agentRunnable,
	}
	supervisor, err := CreateSupervisorTyped(mockLLM, members)
	assert.NoError(t, err)

	initialState := SupervisorState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Single task"),
		},
	}

	res, err := supervisor.Invoke(context.Background(), initialState)
	assert.NoError(t, err)
	assert.Equal(t, "FINISH", res.Next)
	assert.True(t, len(res.Messages) >= 2)

	// Find the worker response
	found := false
	for _, msg := range res.Messages[1:] {
		if msg.Role == llms.ChatMessageTypeAI {
			if txt, ok := msg.Parts[0].(llms.TextContent); ok && txt.Text == "Task completed" {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "Worker response should be in messages")
}

func TestCreateSupervisorTyped_DirectFinish(t *testing.T) {
	// Test supervisor that directly routes to FINISH
	mockLLM := &SupervisorTypedMockLLM{
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

	agent := NewTypedMockAgent("Agent", "Should not be called")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnableTyped[SupervisorState]{
		"Agent": agentRunnable,
	}
	supervisor, err := CreateSupervisorTyped(mockLLM, members)
	assert.NoError(t, err)

	initialState := SupervisorState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Complete immediately"),
		},
	}

	res, err := supervisor.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	// Should have initial message, plus possible supervisor routing messages
	assert.True(t, len(res.Messages) >= 1)
	assert.Equal(t, "Complete immediately", res.Messages[0].Parts[0].(llms.TextContent).Text)
	assert.Equal(t, "FINISH", res.Next)
}

func TestCreateSupervisorTyped_AgentError(t *testing.T) {
	// Test handling of agent errors
	mockLLM := &SupervisorTypedMockLLM{
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

	errorAgent := NewTypedMockErrorAgent("ErrorAgent", "Agent failed to process")
	errorAgentRunnable, err := errorAgent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnableTyped[SupervisorState]{
		"ErrorAgent": errorAgentRunnable,
	}
	supervisor, err := CreateSupervisorTyped(mockLLM, members)
	assert.NoError(t, err)

	initialState := SupervisorState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Trigger error"),
		},
	}

	// Should return error from agent
	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent ErrorAgent failed")
}

func TestCreateSupervisorTyped_NoToolCall(t *testing.T) {
	// Test when LLM doesn't make a tool call
	mockLLM := &SupervisorTypedMockLLM{
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

	agent := NewTypedMockAgent("Agent", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnableTyped[SupervisorState]{
		"Agent": agentRunnable,
	}
	supervisor, err := CreateSupervisorTyped(mockLLM, members)
	assert.NoError(t, err)

	initialState := SupervisorState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Test"),
		},
	}

	// Should return error about not selecting next step
	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "supervisor did not select a next step")
}

func TestCreateSupervisorTyped_InvalidRouteArguments(t *testing.T) {
	// Test when route tool has invalid JSON
	mockLLM := &SupervisorTypedMockLLM{
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

	agent := NewTypedMockAgent("Agent", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnableTyped[SupervisorState]{
		"Agent": agentRunnable,
	}
	supervisor, err := CreateSupervisorTyped(mockLLM, members)
	assert.NoError(t, err)

	initialState := SupervisorState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Test"),
		},
	}

	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse route arguments")
}

func TestCreateSupervisorTyped_LLMError(t *testing.T) {
	// Test when LLM returns an error
	mockLLM := &SupervisorTypedMockLLM{
		responses:   []llms.ContentResponse{},
		returnError: errors.New("LLM connection failed"),
	}

	agent := NewTypedMockAgent("Agent", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnableTyped[SupervisorState]{
		"Agent": agentRunnable,
	}
	supervisor, err := CreateSupervisorTyped(mockLLM, members)
	assert.NoError(t, err)

	initialState := SupervisorState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Test"),
		},
	}

	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM connection failed")
}

func TestCreateSupervisorTyped_MultipleAgents(t *testing.T) {
	// Test supervisor with multiple agents
	mockLLM := &SupervisorTypedMockLLM{
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
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								FunctionCall: &llms.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "Agent2"}`,
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

	agent1 := NewTypedMockAgent("Agent1", "Agent1 completed")
	agent1Runnable, err := agent1.Compile()
	require.NoError(t, err)

	agent2 := NewTypedMockAgent("Agent2", "Agent2 completed")
	agent2Runnable, err := agent2.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnableTyped[SupervisorState]{
		"Agent1": agent1Runnable,
		"Agent2": agent2Runnable,
	}
	supervisor, err := CreateSupervisorTyped(mockLLM, members)
	assert.NoError(t, err)

	initialState := SupervisorState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Process with multiple agents"),
		},
	}

	res, err := supervisor.Invoke(context.Background(), initialState)
	assert.NoError(t, err)
	assert.Equal(t, "FINISH", res.Next)
	assert.True(t, len(res.Messages) >= 3) // Initial + 2 agent responses

	// Verify both agent responses are present
	agent1Found := false
	agent2Found := false
	for _, msg := range res.Messages[1:] {
		if msg.Role == llms.ChatMessageTypeAI {
			if txt, ok := msg.Parts[0].(llms.TextContent); ok {
				if txt.Text == "Agent1 completed" {
					agent1Found = true
				}
				if txt.Text == "Agent2 completed" {
					agent2Found = true
				}
			}
		}
	}
	assert.True(t, agent1Found, "Agent1 response should be in messages")
	assert.True(t, agent2Found, "Agent2 response should be in messages")
}

func TestCreateSupervisorTyped_EmptyMembers(t *testing.T) {
	// Test with no members
	mockLLM := &SupervisorTypedMockLLM{
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
	members := map[string]*graph.StateRunnableTyped[SupervisorState]{}
	supervisor, err := CreateSupervisorTyped(mockLLM, members)
	assert.NoError(t, err)

	initialState := SupervisorState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Test"),
		},
	}

	res, err := supervisor.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	// Should have initial message, plus possible supervisor routing messages
	assert.True(t, len(res.Messages) >= 1)
	assert.Equal(t, "Test", res.Messages[0].Parts[0].(llms.TextContent).Text)
	assert.Equal(t, "FINISH", res.Next)
}

func TestCreateSupervisorTyped_UnknownAgent(t *testing.T) {
	// Test when LLM routes to an unknown agent
	mockLLM := &SupervisorTypedMockLLM{
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

	agent := NewTypedMockAgent("KnownAgent", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnableTyped[SupervisorState]{
		"KnownAgent": agentRunnable,
	}
	supervisor, err := CreateSupervisorTyped(mockLLM, members)
	assert.NoError(t, err)

	initialState := SupervisorState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Test unknown agent"),
		},
	}

	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
}

func TestCreateSupervisorTyped_NoChoices(t *testing.T) {
	// Test when LLM returns no choices
	mockLLM := &SupervisorTypedMockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{}, // Empty choices
			},
		},
	}

	agent := NewTypedMockAgent("Agent", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnableTyped[SupervisorState]{
		"Agent": agentRunnable,
	}
	supervisor, err := CreateSupervisorTyped(mockLLM, members)
	assert.NoError(t, err)

	initialState := SupervisorState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Test"),
		},
	}

	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
}

func TestCreateSupervisorTyped_RouteWithoutFunctionCall(t *testing.T) {
	// Test when tool call has no function call
	mockLLM := &SupervisorTypedMockLLM{
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

	agent := NewTypedMockAgent("Agent", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnableTyped[SupervisorState]{
		"Agent": agentRunnable,
	}
	supervisor, err := CreateSupervisorTyped(mockLLM, members)
	assert.NoError(t, err)

	initialState := SupervisorState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Test"),
		},
	}

	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
}

// Tests for CreateSupervisorWithStateTyped

// CustomTypedMockAgent for testing with custom state
type CustomTypedMockAgent[S any] struct {
	name        string
	response    string
	shouldError bool
	errorMsg    string
}

func NewCustomTypedMockAgent[S any](name, response string) *CustomTypedMockAgent[S] {
	return &CustomTypedMockAgent[S]{
		name:     name,
		response: response,
	}
}

func NewCustomTypedMockErrorAgent[S any](name, errorMsg string) *CustomTypedMockAgent[S] {
	return &CustomTypedMockAgent[S]{
		name:        name,
		shouldError: true,
		errorMsg:    errorMsg,
	}
}

func (a *CustomTypedMockAgent[S]) Invoke(ctx context.Context, state S) (S, error) {
	if a.shouldError {
		var zero S
		return zero, errors.New(a.errorMsg)
	}

	// For custom state, we need to use reflection or type assertion
	// This is a simplified version that assumes the state has Messages field
	// In practice, you'd need to handle different state types appropriately
	return state, nil
}

func TestCreateSupervisorWithStateTyped(t *testing.T) {
	// Define custom state type
	type CustomState struct {
		Messages []llms.MessageContent
		Next     string
		Step     int
	}

	// Setup Mock LLM
	mockLLM := &SupervisorTypedMockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								FunctionCall: &llms.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "Processor"}`,
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

	// Create a simple mock runnable for the custom state
	processorWorkflow := graph.NewStateGraphTyped[CustomState]()
	processorWorkflow.AddNode("process", "Process node", func(ctx context.Context, state CustomState) (CustomState, error) {
		state.Messages = append(state.Messages, llms.TextParts(llms.ChatMessageTypeAI, "Processed"))
		state.Step++
		return state, nil
	})
	processorWorkflow.SetEntryPoint("process")
	processorWorkflow.AddEdge("process", graph.END)
	processorRunnable, err := processorWorkflow.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnableTyped[CustomState]{
		"Processor": processorRunnable,
	}

	// Define state handlers
	getMessages := func(s CustomState) []llms.MessageContent {
		return s.Messages
	}

	updateMessages := func(s CustomState, msgs []llms.MessageContent) CustomState {
		s.Messages = msgs
		return s
	}

	getNext := func(s CustomState) string {
		return s.Next
	}

	setNext := func(s CustomState, next string) CustomState {
		s.Next = next
		return s
	}

	// Create supervisor with custom state
	supervisor, err := CreateSupervisorWithStateTyped(
		mockLLM,
		members,
		getMessages,
		updateMessages,
		getNext,
		setNext,
	)
	assert.NoError(t, err)

	initialState := CustomState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Process this"),
		},
		Step: 0,
	}

	res, err := supervisor.Invoke(context.Background(), initialState)
	assert.NoError(t, err)
	assert.Equal(t, "FINISH", res.Next)
	assert.True(t, len(res.Messages) >= 2)
	assert.True(t, res.Step > 0)
}

func TestCreateSupervisorWithStateTyped_ErrorHandling(t *testing.T) {
	type ErrorState struct {
		Messages []llms.MessageContent
		Next     string
		Failed   bool
	}

	getMessages := func(s ErrorState) []llms.MessageContent {
		return s.Messages
	}

	updateMessages := func(s ErrorState, msgs []llms.MessageContent) ErrorState {
		s.Messages = msgs
		return s
	}

	getNext := func(s ErrorState) string {
		return s.Next
	}

	setNext := func(s ErrorState, next string) ErrorState {
		s.Next = next
		return s
	}

	// Test with LLM error
	mockLLM := &SupervisorTypedMockLLM{
		responses:   []llms.ContentResponse{},
		returnError: errors.New("LLM error in custom state"),
	}

	members := map[string]*graph.StateRunnableTyped[ErrorState]{}

	supervisor, err := CreateSupervisorWithStateTyped(
		mockLLM,
		members,
		getMessages,
		updateMessages,
		getNext,
		setNext,
	)
	assert.NoError(t, err)

	initialState := ErrorState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Test error"),
		},
	}

	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM error in custom state")
}

func TestCreateSupervisorWithStateTyped_MultipleMembers(t *testing.T) {
	type MultiState struct {
		Messages []llms.MessageContent
		Next     string
		Counter  int
	}

	mockLLM := &SupervisorTypedMockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								FunctionCall: &llms.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "Counter"}`,
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
									Arguments: `{"next": "Logger"}`,
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

	// Create member runnables
	counterWorkflow := graph.NewStateGraphTyped[MultiState]()
	counterWorkflow.AddNode("count", "Count node", func(ctx context.Context, state MultiState) (MultiState, error) {
		state.Messages = append(state.Messages, llms.TextParts(llms.ChatMessageTypeAI, "Counted"))
		state.Counter++
		return state, nil
	})
	counterWorkflow.SetEntryPoint("count")
	counterWorkflow.AddEdge("count", graph.END)
	counterRunnable, err := counterWorkflow.Compile()
	require.NoError(t, err)

	loggerWorkflow := graph.NewStateGraphTyped[MultiState]()
	loggerWorkflow.AddNode("log", "Log node", func(ctx context.Context, state MultiState) (MultiState, error) {
		state.Messages = append(state.Messages, llms.TextParts(llms.ChatMessageTypeAI, "Logged"))
		return state, nil
	})
	loggerWorkflow.SetEntryPoint("log")
	loggerWorkflow.AddEdge("log", graph.END)
	loggerRunnable, err := loggerWorkflow.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnableTyped[MultiState]{
		"Counter": counterRunnable,
		"Logger":  loggerRunnable,
	}

	getMessages := func(s MultiState) []llms.MessageContent {
		return s.Messages
	}

	updateMessages := func(s MultiState, msgs []llms.MessageContent) MultiState {
		s.Messages = msgs
		return s
	}

	getNext := func(s MultiState) string {
		return s.Next
	}

	setNext := func(s MultiState, next string) MultiState {
		s.Next = next
		return s
	}

	supervisor, err := CreateSupervisorWithStateTyped(
		mockLLM,
		members,
		getMessages,
		updateMessages,
		getNext,
		setNext,
	)
	assert.NoError(t, err)

	initialState := MultiState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Multi-step process"),
		},
		Counter: 0,
	}

	res, err := supervisor.Invoke(context.Background(), initialState)
	assert.NoError(t, err)
	assert.Equal(t, "FINISH", res.Next)
	assert.True(t, len(res.Messages) >= 3) // Initial + 2 member responses
	assert.True(t, res.Counter > 0)

	// Verify both member responses are present
	countFound := false
	logFound := false
	for _, msg := range res.Messages[1:] {
		if msg.Role == llms.ChatMessageTypeAI {
			if txt, ok := msg.Parts[0].(llms.TextContent); ok {
				if txt.Text == "Counted" {
					countFound = true
				}
				if txt.Text == "Logged" {
					logFound = true
				}
			}
		}
	}
	assert.True(t, countFound, "Counter response should be in messages")
	assert.True(t, logFound, "Logger response should be in messages")
}

func TestCreateSupervisorWithStateTyped_EmptyMembers(t *testing.T) {
	type EmptyState struct {
		Messages []llms.MessageContent
		Next     string
		Data     string
	}

	getMessages := func(s EmptyState) []llms.MessageContent {
		return s.Messages
	}

	updateMessages := func(s EmptyState, msgs []llms.MessageContent) EmptyState {
		s.Messages = msgs
		return s
	}

	getNext := func(s EmptyState) string {
		return s.Next
	}

	setNext := func(s EmptyState, next string) EmptyState {
		s.Next = next
		return s
	}

	mockLLM := &SupervisorTypedMockLLM{
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
	members := map[string]*graph.StateRunnableTyped[EmptyState]{}

	supervisor, err := CreateSupervisorWithStateTyped(
		mockLLM,
		members,
		getMessages,
		updateMessages,
		getNext,
		setNext,
	)
	assert.NoError(t, err)

	initialState := EmptyState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Test with no members"),
		},
		Data: "initial",
	}

	res, err := supervisor.Invoke(context.Background(), initialState)
	assert.NoError(t, err)
	assert.Equal(t, "FINISH", res.Next)
	assert.True(t, len(res.Messages) >= 1) // Initial message plus possible supervisor routing
	assert.Equal(t, "initial", res.Data)   // Data should be preserved
}

// Test SupervisorState structure
func TestSupervisorState(t *testing.T) {
	state := SupervisorState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Test message"),
		},
		Next: "worker1",
	}

	if len(state.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(state.Messages))
	}

	if state.Next != "worker1" {
		t.Errorf("Expected next to be 'worker1', got '%s'", state.Next)
	}
}
