package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mapSchemaAdapterForAny adapts MapSchema for use with StateGraph[any]
type mapSchemaAdapterForAny struct {
	*MapSchema
}

func (m *mapSchemaAdapterForAny) Init() any {
	return m.MapSchema.Init()
}

func (m *mapSchemaAdapterForAny) Update(current, new any) (any, error) {
	return m.MapSchema.Update(current, new)
}

func TestCommandGoto(t *testing.T) {
	// Use any type to allow returning Command
	g := NewStateGraph[any]()

	// Define schema
	schema := NewMapSchema()
	schema.RegisterReducer("count", func(curr, new any) (any, error) {
		if curr == nil {
			return new, nil
		}
		return curr.(int) + new.(int), nil
	})
	g.SetSchema(&mapSchemaAdapterForAny{MapSchema: schema})

	// Node A: Returns Command to update count and go to C (skipping B)
	g.AddNode("A", "A", func(ctx context.Context, state any) (any, error) {
		return &Command{
			Update: map[string]any{"count": 1},
			Goto:   "C",
		}, nil
	})

	// Node B: Should be skipped
	g.AddNode("B", "B", func(ctx context.Context, state any) (any, error) {
		return map[string]any{"count": 10}, nil
	})

	// Node C: Final node
	g.AddNode("C", "C", func(ctx context.Context, state any) (any, error) {
		return map[string]any{"count": 100}, nil
	})

	g.SetEntryPoint("A")
	g.AddEdge("A", "B") // Static edge A -> B
	g.AddEdge("B", "C")
	g.AddEdge("C", END)

	runnable, err := g.Compile()
	assert.NoError(t, err)

	res, err := runnable.Invoke(context.Background(), map[string]any{"count": 0})
	assert.NoError(t, err)

	mRes, ok := res.(map[string]any)
	assert.True(t, ok)

	// Expected: 0 + 1 (A) + 100 (C) = 101. B is skipped.
	assert.Equal(t, 101, mRes["count"])
}
