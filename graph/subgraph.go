package graph

import (
	"context"
	"fmt"
)

// Subgraph represents a nested graph that can be used as a node
type Subgraph struct {
	name     string
	graph    *StateGraph[map[string]any]
	runnable *StateRunnable[map[string]any]
}

// NewSubgraph creates a new subgraph
func NewSubgraph(name string, graph *StateGraph[map[string]any]) (*Subgraph, error) {
	runnable, err := graph.Compile()
	if err != nil {
		return nil, fmt.Errorf("failed to compile subgraph %s: %w", name, err)
	}

	return &Subgraph{
		name:     name,
		graph:    graph,
		runnable: runnable,
	}, nil
}

// Execute runs the subgraph as a node
func (s *Subgraph) Execute(ctx context.Context, state any) (any, error) {
	// Convert state to map[string]any if needed
	var stateMap map[string]any
	if sm, ok := state.(map[string]any); ok {
		stateMap = sm
	} else {
		stateMap = map[string]any{"state": state}
	}

	result, err := s.runnable.Invoke(ctx, stateMap)
	if err != nil {
		return nil, fmt.Errorf("subgraph %s execution failed: %w", s.name, err)
	}
	return result, nil
}

// AddSubgraph adds a subgraph as a node in the parent graph
func AddSubgraph[S any](g *StateGraph[S], name string, subgraph *StateGraph[map[string]any], converter func(S) map[string]any, resultConverter func(map[string]any) S) error {
	sg, err := NewSubgraph(name, subgraph)
	if err != nil {
		return err
	}

	// Wrap the execute function to match the state type
	wrappedFn := func(ctx context.Context, state S) (S, error) {
		// Convert S to map[string]any
		stateMap := converter(state)
		result, err := sg.Execute(ctx, stateMap)
		if err != nil {
			var zero S
			return zero, err
		}
		// Execute returns any, need to assert to map[string]any
		resultMap, ok := result.(map[string]any)
		if !ok {
			var zero S
			return zero, fmt.Errorf("subgraph %s did not return map[string]any", name)
		}
		// Convert result back to S
		return resultConverter(resultMap), nil
	}

	g.AddNode(name, "Subgraph: "+name, wrappedFn)
	return nil
}

// CreateSubgraph creates and adds a subgraph using a builder function
func CreateSubgraph[S any](g *StateGraph[S], name string, builder func(*StateGraph[map[string]any]) error, converter func(S) map[string]any, resultConverter func(map[string]any) S) {
	subgraph := NewStateGraph[map[string]any]()
	builder(subgraph)
	_ = AddSubgraph(g, name, subgraph, converter, resultConverter)
}

// CompositeGraph allows composing multiple graphs together
type CompositeGraph struct {
	graphs map[string]*StateGraph[map[string]any]
	main   *StateGraph[map[string]any]
}

// NewCompositeGraph creates a new composite graph
func NewCompositeGraph() *CompositeGraph {
	return &CompositeGraph{
		graphs: make(map[string]*StateGraph[map[string]any]),
		main:   NewStateGraph[map[string]any](),
	}
}

// AddGraph adds a named graph to the composite
func (cg *CompositeGraph) AddGraph(name string, graph *StateGraph[map[string]any]) {
	cg.graphs[name] = graph
}

// Connect connects two graphs with a transformation function
func (cg *CompositeGraph) Connect(
	fromGraph string,
	fromNode string,
	toGraph string,
	toNode string,
	transform func(any) any,
) error {
	// Create a bridge node that transforms state between graphs
	bridgeName := fmt.Sprintf("%s_%s_to_%s_%s", fromGraph, fromNode, toGraph, toNode)

	cg.main.AddNode(bridgeName, "Bridge: "+bridgeName, func(_ context.Context, state map[string]any) (map[string]any, error) {
		if transform != nil {
			result := transform(state)
			if resultMap, ok := result.(map[string]any); ok {
				return resultMap, nil
			}
		}
		return state, nil
	})

	return nil
}

// Compile compiles the composite graph into a single runnable
func (cg *CompositeGraph) Compile() (*StateRunnable[map[string]any], error) {
	// Add all subgraphs to the main graph
	for name, graph := range cg.graphs {
		if err := AddSubgraph(cg.main, name, graph,
			func(s map[string]any) map[string]any { return s },
			func(s map[string]any) map[string]any { return s }); err != nil {
			return nil, fmt.Errorf("failed to add subgraph %s: %w", name, err)
		}
	}

	return cg.main.Compile()
}

// RecursiveSubgraph allows a subgraph to call itself recursively
type RecursiveSubgraph struct {
	name      string
	graph     *StateGraph[map[string]any]
	maxDepth  int
	condition func(any, int) bool // Should continue recursion?
}

// NewRecursiveSubgraph creates a new recursive subgraph
func NewRecursiveSubgraph(
	name string,
	maxDepth int,
	condition func(any, int) bool,
) *RecursiveSubgraph {
	return &RecursiveSubgraph{
		name:      name,
		graph:     NewStateGraph[map[string]any](),
		maxDepth:  maxDepth,
		condition: condition,
	}
}

// Execute runs the recursive subgraph
func (rs *RecursiveSubgraph) Execute(ctx context.Context, state any) (any, error) {
	return rs.executeRecursive(ctx, state, 0)
}

func (rs *RecursiveSubgraph) executeRecursive(ctx context.Context, state any, depth int) (any, error) {
	// Check max depth
	if depth >= rs.maxDepth {
		return state, nil
	}

	// Check condition
	if !rs.condition(state, depth) {
		return state, nil
	}

	// Convert state to map[string]any
	var stateMap map[string]any
	if sm, ok := state.(map[string]any); ok {
		stateMap = sm
	} else {
		stateMap = map[string]any{"state": state}
	}

	// Compile and execute the graph
	runnable, err := rs.graph.Compile()
	if err != nil {
		return nil, fmt.Errorf("failed to compile recursive subgraph at depth %d: %w", depth, err)
	}

	result, err := runnable.Invoke(ctx, stateMap)
	if err != nil {
		return nil, fmt.Errorf("recursive execution failed at depth %d: %w", depth, err)
	}

	// Recurse with the result
	return rs.executeRecursive(ctx, result, depth+1)
}

// AddRecursiveSubgraph adds a recursive subgraph to the parent graph
func AddRecursiveSubgraph[S any](
	g *StateGraph[S],
	name string,
	maxDepth int,
	condition func(any, int) bool,
	builder func(*StateGraph[map[string]any]),
	converter func(S) map[string]any,
	resultConverter func(map[string]any) S,
) {
	rs := NewRecursiveSubgraph(name, maxDepth, condition)
	builder(rs.graph)

	wrappedFn := func(ctx context.Context, state S) (S, error) {
		stateMap := converter(state)
		result, err := rs.Execute(ctx, stateMap)
		if err != nil {
			var zero S
			return zero, err
		}
		// Execute returns any, need to assert to map[string]any
		resultMap, ok := result.(map[string]any)
		if !ok {
			var zero S
			return zero, fmt.Errorf("recursive subgraph did not return map[string]any")
		}
		return resultConverter(resultMap), nil
	}

	g.AddNode(name, "Recursive subgraph: "+name, wrappedFn)
}

// NestedConditionalSubgraph creates a subgraph with its own conditional routing
func AddNestedConditionalSubgraph[S any](
	g *StateGraph[S],
	name string,
	router func(S) string,
	subgraphs map[string]*StateGraph[map[string]any],
	converter func(S) map[string]any,
	resultConverter func(map[string]any) S,
) error {
	// Create a wrapper node that routes to different subgraphs
	wrappedFn := func(ctx context.Context, state S) (S, error) {
		// Determine which subgraph to use
		subgraphName := router(state)

		subgraph, exists := subgraphs[subgraphName]
		if !exists {
			var zero S
			return zero, fmt.Errorf("subgraph %s not found", subgraphName)
		}

		// Convert state to map[string]any
		stateMap := converter(state)

		// Compile and execute the selected subgraph
		runnable, err := subgraph.Compile()
		if err != nil {
			var zero S
			return zero, fmt.Errorf("failed to compile subgraph %s: %w", subgraphName, err)
		}

		result, err := runnable.Invoke(ctx, stateMap)
		if err != nil {
			var zero S
			return zero, err
		}

		// Convert result back to S
		return resultConverter(result), nil
	}

	g.AddNode(name, "Nested conditional subgraph: "+name, wrappedFn)
	return nil
}
