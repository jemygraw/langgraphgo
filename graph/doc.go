// Package graph provides the core graph construction and execution engine for LangGraph Go.
//
// This package implements the fundamental building blocks for creating stateful, multi-agent applications
// using directed graphs. It offers both untyped and typed interfaces for building workflows,
// with support for parallel execution, checkpointing, streaming, and comprehensive event handling.
//
// # Core Concepts
//
// ## StateGraph
// The primary component for building graphs is StateGraph, which maintains state as it flows
// through nodes. Each node can process and transform the state before passing it to the next node
// based on defined edges.
//
// ## Nodes and Edges
// Nodes represent processing units (functions, agents, tools) that transform state.
// Edges define the flow between nodes, supporting conditional routing based on state content.
//
// ## Typed Support
// For type safety, the package provides StateGraphTyped[S] which uses Go generics to enforce
// state types at compile time, reducing runtime errors and improving code maintainability.
//
// # Key Features
//
//   - Parallel node execution with coordination
//   - Checkpointing for durable execution with resume capability
//   - Streaming for real-time event monitoring
//   - Comprehensive listener system for observability
//   - Built-in retry mechanisms with configurable policies
//   - Subgraph composition for modular design
//   - Graph visualization (Mermaid, PlantUML)
//   - Interrupt support for human-in-the-loop workflows
//
// # Example Usage
//
// ## Basic State Graph
//
//	g := graph.NewStateGraph()
//
//	// Add nodes
//	g.AddNode("process", func(ctx context.Context, state map[string]any) (map[string]any, error) {
//		// Process the state
//		state["processed"] = true
//		return state, nil
//	})
//
//	g.AddNode("validate", func(ctx context.Context, state map[string]any) (map[string]any, error) {
//		// Validate the processed state
//		if state["processed"].(bool) {
//			state["valid"] = true
//		}
//		return state, nil
//	})
//
//	// Set entry point and edges
//	g.SetEntry("process")
//	g.AddEdge("process", "validate")
//	g.AddEdge("validate", graph.END)
//
//	// Compile and run
//	runnable := g.Compile()
//	result, err := runnable.Invoke(context.Background(), map[string]any{
//		"data": "example",
//	})
//
// ## Typed State Graph
//
//	type WorkflowState struct {
//		Input    string `json:"input"`
//		Output   string `json:"output"`
//		Complete bool   `json:"complete"`
//	}
//
//	g := graph.NewStateGraphTyped(func() WorkflowState { return WorkflowState{} })
//
//	g.AddNodeTyped("process", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
//		state.Output = strings.ToUpper(state.Input)
//		state.Complete = true
//		return state, nil
//	})
//
//	// Conditional routing
//	g.AddConditionalEdge("process", func(ctx context.Context, state WorkflowState) string {
//		if state.Complete {
//			return "next"
//		}
//		return "retry"
//	}, "next", "retry")
//
// ## Parallel Execution
//
//	g.AddNodeParallel("parallel_tasks",
//		graph.NewParallelNode(
//			[]graph.Node{
//				{Name: "task1", Function: task1Func},
//				{Name: "task2", Function: task2Func},
//			},
//		),
//	)
//
// ## Checkpointing
//
//	store := graph.NewMemoryCheckpointStore()
//	g.WithCheckpointing(graph.CheckpointConfig{
//		Store: store,
//	})
//
//	// Execute with checkpoint
//	runnable := g.Compile()
//	result, err := runnable.Invoke(context.Background(), initialState,
//		graph.WithExecutionID("workflow-123"))
//
//	// Resume from checkpoint
//	resumed, err := runnable.Resume(context.Background(), "workflow-123", "checkpoint-456")
//
// ## Streaming
//
//	streaming := graph.NewStreamingStateGraph(g, graph.StreamConfig{
//		BufferSize: 100,
//	})
//
//	runnable := streaming.Compile()
//	result, err := runnable.Stream(context.Background(), initialState)
//
//	// Process events
//	for event := range result.Events {
//		fmt.Printf("Event: %v\n", event)
//	}
//
// # Listener System
//
// The package provides a powerful listener system for monitoring and reacting to graph events:
//
//   - ProgressListener: Track execution progress
//   - LoggingListener: Structured logging of events
//   - MetricsListener: Collect performance metrics
//   - ChatListener: Chat-style output formatting
//   - Custom listeners: Implement NodeListener interface
//
// # Error Handling
//
//   - Built-in retry policies with exponential backoff
//   - Custom error filtering for selective retries
//   - Interrupt handling for pausing execution
//   - Comprehensive error context in events
//
// # Visualization
//
// Export graphs for documentation and debugging:
//
//	exporter := graph.NewExporter(g)
//
//	// Mermaid diagram
//	mermaid, err := exporter.Mermaid(graph.MermaidOptions{
//		Direction: "TD",
//	})
//
//	// PlantUML diagram
//	puml, err := exporter.PlantUML()
//
// # Thread Safety
//
// All graph structures are thread-safe for read operations. Write operations (adding nodes,
// edges, or listeners) should be performed before compilation or protected by external synchronization.
//
// # Best Practices
//
//  1. Use typed graphs when possible for better type safety
//  2. Set appropriate buffer sizes for streaming to balance memory and performance
//  3. Implement proper error handling in node functions
//  4. Use checkpoints for long-running or critical workflows
//  5. Add listeners for debugging and monitoring
//  6. Keep node functions pure and stateless when possible
//  7. Use conditional edges for complex routing logic
//  8. Leverage parallel execution for independent tasks
package graph
