package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGraphResume(t *testing.T) {
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

	// Test Resume after InterruptAfter
	t.Run("ResumeAfter", func(t *testing.T) {
		// 1. Run with interrupt after B
		config := &Config{
			InterruptAfter: []string{"B"},
		}
		_, err = runnable.InvokeWithConfig(context.Background(), map[string]any{"value": "Start"}, config)

		assert.Error(t, err)
		var interrupt *GraphInterrupt
		assert.ErrorAs(t, err, &interrupt)
		assert.Equal(t, "B", interrupt.Node)

		interruptState, ok := interrupt.State.(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "StartAB", interruptState["value"])
		assert.Equal(t, []string{"C"}, interrupt.NextNodes)

		// 2. Resume from NextNodes with updated state
		// Simulate user modifying state
		updatedState := map[string]any{"value": interruptState["value"].(string) + "-Modified"}

		resumeConfig := &Config{
			ResumeFrom: interrupt.NextNodes,
		}

		res2, err := runnable.InvokeWithConfig(context.Background(), updatedState, resumeConfig)
		assert.NoError(t, err)
		assert.Equal(t, "StartAB-ModifiedC", res2["value"])
	})

	// Test Resume from InterruptBefore
	t.Run("ResumeBefore", func(t *testing.T) {
		// 1. Run with interrupt before B
		config := &Config{
			InterruptBefore: []string{"B"},
		}
		_, err = runnable.InvokeWithConfig(context.Background(), map[string]any{"value": "Start"}, config)

		assert.Error(t, err)
		var interrupt *GraphInterrupt
		assert.ErrorAs(t, err, &interrupt)
		assert.Equal(t, "B", interrupt.Node)

		interruptState, ok := interrupt.State.(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "StartA", interruptState["value"])

		// 2. Resume from interrupted node
		resumeConfig := &Config{
			ResumeFrom: []string{interrupt.Node},
		}

		res2, err := runnable.InvokeWithConfig(context.Background(), interruptState, resumeConfig)
		assert.NoError(t, err)
		assert.Equal(t, "StartABC", res2["value"])
	})
}
