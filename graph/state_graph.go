package graph

import (
	"context"
)

// This file contains convenience methods for StateGraphUntyped.
// StateGraphUntyped is a wrapper around StateGraph[map[string]any].
// See graph.go for the wrapper definition.

// AddNodeUntyped adds a node with an untyped function signature.
// This is a convenience method that accepts the legacy function signature
// func(ctx context.Context, state any) (any, error).
// Deprecated: Use AddNode with the typed signature func(ctx context.Context, map[string]any) (map[string]any, error).
func (g *StateGraphUntyped) AddNodeUntyped(name string, description string, fn func(ctx context.Context, state any) (any, error)) {
	// Wrap the untyped function to work with map[string]any
	wrappedFn := func(ctx context.Context, state map[string]any) (map[string]any, error) {
		result, err := fn(ctx, state)
		if err != nil {
			return nil, err
		}
		if resultMap, ok := result.(map[string]any); ok {
			return resultMap, nil
		}
		// If result is not a map, wrap it
		return map[string]any{"value": result}, nil
	}
	g.StateGraphMap.AddNode(name, description, wrappedFn)
}
