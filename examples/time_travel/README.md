# Time Travel Example

This example demonstrates the **Time Travel** capabilities of LangGraphGo using `GetState` and `UpdateState`.

## 1. Background

**Time Travel** allows you to inspect and modify the state of a graph execution at any point in time. This is useful for:
- **Debugging**: Inspecting the state at a specific step.
- **Correction**: Fixing incorrect state (e.g., correcting a user input or tool output) and resuming execution.
- **Branching**: Creating alternative execution paths ("what if" scenarios).

## 2. Key Concepts

- **GetState**: Retrieves the current state snapshot of a thread, including values and configuration.
- **UpdateState**: Modifies the state of a thread, creating a new checkpoint. This effectively "forks" the history.

## 3. How It Works

1.  **Initial Run**: We run a simple graph that increments a counter.
2.  **Get State**: We use `GetState` to inspect the final state.
3.  **Update State**: We use `UpdateState` to manually set the counter to a new value (e.g., 10), simulating a correction or branch.
4.  **Resume**: We can then resume execution from this new state (demonstrated here by inspecting the resumed state).

## 4. Code Highlights

### Getting State
```go
snapshot, err := app.GetState(ctx, nil)
fmt.Printf("Current State: %v\n", snapshot.Values)
```

### Updating State
```go
newValues := map[string]interface{}{"count": 10}
// Update state as if we are at "step_1"
newConfig, err := app.UpdateState(ctx, nil, newValues, "step_1")
```

## 5. Running the Example

```bash
go run main.go
```
