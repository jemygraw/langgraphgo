package prebuilt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// CreateAgentTyped creates a new agent graph with full type safety.
// This is a simplified version without skill discovery support.
// For full functionality including skill discovery, use the original CreateAgent.
func CreateAgentTyped(model llms.Model, inputTools []tools.Tool, opts ...CreateAgentOption) (*graph.StateRunnable[AgentState], error) {
	options := &CreateAgentOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Define the graph with generic state type
	workflow := graph.NewStateGraph[AgentState]()

	// Define the state schema for merging
	schema := graph.NewStructSchema(
		AgentState{},
		func(current, new AgentState) (AgentState, error) {
			// Append messages and extra tools
			current.Messages = append(current.Messages, new.Messages...)
			current.ExtraTools = append(current.ExtraTools, new.ExtraTools...)
			return current, nil
		},
	)
	workflow.SetSchema(schema)

	// Define the agent node
	workflow.AddNode("agent", "Agent decision node with LLM", func(ctx context.Context, state AgentState) (AgentState, error) {
		if len(state.Messages) == 0 {
			return state, fmt.Errorf("no messages in state")
		}

		// Combine input tools with extra tools
		var allTools []tools.Tool
		allTools = append(allTools, inputTools...)
		allTools = append(allTools, state.ExtraTools...)

		// Convert tools to ToolInfo for the model
		var toolDefs []llms.Tool
		for _, t := range allTools {
			toolDefs = append(toolDefs, llms.Tool{
				Type: "function",
				Function: &llms.FunctionDefinition{
					Name:        t.Name(),
					Description: t.Description(),
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"input": map[string]any{
								"type":        "string",
								"description": "The input query for the tool",
							},
						},
						"required":             []string{"input"},
						"additionalProperties": false,
					},
				},
			})
		}

		// We need to pass tools to the model
		callOpts := []llms.CallOption{
			llms.WithTools(toolDefs),
		}

		// Apply StateModifier if provided
		msgsToSend := state.Messages

		// Prepend system message if provided
		if options.SystemMessage != "" {
			sysMsg := llms.TextParts(llms.ChatMessageTypeSystem, options.SystemMessage)
			msgsToSend = append([]llms.MessageContent{sysMsg}, msgsToSend...)
		}

		// Apply StateModifier if provided
		if options.StateModifier != nil {
			msgsToSend = options.StateModifier(msgsToSend)
		}

		resp, err := model.GenerateContent(ctx, msgsToSend, callOpts...)
		if err != nil {
			return state, err
		}

		choice := resp.Choices[0]

		// Create AIMessage
		aiMsg := llms.MessageContent{
			Role: llms.ChatMessageTypeAI,
		}

		if choice.Content != "" {
			aiMsg.Parts = append(aiMsg.Parts, llms.TextPart(choice.Content))
		}

		// Handle tool calls
		if len(choice.ToolCalls) > 0 {
			for _, tc := range choice.ToolCalls {
				aiMsg.Parts = append(aiMsg.Parts, tc)
			}
		}

		return AgentState{
			Messages:    []llms.MessageContent{aiMsg},
			ExtraTools: state.ExtraTools,
		}, nil
	})

	// Define the tools node
	workflow.AddNode("tools", "Tool execution node", func(ctx context.Context, state AgentState) (AgentState, error) {
		if len(state.Messages) == 0 {
			return state, fmt.Errorf("no messages in state")
		}

		lastMsg := state.Messages[len(state.Messages)-1]
		if lastMsg.Role != llms.ChatMessageTypeAI {
			return state, fmt.Errorf("last message is not an AI message")
		}

		var toolMessages []llms.MessageContent

		for _, part := range lastMsg.Parts {
			if tc, ok := part.(llms.ToolCall); ok {
				// Parse arguments to get input
				var args map[string]any
				_ = json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args)

				inputVal := ""
				if val, ok := args["input"].(string); ok {
					inputVal = val
				} else {
					inputVal = tc.FunctionCall.Arguments
				}

				// Combine input tools with extra tools for execution
				var allTools []tools.Tool
				allTools = append(allTools, inputTools...)
				allTools = append(allTools, state.ExtraTools...)

				// Create a temporary executor for this run
				currentToolExecutor := NewToolExecutor(allTools)

				// Execute tool
				res, err := currentToolExecutor.Execute(ctx, ToolInvocation{
					Tool:      tc.FunctionCall.Name,
					ToolInput: inputVal,
				})
				if err != nil {
					res = fmt.Sprintf("Error: %v", err)
				}

				// Create ToolMessage
				toolMsg := llms.MessageContent{
					Role: llms.ChatMessageTypeTool,
					Parts: []llms.ContentPart{
						llms.ToolCallResponse{
							ToolCallID: tc.ID,
							Name:       tc.FunctionCall.Name,
							Content:    res,
						},
					},
				}
				toolMessages = append(toolMessages, toolMsg)
			}
		}

		return AgentState{
			Messages:    toolMessages,
			ExtraTools: state.ExtraTools,
		}, nil
	})

	// Define edges
	workflow.SetEntryPoint("agent")

	workflow.AddConditionalEdge("agent", func(ctx context.Context, state AgentState) string {
		if len(state.Messages) == 0 {
			return graph.END
		}

		lastMsg := state.Messages[len(state.Messages)-1]
		hasToolCalls := false
		for _, part := range lastMsg.Parts {
			if _, ok := part.(llms.ToolCall); ok {
				hasToolCalls = true
				break
			}
		}

		if hasToolCalls {
			return "tools"
		}
		return graph.END
	})

	workflow.AddEdge("tools", "agent")

	return workflow.Compile()
}
