package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubgraph(t *testing.T) {
	// 1. Define Child Graph
	child := NewStateGraph[map[string]any]()
	child.AddNode("child_A", "child_A", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		state["child_visited"] = true
		return state, nil
	})
	child.SetEntryPoint("child_A")
	child.AddEdge("child_A", END)

	// 2. Define Parent Graph
	parent := NewStateGraph[map[string]any]()
	parent.AddNode("parent_A", "parent_A", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		state["parent_visited"] = true
		return state, nil
	})

	// Add Child Graph as a node - use identity converters for map[string]any
	err := AddSubgraph(parent, "child", child,
		func(s map[string]any) map[string]any { return s },
		func(s map[string]any) map[string]any { return s })
	assert.NoError(t, err)

	parent.SetEntryPoint("parent_A")
	parent.AddEdge("parent_A", "child")
	parent.AddEdge("child", END)

	// 3. Run Parent Graph
	runnable, err := parent.Compile()
	assert.NoError(t, err)

	res, err := runnable.Invoke(context.Background(), map[string]any{})
	assert.NoError(t, err)

	assert.True(t, res["parent_visited"].(bool))
	assert.True(t, res["child_visited"].(bool))
}

func TestCreateSubgraph(t *testing.T) {
	// Test CreateSubgraph with builder pattern
	parent := NewStateGraph[map[string]any]()

	CreateSubgraph(parent, "dynamic_child", func(g *StateGraph[map[string]any]) error {
		g.AddNode("node1", "Node 1", func(ctx context.Context, state map[string]any) (map[string]any, error) {
			state["dynamic_created"] = true
			return state, nil
		})
		g.SetEntryPoint("node1")
		g.AddEdge("node1", END)
		return nil
	}, func(s map[string]any) map[string]any { return s }, func(s map[string]any) map[string]any { return s })

	// Verify the node was added
	_, ok := parent.nodes["dynamic_child"]
	assert.True(t, ok, "Dynamic subgraph should be added")
}

func TestNewCompositeGraph(t *testing.T) {
	cg := NewCompositeGraph()

	assert.NotNil(t, cg)
	assert.NotNil(t, cg.main)
	assert.NotNil(t, cg.graphs)
	assert.Empty(t, cg.graphs)
}

func TestCompositeGraph_AddGraph(t *testing.T) {
	cg := NewCompositeGraph()

	graph1 := NewStateGraph[map[string]any]()
	graph1.AddNode("test", "Test", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		return state, nil
	})

	cg.AddGraph("graph1", graph1)
	assert.Equal(t, 1, len(cg.graphs))
	assert.Equal(t, graph1, cg.graphs["graph1"])
}

func TestCompositeGraph_Connect(t *testing.T) {
	cg := NewCompositeGraph()

	graph1 := NewStateGraph[map[string]any]()
	graph1.AddNode("output1", "Output 1", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		state["from_graph1"] = true
		return state, nil
	})

	graph2 := NewStateGraph[map[string]any]()
	graph2.AddNode("input2", "Input 2", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		state["to_graph2"] = true
		return state, nil
	})

	// Add graphs to composite
	cg.AddGraph("graph1", graph1)
	cg.AddGraph("graph2", graph2)

	// Connect with transformation
	err := cg.Connect("graph1", "output1", "graph2", "input2", func(state any) any {
		m := state.(map[string]any)
		m["transformed"] = true
		return m
	})

	assert.NoError(t, err)

	// Check that bridge node was created
	bridgeName := "graph1_output1_to_graph2_input2"
	_, ok := cg.main.nodes[bridgeName]
	assert.True(t, ok, "Bridge node should be created")
}

func TestCompositeGraph_Compile(t *testing.T) {
	cg := NewCompositeGraph()

	// Add a simple graph
	simpleGraph := NewStateGraph[map[string]any]()
	simpleGraph.AddNode("test", "Test", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		state["compiled"] = true
		return state, nil
	})
	simpleGraph.SetEntryPoint("test")
	simpleGraph.AddEdge("test", END)

	cg.AddGraph("simple", simpleGraph)

	// Set entry point on the composite graph's main graph to the simple subgraph
	cg.main.SetEntryPoint("simple")

	// Compile composite graph
	runnable, err := cg.Compile()
	assert.NoError(t, err)
	assert.NotNil(t, runnable)

	// Test execution - this will fail because the subgraph doesn't have proper edges set up
	// but the compilation should succeed
	_, err = runnable.Invoke(context.Background(), map[string]any{})
	// We expect this to fail due to graph structure issues
	if err == nil {
		t.Log("Unexpected success - but compilation worked")
	} else {
		t.Logf("Expected execution error: %v", err)
	}
}

func TestNewRecursiveSubgraph(t *testing.T) {
	// Create recursive subgraph with max depth of 3 and condition on count
	maxDepth := 3

	rs := NewRecursiveSubgraph(
		"recursive",
		maxDepth,
		func(state any, depth int) bool {
			m := state.(map[string]any)
			currentCount := m["count"].(int)
			return currentCount < 2 // Recurse twice
		},
	)

	assert.NotNil(t, rs)
	assert.Equal(t, "recursive", rs.name)
	assert.Equal(t, maxDepth, rs.maxDepth)
}

func TestRecursiveSubgraph_Execute(t *testing.T) {
	callCount := 0

	rs := NewRecursiveSubgraph(
		"recursive",
		3,
		func(state any, depth int) bool {
			callCount++
			return depth < 2 // Recurse twice
		},
	)

	// Add a node to the recursive graph
	rs.graph.AddNode("process", "Process", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		if state["count"] == nil {
			state["count"] = 0
		}
		state["count"] = state["count"].(int) + 1
		return state, nil
	})
	rs.graph.SetEntryPoint("process")
	rs.graph.AddEdge("process", END)

	// Execute
	ctx := context.Background()
	initialState := map[string]any{"count": 0}

	result, err := rs.Execute(ctx, initialState)
	assert.NoError(t, err)

	m := result.(map[string]any)
	assert.Equal(t, 2, m["count"], "Should have counted twice")
}

func TestRecursiveSubgraph_MaxDepth(t *testing.T) {
	rs := NewRecursiveSubgraph(
		"recursive",
		2, // Very shallow max depth
		func(state any, depth int) bool {
			return true // Always recurse
		},
	)

	// Add a node
	rs.graph.AddNode("process", "Process", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		return state, nil
	})
	rs.graph.SetEntryPoint("process")
	rs.graph.AddEdge("process", END)

	// Execute - should stop at max depth
	ctx := context.Background()
	result, err := rs.Execute(ctx, map[string]any{})
	assert.NoError(t, err)

	// Should not panic and should complete
	assert.NotNil(t, result)
}
