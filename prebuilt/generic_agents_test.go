package prebuilt

import (
	"context"
	"testing"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// TestCreateReflectionAgentTyped tests the typed version of ReflectionAgent
func TestCreateReflectionAgentTyped(t *testing.T) {
	mockLLM := &MockReflectionLLM{
		responses: []string{
			"This is my initial response to the question.",
			"**Strengths:** The response is clear.\n**Weaknesses:** Could be more detailed.\n**Suggestions for improvement:** Add more examples.",
			"This is my improved response with more details and examples.",
			"**Strengths:** Excellent improvement. Comprehensive and detailed.\n**Weaknesses:** No major issues.\n**Suggestions for improvement:** No improvements needed.",
		},
	}

	config := ReflectionAgentConfig{
		Model:         mockLLM,
		MaxIterations: 3,
		Verbose:       false,
	}

	agent, err := CreateReflectionAgentTyped(config)
	if err != nil {
		t.Fatalf("Failed to create reflection agent typed: %v", err)
	}

	if agent == nil {
		t.Fatal("Agent is nil")
	}

	// Test invocation
	initialState := ReflectionAgentState{
		Messages: []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{llms.TextPart("Explain quantum computing")},
			},
		},
	}

	result, err := agent.Invoke(context.Background(), initialState)
	if err != nil {
		t.Fatalf("Failed to invoke agent: %v", err)
	}

	// Verify final state has messages
	if len(result.Messages) == 0 {
		t.Fatal("No messages in final state")
	}

	// Verify we have a draft
	if result.Draft == "" {
		t.Fatal("Final state does not contain draft")
	}

	// Verify iterations occurred
	if result.Iteration < 1 {
		t.Fatal("Expected at least one iteration")
	}
}

// TestCreateReflectionAgentTypedWithoutModel tests error case
func TestCreateReflectionAgentTypedWithoutModel(t *testing.T) {
	config := ReflectionAgentConfig{
		Model: nil,
	}

	_, err := CreateReflectionAgentTyped(config)
	if err == nil {
		t.Fatal("Expected error when creating agent without model")
	}
}

// TestCreatePEVAgentTyped tests the typed version of PEVAgent
func TestCreatePEVAgentTyped(t *testing.T) {
	mockLLM := &PEVMockLLM{
		responses: []string{
			"1. Calculate 2 + 2",
			`{"tool": "calculator", "tool_input": "2+2"}`,
			`{"is_successful": true, "reasoning": "Calculation completed successfully"}`,
			"The result of 2 + 2 is 4",
		},
	}

	mockTool := PEVMockTool{
		name:        "calculator",
		description: "Performs calculations",
		response:    "4",
	}

	config := PEVAgentConfig{
		Model:      mockLLM,
		Tools:      []tools.Tool{mockTool},
		MaxRetries: 3,
		Verbose:    false,
	}

	agent, err := CreatePEVAgentTyped(config)
	if err != nil {
		t.Fatalf("Failed to create PEV agent typed: %v", err)
	}

	if agent == nil {
		t.Fatal("Expected non-nil agent")
	}

	// Test invocation with typed state
	initialState := PEVAgentState{
		Messages: []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{llms.TextPart("Calculate 2+2")},
			},
		},
	}

	result, err := agent.Invoke(context.Background(), initialState)
	if err != nil {
		t.Fatalf("Failed to invoke agent: %v", err)
	}

	// Verify state was processed
	if len(result.Messages) == 0 {
		t.Fatal("Expected messages in result")
	}
}

// TestCreateAgentTyped tests the typed version of CreateAgent
func TestCreateAgentTyped(t *testing.T) {
	mockLLM := &PEVMockLLM{
		responses: []string{
			"I can help with that.",
		},
	}

	mockTool := PEVMockTool{
		name:        "test_tool",
		description: "A test tool",
		response:    "Tool result",
	}

	agent, err := CreateAgentTyped(mockLLM, []tools.Tool{mockTool})
	if err != nil {
		t.Fatalf("Failed to create agent typed: %v", err)
	}

	if agent == nil {
		t.Fatal("Expected non-nil agent")
	}

	// Test invocation
	initialState := AgentState{
		Messages: []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{llms.TextPart("Hello")},
			},
		},
	}

	result, err := agent.Invoke(context.Background(), initialState)
	if err != nil {
		t.Fatalf("Failed to invoke agent: %v", err)
	}

	// Verify state was processed
	if len(result.Messages) == 0 {
		t.Fatal("Expected messages in result")
	}
}

// TestParsePlanStepsTyped tests the typed helper function
func TestParsePlanStepsTyped(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "numbered list",
			input: `1. First step
2. Second step
3. Third step`,
			expected: []string{"First step", "Second step", "Third step"},
		},
		{
			name: "bullet points",
			input: `- First step
- Second step
- Third step`,
			expected: []string{"First step", "Second step", "Third step"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePlanStepsTyped(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d steps, got %d", len(tt.expected), len(result))
			}
			for i, step := range result {
				if i < len(tt.expected) && step != tt.expected[i] {
					t.Errorf("Step %d: expected %q, got %q", i, tt.expected[i], step)
				}
			}
		})
	}
}

// TestParseVerificationResultTyped tests the typed helper function
func TestParseVerificationResultTyped(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expected    VerificationResult
	}{
		{
			name:        "valid JSON",
			input:       `{"is_successful": true, "reasoning": "All good"}`,
			expectError: false,
			expected:    VerificationResult{IsSuccessful: true, Reasoning: "All good"},
		},
		{
			name:        "JSON with surrounding text",
			input:       `Here is the result: {"is_successful": false, "reasoning": "Failed"} Done.`,
			expectError: false,
			expected:    VerificationResult{IsSuccessful: false, Reasoning: "Failed"},
		},
		{
			name:        "no JSON",
			input:       "This is not JSON",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result VerificationResult
			err := parseVerificationResultTyped(tt.input, &result)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result.IsSuccessful != tt.expected.IsSuccessful {
					t.Errorf("Expected IsSuccessful=%v, got %v", tt.expected.IsSuccessful, result.IsSuccessful)
				}
				if result.Reasoning != tt.expected.Reasoning {
					t.Errorf("Expected Reasoning=%q, got %q", tt.expected.Reasoning, result.Reasoning)
				}
			}
		})
	}
}

// TestTruncateStringTyped tests the typed helper function
func TestTruncateStringTyped(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string",
			input:    "Hello",
			maxLen:   10,
			expected: "Hello",
		},
		{
			name:     "exact length",
			input:    "Hello",
			maxLen:   5,
			expected: "Hello",
		},
		{
			name:     "long string",
			input:    "This is a very long string",
			maxLen:   10,
			expected: "This is a ...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateStringTyped(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestIsResponseSatisfactoryTyped tests the typed helper function
func TestIsResponseSatisfactoryTyped(t *testing.T) {
	tests := []struct {
		name       string
		reflection string
		want       bool
	}{
		{
			name:       "Excellent response",
			reflection: "**Strengths:** Excellent response. **Weaknesses:** No major issues. **Suggestions:** No improvements needed.",
			want:       true,
		},
		{
			name:       "Has issues",
			reflection: "**Strengths:** Good start. **Weaknesses:** Missing key details. **Suggestions:** Should include more examples.",
			want:       false,
		},
		{
			name:       "Satisfactory",
			reflection: "The response is satisfactory and meets all requirements.",
			want:       true,
		},
		{
			name:       "Needs improvement",
			reflection: "The response is incomplete and lacks important information.",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isResponseSatisfactoryTyped(tt.reflection)
			if got != tt.want {
				t.Errorf("isResponseSatisfactoryTyped() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetOriginalRequestTyped tests the typed helper function
func TestGetOriginalRequestTyped(t *testing.T) {
	messages := []llms.MessageContent{
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextPart("What is AI?")},
		},
		{
			Role:  llms.ChatMessageTypeAI,
			Parts: []llms.ContentPart{llms.TextPart("AI is...")},
		},
	}

	request := getOriginalRequestTyped(messages)
	if request != "What is AI?" {
		t.Errorf("Expected 'What is AI?', got '%s'", request)
	}
}

// TestTreeOfThoughtsAgentTyped tests the typed version of TreeOfThoughts
func TestTreeOfThoughtsAgentTyped(t *testing.T) {
	// Simple test state that reaches goal immediately
	initialState := &SimpleThoughtState{
		value:      0,
		isGoal:     true,
		isValid:    true,
		desc:       "Start state",
		parentHash: "",
	}

	generator := &SimpleThoughtGenerator{}
	evaluator := &SimpleThoughtEvaluator{}

	config := TreeOfThoughtsConfig{
		Generator:   generator,
		Evaluator:   evaluator,
		InitialState: initialState,
		MaxDepth:    10,
		MaxPaths:    5,
		Verbose:     false,
	}

	agent, err := CreateTreeOfThoughtsAgentTyped(config)
	if err != nil {
		t.Fatalf("Failed to create TreeOfThoughts agent typed: %v", err)
	}

	if agent == nil {
		t.Fatal("Expected non-nil agent")
	}

	// Test invocation
	initialAgentState := TreeOfThoughtsState{}

	result, err := agent.Invoke(context.Background(), initialAgentState)
	if err != nil {
		t.Fatalf("Failed to invoke agent: %v", err)
	}

	// Should have found solution since initial state is goal
	if result.Solution == "" {
		t.Fatal("Expected solution from initial goal state")
	}
}

// SimpleThoughtState for testing
type SimpleThoughtState struct {
	value      int
	isGoal     bool
	isValid    bool
	desc       string
	parentHash string
}

func (s *SimpleThoughtState) IsValid() bool {
	return s.isValid
}

func (s *SimpleThoughtState) IsGoal() bool {
	return s.isGoal
}

func (s *SimpleThoughtState) GetDescription() string {
	return s.desc
}

func (s *SimpleThoughtState) Hash() string {
	return s.desc
}

// SimpleThoughtGenerator for testing
type SimpleThoughtGenerator struct{}

func (g *SimpleThoughtGenerator) Generate(ctx context.Context, state ThoughtState) ([]ThoughtState, error) {
	// Return empty slice to stop exploration
	return []ThoughtState{}, nil
}

// SimpleThoughtEvaluator for testing
type SimpleThoughtEvaluator struct{}

func (e *SimpleThoughtEvaluator) Evaluate(ctx context.Context, state ThoughtState, depth int) (float64, error) {
	return 1.0, nil
}
