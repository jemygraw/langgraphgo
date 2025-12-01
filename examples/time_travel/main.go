package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smallnest/langgraphgo/graph"
)

// State is a simple map for this example
type State map[string]interface{}

func main() {
	// 1. Create a Checkpointable Graph
	// We use CheckpointableMessageGraph for convenience, but we'll use MapSchema
	g := graph.NewCheckpointableMessageGraph()

	// Use MapSchema
	schema := graph.NewMapSchema()
	schema.RegisterReducer("count", func(current, new interface{}) (interface{}, error) {
		// Simple overwrite or increment logic could go here
		// For this example, we'll just overwrite
		return new, nil
	})
	g.SetSchema(schema)

	// Define Nodes
	g.AddNode("step_1", func(ctx context.Context, state interface{}) (interface{}, error) {
		fmt.Println("Executing Step 1")
		m := state.(map[string]interface{})
		count := m["count"].(int)
		return map[string]interface{}{"count": count + 1}, nil
	})

	g.AddNode("step_2", func(ctx context.Context, state interface{}) (interface{}, error) {
		fmt.Println("Executing Step 2")
		m := state.(map[string]interface{})
		count := m["count"].(int)
		return map[string]interface{}{"count": count + 1}, nil
	})

	g.SetEntryPoint("step_1")
	g.AddEdge("step_1", "step_2")
	g.AddEdge("step_2", graph.END)

	// Compile
	app, err := g.CompileCheckpointable()
	if err != nil {
		log.Fatal(err)
	}

	// 2. Run Initial Execution
	fmt.Println("--- Initial Run ---")
	initialState := map[string]interface{}{"count": 0}

	// We need to define a ThreadID to track state
	// threadID := "thread-1"
	// config := &graph.Config{
	// 	Configurable: map[string]interface{}{
	// 		"thread_id": threadID,
	// 	},
	// }

	// Note: Currently Invoke on CheckpointableRunnable uses internal executionID if not passed?
	// The implementation of Invoke in CheckpointableRunnable doesn't explicitly use config.Configurable["thread_id"]
	// to set executionID. It generates a new one.

	// So we just run it. The app instance holds the executionID.

	res, err := app.Invoke(context.Background(), initialState)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Initial Result: %v\n", res)

	// 3. Get Current State
	fmt.Println("\n--- Get Current State ---")
	// We pass nil config to use the app's internal executionID
	snapshot, err := app.GetState(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Snapshot Values: %v\n", snapshot.Values)
	fmt.Printf("Snapshot Config: %v\n", snapshot.Config)

	// 4. Time Travel / Update State
	// Let's say we want to "go back" and change the count to 10, effectively forking the history.
	fmt.Println("\n--- Update State (Time Travel) ---")

	newValues := map[string]interface{}{"count": 10}
	// We update the state, pretending we are at "step_1" (or before step_2)
	// This creates a new checkpoint.
	newConfig, err := app.UpdateState(context.Background(), nil, newValues, "step_1")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("New Config: %v\n", newConfig)

	// 5. Continue Execution from new state
	// To "resume", we invoke again. But Invoke() starts a NEW executionID by default in current impl.
	// We need a way to Invoke on an EXISTING executionID or Resume.

	// ResumeFromCheckpoint is available.
	fmt.Println("\n--- Resume from New State ---")
	checkpointID := newConfig.Configurable["checkpoint_id"].(string)

	// Note: ResumeFromCheckpoint implementation currently just loads state.
	// It doesn't re-run the graph from that point automatically in the current simple implementation.
	// But let's see what it returns.
	resumedState, err := app.ResumeFromCheckpoint(context.Background(), checkpointID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Resumed State: %v\n", resumedState)

	// In a full implementation, we would want to run the graph starting from the next node.
	// The current ResumeFromCheckpoint just returns the state.
	// To actually run, we might need to create a new Runnable or use a method that accepts start_node.

	// For this example, we demonstrate that the state was indeed updated and persisted.
}
