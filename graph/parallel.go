package graph

import (
	"context"
	"fmt"
	"sync"
)

// ParallelNode represents a set of nodes that can execute in parallel
type ParallelNode struct {
	nodes []Node
	name  string
}

// NewParallelNode creates a new parallel node
func NewParallelNode(name string, nodes ...Node) *ParallelNode {
	return &ParallelNode{
		name:  name,
		nodes: nodes,
	}
}

// Execute runs all nodes in parallel and collects results
func (pn *ParallelNode) Execute(ctx context.Context, state any) (any, error) {
	// Extract actual state if it's wrapped in map[string]any
	var actualState any
	if stateMap, ok := state.(map[string]any); ok {
		// Try to extract from common keys
		if val, exists := stateMap["input"]; exists {
			actualState = val
		} else if val, exists := stateMap["state"]; exists {
			actualState = val
		} else if val, exists := stateMap["value"]; exists {
			actualState = val
		} else {
			actualState = state
		}
	} else {
		actualState = state
	}

	// Create channels for results and errors
	type result struct {
		index int
		value any
		err   error
	}

	results := make(chan result, len(pn.nodes))
	var wg sync.WaitGroup

	// Execute all nodes in parallel
	for i, node := range pn.nodes {
		wg.Add(1)
		go func(idx int, n Node) {
			defer wg.Done()

			// Execute with panic recovery
			defer func() {
				if r := recover(); r != nil {
					results <- result{
						index: idx,
						err:   fmt.Errorf("panic in parallel node %s[%d]: %v", pn.name, idx, r),
					}
				}
			}()

			value, err := n.Function(ctx, actualState)
			results <- result{
				index: idx,
				value: value,
				err:   err,
			}
		}(i, node)
	}

	// Wait for all nodes to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	outputs := make([]any, len(pn.nodes))
	var firstError error

	for res := range results {
		if res.err != nil && firstError == nil {
			firstError = res.err
		}
		outputs[res.index] = res.value
	}

	if firstError != nil {
		return nil, fmt.Errorf("parallel execution failed: %w", firstError)
	}

	// Return collected results
	return outputs, nil
}

// AddParallelNodes adds a set of nodes that execute in parallel
func (g *StateGraphUntyped) AddParallelNodes(groupName string, nodes map[string]func(context.Context, any) (any, error)) {
	// Create parallel node group
	parallelNodes := make([]Node, 0, len(nodes))
	for name, fn := range nodes {
		parallelNodes = append(parallelNodes, Node{
			Name:     name,
			Function: fn,
		})
	}

	// Add as a single parallel node using AddNodeUntyped for compatibility
	parallelNode := NewParallelNode(groupName, parallelNodes...)
	g.AddNodeUntyped(groupName, "Parallel execution group: "+groupName, parallelNode.Execute)
}

// MapReduceNode executes nodes in parallel and reduces results
type MapReduceNode struct {
	name     string
	mapNodes []Node
	reducer  func([]any) (any, error)
}

// NewMapReduceNode creates a new map-reduce node
func NewMapReduceNode(name string, reducer func([]any) (any, error), mapNodes ...Node) *MapReduceNode {
	return &MapReduceNode{
		name:     name,
		mapNodes: mapNodes,
		reducer:  reducer,
	}
}

// Execute runs map nodes in parallel and reduces results
func (mr *MapReduceNode) Execute(ctx context.Context, state any) (any, error) {
	// Extract actual state if it's wrapped in map[string]any
	var actualState any
	if stateMap, ok := state.(map[string]any); ok {
		// Try to extract from common keys
		if val, exists := stateMap["input"]; exists {
			actualState = val
		} else if val, exists := stateMap["state"]; exists {
			actualState = val
		} else if val, exists := stateMap["value"]; exists {
			actualState = val
		} else {
			actualState = state
		}
	} else {
		actualState = state
	}

	// Execute map phase in parallel
	pn := NewParallelNode(mr.name+"_map", mr.mapNodes...)
	results, err := pn.Execute(ctx, actualState)
	if err != nil {
		return nil, fmt.Errorf("map phase failed: %w", err)
	}

	// Execute reduce phase
	if mr.reducer != nil {
		return mr.reducer(results.([]any))
	}

	return results, nil
}

// AddMapReduceNode adds a map-reduce pattern node
func (g *StateGraphUntyped) AddMapReduceNode(
	name string,
	mapFunctions map[string]func(context.Context, any) (any, error),
	reducer func([]any) (any, error),
) {
	// Create map nodes
	mapNodes := make([]Node, 0, len(mapFunctions))
	for nodeName, fn := range mapFunctions {
		mapNodes = append(mapNodes, Node{
			Name:     nodeName,
			Function: fn,
		})
	}

	// Create and add map-reduce node
	mrNode := NewMapReduceNode(name, reducer, mapNodes...)
	g.AddNodeUntyped(name, "Map-reduce node: "+name, mrNode.Execute)
}

// FanOutFanIn creates a fan-out/fan-in pattern
func (g *StateGraphUntyped) FanOutFanIn(
	source string,
	_ []string, // workers parameter kept for API compatibility
	collector string,
	workerFuncs map[string]func(context.Context, any) (any, error),
	collectFunc func([]any) (any, error),
) {
	// Add parallel worker nodes
	g.AddParallelNodes(source+"_workers", workerFuncs)

	// Add collector node
	g.AddNodeUntyped(collector, "Collector node: "+collector, func(_ context.Context, state any) (any, error) {
		// State should be array of results from parallel workers
		// AddNodeUntyped wraps input as map[string]any, so we need to extract the actual results
		var results []any
		if stateMap, ok := state.(map[string]any); ok {
			// Try to get results from common keys
			if val, exists := stateMap["value"]; exists {
				if arr, ok := val.([]any); ok {
					results = arr
				}
			} else if val, exists := stateMap["results"]; exists {
				if arr, ok := val.([]any); ok {
					results = arr
				}
			}
		} else if arr, ok := state.([]any); ok {
			results = arr
		}

		if results != nil {
			return collectFunc(results)
		}
		return nil, fmt.Errorf("invalid state for collector: expected []any, got %T", state)
	})

	// Connect source to workers and workers to collector
	g.AddEdge(source, source+"_workers")
	g.AddEdge(source+"_workers", collector)
}
