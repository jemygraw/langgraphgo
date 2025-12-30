package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuntimeConfiguration(t *testing.T) {
	g := NewStateGraph[map[string]any]()

	// Define a node that reads config from context
	g.AddNode("reader", "reader", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		config := GetConfig(ctx)
		if config == nil {
			return map[string]any{"result": "no config"}, nil
		}

		if val, ok := config.Configurable["model"]; ok {
			return map[string]any{"result": val}, nil
		}
		return map[string]any{"result": "key not found"}, nil
	})

	g.SetEntryPoint("reader")
	g.AddEdge("reader", END)

	runnable, err := g.Compile()
	assert.NoError(t, err)

	// Test with config
	config := &Config{
		Configurable: map[string]any{
			"model": "gpt-4",
		},
	}

	result, err := runnable.InvokeWithConfig(context.Background(), nil, config)
	assert.NoError(t, err)
	assert.Equal(t, "gpt-4", result["result"])

	// Test without config
	result, err = runnable.Invoke(context.Background(), nil)
	assert.NoError(t, err)
	assert.Equal(t, "no config", result["result"])
}

func TestStateGraph_RuntimeConfiguration(t *testing.T) {
	g := NewStateGraph[map[string]any]()

	g.AddNode("reader", "reader", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		config := GetConfig(ctx)
		if config == nil {
			return map[string]any{"result": "no config"}, nil
		}

		if val, ok := config.Configurable["api_key"]; ok {
			return map[string]any{"result": val}, nil
		}
		return map[string]any{"result": "key not found"}, nil
	})

	g.SetEntryPoint("reader")
	g.AddEdge("reader", END)

	runnable, err := g.Compile()
	assert.NoError(t, err)

	config := &Config{
		Configurable: map[string]any{
			"api_key": "secret-123",
		},
	}

	result, err := runnable.InvokeWithConfig(context.Background(), nil, config)
	assert.NoError(t, err)
	assert.Equal(t, "secret-123", result["result"])
}
