package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smallnest/langgraphgo/graph"
)

// Custom Set Reducer
// Merges two values and removes duplicates
func SetReducer(current interface{}, new interface{}) (interface{}, error) {
	// Initialize set with current values
	set := make(map[string]bool)

	// Handle current state
	if current != nil {
		if currentList, ok := current.([]string); ok {
			for _, item := range currentList {
				set[item] = true
			}
		}
	}

	// Merge new values
	if newList, ok := new.([]string); ok {
		for _, item := range newList {
			set[item] = true
		}
	} else if item, ok := new.(string); ok {
		set[item] = true
	}

	// Convert back to slice
	result := make([]string, 0, len(set))
	for item := range set {
		result = append(result, item)
	}

	return result, nil
}

func main() {
	g := graph.NewStateGraph()

	// Define Schema with Custom Reducer
	schema := graph.NewMapSchema()
	schema.RegisterReducer("tags", SetReducer)
	g.SetSchema(schema)

	// Define Nodes
	g.AddNode("start", func(ctx context.Context, state interface{}) (interface{}, error) {
		return map[string]interface{}{
			"tags": []string{"initial"},
		}, nil
	})

	g.AddNode("tagger_a", func(ctx context.Context, state interface{}) (interface{}, error) {
		return map[string]interface{}{
			"tags": []string{"go", "langgraph"},
		}, nil
	})

	g.AddNode("tagger_b", func(ctx context.Context, state interface{}) (interface{}, error) {
		return map[string]interface{}{
			"tags": []string{"ai", "agent", "go"}, // "go" is duplicate
		}, nil
	})

	g.SetEntryPoint("start")
	g.AddEdge("start", "tagger_a")
	g.AddEdge("start", "tagger_b")
	g.AddEdge("tagger_a", graph.END)
	g.AddEdge("tagger_b", graph.END)

	runnable, err := g.Compile()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Custom Reducer Example (Set Merge) ===")
	res, err := runnable.Invoke(context.Background(), map[string]interface{}{
		"tags": []string{},
	})
	if err != nil {
		log.Fatal(err)
	}

	mState := res.(map[string]interface{})
	fmt.Printf("Final Tags: %v\n", mState["tags"])
}
