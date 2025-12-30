package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVisualization(t *testing.T) {
	g := NewStateGraph[map[string]any]()
	g.AddNode("A", "A", func(ctx context.Context, state map[string]any) (map[string]any, error) { return state, nil })
	g.AddNode("B", "B", func(ctx context.Context, state map[string]any) (map[string]any, error) { return state, nil })
	g.AddNode("C", "C", func(ctx context.Context, state map[string]any) (map[string]any, error) { return state, nil })

	g.SetEntryPoint("A")
	g.AddEdge("A", "B")
	g.AddConditionalEdge("B", func(ctx context.Context, state map[string]any) string { return "C" })
	g.AddEdge("C", END)

	_, err := g.Compile()
	assert.NoError(t, err)

	// Create an exporter for the graph
	exporter := NewExporter(g)

	// Test Mermaid
	mermaid := exporter.DrawMermaid()
	assert.Contains(t, mermaid, "A --> B")
	assert.Contains(t, mermaid, "B -.-> B_condition((?))")
	assert.Contains(t, mermaid, "C --> END")

	// Test Mermaid with Options
	mermaidLR := exporter.DrawMermaidWithOptions(MermaidOptions{Direction: "LR"})
	assert.Contains(t, mermaidLR, "flowchart LR")

	// Test DOT
	dot := exporter.DrawDOT()
	assert.Contains(t, dot, "A -> B")
	assert.Contains(t, dot, "B -> B_condition [style=dashed, label=\"?\"]")

	// Test ASCII
	ascii := exporter.DrawASCII()
	assert.Contains(t, ascii, "A")
	assert.Contains(t, ascii, "B")
	assert.Contains(t, ascii, "(?)")
	// Since C is not directly linked from B in static analysis (it's conditional), it might not appear in ASCII tree under B
	// But B has a conditional edge, so we show (?)
	// C is not reachable via static edges from B, so it won't be shown under B.
	// This is expected behavior for static visualization of dynamic graphs.
}
