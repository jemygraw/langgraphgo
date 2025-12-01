# Dynamic Interrupt Example

This example demonstrates the **Dynamic Interrupt** feature in LangGraphGo, similar to Python LangGraph's `interrupt` function.

## 1. Background

Dynamic interrupts allow a node to pause execution at runtime and wait for external input (e.g., from a human). Unlike static `InterruptBefore` configuration, this allows the decision to interrupt to be made inside the node's logic.

## 2. Key Concepts

- **`graph.Interrupt(ctx, value)`**: A function that can be called inside a node.
  - If called for the first time, it stops execution and returns a `GraphInterrupt` error containing the `value`.
  - If called during resumption (with a `ResumeValue` provided), it returns the `ResumeValue` and continues execution.
- **`Config.ResumeValue`**: The value provided when resuming execution, which will be returned by `graph.Interrupt`.

## 3. How It Works

1.  **Initial Run**: The graph runs until it hits `graph.Interrupt("What is your name?")`.
2.  **Interruption**: The `Invoke` method returns a `GraphInterrupt` error. We catch this error and see the query.
3.  **User Input**: We simulate getting input ("Alice") from the user.
4.  **Resume**: We call `Invoke` again, but this time passing `ResumeValue: "Alice"` in the config.
5.  **Replay**: The node runs again. `graph.Interrupt` sees the resume value and returns "Alice" immediately. The node continues to finish.

## 4. Code Highlights

### Inside the Node
```go
name, err := graph.Interrupt(ctx, "What is your name?")
if err != nil {
    return nil, err // Propagate the interrupt
}
// Use the input
return fmt.Sprintf("Hello, %v!", name), nil
```

### Handling the Interrupt
```go
_, err := runnable.Invoke(ctx, nil)
if interrupt, ok := err.(*graph.GraphInterrupt); ok {
    // Get user input...
    config := &graph.Config{ResumeValue: userInput}
    // Resume
    runnable.InvokeWithConfig(ctx, nil, config)
}
```

## 5. Running the Example

```bash
go run main.go
```
