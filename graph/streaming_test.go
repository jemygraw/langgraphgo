package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStreamingModes(t *testing.T) {
	g := NewStreamingStateGraph()

	// Setup simple graph using map-based state
	g.AddNodeUntyped("A", "A", func(ctx context.Context, state any) (any, error) {
		return map[string]any{"state": "A"}, nil
	})
	g.AddNodeUntyped("B", "B", func(ctx context.Context, state any) (any, error) {
		return map[string]any{"state": "B"}, nil
	})
	g.SetEntryPoint("A")
	g.AddEdge("A", "B")
	g.AddEdge("B", END)

	// Test StreamModeValues
	t.Run("Values", func(t *testing.T) {
		g.SetStreamConfig(StreamConfig{
			BufferSize: 100,
			Mode:       StreamModeValues,
		})

		runnable, err := g.CompileStreaming()
		assert.NoError(t, err)

		res := runnable.Stream(context.Background(), map[string]any{"state": "Start"})

		var events []StreamEvent
		for event := range res.Events {
			events = append(events, event)
		}

		// Expect "graph_step" events
		// A runs -> state map{"state": "A"}
		// B runs -> state map{"state": "B"}

		assert.NotEmpty(t, events)
		for _, e := range events {
			assert.Equal(t, "graph_step", string(e.Event))
		}

		lastEvent := events[len(events)-1]
		// Extract state from map
		lastStateMap, ok := lastEvent.State.(map[string]any)
		assert.True(t, ok, "Expected map state, got %T", lastEvent.State)
		assert.Equal(t, "B", lastStateMap["state"])
	})

	// Test StreamModeUpdates
	t.Run("Updates", func(t *testing.T) {
		g.SetStreamConfig(StreamConfig{
			BufferSize: 100,
			Mode:       StreamModeUpdates,
		})

		runnable, err := g.CompileStreaming()
		assert.NoError(t, err)

		res := runnable.Stream(context.Background(), map[string]any{"state": "Start"})

		var events []StreamEvent
		for event := range res.Events {
			events = append(events, event)
		}

		// Expect ToolEnd events (since nodes are treated as tools)
		// A -> map{"state": "A"}
		// B -> map{"state": "B"}

		foundA := false
		foundB := false
		for _, e := range events {
			if e.Event == NodeEventComplete {
				if stateMap, ok := e.State.(map[string]any); ok {
					if stateMap["state"] == "A" {
						foundA = true
					}
					if stateMap["state"] == "B" {
						foundB = true
					}
				}
			}
		}
		assert.True(t, foundA)
		assert.True(t, foundB)
	})
}
