package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGraphInterrupt(t *testing.T) {
	g := NewStateGraph[map[string]any]()
	g.AddNode("A", "A", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		state["value"] = state["value"].(string) + "A"
		return state, nil
	})
	g.AddNode("B", "B", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		state["value"] = state["value"].(string) + "B"
		return state, nil
	})
	g.AddNode("C", "C", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		state["value"] = state["value"].(string) + "C"
		return state, nil
	})

	g.SetEntryPoint("A")
	g.AddEdge("A", "B")
	g.AddEdge("B", "C")
	g.AddEdge("C", END)

	runnable, err := g.Compile()
	assert.NoError(t, err)

	// Test InterruptBefore
	t.Run("InterruptBefore", func(t *testing.T) {
		config := &Config{
			InterruptBefore: []string{"B"},
		}
		res, err := runnable.InvokeWithConfig(context.Background(), map[string]any{"value": "Start"}, config)

		assert.Error(t, err)
		var interrupt *GraphInterrupt
		assert.ErrorAs(t, err, &interrupt)
		assert.Equal(t, "B", interrupt.Node)

		// State is stored as map[string]any in the interrupt
		interruptState, ok := interrupt.State.(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "StartA", interruptState["value"])

		// Result should be the state at interruption
		assert.Equal(t, "StartA", res["value"])
	})

	// Test InterruptAfter
	t.Run("InterruptAfter", func(t *testing.T) {
		config := &Config{
			InterruptAfter: []string{"B"},
		}
		res, err := runnable.InvokeWithConfig(context.Background(), map[string]any{"value": "Start"}, config)

		assert.Error(t, err)
		var interrupt *GraphInterrupt
		assert.ErrorAs(t, err, &interrupt)
		assert.Equal(t, "B", interrupt.Node)

		// State is stored as map[string]any in the interrupt
		interruptState, ok := interrupt.State.(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "StartAB", interruptState["value"])

		// Result should be the state at interruption
		assert.Equal(t, "StartAB", res["value"])
	})
}
