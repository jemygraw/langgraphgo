package prebuilt

import (
	"context"
	"strings"
	"testing"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// PEVMockTool for testing
type PEVMockTool struct {
	name        string
	description string
	response    string
}

func (m PEVMockTool) Name() string {
	return m.name
}

func (m PEVMockTool) Description() string {
	return m.description
}

func (m PEVMockTool) Call(ctx context.Context, input string) (string, error) {
	return m.response, nil
}

// PEVMockLLM for testing
type PEVMockLLM struct {
	responses []string
	callCount int
}

func (m *PEVMockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if m.callCount >= len(m.responses) {
		m.callCount++
		return &llms.ContentResponse{
			Choices: []*llms.ContentChoice{
				{Content: "Default response"},
			},
		}, nil
	}

	response := m.responses[m.callCount]
	m.callCount++

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{Content: response},
		},
	}, nil
}

func (m *PEVMockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return "mock response", nil
}

func TestCreatePEVAgent(t *testing.T) {
	mockLLM := &PEVMockLLM{
		responses: []string{
			"1. Calculate 2 + 2",
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

	agent, err := CreatePEVAgent(config)
	if err != nil {
		t.Fatalf("Failed to create PEV agent: %v", err)
	}

	if agent == nil {
		t.Fatal("Expected non-nil agent")
	}
}

func TestPEVAgentRequiresModel(t *testing.T) {
	config := PEVAgentConfig{
		Tools:      []tools.Tool{},
		MaxRetries: 3,
	}

	_, err := CreatePEVAgent(config)
	if err == nil {
		t.Fatal("Expected error when model is nil")
	}
}

func TestParsePlanSteps(t *testing.T) {
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
			name: "mixed format",
			input: `1. First step
- Second step
* Third step`,
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
			result := parsePlanSteps(tt.input)
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

func TestParseVerificationResult(t *testing.T) {
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
			err := parseVerificationResult(tt.input, &result)

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

func TestTruncateString(t *testing.T) {
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
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Additional tests for uncovered functions

func TestPlannerNode(t *testing.T) {
	tests := []struct {
		name          string
		state         map[string]any
		systemMessage string
		verbose       bool
		expectError   bool
	}{
		{
			name: "initial planning",
			state: map[string]any{
				"messages": []llms.MessageContent{
					{
						Role:  llms.ChatMessageTypeHuman,
						Parts: []llms.ContentPart{llms.TextPart("Calculate 2+2")},
					},
				},
				"retries": 0,
			},
			systemMessage: "You are a planner. Break down tasks into steps.",
			verbose:       false,
			expectError:   false,
		},
		{
			name: "re-planning after failure",
			state: map[string]any{
				"messages": []llms.MessageContent{
					{
						Role:  llms.ChatMessageTypeHuman,
						Parts: []llms.ContentPart{llms.TextPart("Calculate 2+2")},
					},
				},
				"retries":          1,
				"last_tool_result": "Error: calculation failed",
				"verification_result": VerificationResult{
					IsSuccessful: false,
					Reasoning:    "The calculation was incorrect",
				},
			},
			systemMessage: "You are a planner. Break down tasks into steps.",
			verbose:       true,
			expectError:   false,
		},
		{
			name: "no messages",
			state: map[string]any{
				"messages": []llms.MessageContent{},
				"retries":  0,
			},
			systemMessage: "You are a planner.",
			verbose:       false,
			expectError:   true,
		},
		{
			name: "nil messages",
			state: map[string]any{
				"retries": 0,
			},
			systemMessage: "You are a planner.",
			verbose:       false,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := &PEVMockLLM{
				responses: []string{
					"1. Use calculator tool\n2. Verify result",
				},
			}

			ctx := context.Background()
			result, err := plannerNode(ctx, tt.state, mockLLM, tt.systemMessage, tt.verbose)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected result but got nil")
				}
			}
		})
	}
}

func TestExecutorNode(t *testing.T) {
	tests := []struct {
		name        string
		state       map[string]any
		verbose     bool
		expectError bool
	}{
		{
			name: "valid execution",
			state: map[string]any{
				"plan": []string{
					"Use calculator to compute 2+2",
					"Verify the result",
				},
				"current_step": 0,
			},
			verbose:     false,
			expectError: false,
		},
		{
			name: "no plan",
			state: map[string]any{
				"plan": []string{},
			},
			verbose:     false,
			expectError: true,
		},
		{
			name: "nil plan",
			state: map[string]any{
				"plan": nil,
			},
			verbose:     false,
			expectError: true,
		},
		{
			name: "step out of bounds",
			state: map[string]any{
				"plan": []string{
					"Step 1",
				},
				"current_step": 5,
			},
			verbose:     false,
			expectError: true,
		},
		{
			name: "second step",
			state: map[string]any{
				"plan": []string{
					"Step 1",
					"Step 2",
					"Step 3",
				},
				"current_step": 1,
			},
			verbose:     true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTool := PEVMockTool{
				name:        "calculator",
				description: "Performs calculations",
				response:    "4",
			}
			toolExecutor := NewToolExecutor([]tools.Tool{mockTool})

			mockLLM := &PEVMockLLM{
				responses: []string{
					`{"tool": "calculator", "tool_input": "2+2"}`,
				},
			}

			ctx := context.Background()
			result, err := executorNode(ctx, tt.state, toolExecutor, mockLLM, tt.verbose)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected result but got nil")
				}
			}
		})
	}
}

func TestVerifierNode(t *testing.T) {
	tests := []struct {
		name               string
		state              map[string]any
		verificationPrompt string
		verbose            bool
		expectError        bool
	}{
		{
			name: "successful verification",
			state: map[string]any{
				"last_tool_result": "The result is 4",
				"plan": []string{
					"Calculate 2+2",
				},
				"current_step": 0,
			},
			verificationPrompt: "Verify if the calculation is correct",
			verbose:            false,
			expectError:        false,
		},
		{
			name: "failed verification",
			state: map[string]any{
				"last_tool_result": "Error: calculation failed",
				"plan": []string{
					"Calculate 2+2",
				},
				"current_step": 0,
			},
			verificationPrompt: "Verify if the calculation is correct",
			verbose:            true,
			expectError:        false,
		},
		{
			name: "no tool result",
			state: map[string]any{
				"plan": []string{
					"Calculate 2+2",
				},
				"current_step": 0,
			},
			verificationPrompt: "Verify the result",
			verbose:            false,
			expectError:        true,
		},
		{
			name: "invalid verification response",
			state: map[string]any{
				"last_tool_result": "Some result",
				"plan": []string{
					"Do something",
				},
				"current_step": 0,
			},
			verificationPrompt: "Verify",
			verbose:            false,
			expectError:        false, // Should not error, but will assume failure
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := &PEVMockLLM{
				responses: []string{
					`{"is_successful": true, "reasoning": "The calculation is correct"}`,
				},
			}

			ctx := context.Background()
			result, err := verifierNode(ctx, tt.state, mockLLM, tt.verificationPrompt, tt.verbose)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected result but got nil")
				}
			}
		})
	}
}

func TestSynthesizerNode(t *testing.T) {
	tests := []struct {
		name        string
		state       map[string]any
		verbose     bool
		expectError bool
	}{
		{
			name: "normal synthesis",
			state: map[string]any{
				"messages": []llms.MessageContent{
					{
						Role:  llms.ChatMessageTypeHuman,
						Parts: []llms.ContentPart{llms.TextPart("What is 2+2?")},
					},
				},
				"intermediate_steps": []string{
					"Step 1: Calculate 2+2 -> 4",
					"Step 2: Verify result -> Correct",
				},
			},
			verbose:     false,
			expectError: false,
		},
		{
			name: "no intermediate steps",
			state: map[string]any{
				"messages": []llms.MessageContent{
					{
						Role:  llms.ChatMessageTypeHuman,
						Parts: []llms.ContentPart{llms.TextPart("Simple task")}},
				},
				"intermediate_steps": []string{},
			},
			verbose:     false,
			expectError: false,
		},
		{
			name: "nil intermediate steps",
			state: map[string]any{
				"messages": []llms.MessageContent{
					{
						Role:  llms.ChatMessageTypeHuman,
						Parts: []llms.ContentPart{llms.TextPart("Task")}},
				},
			},
			verbose:     true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := &PEVMockLLM{
				responses: []string{
					"The answer to 2+2 is 4",
				},
			}

			ctx := context.Background()
			result, err := synthesizerNode(ctx, tt.state, mockLLM, tt.verbose)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected result but got nil")
				}
			}
		})
	}
}

func TestRouteAfterPlanner(t *testing.T) {
	tests := []struct {
		name          string
		state         map[string]any
		verbose       bool
		expectedRoute string
	}{
		{
			name: "valid plan",
			state: map[string]any{
				"plan": []string{"Step 1", "Step 2"},
			},
			verbose:       false,
			expectedRoute: "executor",
		},
		{
			name: "empty plan",
			state: map[string]any{
				"plan": []string{},
			},
			verbose:       false,
			expectedRoute: graph.END,
		},
		{
			name: "nil plan",
			state: map[string]any{
				"plan": nil,
			},
			verbose:       false,
			expectedRoute: graph.END,
		},
		{
			name: "plan not a slice",
			state: map[string]any{
				"plan": "not a slice",
			},
			verbose:       true,
			expectedRoute: graph.END,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := routeAfterPlanner(tt.state, tt.verbose)
			if route != tt.expectedRoute {
				t.Errorf("Expected route %q, got %q", tt.expectedRoute, route)
			}
		})
	}
}

func TestRouteAfterExecutor(t *testing.T) {
	tests := []struct {
		name          string
		state         map[string]any
		expectedRoute string
	}{
		{
			name: "always go to verifier",
			state: map[string]any{
				"current_step": 0,
				"plan":         []string{"Step 1"},
			},
			expectedRoute: "verifier",
		},
		{
			name:          "empty state still goes to verifier",
			state:         map[string]any{},
			expectedRoute: "verifier",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := routeAfterExecutor(tt.state, false)
			if route != tt.expectedRoute {
				t.Errorf("Expected route %q, got %q", tt.expectedRoute, route)
			}
		})
	}
}

func TestRouteAfterVerifier(t *testing.T) {
	tests := []struct {
		name          string
		state         map[string]any
		maxRetries    int
		verbose       bool
		expectedRoute string
	}{
		{
			name: "successful verification - more steps",
			state: map[string]any{
				"verification_result": VerificationResult{
					IsSuccessful: true,
				},
				"current_step": 0,
				"plan":         []string{"Step 1", "Step 2"},
				"retries":      0,
			},
			maxRetries:    3,
			verbose:       false,
			expectedRoute: "executor",
		},
		{
			name: "successful verification - last step",
			state: map[string]any{
				"verification_result": VerificationResult{
					IsSuccessful: true,
				},
				"current_step": 1,
				"plan":         []string{"Step 1", "Step 2"},
				"retries":      0,
			},
			maxRetries:    3,
			verbose:       false,
			expectedRoute: "synthesizer",
		},
		{
			name: "failed verification - can retry",
			state: map[string]any{
				"verification_result": VerificationResult{
					IsSuccessful: false,
				},
				"current_step": 0,
				"plan":         []string{"Step 1"},
				"retries":      1,
			},
			maxRetries:    3,
			verbose:       false,
			expectedRoute: "planner",
		},
		{
			name: "failed verification - max retries reached",
			state: map[string]any{
				"verification_result": VerificationResult{
					IsSuccessful: false,
				},
				"current_step": 0,
				"plan":         []string{"Step 1"},
				"retries":      3,
			},
			maxRetries:    3,
			verbose:       true,
			expectedRoute: "synthesizer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of state for mutable operations
			stateCopy := make(map[string]any)
			for k, v := range tt.state {
				stateCopy[k] = v
			}

			route := routeAfterVerifier(stateCopy, tt.maxRetries, tt.verbose)
			if route != tt.expectedRoute {
				t.Errorf("Expected route %q, got %q", tt.expectedRoute, route)
			}
		})
	}
}

func TestExecuteStep(t *testing.T) {
	tests := []struct {
		name           string
		stepDesc       string
		toolExecutor   *ToolExecutor
		expectError    bool
		expectContains string
	}{
		{
			name:     "no tools available",
			stepDesc: "Calculate 2+2",
			toolExecutor: &ToolExecutor{
				tools: map[string]tools.Tool{},
			},
			expectError:    false,
			expectContains: "No tools available",
		},
		{
			name:           "nil tool executor",
			stepDesc:       "Do something",
			toolExecutor:   nil,
			expectError:    false,
			expectContains: "No tools available",
		},
		{
			name:     "valid execution",
			stepDesc: "Calculate 2+2",
			toolExecutor: NewToolExecutor([]tools.Tool{
				PEVMockTool{
					name:        "calculator",
					description: "Performs calculations",
					response:    "4",
				},
			}),
			expectError:    false,
			expectContains: "4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := &PEVMockLLM{
				responses: []string{
					`{"tool": "calculator", "tool_input": "2+2"}`,
				},
			}

			ctx := context.Background()
			result, err := executeStep(ctx, tt.stepDesc, tt.toolExecutor, mockLLM)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if tt.expectContains != "" {
				if !strings.Contains(result, tt.expectContains) {
					t.Errorf("Expected result to contain %q but got %q", tt.expectContains, result)
				}
			}
		})
	}
}

func TestParseToolChoice(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expected    ToolInvocation
	}{
		{
			name:        "valid JSON",
			input:       `{"tool": "calculator", "tool_input": "2+2"}`,
			expectError: false,
			expected: ToolInvocation{
				Tool:      "calculator",
				ToolInput: "2+2",
			},
		},
		{
			name:        "JSON with surrounding text",
			input:       `Here is my choice: {"tool": "search", "tool_input": "query"} Done.`,
			expectError: false,
			expected: ToolInvocation{
				Tool:      "search",
				ToolInput: "query",
			},
		},
		{
			name:        "no JSON",
			input:       "This is not JSON",
			expectError: true,
		},
		{
			name:        "invalid JSON",
			input:       `{"tool": "test", "tool_input":}`,
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result ToolInvocation
			err := parseToolChoice(tt.input, &result)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result.Tool != tt.expected.Tool {
					t.Errorf("Expected Tool=%q, got %q", tt.expected.Tool, result.Tool)
				}
				if result.ToolInput != tt.expected.ToolInput {
					t.Errorf("Expected ToolInput=%q, got %q", tt.expected.ToolInput, result.ToolInput)
				}
			}
		})
	}
}

func TestBuildDefaultPrompts(t *testing.T) {
	// Test buildDefaultPlannerPrompt
	plannerPrompt := buildDefaultPlannerPrompt()
	if plannerPrompt == "" {
		t.Error("Planner prompt should not be empty")
	}
	if !strings.Contains(plannerPrompt, "breaks down user requests") {
		t.Error("Planner prompt should mention breaking down requests")
	}

	// Test buildDefaultVerificationPrompt
	verificationPrompt := buildDefaultVerificationPrompt()
	if verificationPrompt == "" {
		t.Error("Verification prompt should not be empty")
	}
	if !strings.Contains(verificationPrompt, "verification") {
		t.Error("Verification prompt should mention verification")
	}
	if !strings.Contains(verificationPrompt, "is_successful") {
		t.Error("Verification prompt should mention is_successful field")
	}
}
