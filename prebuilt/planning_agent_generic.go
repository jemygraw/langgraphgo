package prebuilt

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/log"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// CreatePlanningAgentTyped creates a generic planning agent that first plans the workflow using LLM,
// then executes according to the generated plan.
// This version uses fixed PlanningAgentState for full type safety.
func CreatePlanningAgentTyped(model llms.Model, nodes []*graph.Node, inputTools []tools.Tool, opts ...CreateAgentOption) (*graph.StateRunnable[PlanningAgentState], error) {
	options := &CreateAgentOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Create a map of node names to nodes for easy lookup
	nodeMap := make(map[string]*graph.Node)
	for _, node := range nodes {
		nodeMap[node.Name] = node
	}

	// Define the workflow with generic state type
	workflow := graph.NewStateGraph[PlanningAgentState]()

	// Define the state schema for merging
	schema := graph.NewStructSchema(
		PlanningAgentState{},
		func(current, new PlanningAgentState) (PlanningAgentState, error) {
			// Append new messages to current messages
			current.Messages = append(current.Messages, new.Messages...)
			// Overwrite workflow plan
			current.WorkflowPlan = new.WorkflowPlan
			return current, nil
		},
	)
	workflow.SetSchema(schema)

	// Add planning node - this is where LLM generates the workflow
	workflow.AddNode("planner", "Generates workflow plan based on user request", func(ctx context.Context, state PlanningAgentState) (PlanningAgentState, error) {
		if len(state.Messages) == 0 {
			return state, fmt.Errorf("no messages in state")
		}

		// Build the planning prompt
		nodeDescriptions := buildNodeDescriptionsTyped(nodes)
		planningPrompt := buildPlanningPromptTyped(nodeDescriptions)

		// Prepare messages for LLM
		planningMessages := []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeSystem,
				Parts: []llms.ContentPart{llms.TextPart(planningPrompt)},
			},
		}
		planningMessages = append(planningMessages, state.Messages...)

		if options.Verbose {
			log.Info("planning workflow...")
		}

		// Call LLM to generate the plan
		resp, err := model.GenerateContent(ctx, planningMessages)
		if err != nil {
			return state, fmt.Errorf("failed to generate plan: %w", err)
		}

		planText := resp.Choices[0].Content
		if options.Verbose {
			log.Info("generated plan:\n%s\n", planText)
		}

		// Parse the workflow plan
		workflowPlan, err := parseWorkflowPlanTyped(planText)
		if err != nil {
			return state, fmt.Errorf("failed to parse workflow plan: %w", err)
		}

		// Store the plan in state
		aiMsg := llms.MessageContent{
			Role:  llms.ChatMessageTypeAI,
			Parts: []llms.ContentPart{llms.TextPart(fmt.Sprintf("Workflow plan created with %d nodes and %d edges", len(workflowPlan.Nodes), len(workflowPlan.Edges)))},
		}

		return PlanningAgentState{
			Messages:      []llms.MessageContent{aiMsg},
			WorkflowPlan:  workflowPlan,
		}, nil
	})

	// Add executor node - this builds and executes the planned workflow
	workflow.AddNode("executor", "Executes the planned workflow", func(ctx context.Context, state PlanningAgentState) (PlanningAgentState, error) {
		if state.WorkflowPlan == nil {
			return state, fmt.Errorf("workflow_plan not found in state")
		}

		if options.Verbose {
			log.Info("executing planned workflow...")
		}

		// Build the dynamic workflow using untyped graph for flexibility
		// Note: This is a special case where we need dynamic graph construction
		dynamicWorkflow := graph.NewStateGraph[map[string]any]()
		dynamicSchema := graph.NewMapSchema()
		dynamicSchema.RegisterReducer("messages", graph.AppendReducer)
		// Wrap in adapter to match StateSchemaTyped[map[string]any]
		schemaAdapter := &graph.MapSchemaAdapter{Schema: dynamicSchema}
		dynamicWorkflow.SetSchema(schemaAdapter)

		// Add nodes from the plan
		for _, planNode := range state.WorkflowPlan.Nodes {
			if planNode.Name == "START" || planNode.Name == "END" {
				continue // Skip special nodes
			}

			actualNode, exists := nodeMap[planNode.Name]
			if !exists {
				return state, fmt.Errorf("node %s not found in available nodes", planNode.Name)
			}

			// Wrap the node function to match the typed signature
			wrappedFn := func(ctx context.Context, s map[string]any) (map[string]any, error) {
				result, err := actualNode.Function(ctx, s)
				if err != nil {
					return nil, err
				}
				if resultMap, ok := result.(map[string]any); ok {
					return resultMap, nil
				}
				return s, nil
			}
			dynamicWorkflow.AddNode(actualNode.Name, actualNode.Description, wrappedFn)

			if options.Verbose {
				log.Info("added node: %s", actualNode.Name)
			}
		}

		// Add edges from the plan
		var entryPoint string
		endNodes := make(map[string]bool) // Track nodes that should end

		for _, edge := range state.WorkflowPlan.Edges {
			if edge.From == "START" {
				entryPoint = edge.To
				continue
			}
			if edge.To == "END" {
				endNodes[edge.From] = true
				continue // Will be handled after all edges are added
			}

			if edge.Condition != "" {
				// This is a conditional edge
				// For now, we'll add a simple conditional edge
				// In a real implementation, you might want to parse the condition
				dynamicWorkflow.AddConditionalEdge(edge.From, func(ctx context.Context, s map[string]any) string {
					// Simple condition evaluation
					// You can enhance this to evaluate the actual condition
					return edge.To
				})
			} else {
				dynamicWorkflow.AddEdge(edge.From, edge.To)
			}

			if options.Verbose {
				log.Info("  added edge: %s -> %s", edge.From, edge.To)
			}
		}

		// Add edges to END for terminal nodes
		for nodeName := range endNodes {
			dynamicWorkflow.AddEdge(nodeName, graph.END)
			if options.Verbose {
				log.Info("  added edge: %s -> END", nodeName)
			}
		}

		if entryPoint == "" {
			return state, fmt.Errorf("no entry point found in workflow plan")
		}

		dynamicWorkflow.SetEntryPoint(entryPoint)

		// Compile and execute the dynamic workflow
		runnable, err := dynamicWorkflow.Compile()
		if err != nil {
			return state, fmt.Errorf("failed to compile dynamic workflow: %w", err)
		}

		// Convert PlanningAgentState to map[string]any for execution
		stateMap := map[string]any{
			"messages":      state.Messages,
			"workflow_plan": state.WorkflowPlan,
		}

		// Execute the dynamic workflow with current state
		result, err := runnable.Invoke(ctx, stateMap)
		if err != nil {
			return state, fmt.Errorf("failed to execute dynamic workflow: %w", err)
		}

		if options.Verbose {
			log.Info("workflow execution completed")
		}

		// result is already map[string]any
		resultMap := result

		messages, ok := resultMap["messages"].([]llms.MessageContent)
		if !ok {
			messages = state.Messages
		}

		return PlanningAgentState{
			Messages:     messages,
			WorkflowPlan: state.WorkflowPlan,
		}, nil
	})

	// Define edges
	workflow.SetEntryPoint("planner")
	workflow.AddEdge("planner", "executor")
	workflow.AddEdge("executor", graph.END)

	return workflow.Compile()
}

// buildNodeDescriptionsTyped creates a formatted string describing all available nodes
func buildNodeDescriptionsTyped(nodes []*graph.Node) string {
	var sb strings.Builder
	sb.WriteString("Available nodes:\n")
	for i, node := range nodes {
		sb.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, node.Name, node.Description))
	}
	return sb.String()
}

// buildPlanningPromptTyped creates the prompt for the LLM to generate a workflow plan
func buildPlanningPromptTyped(nodeDescriptions string) string {
	return fmt.Sprintf(`You are a workflow planning assistant. Based on the user's request, create a workflow plan using the available nodes.

%s

Generate a workflow plan in the following JSON format:
{
  "nodes": [
    {"name": "node_name", "type": "process"}
  ],
  "edges": [
    {"from": "START", "to": "first_node"},
    {"from": "first_node", "to": "second_node"},
    {"from": "last_node", "to": "END"}
  ]
}

Rules:
1. The workflow must start with an edge from "START"
2. The workflow must end with an edge to "END"
3. Only use nodes from the available nodes list
4. Each node should appear in the nodes array
5. Create a logical flow based on the user's request
6. Return ONLY the JSON object, no additional text

Example:
{
  "nodes": [
    {"name": "research", "type": "process"},
    {"name": "analyze", "type": "process"}
  ],
  "edges": [
    {"from": "START", "to": "research"},
    {"from": "research", "to": "analyze"},
    {"from": "analyze", "to": "END"}
  ]
}`, nodeDescriptions)
}

// parseWorkflowPlanTyped parses the LLM response to extract the workflow plan
func parseWorkflowPlanTyped(planText string) (*WorkflowPlan, error) {
	// Extract JSON from the response (handle markdown code blocks)
	jsonText := extractJSONTyped(planText)

	var plan WorkflowPlan
	if err := json.Unmarshal([]byte(jsonText), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate the plan
	if len(plan.Nodes) == 0 {
		return nil, fmt.Errorf("workflow plan has no nodes")
	}
	if len(plan.Edges) == 0 {
		return nil, fmt.Errorf("workflow plan has no edges")
	}

	return &plan, nil
}

// extractJSONTyped extracts JSON from a text that might contain markdown code blocks
func extractJSONTyped(text string) string {
	// Try to find JSON in markdown code block
	codeBlockRegex := regexp.MustCompile("(?s)```(?:json)?\\s*({.*?})\\s*```")
	matches := codeBlockRegex.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}

	// Try to find JSON object directly
	jsonRegex := regexp.MustCompile("(?s){.*}")
	matches = jsonRegex.FindStringSubmatch(text)
	if len(matches) > 0 {
		return matches[0]
	}

	return text
}
