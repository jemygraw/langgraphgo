package prebuilt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/log"
	"github.com/tmc/langchaingo/llms"
)

// CreatePEVAgentTyped creates a new PEV (Plan, Execute, Verify) Agent with full type safety.
// The agent implements a robust, self-correcting loop for reliable task execution.
//
// The PEV pattern involves:
// 1. Plan: Break down the user request into executable steps
// 2. Execute: Run each step using available tools
// 3. Verify: Check if the execution was successful
// 4. Retry: If verification fails, re-plan and execute again
func CreatePEVAgentTyped(config PEVAgentConfig) (*graph.StateRunnable[PEVAgentState], error) {
	if config.Model == nil {
		return nil, fmt.Errorf("model is required")
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = 3 // Default to 3 retries
	}

	// Default system messages
	if config.SystemMessage == "" {
		config.SystemMessage = buildDefaultPlannerPromptTyped()
	}

	if config.VerificationPrompt == "" {
		config.VerificationPrompt = buildDefaultVerificationPromptTyped()
	}

	// Create tool executor
	toolExecutor := NewToolExecutor(config.Tools)

	// Create the workflow with generic state type
	workflow := graph.NewStateGraph[PEVAgentState]()

	// Define state schema for merging
	schema := graph.NewStructSchema(
		PEVAgentState{},
		func(current, new PEVAgentState) (PEVAgentState, error) {
			// Append messages and intermediate steps
			current.Messages = append(current.Messages, new.Messages...)
			current.IntermediateSteps = append(current.IntermediateSteps, new.IntermediateSteps...)
			// Overwrite other fields
			current.Plan = new.Plan
			current.CurrentStep = new.CurrentStep
			current.LastToolResult = new.LastToolResult
			current.Retries = new.Retries
			current.VerificationResult = new.VerificationResult
			current.FinalAnswer = new.FinalAnswer
			return current, nil
		},
	)
	workflow.SetSchema(schema)

	// Add planner node
	workflow.AddNode("planner", "Create or revise execution plan", func(ctx context.Context, state PEVAgentState) (PEVAgentState, error) {
		return plannerNodeTyped(ctx, state, config.Model, config.SystemMessage, config.Verbose)
	})

	// Add executor node
	workflow.AddNode("executor", "Execute the current step using tools", func(ctx context.Context, state PEVAgentState) (PEVAgentState, error) {
		return executorNodeTyped(ctx, state, toolExecutor, config.Model, config.Verbose)
	})

	// Add verifier node
	workflow.AddNode("verifier", "Verify the execution result", func(ctx context.Context, state PEVAgentState) (PEVAgentState, error) {
		return verifierNodeTyped(ctx, state, config.Model, config.VerificationPrompt, config.Verbose)
	})

	// Add synthesizer node
	workflow.AddNode("synthesizer", "Synthesize final answer from all steps", func(ctx context.Context, state PEVAgentState) (PEVAgentState, error) {
		return synthesizerNodeTyped(ctx, state, config.Model, config.Verbose)
	})

	// Set entry point
	workflow.SetEntryPoint("planner")

	// Add conditional edges
	workflow.AddConditionalEdge("planner", func(ctx context.Context, state PEVAgentState) string {
		return routeAfterPlannerTyped(state, config.Verbose)
	})

	workflow.AddConditionalEdge("executor", func(ctx context.Context, state PEVAgentState) string {
		return routeAfterExecutorTyped(state, config.Verbose)
	})

	workflow.AddConditionalEdge("verifier", func(ctx context.Context, state PEVAgentState) string {
		return routeAfterVerifierTyped(state, config.MaxRetries, config.Verbose)
	})

	workflow.AddEdge("synthesizer", graph.END)

	return workflow.Compile()
}

// plannerNodeTyped creates or revises an execution plan (typed version)
func plannerNodeTyped(ctx context.Context, state PEVAgentState, model llms.Model, systemMessage string, verbose bool) (PEVAgentState, error) {
	if len(state.Messages) == 0 {
		return state, fmt.Errorf("no messages in state")
	}

	if verbose {
		if state.Retries == 0 {
			log.Info("planning execution steps...")
		} else {
			log.Info("re-planning (attempt %d)...", state.Retries+1)
		}
	}

	// Build planning prompt
	var promptMessages []llms.MessageContent

	if state.Retries == 0 {
		// Initial planning
		promptMessages = []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeSystem,
				Parts: []llms.ContentPart{llms.TextPart(systemMessage)},
			},
		}
		promptMessages = append(promptMessages, state.Messages...)
	} else {
		// Re-planning after verification failure
		replanPrompt := fmt.Sprintf(`The previous execution failed verification. Please create a revised plan.

Original request:
%s

Previous execution result:
%s

Verification feedback:
%s

Create a new plan that addresses the issues identified.`,
			getOriginalRequest(state.Messages), state.LastToolResult, state.VerificationResult)

		promptMessages = []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeSystem,
				Parts: []llms.ContentPart{llms.TextPart(systemMessage)},
			},
			{
				Role:  llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{llms.TextPart(replanPrompt)},
			},
		}
	}

	// Generate plan
	resp, err := model.GenerateContent(ctx, promptMessages)
	if err != nil {
		return state, fmt.Errorf("failed to generate plan: %w", err)
	}

	planText := resp.Choices[0].Content

	// Parse plan into steps
	steps := parsePlanStepsTyped(planText)

	if verbose {
		log.Info("plan created with %d steps", len(steps))
		for i, step := range steps {
			log.Info("  %d. %s", i+1, step)
		}
		log.Info("")
	}

	return PEVAgentState{
		Messages:     state.Messages,
		Plan:         steps,
		CurrentStep:  0,
		Retries:      state.Retries,
	}, nil
}

// executorNodeTyped executes the current step (typed version)
func executorNodeTyped(ctx context.Context, state PEVAgentState, toolExecutor *ToolExecutor, model llms.Model, verbose bool) (PEVAgentState, error) {
	if len(state.Plan) == 0 {
		return state, fmt.Errorf("no plan found in state")
	}

	if state.CurrentStep >= len(state.Plan) {
		return state, fmt.Errorf("current step index out of bounds")
	}

	stepDescription := state.Plan[state.CurrentStep]

	if verbose {
		log.Info("executing step %d/%d: %s", state.CurrentStep+1, len(state.Plan), stepDescription)
	}

	// Use LLM to decide which tool to call
	result, err := executeStepPEVTyped(ctx, stepDescription, toolExecutor, model)
	if err != nil {
		result = fmt.Sprintf("Error: %v", err)
	}

	if verbose {
		log.Info("result: %s\n", truncateStringTyped(result, 200))
	}

	return PEVAgentState{
		Messages:          state.Messages,
		Plan:              state.Plan,
		CurrentStep:       state.CurrentStep,
		LastToolResult:    result,
		IntermediateSteps: []string{fmt.Sprintf("Step %d: %s -> %s", state.CurrentStep+1, stepDescription, truncateStringTyped(result, 100))},
		Retries:           state.Retries,
	}, nil
}

// verifierNodeTyped verifies the execution result (typed version)
func verifierNodeTyped(ctx context.Context, state PEVAgentState, model llms.Model, verificationPrompt string, verbose bool) (PEVAgentState, error) {
	stepDescription := state.Plan[state.CurrentStep]

	if verbose {
		log.Info("verifying execution result...")
	}

	// Build verification prompt
	verifyPrompt := fmt.Sprintf(`Verify if the following execution was successful:

Intended action:
%s

Execution result:
%s

Determine if this result indicates success or failure. Respond with JSON in this exact format:
{
  "is_successful": true or false,
  "reasoning": "your explanation here"
}`,
		stepDescription, state.LastToolResult)

	promptMessages := []llms.MessageContent{
		{
			Role:  llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{llms.TextPart(verificationPrompt)},
		},
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextPart(verifyPrompt)},
		},
	}

	// Generate verification
	resp, err := model.GenerateContent(ctx, promptMessages)
	if err != nil {
		return state, fmt.Errorf("failed to generate verification: %w", err)
	}

	verificationText := resp.Choices[0].Content

	// Parse verification result
	var verificationResult VerificationResult
	if err := parseVerificationResultTyped(verificationText, &verificationResult); err != nil {
		// If parsing fails, assume failure for safety
		verificationResult = VerificationResult{
			IsSuccessful: false,
			Reasoning:    fmt.Sprintf("Failed to parse verification result: %v", err),
		}
	}

	if verbose {
		if verificationResult.IsSuccessful {
			log.Info("verification passed: %s\n", verificationResult.Reasoning)
		} else {
			log.Error("verification failed: %s\n", verificationResult.Reasoning)
		}
	}

	return PEVAgentState{
		Messages:           state.Messages,
		Plan:               state.Plan,
		CurrentStep:        state.CurrentStep,
		LastToolResult:     state.LastToolResult,
		IntermediateSteps:  state.IntermediateSteps,
		Retries:            state.Retries,
		VerificationResult: verificationResult.Reasoning,
	}, nil
}

// synthesizerNodeTyped creates the final answer from all intermediate steps (typed version)
func synthesizerNodeTyped(ctx context.Context, state PEVAgentState, model llms.Model, verbose bool) (PEVAgentState, error) {
	if verbose {
		log.Info("synthesizing final answer...")
	}

	originalRequest := getOriginalRequest(state.Messages)

	// Build synthesis prompt
	stepsText := strings.Join(state.IntermediateSteps, "\n")
	synthesisPrompt := fmt.Sprintf(`Based on the following execution steps, provide a final answer to the user's request.

User request:
%s

Execution steps:
%s

Provide a clear, concise final answer that directly addresses the user's request.`,
		originalRequest, stepsText)

	promptMessages := []llms.MessageContent{
		{
			Role:  llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{llms.TextPart("You are a helpful assistant synthesizing results from a multi-step execution.")},
		},
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextPart(synthesisPrompt)},
		},
	}

	// Generate final answer
	resp, err := model.GenerateContent(ctx, promptMessages)
	if err != nil {
		return state, fmt.Errorf("failed to generate final answer: %w", err)
	}

	finalAnswer := resp.Choices[0].Content

	if verbose {
		log.Info("final answer generated\n")
	}

	// Create AI message
	aiMsg := llms.MessageContent{
		Role:  llms.ChatMessageTypeAI,
		Parts: []llms.ContentPart{llms.TextPart(finalAnswer)},
	}

	return PEVAgentState{
		Messages:           []llms.MessageContent{aiMsg},
		Plan:               state.Plan,
		CurrentStep:        state.CurrentStep,
		LastToolResult:     state.LastToolResult,
		IntermediateSteps:  state.IntermediateSteps,
		Retries:            state.Retries,
		VerificationResult: state.VerificationResult,
		FinalAnswer:        finalAnswer,
	}, nil
}

// Routing functions (typed versions)

func routeAfterPlannerTyped(state PEVAgentState, verbose bool) string {
	if len(state.Plan) == 0 {
		if verbose {
			log.Warn("no plan created, ending")
		}
		return graph.END
	}

	return "executor"
}

func routeAfterExecutorTyped(state PEVAgentState, verbose bool) string {
	// After execution, always verify
	return "verifier"
}

func routeAfterVerifierTyped(state PEVAgentState, maxRetries int, verbose bool) string {
	isSuccessful := strings.Contains(strings.ToLower(state.VerificationResult), "success") ||
		!strings.Contains(strings.ToLower(state.VerificationResult), "fail")

	nextStep := state.CurrentStep + 1

	if isSuccessful {
		// Move to next step
		if nextStep >= len(state.Plan) {
			// All steps completed successfully
			if verbose {
				log.Info("all steps completed successfully, synthesizing final answer")
			}
			return "synthesizer"
		}

		// Continue to next step (need to update state in place)
		return "executor"
	}

	// Verification failed
	if state.Retries >= maxRetries {
		if verbose {
			log.Error("max retries (%d) reached, synthesizing with partial results\n", maxRetries)
		}
		return "synthesizer"
	}

	// Retry with re-planning
	return "planner"
}

// Helper functions (typed versions)

func parsePlanStepsTyped(planText string) []string {
	lines := strings.Split(planText, "\n")
	var steps []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Remove common step prefixes (1., -, *, etc.)
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")

		// Remove numbered prefixes like "1.", "2.", etc.
		parts := strings.SplitN(line, ".", 2)
		if len(parts) == 2 {
			if _, err := fmt.Sscanf(parts[0], "%d", new(int)); err == nil {
				line = strings.TrimSpace(parts[1])
			}
		}

		if line != "" {
			steps = append(steps, line)
		}
	}

	return steps
}

func executeStepPEVTyped(ctx context.Context, stepDescription string, toolExecutor *ToolExecutor, model llms.Model) (string, error) {
	if toolExecutor == nil || len(toolExecutor.tools) == 0 {
		return fmt.Sprintf("Error: No tools available to execute %s", stepDescription), nil
	}

	// 1. Build tool definitions string
	var toolsInfo strings.Builder
	for name, tool := range toolExecutor.tools {
		toolsInfo.WriteString(fmt.Sprintf("- %s: %s\n", name, tool.Description()))
	}

	// 2. Build prompt
	prompt := fmt.Sprintf(`You are an autonomous agent execution step.
Task: %s

Available Tools:
%s

Select the most appropriate tool to execute this task.
Return ONLY a JSON object with the following format:
{
  "tool": "tool_name",
  "tool_input": "input_string"
}
`, stepDescription, toolsInfo.String())

	// 3. Call LLM
	promptMessages := []llms.MessageContent{
		{
			Role:  llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{llms.TextPart("You are a helpful assistant that selects the best tool for a task.")},
		},
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextPart(prompt)},
		},
	}

	resp, err := model.GenerateContent(ctx, promptMessages)
	if err != nil {
		return "", fmt.Errorf("failed to generate tool choice: %w", err)
	}

	choiceText := resp.Choices[0].Content

	// 4. Parse response
	var invocation ToolInvocation
	if err := parseToolChoiceTyped(choiceText, &invocation); err != nil {
		return "", fmt.Errorf("failed to parse tool choice: %w (Response: %s)", err, choiceText)
	}

	// 5. Execute tool
	return toolExecutor.Execute(ctx, invocation)
}

func parseToolChoiceTyped(text string, invocation *ToolInvocation) error {
	// Try to find JSON in the text
	text = strings.TrimSpace(text)

	// Look for JSON object
	startIdx := strings.Index(text, "{")
	endIdx := strings.LastIndex(text, "}")

	if startIdx == -1 || endIdx == -1 {
		return fmt.Errorf("no JSON object found in text")
	}

	jsonText := text[startIdx : endIdx+1]

	if err := json.Unmarshal([]byte(jsonText), invocation); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	return nil
}

func parseVerificationResultTyped(text string, result *VerificationResult) error {
	// Try to find JSON in the text
	text = strings.TrimSpace(text)

	// Look for JSON object
	startIdx := strings.Index(text, "{")
	endIdx := strings.LastIndex(text, "}")

	if startIdx == -1 || endIdx == -1 {
		return fmt.Errorf("no JSON object found in text")
	}

	jsonText := text[startIdx : endIdx+1]

	if err := json.Unmarshal([]byte(jsonText), result); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	return nil
}

func truncateStringTyped(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func buildDefaultPlannerPromptTyped() string {
	return `You are an expert planner that breaks down user requests into concrete, executable steps.

Your task is to:
1. Analyze the user's request carefully
2. Break it down into clear, sequential steps
3. Each step should be specific and actionable
4. Number each step clearly

Format your plan as a numbered list:
1. First step
2. Second step
3. Third step
...

Be concise but specific. Each step should be something that can be executed using available tools.`
}

func buildDefaultVerificationPromptTyped() string {
	return `You are a verification specialist that checks if executions were successful.

Your task is to:
1. Analyze the intended action and the actual result
2. Determine if the result indicates success or failure
3. Provide clear reasoning for your determination

Indicators of success:
- Valid data returned
- Positive confirmation messages
- Expected format/structure

Indicators of failure:
- Error messages
- Null/empty results when data expected
- Timeout or connection errors
- Invalid or unexpected format

Always respond with JSON in this exact format:
{
  "is_successful": true or false,
  "reasoning": "explain why you determined success or failure"
}`
}
